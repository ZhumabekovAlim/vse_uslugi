package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"naimuBack/internal/models"
	"naimuBack/internal/services"
)

// GlobalSearchHandler exposes endpoints for aggregated listings.
type GlobalSearchHandler struct {
	Service *services.GlobalSearchService
}

// Search executes a mixed listings search across supported domains.
func (h *GlobalSearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		http.Error(w, "service unavailable", http.StatusInternalServerError)
		return
	}

	typesParam := strings.TrimSpace(r.URL.Query().Get("types"))
	if typesParam == "" {
		http.Error(w, "types parameter is required", http.StatusBadRequest)
		return
	}

	allowedTypes := models.AllowedTopTypes()
	rawTypes := strings.Split(typesParam, ",")
	seen := make(map[string]struct{}, len(rawTypes))
	types := make([]string, 0, len(rawTypes))
	for _, t := range rawTypes {
		trimmed := strings.TrimSpace(t)
		if trimmed == "" {
			continue
		}
		if _, ok := allowedTypes[trimmed]; !ok {
			http.Error(w, "unsupported listing type: "+trimmed, http.StatusBadRequest)
			return
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		types = append(types, trimmed)
	}

	if len(types) == 0 {
		http.Error(w, "at least one valid type must be provided", http.StatusBadRequest)
		return
	}
	if len(types) > 6 {
		http.Error(w, "no more than 6 listing types allowed", http.StatusBadRequest)
		return
	}

	categories := parseIntList(r.URL.Query().Get("categories"))
	subcategories := parseIntList(r.URL.Query().Get("subcategories"))

	limit := parsePositiveInt(r.URL.Query().Get("limit"), 20)
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	priceFrom := parseFloat(r.URL.Query().Get("priceFrom"))
	if priceFrom == 0 {
		priceFrom = parseFloat(r.URL.Query().Get("price_from"))
	}
	priceTo := parseFloat(r.URL.Query().Get("priceTo"))
	if priceTo == 0 {
		priceTo = parseFloat(r.URL.Query().Get("price_to"))
	}
	ratings := parseFloatList(r.URL.Query().Get("ratings"))
	sortOption := parsePositiveIntAllowZero(r.URL.Query().Get("sortOption"))
	if sortOption == 0 {
		sortOption = parsePositiveIntAllowZero(r.URL.Query().Get("sort_option"))
	}

	userID := extractUserIDFromRequest(r)

	req := models.GlobalSearchRequest{
		Types:          types,
		CategoryIDs:    categories,
		SubcategoryIDs: subcategories,
		Limit:          limit,
		Page:           page,
		PriceFrom:      priceFrom,
		PriceTo:        priceTo,
		Ratings:        ratings,
		SortOption:     sortOption,
		UserID:         userID,
	}

	response, err := h.Service.Search(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func parseIntList(input string) []int {
	if input == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	result := make([]int, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		if value, err := strconv.Atoi(trimmed); err == nil {
			result = append(result, value)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func parseFloatList(input string) []float64 {
	if input == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	result := make([]float64, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		if value, err := strconv.ParseFloat(trimmed, 64); err == nil {
			result = append(result, value)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func parsePositiveInt(input string, fallback int) int {
	if input == "" {
		return fallback
	}
	if value, err := strconv.Atoi(input); err == nil && value > 0 {
		return value
	}
	return fallback
}

func parsePositiveIntAllowZero(input string) int {
	if input == "" {
		return 0
	}
	if value, err := strconv.Atoi(input); err == nil && value >= 0 {
		return value
	}
	return 0
}

func parseFloat(input string) float64 {
	if input == "" {
		return 0
	}
	if value, err := strconv.ParseFloat(input, 64); err == nil {
		return value
	}
	return 0
}

func extractUserIDFromRequest(r *http.Request) int {
	tokenString := r.Header.Get("Authorization")
	if !strings.HasPrefix(strings.ToLower(tokenString), "bearer ") {
		return 0
	}
	tokenString = strings.TrimSpace(tokenString[len("Bearer "):])
	claims := &models.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(signingKey), nil
	})
	if err != nil || !token.Valid {
		return 0
	}
	return int(claims.UserID)
}
