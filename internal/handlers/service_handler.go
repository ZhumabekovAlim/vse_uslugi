package handlers

import (
	_ "context"
	"encoding/json"
	"errors"
	"log"
	"naimuBack/internal/repositories"
	"net/http"
	"strconv"
	"strings"
	_ "strings"

	"naimuBack/internal/models"
	"naimuBack/internal/services"
)

type ServiceHandler struct {
	Service *services.ServiceService
}

func (h *ServiceHandler) CreateService(w http.ResponseWriter, r *http.Request) {
	var service models.Service
	err := json.NewDecoder(r.Body).Decode(&service)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if service.Name == "" || service.Price <= 0 || service.UserID == 0 {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	createdService, err := h.Service.CreateService(r.Context(), service)
	if err != nil {
		log.Printf("CreateService error: %v", err)
		http.Error(w, "Failed to create service", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdService)
}

//func (h *ServiceHandler) GetServiceByID(w http.ResponseWriter, r *http.Request) {
//	idStr := r.URL.Query().Get(":id")
//	if idStr == "" {
//		http.Error(w, "Missing service ID", http.StatusBadRequest)
//		return
//	}
//
//	id, err := strconv.Atoi(idStr)
//	if err != nil {
//		http.Error(w, "Invalid service ID", http.StatusBadRequest)
//		return
//	}
//
//	service, err := h.Service.GetServiceByID(r.Context(), id)
//	if err != nil {
//		if errors.Is(err, repositories.ErrServiceNotFound) {
//			http.Error(w, err.Error(), http.StatusNotFound)
//			return
//		}
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//		return
//	}
//
//	w.Header().Set("Content-Type", "application/json")
//	json.NewEncoder(w).Encode(service)
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

//func (h *ServiceHandler) GetServices(w http.ResponseWriter, r *http.Request) {
//	categoriesStr := r.URL.Query()["categories"]
//	subcategoriesStr := r.URL.Query()["subcategories"]
//	priceFromStr := r.URL.Query().Get("price_from")
//	priceToStr := r.URL.Query().Get("price_to")
//	ratingsStr := r.URL.Query()["ratings"]
//	sorting := r.URL.Query().Get("sorting")
//	pageStr := r.URL.Query().Get("page")
//	pageSizeStr := r.URL.Query().Get("page_size")
//
//	var categories []int
//	for _, catStr := range categoriesStr {
//		cat, err := strconv.Atoi(catStr)
//		if err == nil {
//			categories = append(categories, cat)
//		}
//	}
//
//	var subcategories []string = subcategoriesStr
//
//	priceFrom, _ := strconv.ParseFloat(priceFromStr, 64)
//	priceTo, _ := strconv.ParseFloat(priceToStr, 64)
//
//	var ratings []float64
//	for _, ratStr := range ratingsStr {
//		rat, err := strconv.ParseFloat(ratStr, 64)
//		if err == nil {
//			ratings = append(ratings, rat)
//		}
//	}
//
//	page, _ := strconv.Atoi(pageStr)
//	if page < 1 {
//		page = 1
//	}
//	pageSize, _ := strconv.Atoi(pageSizeStr)
//	if pageSize < 1 {
//		pageSize = 10
//	}
//
//	filters := models.GetServicesRequest{
//		Categories:    categories,
//		Subcategories: subcategories,
//		PriceFrom:     priceFrom,
//		PriceTo:       priceTo,
//		Ratings:       ratings,
//		Sorting:       sorting,
//		Page:          page,
//		PageSize:      pageSize,
//	}
//
//	services, minPrice, maxPrice, total, err := h.Service.GetServicesWithFilters(r.Context(), filters)
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//		return
//	}
//
//	response := struct {
//		Services []models.ServiceResponse `json:"services"`
//		MinPrice float64                  `json:"min_price"`
//		MaxPrice float64                  `json:"max_price"`
//		Total    int                      `json:"total"`
//	}{
//		Services: services,
//		MinPrice: minPrice,
//		MaxPrice: maxPrice,
//		Total:    total,
//	}
//
//	w.Header().Set("Content-Type", "application/json")
//	json.NewEncoder(w).Encode(response)
//}

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
