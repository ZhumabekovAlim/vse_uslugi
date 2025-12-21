package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	sortOption := parsePositiveIntAllowZero(r.URL.Query().Get("sorting"))
	if sortOption == 0 {
		sortOption = parsePositiveIntAllowZero(r.URL.Query().Get("sortOption"))
	}

	rentTypes := parseStringList(r.URL.Query().Get("rent_types"))
	deposits := parseStringList(r.URL.Query().Get("deposits"))
	workExperience := parseStringList(r.URL.Query().Get("work_experience"))
	workSchedules := parseStringList(r.URL.Query().Get("work_schedule"))
	if len(workSchedules) == 0 {
		workSchedules = parseStringList(r.URL.Query().Get("work_schedules"))
	}
	paymentPeriods := parseStringList(r.URL.Query().Get("payment_period"))
	if len(paymentPeriods) == 0 {
		paymentPeriods = parseStringList(r.URL.Query().Get("payment_periods"))
	}
	languages := parseStringList(r.URL.Query().Get("languages"))
	educations := parseStringList(r.URL.Query().Get("education"))
	if len(educations) == 0 {
		educations = parseStringList(r.URL.Query().Get("educations"))
	}

	remoteWork, ok := parseBoolChoice(r.URL.Query().Get("remote"))
	if !ok {
		http.Error(w, "invalid remote value", http.StatusBadRequest)
		return
	}

	onSite, ok := parseBoolChoice(r.URL.Query().Get("on_site"))
	if !ok {
		http.Error(w, "invalid on_site value", http.StatusBadRequest)
		return
	}

	negotiable, ok := parseBoolChoice(r.URL.Query().Get("negotiable"))
	if !ok {
		http.Error(w, "invalid negotiable value", http.StatusBadRequest)
		return
	}

	latitude, longitude, err := parseCoordinates(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	radius, err := parseRadius(r.URL.Query().Get("radius"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
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
		OnSite:         onSite,
		Negotiable:     negotiable,
		RentTypes:      rentTypes,
		Deposits:       deposits,
		WorkExperience: workExperience,
		WorkSchedules:  workSchedules,
		PaymentPeriods: paymentPeriods,
		RemoteWork:     remoteWork,
		Languages:      languages,
		Educations:     educations,
		Latitude:       latitude,
		Longitude:      longitude,
		RadiusKm:       radius,
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

func parseStringList(input string) []string {
	if input == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// parseBoolChoice returns a pointer to bool when the input represents a yes/no choice.
// Supported truthy values: "yes", "true", "1", "да". False values: "no", "false", "0", "нет".
// Empty input returns nil. Any other value leads to ok=false.
func parseBoolChoice(input string) (*bool, bool) {
	if input == "" {
		return nil, true
	}

	normalized := strings.ToLower(strings.TrimSpace(input))
	switch normalized {
	case "yes", "true", "1", "да":
		value := true
		return &value, true
	case "no", "false", "0", "нет":
		value := false
		return &value, true
	default:
		return nil, false
	}
}

func parseCoordinates(values url.Values) (*float64, *float64, error) {
	latParam := strings.TrimSpace(values.Get("latitude"))
	lonParam := strings.TrimSpace(values.Get("longitude"))

	if latParam == "" && lonParam == "" {
		return nil, nil, nil
	}

	if latParam == "" || lonParam == "" {
		return nil, nil, fmt.Errorf("both latitude and longitude must be provided")
	}

	lat, err := strconv.ParseFloat(latParam, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid latitude")
	}
	lon, err := strconv.ParseFloat(lonParam, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid longitude")
	}

	return &lat, &lon, nil
}

func parseRadius(input string) (*float64, error) {
	if strings.TrimSpace(input) == "" {
		return nil, nil
	}

	radius, err := strconv.ParseFloat(strings.TrimSpace(input), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid radius")
	}
	if radius <= 0 {
		return nil, fmt.Errorf("radius must be greater than zero")
	}

	return &radius, nil
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
