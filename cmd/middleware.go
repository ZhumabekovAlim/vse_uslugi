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
		if app.userRepo == nil {
			app.serverError(w, fmt.Errorf("users repo is not initialized"))
			return
		}

		// 1) Достаём access token из Authorization: Bearer xxx
		auth := r.Header.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, "Authorization header missing", http.StatusUnauthorized)
			return
		}
		accessToken := strings.TrimPrefix(auth, "Bearer ")

		// 2) Пытаемся валидировать access token
		accessClaims := &models.Claims{}
		at, atErr := jwt.ParseWithClaims(accessToken, accessClaims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte("asdadsadadaadsasd"), nil
		})

		var effUserID uint
		var effRole string

		if atErr == nil && at != nil && at.Valid {
			// Access valid — берём данные из него
			effUserID = accessClaims.UserID
			effRole = accessClaims.Role
		} else {
			// 3) Access недействителен — проверяем refresh
			refreshToken := r.Header.Get("Refresh-Token")
			if refreshToken == "" {
				http.Error(w, "Refresh token missing", http.StatusUnauthorized)
				return
			}

			// 3.1) Парсим refresh-токен и берём user_id прямо из его клеймов (НЕ из битого access)
			refreshClaims := &models.Claims{}
			rt, rtErr := jwt.ParseWithClaims(refreshToken, refreshClaims, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte("asdadsadadaadsasd"), nil
			})
			if rtErr != nil || rt == nil || !rt.Valid {
				http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
				return
			}

			// 3.2) Достаём сессию из БД. ВАЖНО: лучше искать по самому refreshToken.
			// Если твой репозиторий ждёт userID, можно сделать GetSessionByUserID(ctx, userID).
			// Но надёжнее иметь метод: GetSessionByToken(ctx, refreshToken).
			// Здесь оставлю как у тебя — через user_id (поменяй при необходимости на ByToken).
			userIDStr := strconv.Itoa(int(refreshClaims.UserID))
			session, err := app.userRepo.GetSession(r.Context(), userIDStr)
			if err != nil || session == (models.Session{}) {
				log.Printf("Failed to fetch session for user %v: %v", refreshClaims.UserID, err)
				http.Error(w, "Invalid session", http.StatusUnauthorized)
				return
			}

			if session.RefreshToken != refreshToken {
				http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
				return
			}
			if session.ExpiresAt.Before(time.Now()) {
				http.Error(w, "Expired refresh token", http.StatusUnauthorized)
				return
			}

			// 3.3) Генерим новый access token на основе данных refresh-клеймов
			newAccessToken, err := generateAccessToken(int(refreshClaims.UserID), refreshClaims.Role)
			if err != nil {
				log.Printf("Error generating new access token: %v", err)
				http.Error(w, "Error generating new access token", http.StatusInternalServerError)
				return
			}
			// Отдаём новый access-токен в заголовке ответа
			w.Header().Set("Authorization", "Bearer "+newAccessToken)

			effUserID = refreshClaims.UserID
			effRole = refreshClaims.Role
			log.Printf("New access token issued for user: %v", effUserID)
		}

		// 4) Проверка ролей (на эффективной роли)
		switch requiredRole {
		case "admin":
			if effRole != "admin" {
				http.Error(w, "Forbidden: only admins allowed", http.StatusForbidden)
				return
			}
		case "client":
			if effRole != "client" && effRole != "admin" {
				http.Error(w, "Forbidden: only clients or admins allowed", http.StatusForbidden)
				return
			}
		case "worker":
			if effRole != "worker" && effRole != "admin" {
				http.Error(w, "Forbidden: only workers or admins allowed", http.StatusForbidden)
				return
			}
		}

		// 5) Прокидываем user_id и role в контекст и НЕ забываем r.WithContext(ctx)
		ctx := context.WithValue(r.Context(), "user_id", int(effUserID))
		ctx = context.WithValue(ctx, "role", effRole)
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
