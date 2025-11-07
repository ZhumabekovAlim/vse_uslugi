package taxihttp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"naimuBack/internal/taxi/dispatch"
	"naimuBack/internal/taxi/fsm"
	"naimuBack/internal/taxi/geo"
	"naimuBack/internal/taxi/pay"
	"naimuBack/internal/taxi/pricing"
	"naimuBack/internal/taxi/repo"
	"naimuBack/internal/taxi/timeutil"
	"naimuBack/internal/taxi/ws"
)

// Server handles HTTP endpoints for taxi module.
type Server struct {
	logger        dispatch.Logger
	cfg           dispatch.Config
	geoClient     *geo.DGISClient
	driversRepo   *repo.DriversRepo
	ordersRepo    *repo.OrdersRepo
	intercityRepo *repo.IntercityOrdersRepo
	offersRepo    *repo.OffersRepo
	paymentsRepo  *repo.PaymentsRepo
	driverHub     *ws.DriverHub
	passengerHub  *ws.PassengerHub
	dispatcher    *dispatch.Dispatcher
	payClient     *pay.Client
}

// NewServer constructs Server.
func NewServer(logger dispatch.Logger, cfg dispatch.Config, geoClient *geo.DGISClient, drivers *repo.DriversRepo, orders *repo.OrdersRepo, intercity *repo.IntercityOrdersRepo, offers *repo.OffersRepo, payments *repo.PaymentsRepo, driverHub *ws.DriverHub, passengerHub *ws.PassengerHub, dispatcher *dispatch.Dispatcher, payClient *pay.Client) *Server {
	return &Server{
		logger:        logger,
		cfg:           cfg,
		geoClient:     geoClient,
		driversRepo:   drivers,
		ordersRepo:    orders,
		intercityRepo: intercity,
		offersRepo:    offers,
		paymentsRepo:  payments,
		driverHub:     driverHub,
		passengerHub:  passengerHub,
		dispatcher:    dispatcher,
		payClient:     payClient,
	}
}

// RegisterRoutes registers HTTP routes on mux.
type driverPayload struct {
	UserID        int64  `json:"user_id"`
	Status        string `json:"status"`
	CarModel      string `json:"car_model"`
	CarColor      string `json:"car_color"`
	CarNumber     string `json:"car_number"`
	TechPassport  string `json:"tech_passport"`
	CarPhotoFront string `json:"car_photo_front"`
	CarPhotoBack  string `json:"car_photo_back"`
	CarPhotoLeft  string `json:"car_photo_left"`
	CarPhotoRight string `json:"car_photo_right"`
	DriverPhoto   string `json:"driver_photo"`
	Phone         string `json:"phone"`
	IIN           string `json:"iin"`
	IDCardFront   string `json:"id_card_front"`
	IDCardBack    string `json:"id_card_back"`
}

func (p *driverPayload) normalize() {
	p.Status = strings.TrimSpace(p.Status)
	p.CarModel = strings.TrimSpace(p.CarModel)
	p.CarColor = strings.TrimSpace(p.CarColor)
	p.CarNumber = strings.TrimSpace(p.CarNumber)
	p.TechPassport = strings.TrimSpace(p.TechPassport)
	p.CarPhotoFront = strings.TrimSpace(p.CarPhotoFront)
	p.CarPhotoBack = strings.TrimSpace(p.CarPhotoBack)
	p.CarPhotoLeft = strings.TrimSpace(p.CarPhotoLeft)
	p.CarPhotoRight = strings.TrimSpace(p.CarPhotoRight)
	p.DriverPhoto = strings.TrimSpace(p.DriverPhoto)
	p.Phone = strings.TrimSpace(p.Phone)
	p.IIN = strings.TrimSpace(p.IIN)
	p.IDCardFront = strings.TrimSpace(p.IDCardFront)
	p.IDCardBack = strings.TrimSpace(p.IDCardBack)
	if p.Status == "" {
		p.Status = "offline"
	}
}

func (p driverPayload) validate() string {
	if p.UserID <= 0 {
		return "user_id is required"
	}
	switch p.Status {
	case "offline", "free", "busy":
	default:
		return "invalid status"
	}
	if p.CarNumber == "" {
		return "car_number is required"
	}
	if p.TechPassport == "" {
		return "tech_passport is required"
	}
	if p.CarPhotoFront == "" || p.CarPhotoBack == "" || p.CarPhotoLeft == "" || p.CarPhotoRight == "" {
		return "all car photos are required"
	}
	if p.DriverPhoto == "" {
		return "driver_photo is required"
	}
	if p.Phone == "" {
		return "phone is required"
	}
	if p.IIN == "" {
		return "iin is required"
	}
	if p.IDCardFront == "" || p.IDCardBack == "" {
		return "id card photos are required"
	}
	return ""
}

type driverResponse struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	Status        string    `json:"status"`
	CarModel      string    `json:"car_model,omitempty"`
	CarColor      string    `json:"car_color,omitempty"`
	CarNumber     string    `json:"car_number"`
	TechPassport  string    `json:"tech_passport"`
	CarPhotoFront string    `json:"car_photo_front"`
	CarPhotoBack  string    `json:"car_photo_back"`
	CarPhotoLeft  string    `json:"car_photo_left"`
	CarPhotoRight string    `json:"car_photo_right"`
	DriverPhoto   string    `json:"driver_photo"`
	Phone         string    `json:"phone"`
	IIN           string    `json:"iin"`
	IDCardFront   string    `json:"id_card_front"`
	IDCardBack    string    `json:"id_card_back"`
	Rating        float64   `json:"rating"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func newDriverResponse(d repo.Driver) driverResponse {
	return driverResponse{
		ID:            d.ID,
		UserID:        d.UserID,
		Status:        d.Status,
		CarModel:      d.CarModel.String,
		CarColor:      d.CarColor.String,
		CarNumber:     d.CarNumber,
		TechPassport:  d.TechPassport,
		CarPhotoFront: d.CarPhotoFront,
		CarPhotoBack:  d.CarPhotoBack,
		CarPhotoLeft:  d.CarPhotoLeft,
		CarPhotoRight: d.CarPhotoRight,
		DriverPhoto:   d.DriverPhoto,
		Phone:         d.Phone,
		IIN:           d.IIN,
		IDCardFront:   d.IDCardFront,
		IDCardBack:    d.IDCardBack,
		Rating:        d.Rating,
		UpdatedAt:     d.UpdatedAt,
	}
}

type orderAddressResponse struct {
	ID      int64   `json:"id"`
	Seq     int     `json:"seq"`
	Lon     float64 `json:"lon"`
	Lat     float64 `json:"lat"`
	Address string  `json:"address,omitempty"`
}

type orderResponse struct {
	ID               int64                  `json:"id"`
	PassengerID      int64                  `json:"passenger_id"`
	DriverID         *int64                 `json:"driver_id,omitempty"`
	FromLon          float64                `json:"from_lon"`
	FromLat          float64                `json:"from_lat"`
	ToLon            float64                `json:"to_lon"`
	ToLat            float64                `json:"to_lat"`
	DistanceM        int                    `json:"distance_m"`
	EtaSeconds       int                    `json:"eta_s"`
	RecommendedPrice int                    `json:"recommended_price"`
	ClientPrice      int                    `json:"client_price"`
	PaymentMethod    string                 `json:"payment_method"`
	Status           string                 `json:"status"`
	Notes            string                 `json:"notes,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	Addresses        []orderAddressResponse `json:"addresses"`
	Driver           *driverResponse        `json:"driver,omitempty"`
}

func newOrderResponse(o repo.Order, driver *repo.Driver) orderResponse {
	var driverID *int64
	if o.DriverID.Valid {
		driverID = &o.DriverID.Int64
	}
	resp := orderResponse{
		ID:               o.ID,
		PassengerID:      o.PassengerID,
		DriverID:         driverID,
		FromLon:          o.FromLon,
		FromLat:          o.FromLat,
		ToLon:            o.ToLon,
		ToLat:            o.ToLat,
		DistanceM:        o.DistanceM,
		EtaSeconds:       o.EtaSeconds,
		RecommendedPrice: o.RecommendedPrice,
		ClientPrice:      o.ClientPrice,
		PaymentMethod:    o.PaymentMethod,
		Status:           o.Status,
		CreatedAt:        o.CreatedAt,
		UpdatedAt:        o.UpdatedAt,
	}
	if o.Notes.Valid {
		resp.Notes = o.Notes.String
	}
	if len(o.Addresses) > 0 {
		resp.Addresses = make([]orderAddressResponse, 0, len(o.Addresses))
		for _, addr := range o.Addresses {
			addrResp := orderAddressResponse{
				ID:  addr.ID,
				Seq: addr.Seq,
				Lon: addr.Lon,
				Lat: addr.Lat,
			}
			if addr.Address.Valid {
				addrResp.Address = addr.Address.String
			}
			resp.Addresses = append(resp.Addresses, addrResp)
		}
	}
	if driver != nil {
		d := newDriverResponse(*driver)
		resp.Driver = &d
	}
	return resp
}

var allowedIntercityTripTypes = map[string]struct{}{
	"companions": {},
	"parcel":     {},
	"solo":       {},
}

var allowedIntercityStatuses = map[string]struct{}{
	"open":   {},
	"closed": {},
}

type intercityOrderPayload struct {
	PassengerID   int64  `json:"passenger_id"`
	DriverID      int64  `json:"driver_id"`
	FromLocation  string `json:"from"`
	ToLocation    string `json:"to"`
	TripType      string `json:"trip_type"`
	Comment       string `json:"comment"`
	Price         int    `json:"price"`
	DepartureDate string `json:"departure_date"`
	DepartureTime string `json:"departure_time"`
}

type intercityClosePayload struct {
	PassengerID int64 `json:"passenger_id"`
}

func (p *intercityOrderPayload) normalize() {
	p.FromLocation = strings.TrimSpace(p.FromLocation)
	p.ToLocation = strings.TrimSpace(p.ToLocation)
	p.TripType = strings.TrimSpace(strings.ToLower(p.TripType))
	p.Comment = strings.TrimSpace(p.Comment)
	p.DepartureDate = strings.TrimSpace(p.DepartureDate)
	p.DepartureTime = strings.TrimSpace(p.DepartureTime)
}

func (p intercityOrderPayload) validate() string {
	hasPassenger := p.PassengerID > 0
	hasDriver := p.DriverID > 0
	if hasPassenger == hasDriver {
		return "either passenger_id or driver_id is required"
	}
	if p.FromLocation == "" {
		return "from is required"
	}
	if p.ToLocation == "" {
		return "to is required"
	}
	if _, ok := allowedIntercityTripTypes[p.TripType]; !ok {
		return "invalid trip_type"
	}
	if p.DepartureDate == "" {
		return "departure_date is required"
	}
	if p.DepartureTime != "" {
		if _, err := time.Parse("15:04", p.DepartureTime); err != nil {
			return "invalid departure_time"
		}
	}
	if p.Price < 0 {
		return "price must be >= 0"
	}
	return ""
}

type intercityOrderResponse struct {
	ID            int64                    `json:"id"`
	PassengerID   int64                    `json:"passenger_id"`
	DriverID      *int64                   `json:"driver_id,omitempty"`
	FromLocation  string                   `json:"from"`
	ToLocation    string                   `json:"to"`
	TripType      string                   `json:"trip_type"`
	Comment       string                   `json:"comment,omitempty"`
	Price         int                      `json:"price"`
	ContactPhone  string                   `json:"contact_phone"`
	DepartureDate string                   `json:"departure_date"`
	DepartureTime string                   `json:"departure_time,omitempty"`
	Status        string                   `json:"status"`
	CreatedAt     time.Time                `json:"created_at"`
	UpdatedAt     time.Time                `json:"updated_at"`
	ClosedAt      *time.Time               `json:"closed_at,omitempty"`
	CreatorRole   string                   `json:"creator_role"`
	Driver        *intercityDriverResponse `json:"driver,omitempty"`
}

type intercityDriverResponse struct {
	ID               int64      `json:"id"`
	CarModel         string     `json:"car_model,omitempty"`
	FullName         string     `json:"full_name,omitempty"`
	Rating           *float64   `json:"rating,omitempty"`
	Photo            string     `json:"photo,omitempty"`
	ProfileUpdatedAt *time.Time `json:"profile_updated_at,omitempty"`
}

func newIntercityOrderResponse(o repo.IntercityOrder) intercityOrderResponse {
	resp := intercityOrderResponse{
		ID:            o.ID,
		PassengerID:   0,
		FromLocation:  o.FromLocation,
		ToLocation:    o.ToLocation,
		TripType:      o.TripType,
		Price:         o.Price,
		ContactPhone:  o.ContactPhone,
		DepartureDate: o.DepartureDate.Format("2006-01-02"),
		Status:        o.Status,
		CreatedAt:     o.CreatedAt,
		UpdatedAt:     o.UpdatedAt,
		CreatorRole:   o.CreatorRole,
	}
	if o.PassengerID.Valid {
		resp.PassengerID = o.PassengerID.Int64
	}
	if o.DepartureTime.Valid {
		t := strings.TrimSpace(o.DepartureTime.String)
		if t != "" {
			if parsed, err := time.Parse("15:04:05", t); err == nil {
				resp.DepartureTime = parsed.Format("15:04")
			} else if parsed, err := time.Parse("15:04", t); err == nil {
				resp.DepartureTime = parsed.Format("15:04")
			} else {
				resp.DepartureTime = t
			}
		}
	}
	if o.Comment.Valid {
		resp.Comment = o.Comment.String
	}
	if o.ClosedAt.Valid {
		closedAt := o.ClosedAt.Time
		resp.ClosedAt = &closedAt
	}
	if o.DriverID.Valid {
		resp.DriverID = &o.DriverID.Int64
		driver := intercityDriverResponse{ID: o.DriverID.Int64}
		if o.DriverCarModel.Valid {
			driver.CarModel = o.DriverCarModel.String
		}
		if o.DriverFullName.Valid {
			driver.FullName = strings.TrimSpace(o.DriverFullName.String)
		}
		if o.DriverRating.Valid {
			rating := o.DriverRating.Float64
			driver.Rating = &rating
		}
		if o.DriverPhoto.Valid {
			driver.Photo = o.DriverPhoto.String
		}
		if o.DriverProfileStamp.Valid {
			ts := o.DriverProfileStamp.Time
			driver.ProfileUpdatedAt = &ts
		}
		resp.Driver = &driver
	}
	return resp
}

type intercityListPayload struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Date        string `json:"date"`
	Time        string `json:"time"`
	Status      string `json:"status"`
	PassengerID *int64 `json:"passenger_id"`
	DriverID    *int64 `json:"driver_id"`
	Limit       *int   `json:"limit"`
	Offset      *int   `json:"offset"`
}

func (p *intercityListPayload) normalize() {
	p.From = strings.TrimSpace(p.From)
	p.To = strings.TrimSpace(p.To)
	p.Date = strings.TrimSpace(p.Date)
	p.Time = strings.TrimSpace(p.Time)
	p.Status = strings.TrimSpace(strings.ToLower(p.Status))
}

func (p intercityListPayload) validate() string {
	if p.Limit != nil && *p.Limit < 0 {
		return "invalid limit"
	}
	if p.Offset != nil && *p.Offset < 0 {
		return "invalid offset"
	}
	if p.PassengerID != nil && *p.PassengerID <= 0 {
		return "invalid passenger_id"
	}
	if p.DriverID != nil && *p.DriverID <= 0 {
		return "invalid driver_id"
	}
	if p.Date != "" {
		if _, err := time.Parse("2006-01-02", p.Date); err != nil {
			return "invalid date"
		}
	}
	if p.Time != "" {
		if _, err := time.Parse("15:04", p.Time); err != nil {
			return "invalid time"
		}
	}
	if p.Status != "" && p.Status != "all" {
		if _, ok := allowedIntercityStatuses[p.Status]; !ok {
			return "invalid status"
		}
	}
	return ""
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/drivers", s.handleDrivers)
	mux.HandleFunc("/api/v1/drivers/", s.handleDriver)
	mux.HandleFunc("/api/v1/route/quote", s.handleRouteQuote)
	mux.HandleFunc("/api/v1/orders", s.handleOrders)
	mux.HandleFunc("/api/v1/driver/orders", s.handleDriverOrders)
	mux.HandleFunc("/api/v1/orders/", s.handleOrderSubroutes)
	mux.HandleFunc("/api/v1/intercity/orders", s.handleIntercityOrders)
	mux.HandleFunc("/api/v1/intercity/orders/list", s.listIntercityOrders)
	mux.HandleFunc("/api/v1/intercity/orders/", s.handleIntercityOrderSubroutes)
	mux.HandleFunc("/api/v1/offers/accept", s.handleOfferAccept)
	mux.HandleFunc("/api/v1/payments/airbapay/webhook", s.handleAirbaPayWebhook)
	mux.HandleFunc("/ws/driver", s.handleDriverWS)
	mux.HandleFunc("/ws/passenger", s.handlePassengerWS)
}

func (s *Server) handleDrivers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listDrivers(w, r)
	case http.MethodPost:
		s.createDriver(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleDriver(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/drivers/")
	path = strings.TrimSuffix(path, "/")
	if path == "" || strings.Contains(path, "/") {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.getDriver(w, r, id)
	case http.MethodPut:
		s.updateDriver(w, r, id)
	case http.MethodDelete:
		s.deleteDriver(w, r, id)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) listDrivers(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = n
	}
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, "invalid offset")
			return
		}
		offset = n
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	drivers, err := s.driversRepo.List(ctx, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list drivers failed")
		return
	}
	resp := make([]driverResponse, 0, len(drivers))
	for _, d := range drivers {
		resp = append(resp, newDriverResponse(d))
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"drivers": resp})
}

func (s *Server) createDriver(w http.ResponseWriter, r *http.Request) {
	var payload driverPayload

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(50 << 20); err != nil {
			writeError(w, http.StatusBadRequest, "invalid multipart form")
			return
		}
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
		}

		if userIDStr := strings.TrimSpace(r.FormValue("user_id")); userIDStr != "" {
			userID, err := strconv.ParseInt(userIDStr, 10, 64)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid user_id")
				return
			}
			payload.UserID = userID
		}
		payload.Status = r.FormValue("status")
		payload.CarModel = r.FormValue("car_model")
		payload.CarColor = r.FormValue("car_color")
		payload.CarNumber = r.FormValue("car_number")
		payload.Phone = r.FormValue("phone")
		payload.IIN = r.FormValue("iin")

		var err error
		if payload.TechPassport, err = saveDriverAsset(r, "tech_passport", "TechPassport"); err != nil {
			s.logger.Errorf("failed to save tech_passport: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to save tech_passport")
			return
		}
		if payload.CarPhotoFront, err = saveDriverAsset(r, "car_photo_front", "CarPhotoFront"); err != nil {
			s.logger.Errorf("failed to save car_photo_front: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to save car_photo_front")
			return
		}
		if payload.CarPhotoBack, err = saveDriverAsset(r, "car_photo_back", "CarPhotoBack"); err != nil {
			s.logger.Errorf("failed to save car_photo_back: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to save car_photo_back")
			return
		}
		if payload.CarPhotoLeft, err = saveDriverAsset(r, "car_photo_left", "CarPhotoLeft"); err != nil {
			s.logger.Errorf("failed to save car_photo_left: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to save car_photo_left")
			return
		}
		if payload.CarPhotoRight, err = saveDriverAsset(r, "car_photo_right", "CarPhotoRight"); err != nil {
			s.logger.Errorf("failed to save car_photo_right: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to save car_photo_right")
			return
		}
		if payload.DriverPhoto, err = saveDriverAsset(r, "driver_photo", "DriverPhoto"); err != nil {
			s.logger.Errorf("failed to save driver_photo: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to save driver_photo")
			return
		}
		if payload.IDCardFront, err = saveDriverAsset(r, "id_card_front", "IDCardFront"); err != nil {
			s.logger.Errorf("failed to save id_card_front: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to save id_card_front")
			return
		}
		if payload.IDCardBack, err = saveDriverAsset(r, "id_card_back", "IDCardBack"); err != nil {
			s.logger.Errorf("failed to save id_card_back: %v", err)
			writeError(w, http.StatusInternalServerError, "failed to save id_card_back")
			return
		}
	} else {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
	}

	payload.normalize()
	if msg := payload.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	driver := repo.Driver{
		UserID:        payload.UserID,
		Status:        payload.Status,
		CarModel:      toNullString(payload.CarModel),
		CarColor:      toNullString(payload.CarColor),
		CarNumber:     payload.CarNumber,
		TechPassport:  payload.TechPassport,
		CarPhotoFront: payload.CarPhotoFront,
		CarPhotoBack:  payload.CarPhotoBack,
		CarPhotoLeft:  payload.CarPhotoLeft,
		CarPhotoRight: payload.CarPhotoRight,
		DriverPhoto:   payload.DriverPhoto,
		Phone:         payload.Phone,
		IIN:           payload.IIN,
		IDCardFront:   payload.IDCardFront,
		IDCardBack:    payload.IDCardBack,
	}

	id, err := s.driversRepo.Create(ctx, driver)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create driver failed")
		return
	}
	driver, err = s.driversRepo.Get(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "fetch driver failed")
		return
	}
	writeJSON(w, http.StatusCreated, newDriverResponse(driver))
}

func (s *Server) getDriver(w http.ResponseWriter, r *http.Request, id int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	driver, err := s.driversRepo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "driver not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "fetch driver failed")
		return
	}
	writeJSON(w, http.StatusOK, newDriverResponse(driver))
}

func (s *Server) updateDriver(w http.ResponseWriter, r *http.Request, id int64) {
	var payload driverPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	payload.normalize()
	if msg := payload.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	driver := repo.Driver{
		ID:            id,
		UserID:        payload.UserID,
		Status:        payload.Status,
		CarModel:      toNullString(payload.CarModel),
		CarColor:      toNullString(payload.CarColor),
		CarNumber:     payload.CarNumber,
		TechPassport:  payload.TechPassport,
		CarPhotoFront: payload.CarPhotoFront,
		CarPhotoBack:  payload.CarPhotoBack,
		CarPhotoLeft:  payload.CarPhotoLeft,
		CarPhotoRight: payload.CarPhotoRight,
		DriverPhoto:   payload.DriverPhoto,
		Phone:         payload.Phone,
		IIN:           payload.IIN,
		IDCardFront:   payload.IDCardFront,
		IDCardBack:    payload.IDCardBack,
	}

	if err := s.driversRepo.Update(ctx, driver); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "driver not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "update driver failed")
		return
	}

	driver, err := s.driversRepo.Get(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "fetch driver failed")
		return
	}
	writeJSON(w, http.StatusOK, newDriverResponse(driver))
}

func saveDriverAsset(r *http.Request, field string, alt ...string) (string, error) {
	keys := append([]string{field}, alt...)
	for _, key := range keys {
		if v := strings.TrimSpace(r.FormValue(key)); v != "" {
			return v, nil
		}
	}

	for _, key := range keys {
		file, header, err := r.FormFile(key)
		if err != nil {
			if errors.Is(err, http.ErrMissingFile) {
				continue
			}
			return "", err
		}

		dirName := strings.ToLower(field)
		dirName = strings.ReplaceAll(dirName, " ", "_")
		saveDir := filepath.Join("uploads", "taxi", dirName)
		if err := os.MkdirAll(saveDir, 0o755); err != nil {
			file.Close()
			return "", err
		}

		ext := filepath.Ext(header.Filename)
		safeField := strings.ReplaceAll(dirName, " ", "_")
		filename := fmt.Sprintf("%s_%d%s", safeField, time.Now().UnixNano(), ext)
		diskPath := filepath.Join(saveDir, filename)

		dst, err := os.Create(diskPath)
		if err != nil {
			file.Close()
			return "", err
		}

		if _, err := io.Copy(dst, file); err != nil {
			dst.Close()
			file.Close()
			_ = os.Remove(diskPath)
			return "", err
		}

		dst.Close()
		file.Close()

		publicPath := "/" + filepath.ToSlash(filepath.Join("uploads", "taxi", dirName, filename))
		return publicPath, nil
	}

	return "", nil
}

func (s *Server) deleteDriver(w http.ResponseWriter, r *http.Request, id int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := s.driversRepo.Delete(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "driver not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "delete driver failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func toNullString(v string) sql.NullString {
	if v == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: v, Valid: true}
}

func roundDownToStep(n, step int) int {
	if step <= 0 {
		return n
	}
	if n < 0 {
		// на всякий случай корректно обрабатываем отрицательные
		return -roundDownToStep(-n, step)
	}
	return (n / step) * step
}

func (s *Server) handleRouteQuote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	type quotePoint struct {
		Lon     float64 `json:"lon"`
		Lat     float64 `json:"lat"`
		Address string  `json:"address"`
	}

	var req struct {
		FromAddress string       `json:"from_address"`
		ToAddress   string       `json:"to_address"`
		From        *quotePoint  `json:"from"`
		To          *quotePoint  `json:"to"`
		Stops       []quotePoint `json:"stops"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 7*time.Second)
	defer cancel()

	// helper: получить lon/lat либо из координат, либо геокодировать адрес
	type resolvedPoint struct {
		lon     float64
		lat     float64
		address string
	}

	resolvePoint := func(fallbackAddr string, pt *quotePoint) (resolvedPoint, error) {
		var (
			addr   = strings.TrimSpace(fallbackAddr)
			hasPt  = pt != nil
			result resolvedPoint
		)

		if hasPt {
			if pt.Lon != 0 && pt.Lat != 0 {
				result.lon = pt.Lon
				result.lat = pt.Lat
				result.address = strings.TrimSpace(pt.Address)
				if result.address == "" {
					result.address = addr
				}
				return result, nil
			}
			if trimmed := strings.TrimSpace(pt.Address); trimmed != "" {
				addr = trimmed
			}
		}

		if addr != "" {
			lon, lat, err := s.geoClient.Geocode(ctx, addr)
			if err != nil {
				return resolvedPoint{}, err
			}
			result.lon = lon
			result.lat = lat
			result.address = addr
			return result, nil
		}

		return resolvedPoint{}, errors.New("point required: pass either coordinates or address")
	}

	points := make([]resolvedPoint, 0, len(req.Stops)+2)

	from, err := resolvePoint(req.FromAddress, req.From)
	if err != nil {
		writeError(w, http.StatusBadRequest, "from: "+err.Error())
		return
	}
	points = append(points, from)

	for i := range req.Stops {
		stop := req.Stops[i]
		resolved, err := resolvePoint(stop.Address, &stop)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("stop %d: %v", i, err))
			return
		}
		points = append(points, resolved)
	}

	to, err := resolvePoint(req.ToAddress, req.To)
	if err != nil {
		writeError(w, http.StatusBadRequest, "to: "+err.Error())
		return
	}
	points = append(points, to)

	if len(points) < 2 {
		writeError(w, http.StatusBadRequest, "at least two points required")
		return
	}

	// Лог на всякий случай
	fmt.Println("RouteQuote points:")
	for idx, p := range points {
		fmt.Printf("  #%d: %.6f %.6f\n", idx, p.lon, p.lat)
	}

	totalDistance := 0
	totalEta := 0
	for i := 0; i < len(points)-1; i++ {
		distance, eta, err := s.geoClient.RouteMatrix(ctx, points[i].lon, points[i].lat, points[i+1].lon, points[i+1].lat)
		if err != nil {
			// Оставляю подробный ответ — удобно для дебага.
			writeError(w, http.StatusBadGateway, fmt.Sprintf("route matrix failed: %v", err))
			return
		}
		totalDistance += distance
		totalEta += eta
	}

	rec := pricing.Recommended(totalDistance, s.cfg.GetPricePerKM(), s.cfg.GetMinPrice())
	minPrice := s.cfg.GetMinPrice()
	if rec <= minPrice {
		rec = minPrice // не опускаем ниже минимума
	} else {
		rec = roundDownToStep(rec, 50) // округляем вниз до 50
		if rec < minPrice {
			rec = minPrice
		}
	}
	makePayloadPoint := func(p resolvedPoint) map[string]interface{} {
		point := map[string]interface{}{"lon": p.lon, "lat": p.lat}
		if p.address != "" {
			point["address"] = p.address
		}
		return point
	}

	resp := map[string]interface{}{
		"from":              makePayloadPoint(points[0]),
		"to":                makePayloadPoint(points[len(points)-1]),
		"distance_m":        totalDistance,
		"eta_s":             totalEta,
		"recommended_price": rec,
		"min_price":         s.cfg.GetMinPrice(),
	}
	if len(points) > 2 {
		stops := make([]map[string]interface{}, 0, len(points)-2)
		for _, p := range points[1 : len(points)-1] {
			stops = append(stops, makePayloadPoint(p))
		}
		resp["stops"] = stops
	}

	writeJSON(w, http.StatusOK, resp)
}
func (s *Server) handleOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListOrders(w, r)
	case http.MethodPost:
		s.handleCreateOrder(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleDriverOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListDriverOrders(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleListOrders(w http.ResponseWriter, r *http.Request) {
	passengerID, err := parseAuthID(r, "X-Passenger-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing passenger id")
		return
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = n
	}
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, "invalid offset")
			return
		}
		offset = n
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	orders, err := s.ordersRepo.ListByPassenger(ctx, passengerID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list orders failed")
		return
	}

	driverCache := make(map[int64]repo.Driver)
	resp := make([]orderResponse, 0, len(orders))
	for _, order := range orders {
		var driver *repo.Driver
		if order.DriverID.Valid {
			if cached, ok := driverCache[order.DriverID.Int64]; ok {
				driver = &cached
			} else {
				d, err := s.driversRepo.Get(ctx, order.DriverID.Int64)
				if err == nil {
					driverCache[order.DriverID.Int64] = d
					driver = &d
				}
			}
		}
		resp = append(resp, newOrderResponse(order, driver))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"orders": resp, "limit": limit, "offset": offset})
}

func (s *Server) handleListDriverOrders(w http.ResponseWriter, r *http.Request) {
	driverID, err := parseAuthID(r, "X-Driver-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing driver id")
		return
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = n
	}
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, "invalid offset")
			return
		}
		offset = n
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	orders, err := s.ordersRepo.ListByDriver(ctx, driverID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list orders failed")
		return
	}

	var driver *repo.Driver
	if d, err := s.driversRepo.Get(ctx, driverID); err == nil {
		driver = &d
	}

	resp := make([]orderResponse, 0, len(orders))
	for _, order := range orders {
		resp = append(resp, newOrderResponse(order, driver))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"orders": resp, "limit": limit, "offset": offset})
}

func (s *Server) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	passengerID, err := parseAuthID(r, "X-Passenger-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing passenger id")
		return
	}
	var req struct {
		From struct {
			Lon     float64 `json:"lon"`
			Lat     float64 `json:"lat"`
			Address string  `json:"address"`
		} `json:"from"`
		To struct {
			Lon     float64 `json:"lon"`
			Lat     float64 `json:"lat"`
			Address string  `json:"address"`
		} `json:"to"`
		Stops []struct {
			Lon     float64 `json:"lon"`
			Lat     float64 `json:"lat"`
			Address string  `json:"address"`
		} `json:"stops"`
		DistanceM     int    `json:"distance_m"`
		EtaSeconds    int    `json:"eta_s"`
		ClientPrice   int    `json:"client_price"`
		PaymentMethod string `json:"payment_method"`
		Notes         string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.ClientPrice < s.cfg.GetMinPrice() {
		writeError(w, http.StatusBadRequest, "price below minimum")
		return
	}
	if req.PaymentMethod != "online" && req.PaymentMethod != "cash" {
		writeError(w, http.StatusBadRequest, "invalid payment method")
		return
	}

	type waypoint struct {
		lon     float64
		lat     float64
		address string
	}
	points := []waypoint{{lon: req.From.Lon, lat: req.From.Lat, address: strings.TrimSpace(req.From.Address)}}
	for i, stop := range req.Stops {
		if stop.Lon == 0 && stop.Lat == 0 {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("stop %d has empty coordinates", i))
			return
		}
		points = append(points, waypoint{lon: stop.Lon, lat: stop.Lat, address: strings.TrimSpace(stop.Address)})
	}
	points = append(points, waypoint{lon: req.To.Lon, lat: req.To.Lat, address: strings.TrimSpace(req.To.Address)})
	if len(points) < 2 {
		writeError(w, http.StatusBadRequest, "at least two points required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	totalDistance := 0
	totalEta := 0
	for i := 0; i < len(points)-1; i++ {
		distance, eta, err := s.geoClient.RouteMatrix(ctx, points[i].lon, points[i].lat, points[i+1].lon, points[i+1].lat)
		if err != nil {
			writeError(w, http.StatusBadGateway, "route validation failed")
			return
		}
		totalDistance += distance
		totalEta += eta
	}

	if !validateInt(totalDistance, req.DistanceM) {
		writeError(w, http.StatusBadRequest, "distance mismatch")
		return
	}
	if !validateInt(totalEta, req.EtaSeconds) {
		writeError(w, http.StatusBadRequest, "eta mismatch")
		return
	}

	rec := pricing.Recommended(totalDistance, s.cfg.GetPricePerKM(), s.cfg.GetMinPrice())
	minPrice := s.cfg.GetMinPrice()
	if rec <= minPrice {
		rec = minPrice
	} else {
		rec = roundDownToStep(rec, 50)
		if rec < minPrice {
			rec = minPrice
		}
	}
	order := repo.Order{
		PassengerID:      passengerID,
		FromLon:          req.From.Lon,
		FromLat:          req.From.Lat,
		ToLon:            req.To.Lon,
		ToLat:            req.To.Lat,
		DistanceM:        totalDistance,
		EtaSeconds:       totalEta,
		RecommendedPrice: rec,
		ClientPrice:      req.ClientPrice,
		PaymentMethod:    req.PaymentMethod,
	}
	if req.Notes != "" {
		order.Notes = sql.NullString{String: req.Notes, Valid: true}
	}

	addresses := make([]repo.OrderAddress, 0, len(points))
	for idx, point := range points {
		addr := repo.OrderAddress{Seq: idx, Lon: point.lon, Lat: point.lat}
		if point.address != "" {
			addr.Address = sql.NullString{String: point.address, Valid: true}
		}
		addresses = append(addresses, addr)
	}
	order.Addresses = addresses

	dispatchRec := repo.DispatchRecord{RadiusM: s.cfg.GetSearchRadiusStart(), NextTickAt: timeutil.Now(), State: "searching"}
	orderID, err := s.ordersRepo.CreateWithDispatch(ctx, order, dispatchRec)
	if err != nil {
		s.logger.Errorf("create order failed: %v", err)
		writeError(w, http.StatusInternalServerError, "create failed")
		return
	}

	if s.dispatcher != nil {
		_ = s.dispatcher.TriggerImmediate(context.Background(), orderID)
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{"order_id": orderID, "recommended_price": rec})
}

func validateInt(expected, actual int) bool {
	if actual == 0 {
		return true
	}
	diff := expected - actual
	if diff < 0 {
		diff = -diff
	}
	return diff <= expected/10+1
}

func (s *Server) handleOrderSubroutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/orders/")
	path = strings.Trim(path, "/")
	if path == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	parts := strings.Split(path, "/")
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if len(parts) == 1 {
		if r.Method == http.MethodGet {
			s.handleGetOrder(w, r, id)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	switch parts[1] {
	case "reprice":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleReprice(w, r, id)
	case "status":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleStatus(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request, orderID int64) {
	passengerID, err := parseAuthID(r, "X-Passenger-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing passenger id")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	order, driver, err := s.ordersRepo.GetWithDriver(ctx, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "fetch failed")
		return
	}
	if order.PassengerID != passengerID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}
	writeJSON(w, http.StatusOK, newOrderResponse(order, driver))
}

func (s *Server) handleReprice(w http.ResponseWriter, r *http.Request, orderID int64) {
	var req struct {
		ClientPrice int `json:"client_price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.ClientPrice < s.cfg.GetMinPrice() {
		writeError(w, http.StatusBadRequest, "price below minimum")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	order, err := s.ordersRepo.Get(ctx, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "fetch order failed")
		return
	}
	if err := s.ordersRepo.UpdatePrice(ctx, orderID, order.ClientPrice, req.ClientPrice); err != nil {
		writeError(w, http.StatusInternalServerError, "update price failed")
		return
	}
	if s.dispatcher != nil {
		_ = s.dispatcher.TriggerImmediate(context.Background(), orderID)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"order_id": orderID, "client_price": req.ClientPrice})
}

func (s *Server) handleIntercityOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.createIntercityOrder(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleIntercityOrderSubroutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/intercity/orders/")
	path = strings.Trim(path, "/")
	if path == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	parts := strings.Split(path, "/")
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if len(parts) == 1 {
		if r.Method == http.MethodGet {
			s.getIntercityOrder(w, r, id)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if len(parts) == 2 && parts[1] == "close" {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.closeIntercityOrder(w, r, id)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func (s *Server) listIntercityOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var payload intercityListPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		if !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
	}
	payload.normalize()
	if msg := payload.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	limit := 100
	if payload.Limit != nil {
		limit = *payload.Limit
	}
	offset := 0
	if payload.Offset != nil {
		offset = *payload.Offset
	}

	filter := repo.IntercityOrdersFilter{
		Limit:  limit,
		Offset: offset,
		From:   payload.From,
		To:     payload.To,
	}

	if payload.PassengerID != nil {
		filter.PassengerID = *payload.PassengerID
	}
	if payload.DriverID != nil {
		filter.DriverID = *payload.DriverID
	}
	if payload.Date != "" {
		date, _ := time.Parse("2006-01-02", payload.Date)
		filter.Date = &date
	}
	if payload.Time != "" {
		departureTime, _ := time.Parse("15:04", payload.Time)
		filter.Time = &departureTime
	}

	status := payload.Status
	if status == "" {
		filter.Status = "open"
	} else if status == "all" {
		filter.Status = ""
	} else {
		filter.Status = status
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	orders, err := s.intercityRepo.List(ctx, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list failed")
		return
	}

	resp := make([]intercityOrderResponse, 0, len(orders))
	for _, order := range orders {
		resp = append(resp, newIntercityOrderResponse(order))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"orders": resp, "limit": limit, "offset": offset})
}

func (s *Server) createIntercityOrder(w http.ResponseWriter, r *http.Request) {
	var payload intercityOrderPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	payload.normalize()
	if msg := payload.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	departure, err := time.Parse("2006-01-02", payload.DepartureDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid departure_date")
		return
	}
	order := repo.IntercityOrder{
		FromLocation:  payload.FromLocation,
		ToLocation:    payload.ToLocation,
		TripType:      payload.TripType,
		Price:         payload.Price,
		DepartureDate: departure,
		Status:        "open",
	}
	if payload.PassengerID > 0 {
		order.PassengerID = sql.NullInt64{Int64: payload.PassengerID, Valid: true}
		order.CreatorRole = "passenger"
	}
	if payload.DriverID > 0 {
		order.DriverID = sql.NullInt64{Int64: payload.DriverID, Valid: true}
		order.CreatorRole = "driver"
	}
	if payload.DepartureTime != "" {
		departureTime, err := time.Parse("15:04", payload.DepartureTime)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid departure_time")
			return
		}
		order.DepartureTime = sql.NullString{String: departureTime.Format("15:04:05"), Valid: true}
	}
	if payload.Comment != "" {
		order.Comment = sql.NullString{String: payload.Comment, Valid: true}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	id, err := s.intercityRepo.Create(ctx, order)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	created, err := s.intercityRepo.Get(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "fetch failed")
		return
	}

	resp := newIntercityOrderResponse(created)
	event := ws.IntercityEvent{Type: "intercity_order", Action: "created", Order: resp}
	s.passengerHub.BroadcastEvent(event)
	s.driverHub.BroadcastEvent(event)

	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) getIntercityOrder(w http.ResponseWriter, r *http.Request, id int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	order, err := s.intercityRepo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "fetch failed")
		return
	}
	writeJSON(w, http.StatusOK, newIntercityOrderResponse(order))
}

func (s *Server) closeIntercityOrder(w http.ResponseWriter, r *http.Request, id int64) {
	var payload intercityClosePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if payload.PassengerID <= 0 {
		writeError(w, http.StatusBadRequest, "passenger_id is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := s.intercityRepo.Close(ctx, id, payload.PassengerID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "close failed")
		return
	}
	order, err := s.intercityRepo.Get(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "fetch failed")
		return
	}

	resp := newIntercityOrderResponse(order)
	event := ws.IntercityEvent{Type: "intercity_order", Action: "closed", Order: resp}
	s.passengerHub.BroadcastEvent(event)
	s.driverHub.BroadcastEvent(event)

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleOfferAccept(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	driverID, err := parseAuthID(r, "X-Driver-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing driver id")
		return
	}
	var req struct {
		OrderID int64 `json:"order_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, err := s.driversRepo.Get(ctx, driverID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, "driver not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "driver lookup failed")
		return
	}

	closedDrivers, err := s.offersRepo.AcceptOffer(ctx, req.OrderID, driverID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusConflict, "offer not available")
			return
		}
		writeError(w, http.StatusInternalServerError, "accept failed")
		return
	}
	if err := s.ordersRepo.AssignDriver(ctx, req.OrderID, driverID); err != nil {
		writeError(w, http.StatusInternalServerError, "assign failed")
		return
	}

	order, err := s.ordersRepo.Get(ctx, req.OrderID)
	if err == nil {
		s.passengerHub.PushOrderEvent(order.PassengerID, ws.PassengerEvent{Type: "order_assigned", OrderID: order.ID, Status: "accepted"})
	}

	if len(closedDrivers) > 0 {
		s.driverHub.NotifyOfferClosed(req.OrderID, closedDrivers, "accepted_by_other")
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request, orderID int64) {
	var req struct {
		Status string   `json:"status"`
		Lon    *float64 `json:"lon"`
		Lat    *float64 `json:"lat"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	order, err := s.ordersRepo.Get(ctx, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "fetch failed")
		return
	}
	if !fsm.CanTransition(order.Status, req.Status) {
		writeError(w, http.StatusConflict, "invalid transition")
		return
	}

	if req.Status == "completed" {
		if !order.DriverID.Valid {
			writeError(w, http.StatusBadRequest, "driver not assigned")
			return
		}
		if req.Lon == nil || req.Lat == nil {
			writeError(w, http.StatusBadRequest, "location required to complete")
			return
		}
		if distanceMeters(order.ToLon, order.ToLat, *req.Lon, *req.Lat) > 300 {
			writeError(w, http.StatusBadRequest, "driver location mismatch")
			return
		}
	}

	if err := s.ordersRepo.UpdateStatusCAS(ctx, orderID, order.Status, req.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusConflict, "order status changed")
			return
		}
		writeError(w, http.StatusInternalServerError, "update status failed")
		return
	}

	s.passengerHub.PushOrderEvent(order.PassengerID, ws.PassengerEvent{Type: "order_status", OrderID: order.ID, Status: req.Status})

	if req.Status == "completed" && order.PaymentMethod == "online" && s.payClient != nil {
		go s.createPayment(orderID, order.ClientPrice)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": req.Status})
}

func distanceMeters(lon1, lat1, lon2, lat2 float64) float64 {
	const earthRadius = 6371000.0
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c
}

func (s *Server) createPayment(orderID int64, amount int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	payload, _ := json.Marshal(map[string]interface{}{"order_id": orderID, "amount": amount})
	paymentID, err := s.paymentsRepo.Create(ctx, orderID, amount, "airbapay", payload)
	if err != nil {
		s.logger.Errorf("create payment record failed: %v", err)
		return
	}
	if s.payClient == nil {
		return
	}
	resp, err := s.payClient.CreatePayment(ctx, pay.CreatePaymentRequest{OrderID: orderID, Amount: amount, Currency: "KZT", Description: "Taxi ride"})
	if err != nil {
		s.logger.Errorf("airbapay request failed: %v", err)
		_ = s.paymentsRepo.UpdateState(ctx, paymentID, "failed", "")
		return
	}
	_ = s.paymentsRepo.UpdateState(ctx, paymentID, "created", resp.InvoiceID)
}

func (s *Server) handleAirbaPayWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	signature := r.Header.Get("X-AirbaPay-Signature")
	if signature == "" {
		writeError(w, http.StatusBadRequest, "missing signature")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	secret := ""
	if s.payClient != nil {
		secret = s.payClient.Secret()
	}
	if secret == "" || !pay.VerifyHMAC(body, signature, secret) {
		writeError(w, http.StatusUnauthorized, "invalid signature")
		return
	}
	var payload struct {
		OrderID int64  `json:"order_id"`
		Status  string `json:"status"`
		TxnID   string `json:"transaction_id"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := s.paymentsRepo.SaveWebhook(ctx, "airbapay", signature, body); err != nil {
		s.logger.Errorf("save webhook failed: %v", err)
	}
	if payload.Status == "paid" {
		if err := s.ordersRepo.UpdateStatusCAS(ctx, payload.OrderID, "completed", "paid"); err != nil {
			s.logger.Errorf("order paid update failed: %v", err)
		}
		if err := s.paymentsRepo.UpdateStateByOrder(ctx, payload.OrderID, "paid", payload.TxnID); err != nil {
			s.logger.Errorf("payment state update failed: %v", err)
		}
		if order, err := s.ordersRepo.Get(ctx, payload.OrderID); err == nil {
			s.passengerHub.PushOrderEvent(order.PassengerID, ws.PassengerEvent{Type: "order_status", OrderID: order.ID, Status: "paid"})
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleDriverWS(w http.ResponseWriter, r *http.Request) {
	s.driverHub.ServeWS(w, r)
}

func (s *Server) handlePassengerWS(w http.ResponseWriter, r *http.Request) {
	s.passengerHub.ServeWS(w, r)
}

func parseAuthID(r *http.Request, header string) (int64, error) {
	v := r.Header.Get(header)
	if v == "" {
		return 0, errors.New("missing header")
	}
	return strconv.ParseInt(v, 10, 64)
}
