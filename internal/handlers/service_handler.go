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

type ServiceHandler struct {
	Service *services.ServiceService
}

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

	service, err := h.Service.GetServiceByID(r.Context(), id, userID)
	if err != nil {
		http.Error(w, "Service not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(service)
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

func (h *ServiceHandler) ArchiveService(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ServiceID int `json:"service_id"`
		Archive   int `json:"archive"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if err := h.Service.ArchiveService(r.Context(), req.ServiceID, req.Archive == 1); err != nil {
		http.Error(w, "Failed to archive service", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
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

	result, err := h.Service.GetFilteredServices(r.Context(), filter, 0, cityID)
	if err != nil {
		log.Printf("GetServices error: %v", err)
		http.Error(w, "Failed to fetch services", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *ServiceHandler) GetServicesAdmin(w http.ResponseWriter, r *http.Request) {
	h.GetServices(w, r)
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

	result, err := h.Service.GetFilteredServices(r.Context(), filter, userID, 0)
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

	services, err := h.Service.GetFilteredServicesPost(r.Context(), req)
	if err != nil {
		log.Printf("GetServicesPost error: %v", err)
		http.Error(w, "Failed to fetch services", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"services": services,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *ServiceHandler) GetServicesByStatusAndUserID(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID int    `json:"user_id"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	services, err := h.Service.GetServicesByStatusAndUserID(r.Context(), req.UserID, req.Status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(services)
}

func (h *ServiceHandler) ServeServiceImage(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get(":filename")
	if filename == "" {
		http.Error(w, "filename is required", http.StatusBadRequest)
		return
	}
	imagePath := filepath.Join("cmd/uploads/services", filename)

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

func (h *ServiceHandler) ServeServiceVideo(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get(":filename")
	if filename == "" {
		http.Error(w, "filename is required", http.StatusBadRequest)
		return
	}

	videoPath := filepath.Join("cmd/uploads/services/videos", filename)

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
	service.AvgRating, _ = strconv.ParseFloat(r.FormValue("avg_rating"), 64)
	service.Top = r.FormValue("top")
	service.Status = normalizeListingStatus(r.FormValue("status"))
	if v := r.FormValue("latitude"); v != "" {
		service.Latitude = &v
	}
	if v := r.FormValue("longitude"); v != "" {
		service.Longitude = &v
	}
	service.CreatedAt = time.Now()

	// Сохраняем изображения
	saveDir := "cmd/uploads/services"
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		http.Error(w, "Failed to create image directory", http.StatusInternalServerError)
		return
	}

	imageHeaders := collectImageFiles(r.MultipartForm, "images", "images[]")
	var imageInfos []models.Image

	if parsedImages, ok, err := gatherImagesFromForm[models.Image](r.MultipartForm, "images", "images[]"); err != nil {
		http.Error(w, "Invalid images payload", http.StatusBadRequest)
		return
	} else if ok {
		imageInfos = append(imageInfos, parsedImages...)
	}

	if parsedLinks, ok, err := gatherImagesFromForm[models.Image](r.MultipartForm, "image_links", "image_links[]"); err != nil {
		http.Error(w, "Invalid image links payload", http.StatusBadRequest)
		return
	} else if ok {
		imageInfos = append(imageInfos, parsedLinks...)
	}

	for _, fileHeader := range imageHeaders {
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, "Failed to open image", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		timestamp := time.Now().UnixNano()
		ext := filepath.Ext(fileHeader.Filename)
		imageName := fmt.Sprintf("service_image_%d%s", timestamp, ext)
		savePath := filepath.Join(saveDir, imageName)
		publicURL := fmt.Sprintf("/images/services/%s", imageName)

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

		imageInfos = append(imageInfos, models.Image{
			Name: fileHeader.Filename,
			Path: publicURL,
			Type: fileHeader.Header.Get("Content-Type"),
		})
	}

	service.Images = imageInfos

	videoDir := "cmd/uploads/services/videos"
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
		videoName := fmt.Sprintf("service_video_%d%s", timestamp, ext)
		savePath := filepath.Join(videoDir, videoName)
		publicURL := fmt.Sprintf("/videos/services/%s", videoName)

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

	createdService, err := h.Service.CreateService(r.Context(), service)
	if err != nil {
		if errors.Is(err, services.ErrNoActiveSubscription) {

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

	err = r.ParseMultipartForm(32 << 20) // до 32MB
	if err != nil {
		http.Error(w, "Invalid multipart form", http.StatusBadRequest)
		return
	}

	existingService, err := h.Service.GetServiceByID(r.Context(), id, 0)
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
		service.Price, _ = strconv.ParseFloat(v[0], 64)
	}
	if v, ok := r.MultipartForm.Value["user_id"]; ok {
		service.UserID, _ = strconv.Atoi(v[0])
	}
	if _, ok := r.MultipartForm.Value["description"]; ok {
		service.Description = r.FormValue("description")
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
	if _, ok := r.MultipartForm.Value["top"]; ok {
		service.Top = r.FormValue("top")
	}
	if _, ok := r.MultipartForm.Value["liked"]; ok {
		service.Liked = r.FormValue("liked") == "true"
	}
	if _, ok := r.MultipartForm.Value["status"]; ok {
		service.Status = normalizeListingStatus(r.FormValue("status"))
	}
	if v, ok := r.MultipartForm.Value["latitude"]; ok {
		lat := v[0]
		service.Latitude = &lat
	}
	if v, ok := r.MultipartForm.Value["longitude"]; ok {
		lon := v[0]
		service.Longitude = &lon
	}

	images := service.Images

	if parsedImages, ok, err := gatherImagesFromForm[models.Image](r.MultipartForm, "images", "images[]"); err != nil {
		http.Error(w, "Invalid images payload", http.StatusBadRequest)
		return
	} else if ok {
		images = parsedImages
	} else if parsedExisting, okExisting, err := gatherImagesFromForm[models.Image](r.MultipartForm, "existing_images", "existing_images[]"); err != nil {
		http.Error(w, "Invalid images payload", http.StatusBadRequest)
		return
	} else if okExisting {
		images = parsedExisting
	}

	if parsedLinks, ok, err := gatherImagesFromForm[models.Image](r.MultipartForm, "image_links", "image_links[]"); err != nil {
		http.Error(w, "Invalid image links payload", http.StatusBadRequest)
		return
	} else if ok {
		images = append(images, parsedLinks...)
	}

	saveDir := "cmd/uploads/services"
	err = os.MkdirAll(saveDir, 0755)
	if err != nil {
		http.Error(w, "Failed to create image directory", http.StatusInternalServerError)
		return
	}

	fileHeaders := collectImageFiles(r.MultipartForm, "images", "images[]")
	if len(fileHeaders) > 0 {
		var uploaded []models.Image
		for _, fileHeader := range fileHeaders {
			file, err := fileHeader.Open()
			if err != nil {
				http.Error(w, "Failed to open image", http.StatusInternalServerError)
				return
			}
			defer file.Close()

			timestamp := time.Now().UnixNano()
			ext := filepath.Ext(fileHeader.Filename)
			imageName := fmt.Sprintf("service_image_%d%s", timestamp, ext)
			savePath := filepath.Join(saveDir, imageName)
			publicURL := fmt.Sprintf("/images/services/%s", imageName)

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

			uploaded = append(uploaded, models.Image{
				Name: fileHeader.Filename,
				Path: publicURL,
				Type: fileHeader.Header.Get("Content-Type"),
			})
		}
		images = append(images, uploaded...)
	}

	if len(deletedImageKeys) > 0 {
		var removedImages []models.Image
		images, removedImages = filterServiceImages(images, deletedImageKeys)
		if err := removeServiceImagesFromDisk(removedImages); err != nil {
			log.Printf("Failed to remove service images: %v", err)
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

	videoDir := "cmd/uploads/services/videos"
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
			videoName := fmt.Sprintf("service_video_%d%s", timestamp, ext)
			savePath := filepath.Join(videoDir, videoName)
			publicURL := fmt.Sprintf("/videos/services/%s", videoName)

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
		if err := removeServiceVideosFromDisk(removedVideos); err != nil {
			log.Printf("Failed to remove service videos: %v", err)
		}
	}

	service.Videos = videos

	now := time.Now()
	service.UpdatedAt = &now

	updatedService, err := h.Service.UpdateService(r.Context(), service)
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

func (h *ServiceHandler) GetFilteredServicesWithLikes(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get(":user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	var req models.FilterServicesRequest
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

	services, err := h.Service.GetFilteredServicesWithLikes(r.Context(), req, userID)
	if err != nil {
		log.Printf("GetServicesPost error: %v", err)
		http.Error(w, "Failed to fetch services", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"services": services,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *ServiceHandler) GetServiceByServiceIDAndUserID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	serviceIDStr := r.URL.Query().Get(":service_id")
	if serviceIDStr == "" {
		http.Error(w, "service ID is required", http.StatusBadRequest)
		return
	}

	serviceID, err := strconv.Atoi(serviceIDStr)
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
	service, err := h.Service.GetServiceByServiceIDAndUserID(ctx, serviceID, userID)
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
	if err := json.NewEncoder(w).Encode(service); err != nil {
		log.Printf("[ERROR] Failed to encode response: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func filterServiceImages(images []models.Image, deleteKeys []string) ([]models.Image, []models.Image) {
	removalSet := buildRemovalSet(deleteKeys)
	if len(removalSet) == 0 {
		return images, nil
	}

	var (
		kept    []models.Image
		removed []models.Image
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

func filterServiceVideos(videos []models.Video, deleteKeys []string) ([]models.Video, []models.Video) {
	removalSet := buildRemovalSet(deleteKeys)
	if len(removalSet) == 0 {
		return videos, nil
	}

	var (
		kept    []models.Video
		removed []models.Video
	)

	for _, video := range videos {
		if shouldRemoveMedia(video.Path, video.Name, removalSet) {
			removed = append(removed, video)
			continue
		}
		kept = append(kept, video)
	}

	return kept, removed
}

func removeServiceImagesFromDisk(images []models.Image) error {
	for _, img := range images {
		if img.Type == "link" {
			continue
		}
		if err := removeMediaFile("cmd/uploads/services", "/images/services/", img.Path); err != nil {
			return err
		}
	}
	return nil
}

func removeServiceVideosFromDisk(videos []models.Video) error {
	for _, video := range videos {
		if video.Type == "link" {
			continue
		}
		if err := removeMediaFile("cmd/uploads/services/videos", "/videos/services/", video.Path); err != nil {
			return err
		}
	}
	return nil
}

func buildRemovalSet(keys []string) map[string]struct{} {
	if len(keys) == 0 {
		return nil
	}

	set := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		set[key] = struct{}{}
	}
	return set
}

func shouldRemoveMedia(path, name string, removalSet map[string]struct{}) bool {
	if removalSet == nil {
		return false
	}
	if _, ok := removalSet[path]; ok {
		return true
	}
	if name != "" {
		if _, ok := removalSet[name]; ok {
			return true
		}
	}
	return false
}

func removeMediaFile(baseDir, publicPrefix, publicPath string) error {
	if !strings.HasPrefix(publicPath, publicPrefix) {
		return nil
	}

	relative := strings.TrimPrefix(publicPath, publicPrefix)
	if relative == "" {
		return nil
	}

	cleanRelative := filepath.Clean(relative)
	if cleanRelative == "." || strings.HasPrefix(cleanRelative, "..") || filepath.IsAbs(cleanRelative) {
		return fmt.Errorf("invalid media path: %s", publicPath)
	}

	fsPath := filepath.Join(baseDir, cleanRelative)
	if err := os.Remove(fsPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}
