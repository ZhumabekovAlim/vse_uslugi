package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"naimuBack/internal/models"
	"naimuBack/internal/services"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type RentCategoryHandler struct {
	Service *services.RentCategoryService
}

func (h *RentCategoryHandler) GetCategoryByID(w http.ResponseWriter, r *http.Request) {
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

func (h *RentCategoryHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
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

func (h *RentCategoryHandler) GetAllCategories(w http.ResponseWriter, r *http.Request) {
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

func (h *RentCategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "missing category name", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "missing image file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	timestamp := time.Now().UnixNano()
	ext := filepath.Ext(header.Filename)
	imageName := fmt.Sprintf("rent_category_image_%d%s", timestamp, ext)
	savePath := filepath.Join("cmd/uploads/rent_categories", imageName)
	publicURL := fmt.Sprintf("/images/rent_categories/%s", imageName)

	saveDir := "cmd/uploads/rent_categories"
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		http.Error(w, "failed to create image directory", http.StatusInternalServerError)
		return
	}

	out, err := os.Create(savePath)
	if err != nil {
		http.Error(w, "cannot save image", http.StatusInternalServerError)
		return
	}
	defer out.Close()
	if _, err = io.Copy(out, file); err != nil {
		http.Error(w, "failed to write image", http.StatusInternalServerError)
		return
	}

	category := models.RentCategory{
		Name:      name,
		ImagePath: publicURL,
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

func (h *RentCategoryHandler) ServeImage(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get(":filename")
	if filename == "" {
		http.Error(w, "filename is required", http.StatusBadRequest)
		return
	}
	imagePath := filepath.Join("cmd/uploads/rent_categories", filename)
	log.Println("Serving image from:", imagePath)
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		http.Error(w, "image not found", http.StatusNotFound)
		return
	}

	ext := filepath.Ext(imagePath)
	switch ext {
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	http.ServeFile(w, r, imagePath)
}

func (h *RentCategoryHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "failed to parse multipart form", http.StatusBadRequest)
		return
	}

	idStr := r.URL.Query().Get(":id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid category ID", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	var imagePath string

	file, header, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		timestamp := time.Now().UnixNano()
		ext := filepath.Ext(header.Filename)
		imageName := fmt.Sprintf("rent_category_image_%d%s", timestamp, ext)
		savePath := filepath.Join("cmd/uploads/rent_categories", imageName)
		imagePath = fmt.Sprintf("/images/rent_categories/%s", imageName)

		saveDir := "cmd/uploads/rent_categories"
		if err := os.MkdirAll(saveDir, 0755); err != nil {
			http.Error(w, "failed to create image directory", http.StatusInternalServerError)
			return
		}
		out, err := os.Create(savePath)
		if err != nil {
			http.Error(w, "cannot save image", http.StatusInternalServerError)
			return
		}
		defer out.Close()
		if _, err = io.Copy(out, file); err != nil {
			http.Error(w, "failed to write image", http.StatusInternalServerError)
			return
		}
	}

	category := models.RentCategory{
		ID:        id,
		Name:      name,
		ImagePath: imagePath,
	}

	updated, err := h.Service.UpdateCategory(r.Context(), category)
	if err != nil {
		http.Error(w, "failed to update category", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}
