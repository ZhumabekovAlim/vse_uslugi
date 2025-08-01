package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"naimuBack/internal/models"
	"naimuBack/internal/services"
)

// ErrUserNotFound is returned when a user is not found.
var ErrUserNotFound = errors.New("user not found")

type UserHandler struct {
	Service *services.UserService
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	createdUser, err := h.Service.CreateUser(r.Context(), user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdUser)
}

func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get(":id")
	if idStr == "" {
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := h.Service.GetUserByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get(":id")
	if idStr == "" {
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	user.ID = id

	updatedUser, err := h.Service.UpdateUser(r.Context(), user)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedUser)
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get(":id")
	if idStr == "" {
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	err = h.Service.DeleteUser(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	phone := r.URL.Query().Get("phone")
	role := r.URL.Query().Get("role")

	if phone != "" {
		user, err := h.Service.GetUserByPhone(r.Context(), phone)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
		return
	}

	if role != "" {
		users, err := h.Service.GetUsersByRole(r.Context(), role)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(users); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
		return
	}

	users, err := h.Service.GetAllUsers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(users); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *UserHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	var req struct {
		models.User
		VerificationCode string `json:"verification_code"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.Service.SignUp(r.Context(), req.User, req.VerificationCode)
	if err != nil {
		if errors.Is(err, models.ErrInvalidVerificationCode) {
			http.Error(w, "Неверный код подтверждения", http.StatusUnauthorized)
			return
		}
		log.Printf("SignUp error: %v", err)
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *UserHandler) SignIn(w http.ResponseWriter, r *http.Request) {
	var req models.SignInRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.Service.SignIn(r.Context(), req.Name, req.Phone, req.Email, req.Password)
	if err != nil {
		log.Printf("error: %v", err)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *UserHandler) UserLogOut(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	clientIDStr := r.URL.Query().Get(":id")
	clientID, err := strconv.Atoi(clientIDStr)
	if err != nil {
		http.Error(w, "Invalid client ID", http.StatusBadRequest)
		return
	}

	err = h.Service.UserLogOut(ctx, clientID)
	if err != nil {
		log.Printf("Error getting users: %v", err)
		http.Error(w, "Failed to get users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

}

func (h *UserHandler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	var req models.UpdatePasswordRequest

	// Парсим JSON тело запроса
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Проверяем, что все необходимые данные присутствуют
	if req.UserID == 0 || req.OldPassword == "" || req.NewPassword == "" {
		http.Error(w, "User ID, old password, and new password are required", http.StatusBadRequest)
		return
	}

	// Вызываем сервис для обновления пароля
	err = h.Service.UpdatePassword(r.Context(), req.UserID, req.OldPassword, req.NewPassword)
	if err != nil {
		if errors.Is(err, models.ErrInvalidPassword) {
			http.Error(w, "Invalid old password", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	// Возвращаем успешный статус
	w.WriteHeader(http.StatusOK)
}

func (h *UserHandler) ChangeNumber(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Phone string `json:"phone"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Phone == "" {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.Service.ChangeNumber(r.Context(), req.Phone)
	if err != nil {
		if errors.Is(err, models.ErrDuplicatePhone) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		log.Printf("ChangeNumber error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *UserHandler) ChangeEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Email == "" {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.Service.ChangeEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, models.ErrDuplicateEmail) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		log.Printf("ChangeEmail error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *UserHandler) ChangeCityForUser(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from URL (e.g. /users/5/city)
	idStr := r.URL.Query().Get(":id")
	if idStr == "" {
		http.Error(w, "Missing user ID in URL", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(idStr)
	if err != nil || userID <= 0 {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req struct {
		CityID int `json:"city_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CityID == 0 {
		http.Error(w, "city_id is required", http.StatusBadRequest)
		return
	}

	// Call service
	err = h.Service.ChangeCityForUser(r.Context(), userID, req.CityID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to update city", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "City updated successfully"})
}

func (h *UserHandler) CheckUserDuplicate(w http.ResponseWriter, r *http.Request) {
	var req models.User
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверное тело запроса", http.StatusBadRequest)
		return
	}

	err := h.Service.CheckUserDuplicate(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict) // 409 — если найден дубликат
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Код отправлен на номер",
	})
}

func (h *UserHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req models.PasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	if err := h.Service.SendResetCode(r.Context(), req.Email); err != nil {
		http.Error(w, "Ошибка отправки кода: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Код отправлен на email"}`))
}

func (h *UserHandler) VerifyResetCode(w http.ResponseWriter, r *http.Request) {
	var req models.VerifyResetCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный формат", http.StatusBadRequest)
		return
	}

	valid, err := h.Service.VerifyResetCode(r.Context(), req.Email, req.Code)
	if err != nil || !valid {
		http.Error(w, "Неверный код", http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Код подтвержден"}`))
}

func (h *UserHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req models.NewPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный формат", http.StatusBadRequest)
		return
	}

	if err := h.Service.ResetPassword(r.Context(), req.Email, req.NewPassword); err != nil {
		http.Error(w, "Ошибка смены пароля", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Пароль успешно изменен"}`))
}

func (h *UserHandler) UpdateToWorker(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get(":id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	var user models.User
	user.ID = id
	user.Role = "worker"

	if yearsStr := r.FormValue("years_of_exp"); yearsStr != "" {
		if y, err := strconv.Atoi(yearsStr); err == nil {
			user.YearsOfExp = &y
		}
	}

	user.Skills = r.FormValue("skills")

	categoryIDs := r.Form["category_ids"]
	for _, c := range categoryIDs {
		if cid, err := strconv.Atoi(c); err == nil {
			user.Categories = append(user.Categories, models.Category{ID: cid})
		}
	}

	// === Сохраняем файл подтверждения ===
	saveDir := "cmd/uploads/docs"
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		http.Error(w, "Failed to create doc directory", http.StatusInternalServerError)
		return
	}

	file, handler, err := r.FormFile("doc_of_proof")
	if err == nil {
		defer file.Close()

		timestamp := time.Now().UnixNano()
		ext := filepath.Ext(handler.Filename)
		newName := fmt.Sprintf("doc_of_proof_%d%s", timestamp, ext)
		savePath := filepath.Join(saveDir, newName)
		publicURL := fmt.Sprintf("/docs/%s", newName)

		dst, err := os.Create(savePath)
		if err != nil {
			http.Error(w, "Cannot save document", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, "Failed to write document", http.StatusInternalServerError)
			return
		}

		user.DocOfProof = &publicURL
	}

	updated, err := h.Service.UpdateToWorker(r.Context(), user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func (h *UserHandler) ServeProofDocument(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get(":filename")
	if filename == "" {
		http.Error(w, "filename is required", http.StatusBadRequest)
		return
	}

	docPath := filepath.Join("cmd/uploads/docs", filename)
	if _, err := os.Stat(docPath); os.IsNotExist(err) {
		http.Error(w, "document not found", http.StatusNotFound)
		return
	}

	ext := filepath.Ext(docPath)
	switch ext {
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".pdf":
		w.Header().Set("Content-Type", "application/pdf")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	http.ServeFile(w, r, docPath)
}
