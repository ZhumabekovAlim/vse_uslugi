package main

import (
	"context"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"log"
	"naimuBack/internal/models"
	"net/http"
	"strconv"
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
		// Получаем access token из заголовка
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "Authorization header missing", http.StatusUnauthorized)
			return
		}

		// Удаляем префикс "Bearer "
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")

		// Проверяем access token
		claims := &models.Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				log.Println("Unexpected signing method:", token.Header["alg"])
				return nil, http.ErrNoLocation
			}
			return []byte("asdadsadadaadsasd"), nil
		})

		fmt.Println("middleware claims:", token)
		// Если access token недействителен, проверяем refresh token
		if err != nil || !token.Valid {
			refreshToken := r.Header.Get("Refresh-Token")
			if refreshToken == "" {
				http.Error(w, "Refresh token missing", http.StatusUnauthorized)
				return
			}

			// Получаем refresh token из базы данных
			session, err := app.userRepo.GetSession(r.Context(), strconv.Itoa(int(claims.UserID)))
			if err != nil {
				log.Printf("Failed to fetch session for user %v: %v", claims.UserID, err)
				http.Error(w, "Invalid session", http.StatusUnauthorized)
				return
			}

			// Проверяем совпадение токена из базы и его срок действия
			if session.RefreshToken != refreshToken || session.ExpiresAt.Before(time.Now()) {
				http.Error(w, "Invalid or expired refresh token", http.StatusUnauthorized)
				return
			}

			// Создаем новый access token, если refresh token действителен
			newAccessToken, err := generateAccessToken(int(claims.UserID), claims.Role)
			if err != nil {
				log.Printf("Error generating new access token: %v", err)
				http.Error(w, "Error generating new access token", http.StatusInternalServerError)
				return
			}
			fmt.Println(newAccessToken)
			// Устанавливаем новый access token в заголовке ответа
			w.Header().Set("Authorization", "Bearer "+newAccessToken)
			log.Printf("New access token issued for user: %v", claims.UserID)

			claims = &models.Claims{UserID: claims.UserID, Role: claims.Role} // Обновляем данные пользователя для проверки ролей
		}

		// Проверка ролей
		if requiredRole == "admin" && claims.Role != "admin" {
			http.Error(w, "Forbidden: only admins allowed", http.StatusForbidden)
			return
		}
		if requiredRole == "client" && claims.Role != "client" && claims.Role != "admin" {
			http.Error(w, "Forbidden: only clients or admins allowed", http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), "user_id", int(claims.UserID))
		ctx = context.WithValue(ctx, "role", claims.Role)

		// Передаем управление следующему обработчику
		next.ServeHTTP(w, r)
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
