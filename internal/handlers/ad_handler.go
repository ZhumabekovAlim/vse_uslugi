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

type AdHandler struct {
	Service *services.AdService
}

func (h *AdHandler) GetAdByID(w http.ResponseWriter, r *http.Request) {
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

	ad, err := h.Service.GetAdByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Service not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ad)
}

func (h *AdHandler) DeleteAd(w http.ResponseWriter, r *http.Request) {
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

	err = h.Service.DeleteAd(r.Context(), id)
	if err != nil {
		if errors.Is(err, repositories.ErrAdNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *AdHandler) GetAd(w http.ResponseWriter, r *http.Request) {
	// Чтение query-параметров
	categories := parseIntArrayAd(r.URL.Query().Get("categories"))
	subcategories := parseStringArrayAd(r.URL.Query().Get("subcategories"))
	ratings := parseFloatArrayAd(r.URL.Query().Get("ratings"))

	priceFrom, _ := strconv.ParseFloat(r.URL.Query().Get("price_from"), 64)
	priceTo, _ := strconv.ParseFloat(r.URL.Query().Get("price_to"), 64)
	sortOption, _ := strconv.Atoi(r.URL.Query().Get("sort"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	// Сборка запроса
	filter := models.AdFilterRequest{
		Categories:    categories,
		Subcategories: subcategories,
		PriceFrom:     priceFrom,
		PriceTo:       priceTo,
		Ratings:       ratings,
		SortOption:    sortOption,
		Page:          page,
		Limit:         limit,
	}

	result, err := h.Service.GetFilteredAd(r.Context(), filter, 0)
	if err != nil {
		log.Printf("GetServices error: %v", err)
		http.Error(w, "Failed to fetch services", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
func parseIntArrayAd(input string) []int {
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

func parseFloatArrayAd(input string) []float64 {
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

func parseStringArrayAd(input string) []string {
	if input == "" {
		return nil
	}
	return strings.Split(input, ",")
}

func (h *AdHandler) GetAdSorted(w http.ResponseWriter, r *http.Request) {
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

	filter := models.AdFilterRequest{
		SortOption: sortOption,
		Page:       page,
		Limit:      limit,
	}

	result, err := h.Service.GetFilteredAd(r.Context(), filter, userID)
	if err != nil {
		http.Error(w, "Failed to get services", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *AdHandler) GetAdByUserID(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	ads, err := h.Service.GetAdByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to fetch services", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ads)
}

func (h *AdHandler) GetFilteredAdPost(w http.ResponseWriter, r *http.Request) {
	var req models.FilterAdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ads, err := h.Service.GetFilteredAdPost(r.Context(), req)
	if err != nil {
		log.Printf("GetServicesPost error: %v", err)
		http.Error(w, "Failed to fetch services", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"ads": ads,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AdHandler) GetAdByStatusAndUserID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID int    `json:"user_id"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ads, err := h.Service.GetAdByStatusAndUserID(r.Context(), req.UserID, req.Status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ads)
}

func (h *AdHandler) ServeAdImage(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get(":filename")
	if filename == "" {
		http.Error(w, "filename is required", http.StatusBadRequest)
		return
	}
	imagePath := filepath.Join("cmd/uploads/ad", filename)

	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		http.Error(w, "image not found", http.StatusNotFound)
		return
	}

	// Content-Type по расширению
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

func (h *AdHandler) CreateAd(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(32 << 20) // 32MB
	if err != nil {
		http.Error(w, "Invalid multipart form", http.StatusBadRequest)
		return
	}

	var service models.Ad
	service.Name = r.FormValue("name")
	service.Address = r.FormValue("address")
	service.Price, _ = strconv.ParseFloat(r.FormValue("price"), 64)
	service.UserID, _ = strconv.Atoi(r.FormValue("user_id"))
	service.Description = r.FormValue("description")
	service.CategoryID, _ = strconv.Atoi(r.FormValue("category_id"))
	service.SubcategoryID, _ = strconv.Atoi(r.FormValue("subcategory_id"))
	service.AvgRating, _ = strconv.ParseFloat(r.FormValue("avg_rating"), 64)
	service.Top = r.FormValue("top")
	service.Status = r.FormValue("status")
	service.CreatedAt = time.Now()

	// Сохраняем изображения
	saveDir := "cmd/uploads/ad"
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		http.Error(w, "Failed to create image directory", http.StatusInternalServerError)
		return
	}

	files := r.MultipartForm.File["images"]
	var imageInfos []models.ImageAd

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, "Failed to open image", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		timestamp := time.Now().UnixNano()
		ext := filepath.Ext(fileHeader.Filename)
		imageName := fmt.Sprintf("ad_image_%d%s", timestamp, ext)
		savePath := filepath.Join(saveDir, imageName)
		publicURL := fmt.Sprintf("/images/ad/%s", imageName)

		dst, err := os.Create(savePath)
		if err != nil {
			http.Error(w, "Cannot save image", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, "Failed to write image", http.StatusInternalServerError)
			return
		}

		imageInfos = append(imageInfos, models.ImageAd{
			Name: fileHeader.Filename,
			Path: publicURL,
			Type: fileHeader.Header.Get("Content-Type"),
		})
	}

	service.Images = imageInfos

	createdService, err := h.Service.CreateAd(r.Context(), service)
	if err != nil {
		log.Printf("Failed to create service: %v", err)
		http.Error(w, "Failed to create service", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createdService)
}

func (h *AdHandler) UpdateAd(w http.ResponseWriter, r *http.Request) {
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

	err = r.ParseMultipartForm(32 << 20) // до 32MB
	if err != nil {
		http.Error(w, "Invalid multipart form", http.StatusBadRequest)
		return
	}

	var service models.Ad
	service.ID = id
	service.Name = r.FormValue("name")
	service.Address = r.FormValue("address")
	service.Price, _ = strconv.ParseFloat(r.FormValue("price"), 64)
	service.UserID, _ = strconv.Atoi(r.FormValue("user_id"))
	service.Description = r.FormValue("description")
	service.CategoryID, _ = strconv.Atoi(r.FormValue("category_id"))
	service.SubcategoryID, _ = strconv.Atoi(r.FormValue("subcategory_id"))
	service.AvgRating, _ = strconv.ParseFloat(r.FormValue("avg_rating"), 64)
	service.Top = r.FormValue("top")
	service.Liked = r.FormValue("liked") == "true"
	service.Status = r.FormValue("status")
	now := time.Now()
	service.UpdatedAt = &now

	// Обработка изображений
	saveDir := "cmd/uploads/ad"
	err = os.MkdirAll(saveDir, 0755)
	if err != nil {
		http.Error(w, "Failed to create image directory", http.StatusInternalServerError)
		return
	}

	files := r.MultipartForm.File["images"]
	var imageInfos []models.ImageAd

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, "Failed to open image", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		timestamp := time.Now().UnixNano()
		ext := filepath.Ext(fileHeader.Filename)
		imageName := fmt.Sprintf("ad_image_%d%s", timestamp, ext)
		savePath := filepath.Join(saveDir, imageName)
		publicURL := fmt.Sprintf("/images/ad/%s", imageName)

		dst, err := os.Create(savePath)
		if err != nil {
			http.Error(w, "Cannot save image", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, "Failed to write image", http.StatusInternalServerError)
			return
		}

		imageInfos = append(imageInfos, models.ImageAd{
			Name: fileHeader.Filename,
			Path: publicURL,
			Type: fileHeader.Header.Get("Content-Type"),
		})
	}

	service.Images = imageInfos

	updatedService, err := h.Service.UpdateAd(r.Context(), service)
	if err != nil {
		log.Printf("Failed to update service: %v", err)
		http.Error(w, "Failed to update service", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedService)
}

func (h *AdHandler) GetFilteredAdWithLikes(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	var req models.FilterAdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ads, err := h.Service.GetFilteredAdWithLikes(r.Context(), req, userID)
	if err != nil {
		log.Printf("GetServicesPost error: %v", err)
		http.Error(w, "Failed to fetch services", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"ads": ads,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AdHandler) GetAdByAdIDAndUserID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	adIDStr := r.URL.Query().Get(":ad_id")
	if adIDStr == "" {
		http.Error(w, "service ID is required", http.StatusBadRequest)
		return
	}

	adID, err := strconv.Atoi(adIDStr)
	if err != nil {
		http.Error(w, "invalid service ID", http.StatusBadRequest)
		return
	}
	userIDStr := r.URL.Query().Get(":user_id")

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "unauthorized or missing user ID", http.StatusUnauthorized)
		return
	}

	// Получение сервиса
	ad, err := h.Service.GetAdByAdIDAndUserID(ctx, adID, userID)
	if err != nil {
		if err.Error() == "service not found" {
			http.Error(w, "service not found", http.StatusNotFound)
		} else {
			log.Printf("[ERROR] Failed to get service: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Успешный ответ
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ad); err != nil {
		log.Printf("[ERROR] Failed to encode response: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
