package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"naimuBack/internal/models"
	"naimuBack/internal/services"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type CategoryHandler struct {
	Service *services.CategoryService
}

func (h *CategoryHandler) GetCategoryByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get(":id")
	if idStr == "" {
		http.Error(w, "Missing category ID", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	category, err := h.Service.GetCategoryByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrCategoryNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(category)
}

func (h *CategoryHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get(":id")
	if idStr == "" {
		http.Error(w, "Missing category ID", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	var category models.Category
	if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	category.ID = id

	updatedCategory, err := h.Service.UpdateCategory(r.Context(), category)
	if err != nil {
		if errors.Is(err, models.ErrCategoryNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedCategory)
}

func (h *CategoryHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get(":id")
	if idStr == "" {
		http.Error(w, "Missing category ID", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	err = h.Service.DeleteCategory(r.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrCategoryNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CategoryHandler) GetAllCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.Service.GetAllCategories(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(categories); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

//func (h *CategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
//	err := r.ParseMultipartForm(10 << 20)
//	if err != nil {
//		http.Error(w, "could not parse form", http.StatusBadRequest)
//		return
//	}
//
//	name := r.FormValue("name")
//	if name == "" {
//		http.Error(w, "name is required", http.StatusBadRequest)
//		return
//	}
//
//	subcategoryIDsStr := r.Form["subcategory_ids"]
//	subcategoryIDs := []int{}
//	for _, idStr := range subcategoryIDsStr {
//		id, err := strconv.Atoi(idStr)
//		if err == nil {
//			subcategoryIDs = append(subcategoryIDs, id)
//		}
//	}
//
//	// default min_price to 0
//	category := models.Category{
//		Name:      name,
//		MinPrice:  0,
//		CreatedAt: time.Now(),
//		UpdatedAt: time.Now(),
//	}
//
//	// handle image
//	file, header, err := r.FormFile("image")
//	if err == nil {
//		defer file.Close()
//
//		uploadDir := "cmd/uploads/categories"
//		os.MkdirAll(uploadDir, 0755)
//
//		safeFileName := fmt.Sprintf("category_image_%d%s", time.Now().UnixNano(), filepath.Ext(header.Filename))
//		relativePath := filepath.Join("categories", safeFileName)
//		fullPath := filepath.Join("uploads", relativePath)
//
//		tmpFile, err := os.Create(fullPath)
//		if err != nil {
//			http.Error(w, "cannot save file", http.StatusInternalServerError)
//			return
//		}
//		defer tmpFile.Close()
//
//		_, err = io.Copy(tmpFile, file)
//		if err != nil {
//			http.Error(w, "copy failed", http.StatusInternalServerError)
//			return
//		}
//
//		// сохранение относительного пути, который будет доступен через "/static/..."
//		category.ImagePath = "/static/" + relativePath
//
//	}
//
//	createdCategory, err := h.Service.CreateCategory(r.Context(), category)
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//		return
//	}
//
//	w.WriteHeader(http.StatusCreated)
//	w.Header().Set("Content-Type", "application/json")
//	json.NewEncoder(w).Encode(createdCategory)
//}

func (h *CategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10MB
	if err != nil {
		http.Error(w, "cannot parse form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	// --- Обработка изображения
	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "could not get image file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	timestamp := time.Now().Unix()
	imageName := fmt.Sprintf("category_image_%d%s", timestamp, filepath.Ext(header.Filename))
	uploadDir := "cmd/uploads/categories"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		http.Error(w, "failed to create directory", http.StatusInternalServerError)
		return
	}
	imagePath := filepath.Join(uploadDir, imageName)

	out, err := os.Create(imagePath)
	if err != nil {
		http.Error(w, "failed to save image", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		http.Error(w, "failed to save image content", http.StatusInternalServerError)
		return
	}

	// --- Создание категории
	category := models.Category{
		Name:      name,
		ImagePath: imagePath,
		MinPrice:  0,
	}

	createdCategory, err := h.Service.CreateCategory(r.Context(), category)
	if err != nil {
		http.Error(w, "failed to create category", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdCategory)
}

func (h *CategoryHandler) ServeCategoryImage(w http.ResponseWriter, r *http.Request) {
	imageName := r.URL.Query().Get("filename")
	if imageName == "" {
		http.Error(w, "missing filename", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join("cmd/uploads/categories", filepath.Base(imageName))

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "image not found", http.StatusNotFound)
		return
	}

	ext := filepath.Ext(filePath)
	var contentType string
	switch ext {
	case ".png":
		contentType = "image/png"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".webp":
		contentType = "image/webp"
	default:
		contentType = "application/octet-stream"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "inline; filename=\""+filepath.Base(filePath)+"\"")
	http.ServeFile(w, r, filePath)
}
