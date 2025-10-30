package taxihttp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"naimuBack/internal/taxi/dispatch"
	"naimuBack/internal/taxi/fsm"
	"naimuBack/internal/taxi/geo"
	"naimuBack/internal/taxi/pay"
	"naimuBack/internal/taxi/pricing"
	"naimuBack/internal/taxi/repo"
	"naimuBack/internal/taxi/ws"
)

// Server handles HTTP endpoints for taxi module.
type Server struct {
	logger       dispatch.Logger
	cfg          dispatch.Config
	geoClient    *geo.DGISClient
	driversRepo  *repo.DriversRepo
	ordersRepo   *repo.OrdersRepo
	offersRepo   *repo.OffersRepo
	paymentsRepo *repo.PaymentsRepo
	driverHub    *ws.DriverHub
	passengerHub *ws.PassengerHub
	dispatcher   *dispatch.Dispatcher
	payClient    *pay.Client
}

// NewServer constructs Server.
func NewServer(logger dispatch.Logger, cfg dispatch.Config, geoClient *geo.DGISClient, drivers *repo.DriversRepo, orders *repo.OrdersRepo, offers *repo.OffersRepo, payments *repo.PaymentsRepo, driverHub *ws.DriverHub, passengerHub *ws.PassengerHub, dispatcher *dispatch.Dispatcher, payClient *pay.Client) *Server {
	return &Server{
		logger:       logger,
		cfg:          cfg,
		geoClient:    geoClient,
		driversRepo:  drivers,
		ordersRepo:   orders,
		offersRepo:   offers,
		paymentsRepo: payments,
		driverHub:    driverHub,
		passengerHub: passengerHub,
		dispatcher:   dispatcher,
		payClient:    payClient,
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

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/drivers", s.handleDrivers)
	mux.HandleFunc("/api/v1/drivers/", s.handleDriver)
	mux.HandleFunc("/api/v1/route/quote", s.handleRouteQuote)
	mux.HandleFunc("/api/v1/orders", s.handleOrders)
	mux.HandleFunc("/api/v1/orders/", s.handleOrderSubroutes)
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

	var req struct {
		FromAddress string `json:"from_address"`
		ToAddress   string `json:"to_address"`
		From        *struct {
			Lon float64 `json:"lon"`
			Lat float64 `json:"lat"`
		} `json:"from"`
		To *struct {
			Lon float64 `json:"lon"`
			Lat float64 `json:"lat"`
		} `json:"to"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 7*time.Second)
	defer cancel()

	// helper: получить lon/lat либо из координат, либо геокодировать адрес
	resolvePoint := func(addr string, pt *struct {
		Lon float64 `json:"lon"`
		Lat float64 `json:"lat"`
	}) (lon, lat float64, err error) {
		// Если пришли валидные координаты — используем их
		if pt != nil && pt.Lon != 0 && pt.Lat != 0 {
			return pt.Lon, pt.Lat, nil
		}
		// Иначе, если есть адрес — геокодируем
		if strings.TrimSpace(addr) != "" {
			return s.geoClient.Geocode(ctx, addr)
		}
		// Ни координат, ни адреса — ошибка
		return 0, 0, errors.New("point required: pass either coordinates or address")
	}

	fromLon, fromLat, err := resolvePoint(req.FromAddress, req.From)
	if err != nil {
		writeError(w, http.StatusBadRequest, "from: "+err.Error())
		return
	}
	toLon, toLat, err := resolvePoint(req.ToAddress, req.To)
	if err != nil {
		writeError(w, http.StatusBadRequest, "to: "+err.Error())
		return
	}

	// Лог на всякий случай
	fmt.Println("RouteQuote FROM:", fromLon, fromLat, "TO:", toLon, toLat)

	distance, eta, err := s.geoClient.RouteMatrix(ctx, fromLon, fromLat, toLon, toLat)
	if err != nil {
		// Оставляю подробный ответ — удобно для дебага.
		writeError(w, http.StatusBadGateway, fmt.Sprintf("route matrix failed: %v", err))
		return
	}

	rec := pricing.Recommended(distance, s.cfg.GetPricePerKM(), s.cfg.GetMinPrice())
	minPrice := s.cfg.GetMinPrice()
	if rec <= minPrice {
		rec = minPrice // не опускаем ниже минимума
	} else {
		rec = roundDownToStep(rec, 50) // округляем вниз до 50
		if rec < minPrice {
			rec = minPrice
		}
	}
	resp := map[string]interface{}{
		"from":              map[string]float64{"lon": fromLon, "lat": fromLat},
		"to":                map[string]float64{"lon": toLon, "lat": toLat},
		"distance_m":        distance,
		"eta_s":             eta,
		"recommended_price": rec,
		"min_price":         s.cfg.GetMinPrice(),
	}

	writeJSON(w, http.StatusOK, resp)
}
func (s *Server) handleOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleCreateOrder(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
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

	dispatchRec := repo.DispatchRecord{RadiusM: s.cfg.GetSearchRadiusStart(), NextTickAt: time.Now(), State: "searching"}
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
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
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

	if err := s.offersRepo.AcceptOffer(ctx, req.OrderID, driverID); err != nil {
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

	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request, orderID int64) {
	var req struct {
		Status string `json:"status"`
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
