package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
	"naimuBack/internal/services"
)

type RentHandler struct {
	Service     *services.RentService
	ChatService *services.ChatService
}

func (h *RentHandler) GetRentByID(w http.ResponseWriter, r *http.Request) {
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

	userID := 0

	tokenString := r.Header.Get("Authorization")
	if strings.HasPrefix(tokenString, "Bearer ") {
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		claims := &models.Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(signingKey), nil
		})
		if err == nil && token.Valid {
			userID = int(claims.UserID)

		}
	}

	rent, err := h.Service.GetRentByID(r.Context(), id, userID)
	if err != nil {
		http.Error(w, "Service not found", http.StatusNotFound)
		return
	}

	chatObject := getChatObject(r.Context(), h.ChatService, userID, "rent", id)

	w.Header().Set("Content-Type", "application/json")
	if err := respondWithChatObject(w, rent, chatObject); err != nil {
		log.Printf("Failed to encode rent with chat object: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (h *RentHandler) DeleteRent(w http.ResponseWriter, r *http.Request) {
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

	err = h.Service.DeleteRent(r.Context(), id)
	if err != nil {
		if errors.Is(err, repositories.ErrWorkNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *RentHandler) ArchiveRent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RentID  int `json:"rent_id"`
		Archive int `json:"archive"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if err := h.Service.ArchiveRent(r.Context(), req.RentID, req.Archive == 1); err != nil {
		if errors.Is(err, services.ErrNoActiveSubscriptionRent) {
			http.Error(w, "subscription required", http.StatusForbidden)
			return
		}
		http.Error(w, "Failed to archive rent", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *RentHandler) GetRents(w http.ResponseWriter, r *http.Request) {
	// Чтение query-параметров
	categories := parseIntArrayRent(r.URL.Query().Get("categories"))
	subcategories := parseStringArrayRent(r.URL.Query().Get("subcategories"))
	ratings := parseFloatArrayRent(r.URL.Query().Get("ratings"))

	priceFrom, _ := strconv.ParseFloat(r.URL.Query().Get("price_from"), 64)
	priceTo, _ := strconv.ParseFloat(r.URL.Query().Get("price_to"), 64)
	sortOption, _ := strconv.Atoi(r.URL.Query().Get("sort"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	// Сборка запроса
	filter := models.RentFilterRequest{
		Categories:    categories,
		Subcategories: subcategories,
		PriceFrom:     priceFrom,
		PriceTo:       priceTo,
		Ratings:       ratings,
		SortOption:    sortOption,
		Page:          page,
		Limit:         limit,
	}

	cityID, _ := strconv.Atoi(r.URL.Query().Get("city_id"))
	tokenString := r.Header.Get("Authorization")
	if strings.HasPrefix(tokenString, "Bearer ") {
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		claims := &models.Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(signingKey), nil
		})
		if err == nil && token.Valid && cityID == 0 {
			cityID = claims.CityID
		}
	}

	result, err := h.Service.GetFilteredRents(r.Context(), filter, 0, cityID)
	if err != nil {
		log.Printf("GetServices error: %v", err)
		http.Error(w, "Failed to fetch services", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *RentHandler) GetRentsAdmin(w http.ResponseWriter, r *http.Request) {
	h.GetRents(w, r)
}
func parseIntArrayRent(input string) []int {
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

func parseFloatArrayRent(input string) []float64 {
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

func parseStringArrayRent(input string) []string {
	if input == "" {
		return nil
	}
	return strings.Split(input, ",")
}

func (h *RentHandler) GetRentsSorted(w http.ResponseWriter, r *http.Request) {
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

	filter := models.RentFilterRequest{
		SortOption: sortOption,
		Page:       page,
		Limit:      limit,
	}

	result, err := h.Service.GetFilteredRents(r.Context(), filter, userID, 0)
	if err != nil {
		http.Error(w, "Failed to get services", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *RentHandler) GetRentsByUserID(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	rents, err := h.Service.GetRentsByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to fetch services", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rents)
}

func (h *RentHandler) GetFilteredRentsPost(w http.ResponseWriter, r *http.Request) {
	var req models.FilterRentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	tokenString := r.Header.Get("Authorization")
	if strings.HasPrefix(tokenString, "Bearer ") {
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		claims := &models.Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(signingKey), nil
		})
		if err == nil && token.Valid {
			if req.CityID == 0 {
				req.CityID = claims.CityID
			}
		}
	}

	rents, err := h.Service.GetFilteredRentsPost(r.Context(), req)
	if err != nil {
		log.Printf("GetServicesPost error: %v", err)
		http.Error(w, "Failed to fetch services", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"rents": rents,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *RentHandler) GetRentsByStatusAndUserID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID int    `json:"user_id"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	rents, err := h.Service.GetRentsByStatusAndUserID(r.Context(), req.UserID, req.Status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rents)
}

func (h *RentHandler) ServeRentsImage(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get(":filename")
	if filename == "" {
		http.Error(w, "filename is required", http.StatusBadRequest)
		return
	}
	imagePath := filepath.Join("cmd/uploads/rents", filename)

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

func (h *RentHandler) ServeRentVideo(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get(":filename")
	if filename == "" {
		http.Error(w, "filename is required", http.StatusBadRequest)
		return
	}

	videoPath := filepath.Join("cmd/uploads/rents/videos", filename)

	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		http.Error(w, "video not found", http.StatusNotFound)
		return
	}

	ext := strings.ToLower(filepath.Ext(videoPath))
	var contentType string
	switch ext {
	case ".mp4":
		contentType = "video/mp4"
	case ".mov":
		contentType = "video/quicktime"
	case ".webm":
		contentType = "video/webm"
	case ".mkv":
		contentType = "video/x-matroska"
	default:
		contentType = "application/octet-stream"
	}

	w.Header().Set("Content-Type", contentType)
	http.ServeFile(w, r, videoPath)
}

func (h *RentHandler) CreateRent(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(32 << 20) // 32MB
	if err != nil {
		http.Error(w, "Invalid multipart form", http.StatusBadRequest)
		return
	}

	var service models.Rent
	service.Name = r.FormValue("name")
	service.Address = r.FormValue("address")
	if priceStr := strings.TrimSpace(r.FormValue("price")); priceStr != "" {
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			http.Error(w, "Invalid price", http.StatusBadRequest)
			return
		}
		service.Price = &price
	}
	if priceToStr := strings.TrimSpace(r.FormValue("price_to")); priceToStr != "" {
		priceTo, err := strconv.ParseFloat(priceToStr, 64)
		if err != nil {
			http.Error(w, "Invalid price_to", http.StatusBadRequest)
			return
		}
		service.PriceTo = &priceTo
	}
	service.Negotiable = parseBool(r.FormValue("negotiable"))
	service.HidePhone = parseBool(r.FormValue("hide_phone"))
	if userIDStr := r.FormValue("user_id"); userIDStr != "" {
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			http.Error(w, "Invalid user_id", http.StatusBadRequest)
			return
		}
		service.UserID = userID
	}
	if service.UserID == 0 {
		if ctxUserID, ok := r.Context().Value("user_id").(int); ok && ctxUserID != 0 {
			service.UserID = ctxUserID
		}
	}
	if service.UserID == 0 {
		http.Error(w, "Missing user_id", http.StatusBadRequest)
		return
	}
	service.Description = r.FormValue("description")
	service.CategoryID, _ = strconv.Atoi(r.FormValue("category_id"))
	service.SubcategoryID, _ = strconv.Atoi(r.FormValue("subcategory_id"))
	service.WorkTimeFrom = r.FormValue("work_time_from")
	service.WorkTimeTo = r.FormValue("work_time_to")
	service.Condition = r.FormValue("condition")
	service.Delivery = parseBool(r.FormValue("delivery"))
	service.AvgRating, _ = strconv.ParseFloat(r.FormValue("avg_rating"), 64)
	service.RentType = r.FormValue("rent_type")
	service.Deposit = r.FormValue("deposit")
	service.Latitude = r.FormValue("latitude")
	service.Longitude = r.FormValue("longitude")
	service.Top = r.FormValue("top")
	service.Status = normalizeListingStatus(r.FormValue("status"))
	service.CreatedAt = time.Now()

	// Сохраняем изображения
	saveDir := "cmd/uploads/rents"
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		http.Error(w, "Failed to create image directory", http.StatusInternalServerError)
		return
	}

	files := r.MultipartForm.File["images"]
	var imageInfosRent []models.ImageRent

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, "Failed to open image", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		timestamp := time.Now().UnixNano()
		ext := filepath.Ext(fileHeader.Filename)
		imageName := fmt.Sprintf("rent_image_%d%s", timestamp, ext)
		savePath := filepath.Join(saveDir, imageName)
		publicURL := fmt.Sprintf("/images/rents/%s", imageName)

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

		imageInfosRent = append(imageInfosRent, models.ImageRent{
			Name: fileHeader.Filename,
			Path: publicURL,
			Type: fileHeader.Header.Get("Content-Type"),
		})
	}

	service.Images = imageInfosRent

	videoDir := "cmd/uploads/rents/videos"
	if err := os.MkdirAll(videoDir, 0755); err != nil {
		http.Error(w, "Failed to create video directory", http.StatusInternalServerError)
		return
	}

	videoHeaders := collectImageFiles(r.MultipartForm, "videos", "videos[]")
	var videoInfos []models.Video

	if parsedVideos, ok, err := gatherImagesFromForm[models.Video](r.MultipartForm, "videos", "videos[]"); err != nil {
		http.Error(w, "Invalid videos payload", http.StatusBadRequest)
		return
	} else if ok {
		videoInfos = append(videoInfos, parsedVideos...)
	}

	for _, fileHeader := range videoHeaders {
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, "Failed to open video", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		timestamp := time.Now().UnixNano()
		ext := filepath.Ext(fileHeader.Filename)
		videoName := fmt.Sprintf("rent_video_%d%s", timestamp, ext)
		savePath := filepath.Join(videoDir, videoName)
		publicURL := fmt.Sprintf("/videos/rents/%s", videoName)

		dst, err := os.Create(savePath)
		if err != nil {
			http.Error(w, "Cannot save video", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, "Failed to write video", http.StatusInternalServerError)
			return
		}

		videoInfos = append(videoInfos, models.Video{
			Name: fileHeader.Filename,
			Path: publicURL,
			Type: fileHeader.Header.Get("Content-Type"),
		})
	}

	if parsedLinks, ok, err := gatherImagesFromForm[models.Video](r.MultipartForm, "video_links", "video_links[]"); err != nil {
		http.Error(w, "Invalid video links payload", http.StatusBadRequest)
		return
	} else if ok {
		videoInfos = append(videoInfos, parsedLinks...)
	}

	service.Videos = videoInfos

	createdService, err := h.Service.CreateRent(r.Context(), service)
	if err != nil {
		if errors.Is(err, services.ErrNoActiveSubscriptionRent) {

			http.Error(w, err.Error(), http.StatusForbidden)

			return
		}
		log.Printf("Failed to create service: %v", err)
		http.Error(w, "Failed to create service", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(createdService)
}

func (h *RentHandler) UpdateRent(w http.ResponseWriter, r *http.Request) {
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

	existingService, err := h.Service.GetRentByID(r.Context(), id, 0)
	if err != nil {
		http.Error(w, "Service not found", http.StatusNotFound)
		return
	}

	service := existingService

	deletedImageKeys, _, err := gatherStringsFromForm(r.MultipartForm, "delete_images", "delete_images[]", "removed_images", "removed_images[]")
	if err != nil {
		http.Error(w, "Invalid delete images payload", http.StatusBadRequest)
		return
	}

	if fileKeys, ok, err := gatherStringsFromFormFiles(r.MultipartForm, "delete_images", "delete_images[]", "removed_images", "removed_images[]"); err != nil {
		http.Error(w, "Invalid delete images payload", http.StatusBadRequest)
		return
	} else if ok {
		deletedImageKeys = append(deletedImageKeys, fileKeys...)
	}

	deletedVideoKeys, _, err := gatherStringsFromForm(r.MultipartForm, "delete_videos", "delete_videos[]", "removed_videos", "removed_videos[]")
	if err != nil {
		http.Error(w, "Invalid delete videos payload", http.StatusBadRequest)
		return
	}

	if fileKeys, ok, err := gatherStringsFromFormFiles(r.MultipartForm, "delete_videos", "delete_videos[]", "removed_videos", "removed_videos[]"); err != nil {
		http.Error(w, "Invalid delete videos payload", http.StatusBadRequest)
		return
	} else if ok {
		deletedVideoKeys = append(deletedVideoKeys, fileKeys...)
	}

	if _, ok := r.MultipartForm.Value["name"]; ok {
		service.Name = r.FormValue("name")
	}
	if _, ok := r.MultipartForm.Value["address"]; ok {
		service.Address = r.FormValue("address")
	}
	if v, ok := r.MultipartForm.Value["price"]; ok {
		priceStr := strings.TrimSpace(v[0])
		if priceStr != "" {
			price, err := strconv.ParseFloat(priceStr, 64)
			if err != nil {
				http.Error(w, "Invalid price", http.StatusBadRequest)
				return
			}
			service.Price = &price
		} else {
			service.Price = nil
		}
	}
	if v, ok := r.MultipartForm.Value["price_to"]; ok {
		priceStr := strings.TrimSpace(v[0])
		if priceStr != "" {
			priceTo, err := strconv.ParseFloat(priceStr, 64)
			if err != nil {
				http.Error(w, "Invalid price_to", http.StatusBadRequest)
				return
			}
			service.PriceTo = &priceTo
		} else {
			service.PriceTo = nil
		}
	}
	if v, ok := r.MultipartForm.Value["user_id"]; ok {
		service.UserID, _ = strconv.Atoi(v[0])
	}
	if v, ok := r.MultipartForm.Value["negotiable"]; ok {
		service.Negotiable = parseBool(v[0])
	}
	if v, ok := r.MultipartForm.Value["hide_phone"]; ok {
		service.HidePhone = parseBool(v[0])
	}
	if _, ok := r.MultipartForm.Value["description"]; ok {
		service.Description = r.FormValue("description")
	}
	if _, ok := r.MultipartForm.Value["condition"]; ok {
		service.Condition = r.FormValue("condition")
	}
	if v, ok := r.MultipartForm.Value["delivery"]; ok {
		service.Delivery = parseBool(v[0])
	}
	if _, ok := r.MultipartForm.Value["work_time_from"]; ok {
		service.WorkTimeFrom = r.FormValue("work_time_from")
	}
	if _, ok := r.MultipartForm.Value["work_time_to"]; ok {
		service.WorkTimeTo = r.FormValue("work_time_to")
	}
	if v, ok := r.MultipartForm.Value["category_id"]; ok {
		service.CategoryID, _ = strconv.Atoi(v[0])
	}
	if v, ok := r.MultipartForm.Value["subcategory_id"]; ok {
		service.SubcategoryID, _ = strconv.Atoi(v[0])
	}
	if v, ok := r.MultipartForm.Value["avg_rating"]; ok {
		service.AvgRating, _ = strconv.ParseFloat(v[0], 64)
	}
	if _, ok := r.MultipartForm.Value["rent_type"]; ok {
		service.RentType = r.FormValue("rent_type")
	}
	if _, ok := r.MultipartForm.Value["deposit"]; ok {
		service.Deposit = r.FormValue("deposit")
	}
	if _, ok := r.MultipartForm.Value["latitude"]; ok {
		service.Latitude = r.FormValue("latitude")
	}
	if _, ok := r.MultipartForm.Value["longitude"]; ok {
		service.Longitude = r.FormValue("longitude")
	}
	if _, ok := r.MultipartForm.Value["top"]; ok {
		service.Top = r.FormValue("top")
	}
	if _, ok := r.MultipartForm.Value["liked"]; ok {
		service.Liked = r.FormValue("liked") == "true"
	}
	if _, ok := r.MultipartForm.Value["status"]; ok {
		service.Status = normalizeListingStatus(r.FormValue("status"))
	}

	images := service.Images

	if parsedImages, ok, err := gatherImagesFromForm[models.ImageRent](r.MultipartForm, "images", "images[]"); err != nil {
		http.Error(w, "Invalid images payload", http.StatusBadRequest)
		return
	} else if ok {
		images = parsedImages
	} else if parsedExisting, okExisting, err := gatherImagesFromForm[models.ImageRent](r.MultipartForm, "existing_images", "existing_images[]"); err != nil {
		http.Error(w, "Invalid images payload", http.StatusBadRequest)
		return
	} else if okExisting {
		images = parsedExisting
	}

	if parsedLinks, ok, err := gatherImagesFromForm[models.ImageRent](r.MultipartForm, "image_links", "image_links[]"); err != nil {
		http.Error(w, "Invalid image links payload", http.StatusBadRequest)
		return
	} else if ok {
		images = append(images, parsedLinks...)
	}

	saveDir := "cmd/uploads/rents"
	err = os.MkdirAll(saveDir, 0755)
	if err != nil {
		http.Error(w, "Failed to create image directory", http.StatusInternalServerError)
		return
	}

	fileHeaders := collectImageFiles(r.MultipartForm, "images", "images[]")
	if len(fileHeaders) > 0 {
		var uploaded []models.ImageRent
		for _, fileHeader := range fileHeaders {
			file, err := fileHeader.Open()
			if err != nil {
				http.Error(w, "Failed to open image", http.StatusInternalServerError)
				return
			}
			defer file.Close()

			timestamp := time.Now().UnixNano()
			ext := filepath.Ext(fileHeader.Filename)
			imageName := fmt.Sprintf("rent_image_%d%s", timestamp, ext)
			savePath := filepath.Join(saveDir, imageName)
			publicURL := fmt.Sprintf("/images/rents/%s", imageName)

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

			uploaded = append(uploaded, models.ImageRent{
				Name: fileHeader.Filename,
				Path: publicURL,
				Type: fileHeader.Header.Get("Content-Type"),
			})
		}
		images = append(images, uploaded...)
	}

	if len(deletedImageKeys) > 0 {
		var removedImages []models.ImageRent
		images, removedImages = filterRentImages(images, deletedImageKeys)
		if err := removeRentImagesFromDisk(removedImages); err != nil {
			log.Printf("Failed to remove rent images: %v", err)
		}
	}

	service.Images = images

	videos := service.Videos

	if parsedVideos, ok, err := gatherImagesFromForm[models.Video](r.MultipartForm, "videos", "videos[]"); err != nil {
		http.Error(w, "Invalid videos payload", http.StatusBadRequest)
		return
	} else if ok {
		videos = parsedVideos
	} else if parsedExisting, okExisting, err := gatherImagesFromForm[models.Video](r.MultipartForm, "existing_videos", "existing_videos[]"); err != nil {
		http.Error(w, "Invalid videos payload", http.StatusBadRequest)
		return
	} else if okExisting {
		videos = parsedExisting
	}

	if parsedLinks, ok, err := gatherImagesFromForm[models.Video](r.MultipartForm, "video_links", "video_links[]"); err != nil {
		http.Error(w, "Invalid video links payload", http.StatusBadRequest)
		return
	} else if ok {
		videos = append(videos, parsedLinks...)
	}

	videoDir := "cmd/uploads/rents/videos"
	if err := os.MkdirAll(videoDir, 0755); err != nil {
		http.Error(w, "Failed to create video directory", http.StatusInternalServerError)
		return
	}

	videoHeaders := collectImageFiles(r.MultipartForm, "videos", "videos[]")
	if len(videoHeaders) > 0 {
		var uploaded []models.Video
		for _, fileHeader := range videoHeaders {
			file, err := fileHeader.Open()
			if err != nil {
				http.Error(w, "Failed to open video", http.StatusInternalServerError)
				return
			}
			defer file.Close()

			timestamp := time.Now().UnixNano()
			ext := filepath.Ext(fileHeader.Filename)
			videoName := fmt.Sprintf("rent_video_%d%s", timestamp, ext)
			savePath := filepath.Join(videoDir, videoName)
			publicURL := fmt.Sprintf("/videos/rents/%s", videoName)

			dst, err := os.Create(savePath)
			if err != nil {
				http.Error(w, "Cannot save video", http.StatusInternalServerError)
				return
			}
			defer dst.Close()

			if _, err := io.Copy(dst, file); err != nil {
				http.Error(w, "Failed to write video", http.StatusInternalServerError)
				return
			}

			uploaded = append(uploaded, models.Video{
				Name: fileHeader.Filename,
				Path: publicURL,
				Type: fileHeader.Header.Get("Content-Type"),
			})
		}
		videos = append(videos, uploaded...)
	}

	if len(deletedVideoKeys) > 0 {
		var removedVideos []models.Video
		videos, removedVideos = filterServiceVideos(videos, deletedVideoKeys)
		if err := removeRentVideosFromDisk(removedVideos); err != nil {
			log.Printf("Failed to remove rent videos: %v", err)
		}
	}

	service.Videos = videos

	now := time.Now()
	service.UpdatedAt = &now

	updatedService, err := h.Service.UpdateRent(r.Context(), service)
	if err != nil {
		if errors.Is(err, services.ErrNoActiveSubscription) {

			http.Error(w, err.Error(), http.StatusForbidden)

			return
		}
		log.Printf("Failed to update service: %v", err)
		http.Error(w, "Failed to update service", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedService)
}

func filterRentImages(images []models.ImageRent, deleteKeys []string) ([]models.ImageRent, []models.ImageRent) {
	removalSet := buildRemovalSet(deleteKeys)
	if len(removalSet) == 0 {
		return images, nil
	}

	var (
		kept    []models.ImageRent
		removed []models.ImageRent
	)

	for _, img := range images {
		if shouldRemoveMedia(img.Path, img.Name, removalSet) {
			removed = append(removed, img)
			continue
		}
		kept = append(kept, img)
	}

	return kept, removed
}

func removeRentImagesFromDisk(images []models.ImageRent) error {
	for _, img := range images {
		if img.Type == "link" {
			continue
		}
		if err := removeMediaFile("cmd/uploads/rents", "/images/rents/", img.Path); err != nil {
			return err
		}
	}
	return nil
}

func removeRentVideosFromDisk(videos []models.Video) error {
	for _, video := range videos {
		if video.Type == "link" {
			continue
		}
		if err := removeMediaFile("cmd/uploads/rents/videos", "/videos/rents/", video.Path); err != nil {
			return err
		}
	}
	return nil
}

func (h *RentHandler) GetFilteredRentsWithLikes(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	var req models.FilterRentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	tokenString := r.Header.Get("Authorization")
	if strings.HasPrefix(tokenString, "Bearer ") {
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		claims := &models.Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(signingKey), nil
		})
		if err == nil && token.Valid {
			if req.CityID == 0 {
				req.CityID = claims.CityID
			}
		}
	}

	rents, err := h.Service.GetFilteredRentsWithLikes(r.Context(), req, userID)
	if err != nil {
		log.Printf("GetServicesPost error: %v", err)
		http.Error(w, "Failed to fetch services", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"rents": rents,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *RentHandler) GetRentByRentIDAndUserID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rentIDStr := r.URL.Query().Get(":rent_id")
	if rentIDStr == "" {
		http.Error(w, "service ID is required", http.StatusBadRequest)
		return
	}

	rentID, err := strconv.Atoi(rentIDStr)
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
	rent, err := h.Service.GetRentByRentIDAndUserID(ctx, rentID, userID)
	if err != nil {
		if err.Error() == "service not found" {
			http.Error(w, "service not found", http.StatusNotFound)
		} else {
			log.Printf("[ERROR] Failed to get service: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	chatObject := getChatObject(r.Context(), h.ChatService, userID, "rent", rentID)

	// Успешный ответ
	w.Header().Set("Content-Type", "application/json")
	if err := respondWithChatObject(w, rent, chatObject); err != nil {
		log.Printf("[ERROR] Failed to encode response: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
