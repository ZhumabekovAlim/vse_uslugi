package http

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"naimuBack/internal/courier/repo"
)

// randomHex генерит n байт и возвращает hex-строку (для имени файла)
func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b)
}

// saveUploadedFile сохраняет файл на диск и возвращает относительный путь (для БД)
func saveUploadedFile(file multipart.File, header *multipart.FileHeader, subdir string) (string, error) {
	defer file.Close()

	// Создадим директорию: ./uploads/couriers/<subdir>/
	baseDir := filepath.Join(".", "uploads", "couriers", subdir)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return "", err
	}

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".bin"
	}
	filename := time.Now().Format("20060102") + "_" + randomHex(8) + ext
	fullPath := filepath.Join(baseDir, filename)

	dst, err := os.Create(fullPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return "", err
	}

	// Вернём относительный путь (как обычно кладут в БД)
	rel := filepath.ToSlash(filepath.Join("uploads", "couriers", subdir, filename))
	return rel, nil
}

func (s *Server) handleCourierUpsert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.couriers == nil {
		http.Error(w, "couriers repository unavailable", http.StatusServiceUnavailable)
		return
	}

	ctype := r.Header.Get("Content-Type")
	isMultipart := strings.HasPrefix(strings.ToLower(ctype), "multipart/form-data")

	// Поля
	var (
		userID           int64
		firstName        string
		lastName         string
		middleName       *string
		courierPhotoPath string
		iin              string
		dateOfBirth      string
		idCardFrontPath  string
		idCardBackPath   string
		phone            string
		rating           *float64
		status           *string
	)

	if isMultipart {
		// 20 MiB лимит формы (подстрой при необходимости)
		if err := r.ParseMultipartForm(20 << 20); err != nil {
			writeError(w, http.StatusBadRequest, "invalid multipart form")
			return
		}
		// Читаем текстовые поля
		userIDStr := strings.TrimSpace(r.FormValue("user_id"))
		if userIDStr != "" {
			if v, err := strconv.ParseInt(userIDStr, 10, 64); err == nil {
				userID = v
			}
		}
		firstName = strings.TrimSpace(r.FormValue("first_name"))
		lastName = strings.TrimSpace(r.FormValue("last_name"))
		if v := strings.TrimSpace(r.FormValue("middle_name")); v != "" {
			middleName = &v
		}
		iin = strings.TrimSpace(r.FormValue("iin"))
		dateOfBirth = strings.TrimSpace(r.FormValue("date_of_birth"))
		phone = strings.TrimSpace(r.FormValue("phone"))
		if v := strings.TrimSpace(r.FormValue("rating")); v != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				rating = &f
			}
		}
		if v := strings.TrimSpace(r.FormValue("status")); v != "" {
			status = &[]string{v}[0]
		}

		// Загружаемые файлы: courier_photo, id_card_front, id_card_back
		// — если не пришёл файл, поле остаётся пустым (при апдейте можешь не присылать)
		if f, h, err := r.FormFile("courier_photo"); err == nil {
			if path, err := saveUploadedFile(f, h, "photos"); err == nil {
				courierPhotoPath = path
			} else {
				writeError(w, http.StatusBadRequest, "failed to save courier_photo")
				return
			}
		}
		if f, h, err := r.FormFile("id_card_front"); err == nil {
			if path, err := saveUploadedFile(f, h, "idcards"); err == nil {
				idCardFrontPath = path
			} else {
				writeError(w, http.StatusBadRequest, "failed to save id_card_front")
				return
			}
		}
		if f, h, err := r.FormFile("id_card_back"); err == nil {
			if path, err := saveUploadedFile(f, h, "idcards"); err == nil {
				idCardBackPath = path
			} else {
				writeError(w, http.StatusBadRequest, "failed to save id_card_back")
				return
			}
		}
	} else {
		// Старый путь: JSON
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
		userID = payload.UserID
		firstName = strings.TrimSpace(payload.FirstName)
		lastName = strings.TrimSpace(payload.LastName)
		middleName = payload.MiddleName
		courierPhotoPath = strings.TrimSpace(payload.CourierPhoto)
		iin = strings.TrimSpace(payload.IIN)
		dateOfBirth = strings.TrimSpace(payload.DateOfBirth)
		idCardFrontPath = strings.TrimSpace(payload.IDCardFront)
		idCardBackPath = strings.TrimSpace(payload.IDCardBack)
		phone = strings.TrimSpace(payload.Phone)
		rating = payload.Rating
		status = payload.Status
	}

	// Валидация
	if userID == 0 || firstName == "" || lastName == "" {
		writeError(w, http.StatusBadRequest, "user_id, first_name and last_name are required")
		return
	}
	if iin == "" {
		writeError(w, http.StatusBadRequest, "iin is required")
		return
	}
	// Для первичного создания требуем хотя бы courier_photo, при апдейте можно не присылать файл
	// (определим первичное создание чуть ниже, когда будем получать существующую запись)
	birthDate, err := time.Parse("2006-01-02", dateOfBirth)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid date_of_birth")
		return
	}
	stat := repo.CourierStatusOffline
	if status != nil && strings.TrimSpace(*status) != "" {
		candidate := strings.ToLower(strings.TrimSpace(*status))
		switch candidate {
		case repo.CourierStatusOffline, repo.CourierStatusFree, repo.CourierStatusBusy:
			stat = candidate
		default:
			writeError(w, http.StatusBadRequest, "invalid status")
			return
		}
	}

	var middle sql.NullString
	if middleName != nil && strings.TrimSpace(*middleName) != "" {
		middle = sql.NullString{String: strings.TrimSpace(*middleName), Valid: true}
	}
	var ratingNF sql.NullFloat64
	if rating != nil {
		ratingNF = sql.NullFloat64{Float64: *rating, Valid: true}
	}

	// Если это апдейт по user_id, возможно фото не пришли — оставим прежние.
	// Для этого попробуем найти курьера по user_id (добавим простой геттер).
	ctx, cancel := contextWithTimeout(r)
	defer cancel()

	courier := repo.Courier{
		UserID:      userID,
		FirstName:   firstName,
		LastName:    lastName,
		MiddleName:  middle,
		Photo:       courierPhotoPath,
		IIN:         iin,
		BirthDate:   birthDate,
		IDCardFront: idCardFrontPath,
		IDCardBack:  idCardBackPath,
		Phone:       phone,
		Rating:      ratingNF,
		Status:      stat,
	}

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
