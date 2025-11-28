package main

import (
	"context"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"naimuBack/internal/models"
	"net/http"
	"strings"
	"time"
)

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("X-Frame-Options", "deny")
		next.ServeHTTP(w, r)
	})
}

func makeResponseJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func (app *application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.infoLog.Printf("%s - %s %s %s", r.RemoteAddr, r.Proto, r.Method, r.URL.RequestURI())
		next.ServeHTTP(w, r)
	})
}

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverError(w, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (app *application) JWTMiddleware(next http.Handler, requiredRole string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1) Получаем access token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Authorization header missing or invalid", http.StatusUnauthorized)
			return
		}
		accessToken := strings.TrimPrefix(authHeader, "Bearer ")

		claims := &models.Claims{}
		token, err := jwt.ParseWithClaims(accessToken, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte("asdadsadadaadsasd"), nil
		})

		if err != nil || !token.Valid {
			// 2) Access невалиден — проверяем Refresh-Token
			refreshToken := r.Header.Get("Refresh-Token")
			if refreshToken == "" {
				http.Error(w, "Refresh token missing", http.StatusUnauthorized)
				return
			}

			// 3) Ищем сессию по refresh токену
			session, err := app.userRepo.GetSessionByToken(r.Context(), refreshToken)
			if err != nil || session == (models.Session{}) {
				http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
				return
			}

			// 4) Сверка токена и срока действия
			if session.RefreshToken != refreshToken {
				http.Error(w, "Refresh token mismatch", http.StatusUnauthorized)
				return
			}
			if session.ExpiresAt.Before(time.Now()) {
				http.Error(w, "Expired refresh token", http.StatusUnauthorized)
				return
			}

			// 5) Генерируем новый access token
			newAccessToken, err := generateAccessToken(session.UserID, session.Role)
			if err != nil {
				http.Error(w, "Error generating new access token", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Authorization", "Bearer "+newAccessToken)

			// Обновляем данные claims
			claims.UserID = uint(session.UserID)
			claims.Role = session.Role
		}

		// 6) Проверка ролей
		switch requiredRole {
		case "admin":
			if claims.Role != "admin" {
				http.Error(w, "Forbidden: only admins allowed", http.StatusForbidden)
				return
			}
		case "client":
			if claims.Role != "client" && claims.Role != "admin" {
				http.Error(w, "Forbidden: only clients or admins allowed", http.StatusForbidden)
				return
			}
		case "worker":
			if claims.Role != "worker" && claims.Role != "admin" && claims.Role != "business_worker" {
				http.Error(w, "Forbidden: only workers or admins allowed", http.StatusForbidden)
				return
			}
		case "business":
			if claims.Role != "business" && claims.Role != "admin" {
				http.Error(w, "Forbidden: only business or admins allowed", http.StatusForbidden)
				return
			}
		case "business_worker":
			if claims.Role != "business_worker" && claims.Role != "admin" {
				http.Error(w, "Forbidden: only business workers or admins allowed", http.StatusForbidden)
				return
			}
		}

		// 7) Прокидываем user_id и role в контекст
		ctx := context.WithValue(r.Context(), "user_id", int(claims.UserID))
		ctx = context.WithValue(ctx, "role", claims.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Функция для генерации нового access token
func generateAccessToken(userID int, role string) (string, error) {
	claims := &models.Claims{
		UserID: uint(userID),
		Role:   role,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(20 * time.Hour).Unix(), // Устанавливаем срок годности access token
			IssuedAt:  time.Now().Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte("asdadsadadaadsasd"))
}
