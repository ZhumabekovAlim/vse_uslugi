package http

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"naimuBack/internal/courier/repo"
)

func (s *Server) handleCourierUpsert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.couriers == nil {
		http.Error(w, "couriers repository unavailable", http.StatusServiceUnavailable)
		return
	}
	var payload struct {
		UserID       int64    `json:"user_id"`
		FirstName    string   `json:"first_name"`
		LastName     string   `json:"last_name"`
		MiddleName   *string  `json:"middle_name"`
		CourierPhoto string   `json:"courier_photo"`
		IIN          string   `json:"iin"`
		DateOfBirth  string   `json:"date_of_birth"`
		IDCardFront  string   `json:"id_card_front"`
		IDCardBack   string   `json:"id_card_back"`
		Phone        string   `json:"phone"`
		Rating       *float64 `json:"rating"`
		Status       *string  `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if payload.UserID == 0 || strings.TrimSpace(payload.FirstName) == "" || strings.TrimSpace(payload.LastName) == "" {
		writeError(w, http.StatusBadRequest, "user_id, first_name and last_name are required")
		return
	}
	if strings.TrimSpace(payload.CourierPhoto) == "" || strings.TrimSpace(payload.IIN) == "" {
		writeError(w, http.StatusBadRequest, "courier_photo and iin are required")
		return
	}
	birthDate, err := time.Parse("2006-01-02", strings.TrimSpace(payload.DateOfBirth))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid date_of_birth")
		return
	}
	status := "pending"
	if payload.Status != nil && strings.TrimSpace(*payload.Status) != "" {
		status = strings.ToLower(strings.TrimSpace(*payload.Status))
	}

	var middle sql.NullString
	if payload.MiddleName != nil && strings.TrimSpace(*payload.MiddleName) != "" {
		middle = sql.NullString{String: strings.TrimSpace(*payload.MiddleName), Valid: true}
	}
	var rating sql.NullFloat64
	if payload.Rating != nil {
		rating = sql.NullFloat64{Float64: *payload.Rating, Valid: true}
	}

	courier := repo.Courier{
		UserID:      payload.UserID,
		FirstName:   strings.TrimSpace(payload.FirstName),
		LastName:    strings.TrimSpace(payload.LastName),
		MiddleName:  middle,
		Photo:       strings.TrimSpace(payload.CourierPhoto),
		IIN:         strings.TrimSpace(payload.IIN),
		BirthDate:   birthDate,
		IDCardFront: strings.TrimSpace(payload.IDCardFront),
		IDCardBack:  strings.TrimSpace(payload.IDCardBack),
		Phone:       strings.TrimSpace(payload.Phone),
		Rating:      rating,
		Status:      status,
	}

	ctx, cancel := contextWithTimeout(r)
	defer cancel()

	id, err := s.couriers.Upsert(ctx, courier)
	if err != nil {
		s.logger.Errorf("courier: upsert profile failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to upsert courier")
		return
	}

	created, err := s.couriers.Get(ctx, id)
	if err != nil {
		s.logger.Errorf("courier: fetch courier after upsert failed: %v", err)
		writeJSON(w, http.StatusOK, map[string]int64{"courier_id": id})
		return
	}
	stats, err := s.orders.CourierStats(ctx, created.ID)
	if err != nil {
		s.logger.Errorf("courier: fetch courier stats failed: %v", err)
		stats = repo.CourierOrderStats{}
	}

	resp := courierProfileResponse{Courier: makeCourierResponse(created), Stats: makeCourierStatsResponse(stats)}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleCourierProfileRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/courier/")
	path = strings.Trim(path, "/")
	if path == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid courier id")
		return
	}
	switch parts[1] {
	case "profile":
		s.handleCourierProfile(w, r, id)
	case "reviews":
		s.handleCourierReviews(w, r, id)
	case "stats":
		s.handleCourierStats(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *Server) handleCourierProfile(w http.ResponseWriter, r *http.Request, courierID int64) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.couriers == nil {
		http.Error(w, "couriers repository unavailable", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := contextWithTimeout(r)
	defer cancel()

	courier, err := s.couriers.Get(ctx, courierID)
	if errors.Is(err, repo.ErrNotFound) {
		writeError(w, http.StatusNotFound, "courier not found")
		return
	}
	if err != nil {
		s.logger.Errorf("courier: load profile failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load courier")
		return
	}
	stats, err := s.orders.CourierStats(ctx, courier.ID)
	if err != nil {
		s.logger.Errorf("courier: load courier stats failed: %v", err)
		stats = repo.CourierOrderStats{}
	}
	resp := courierProfileResponse{Courier: makeCourierResponse(courier), Stats: makeCourierStatsResponse(stats)}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleCourierReviews(w http.ResponseWriter, r *http.Request, courierID int64) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.couriers == nil {
		http.Error(w, "courier repository unavailable", http.StatusServiceUnavailable)
		return
	}
	ctx, cancel := contextWithTimeout(r)
	defer cancel()

	if _, err := s.couriers.Get(ctx, courierID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "courier not found")
		} else {
			s.logger.Errorf("courier: load courier for reviews failed: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to load courier")
		}
		return
	}

	reviews, err := s.orders.ListCourierReviews(ctx, courierID)
	if err != nil {
		s.logger.Errorf("courier: list reviews failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load reviews")
		return
	}

	resp := make([]courierReviewResponse, 0, len(reviews))
	for _, review := range reviews {
		item := courierReviewResponse{
			CreatedAt: review.CreatedAt,
			Order:     makeOrderResponse(review.Order),
		}
		if review.SenderRating.Valid {
			v := review.SenderRating.Float64
			item.Rating = &v
		}
		if review.CourierRating.Valid {
			v := review.CourierRating.Float64
			item.CourierRating = &v
		}
		if review.Comment.Valid {
			item.Comment = review.Comment.String
		}
		resp = append(resp, item)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"reviews": resp})
}

func (s *Server) handleCourierStats(w http.ResponseWriter, r *http.Request, courierID int64) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := contextWithTimeout(r)
	defer cancel()

	stats, err := s.orders.CourierStats(ctx, courierID)
	if err != nil {
		s.logger.Errorf("courier: load courier stats failed: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load stats")
		return
	}
	writeJSON(w, http.StatusOK, makeCourierStatsResponse(stats))
}
