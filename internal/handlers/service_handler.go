package handlers

import (
	_ "context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
	"naimuBack/internal/services"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	_ "strings"
	"time"
)

type ServiceHandler struct {
	Service *services.ServiceService
}

func (h *ServiceHandler) CreateService(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(32 << 20) // 32MB
	if err != nil {
		http.Error(w, "Invalid multipart form", http.StatusBadRequest)
		return
	}

	var service models.Service
	service.Name = r.FormValue("name")
	service.Address = r.FormValue("address")
	service.Price, _ = strconv.ParseFloat(r.FormValue("price"), 64)
	service.UserID, _ = strconv.Atoi(r.FormValue("user_id"))
	service.Description = r.FormValue("description")
	service.CategoryID, _ = strconv.Atoi(r.FormValue("category_id"))
	service.SubcategoryID, _ = strconv.Atoi(r.FormValue("subcategory_id"))
	service.AvgRating, _ = strconv.ParseFloat(r.FormValue("avg_rating"), 64)
	service.Top = r.FormValue("top")
	service.CreatedAt = time.Now()

	uploadDir := "./uploads"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
			http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
			return
		}
	}

	files := r.MultipartForm.File["images"]
	var imageInfos []models.Image

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, "Failed to open file", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), fileHeader.Filename)
		filePath := filepath.Join(uploadDir, filename)

		dst, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "Cannot save file", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}

		imageInfos = append(imageInfos, models.Image{
			Name: fileHeader.Filename,
			Path: "/uploads/" + filename,
			Type: fileHeader.Header.Get("Content-Type"),
		})
	}

	service.Images = imageInfos

	createdService, err := h.Service.CreateService(r.Context(), service)
	if err != nil {
		http.Error(w, "Failed to create service", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createdService)
}

//func (h *ServiceHandler) CreateService(w http.ResponseWriter, r *http.Request) {
//	err := r.ParseMultipartForm(32 << 20) // 32 MB
//	if err != nil {
//		http.Error(w, "Invalid multipart form", http.StatusBadRequest)
//		return
//	}
//
//	var service models.Service
//	service.Name = r.FormValue("name")
//	service.Address = r.FormValue("address")
//	service.Price, _ = strconv.ParseFloat(r.FormValue("price"), 64)
//	service.UserID, _ = strconv.Atoi(r.FormValue("user_id"))
//	service.Description = r.FormValue("description")
//	service.CategoryID, _ = strconv.Atoi(r.FormValue("category_id"))
//	service.SubcategoryID, _ = strconv.Atoi(r.FormValue("subcategory_id"))
//	service.CreatedAt = time.Now()
//
//	files := r.MultipartForm.File["images"]
//	var imagePaths []string
//
//	uploadDir := "./uploads"
//	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
//		err = os.MkdirAll(uploadDir, os.ModePerm)
//		if err != nil {
//			http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
//			return
//		}
//	}
//
//	for _, fileHeader := range files {
//		file, err := fileHeader.Open()
//		if err != nil {
//			http.Error(w, "Failed to open uploaded file", http.StatusInternalServerError)
//			return
//		}
//		defer file.Close()
//
//		// Генерируем уникальное имя файла
//		filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), fileHeader.Filename)
//		filePath := filepath.Join(uploadDir, filename)
//
//		dst, err := os.Create(filePath)
//		if err != nil {
//			http.Error(w, "Cannot save file", http.StatusInternalServerError)
//			return
//		}
//		defer dst.Close()
//
//		if _, err := io.Copy(dst, file); err != nil {
//			http.Error(w, "Failed to save uploaded file", http.StatusInternalServerError)
//			return
//		}
//
//		imagePaths = append(imagePaths, "/uploads/"+filename) // путь для фронта
//	}
//
//	service.Images = imagePaths
//
//	created, err := h.Service.CreateService(r.Context(), service)
//	if err != nil {
//		log.Printf("Failed to create service error: %v", err)
//		http.Error(w, "Failed to create service", http.StatusInternalServerError)
//		return
//	}
//
//	w.Header().Set("Content-Type", "application/json")
//	w.WriteHeader(http.StatusCreated)
//	json.NewEncoder(w).Encode(created)
//}

func (h *ServiceHandler) GetServiceByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get(":id")
	if idStr == "" {
		http.Error(w, "Missing service ID", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid service ID", http.StatusBadRequest)
		return
	}

	service, err := h.Service.GetServiceByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Service not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(service)
}

func (h *ServiceHandler) UpdateService(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get(":id")
	if idStr == "" {
		http.Error(w, "Missing service ID", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid service ID", http.StatusBadRequest)
		return
	}

	var service models.Service
	if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	service.ID = id

	updatedService, err := h.Service.UpdateService(r.Context(), service)
	if err != nil {
		if errors.Is(err, repositories.ErrServiceNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedService)
}

func (h *ServiceHandler) DeleteService(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get(":id")
	if idStr == "" {
		http.Error(w, "Missing service ID", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid service ID", http.StatusBadRequest)
		return
	}

	err = h.Service.DeleteService(r.Context(), id)
	if err != nil {
		if errors.Is(err, repositories.ErrServiceNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ServiceHandler) GetServices(w http.ResponseWriter, r *http.Request) {
	// Чтение query-параметров
	categories := parseIntArray(r.URL.Query().Get("categories"))
	subcategories := parseStringArray(r.URL.Query().Get("subcategories"))
	ratings := parseFloatArray(r.URL.Query().Get("ratings"))

	priceFrom, _ := strconv.ParseFloat(r.URL.Query().Get("price_from"), 64)
	priceTo, _ := strconv.ParseFloat(r.URL.Query().Get("price_to"), 64)
	sortOption, _ := strconv.Atoi(r.URL.Query().Get("sort"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	// Сборка запроса
	filter := models.ServiceFilterRequest{
		Categories:    categories,
		Subcategories: subcategories,
		PriceFrom:     priceFrom,
		PriceTo:       priceTo,
		Ratings:       ratings,
		SortOption:    sortOption,
		Page:          page,
		Limit:         limit,
	}

	result, err := h.Service.GetFilteredServices(r.Context(), filter, 0)
	if err != nil {
		log.Printf("GetServices error: %v", err)
		http.Error(w, "Failed to fetch services", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
func parseIntArray(input string) []int {
	if input == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	var result []int
	for _, part := range parts {
		if val, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
			result = append(result, val)
		}
	}
	return result
}

func parseFloatArray(input string) []float64 {
	if input == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	var result []float64
	for _, part := range parts {
		if val, err := strconv.ParseFloat(strings.TrimSpace(part), 64); err == nil {
			result = append(result, val)
		}
	}
	return result
}

func parseStringArray(input string) []string {
	if input == "" {
		return nil
	}
	return strings.Split(input, ",")
}

func (h *ServiceHandler) GetServicesSorted(w http.ResponseWriter, r *http.Request) {
	sortStr := r.URL.Query().Get(":type")
	sortOption, err := strconv.Atoi(sortStr)
	if err != nil || sortOption < 1 || sortOption > 3 {
		http.Error(w, "Invalid sort option", http.StatusBadRequest)
		return
	}

	userStr := r.URL.Query().Get(":user_id")
	userID, err := strconv.Atoi(userStr)
	if err != nil {
		http.Error(w, "Invalid sort option", http.StatusBadRequest)
		return
	}

	// Пагинация
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	filter := models.ServiceFilterRequest{
		SortOption: sortOption,
		Page:       page,
		Limit:      limit,
	}

	result, err := h.Service.GetFilteredServices(r.Context(), filter, userID)
	if err != nil {
		http.Error(w, "Failed to get services", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *ServiceHandler) GetServiceByUserID(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	services, err := h.Service.GetServicesByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to fetch services", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(services)
}

func (h *ServiceHandler) GetFilteredServicesPost(w http.ResponseWriter, r *http.Request) {
	var req models.FilterServicesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	services, err := h.Service.GetFilteredServicesPost(r.Context(), req)
	if err != nil {
		http.Error(w, "Failed to fetch services", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"services": services,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
