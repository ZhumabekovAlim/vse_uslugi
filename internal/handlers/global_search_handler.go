package handlers

import (
	"encoding/json"
	"fmt"
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

type globalSearchPayload struct {
	Types          []string  `json:"types"`
	Categories     []int     `json:"categories"`
	Subcategories  []int     `json:"subcategories"`
	Limit          *int      `json:"limit"`
	Page           *int      `json:"page"`
	PriceFrom      *float64  `json:"price_from"`
	PriceFromAlt   *float64  `json:"priceFrom"`
	PriceTo        *float64  `json:"price_to"`
	PriceToAlt     *float64  `json:"priceTo"`
	Ratings        []float64 `json:"ratings"`
	SortOption     *int      `json:"sort_option"`
	SortOptionAlt  *int      `json:"sortOption"`
	OnSite         *bool     `json:"on_site"`
	Negotiable     *bool     `json:"negotiable"`
	RentTypes      []string  `json:"rent_types"`
	Deposits       []string  `json:"deposits"`
	WorkExperience []string  `json:"work_experience"`
	WorkSchedules  []string  `json:"work_schedules"`
	WorkSchedule   []string  `json:"work_schedule"`
	PaymentPeriods []string  `json:"payment_periods"`
	PaymentPeriod  []string  `json:"payment_period"`
	RemoteWork     *bool     `json:"remote"`
	Languages      []string  `json:"languages"`
	Educations     []string  `json:"educations"`
	Education      []string  `json:"education"`
	OrderDate      *string   `json:"order_date"`
	OrderTime      *string   `json:"order_time"`
	Latitude       *float64  `json:"latitude"`
	Longitude      *float64  `json:"longitude"`
	Radius         *float64  `json:"radius"`
}

// Search executes a mixed listings search across supported domains.
func (h *GlobalSearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		http.Error(w, "service unavailable", http.StatusInternalServerError)
		return
	}

	var payload globalSearchPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	if len(payload.Types) == 0 {
		http.Error(w, "types parameter is required", http.StatusBadRequest)
		return
	}

	allowedTypes := models.AllowedTopTypes()
	seen := make(map[string]struct{}, len(payload.Types))
	types := make([]string, 0, len(payload.Types))
	for _, t := range payload.Types {
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

	categories := payload.Categories
	subcategories := payload.Subcategories

	limit := normalizePositiveInt(payload.Limit, 20)
	page := normalizePositiveInt(payload.Page, 1)
	priceFrom := coalesceFloat(payload.PriceFrom, payload.PriceFromAlt)
	priceTo := coalesceFloat(payload.PriceTo, payload.PriceToAlt)
	ratings := payload.Ratings
	sortOption := normalizeNonNegativeInt(payload.SortOption, payload.SortOptionAlt)

	rentTypes := payload.RentTypes
	deposits := payload.Deposits
	workExperience := payload.WorkExperience
	workSchedules := payload.WorkSchedules
	if len(workSchedules) == 0 {
		workSchedules = payload.WorkSchedule
	}
	paymentPeriods := payload.PaymentPeriods
	if len(paymentPeriods) == 0 {
		paymentPeriods = payload.PaymentPeriod
	}
	languages := payload.Languages
	educations := payload.Educations
	if len(educations) == 0 {
		educations = payload.Education
	}
	orderDate := payload.OrderDate
	orderTime := payload.OrderTime

	remoteWork := payload.RemoteWork
	onSite := payload.OnSite
	negotiable := payload.Negotiable

	latitude, longitude, err := normalizeCoordinates(payload.Latitude, payload.Longitude)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	radius, err := normalizeRadius(payload.Radius)
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
		OrderDate:      orderDate,
		OrderTime:      orderTime,
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

func normalizePositiveInt(value *int, fallback int) int {
	if value != nil && *value > 0 {
		return *value
	}
	return fallback
}

func coalesceFloat(first *float64, second *float64) float64 {
	if first != nil {
		return *first
	}
	if second != nil {
		return *second
	}
	return 0
}

func normalizeNonNegativeInt(first *int, second *int) int {
	if first != nil && *first >= 0 {
		return *first
	}
	if second != nil && *second >= 0 {
		return *second
	}
	return 0
}

func normalizeCoordinates(latitude *float64, longitude *float64) (*float64, *float64, error) {
	if latitude == nil && longitude == nil {
		return nil, nil, nil
	}
	if latitude == nil || longitude == nil {
		return nil, nil, fmt.Errorf("both latitude and longitude must be provided")
	}
	return latitude, longitude, nil
}

func normalizeRadius(radius *float64) (*float64, error) {
	if radius == nil {
		return nil, nil
	}
	if *radius <= 0 {
		return nil, fmt.Errorf("radius must be greater than zero")
	}
	return radius, nil
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
