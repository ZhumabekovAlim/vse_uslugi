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
	ordersRepo   *repo.OrdersRepo
	offersRepo   *repo.OffersRepo
	paymentsRepo *repo.PaymentsRepo
	driverHub    *ws.DriverHub
	passengerHub *ws.PassengerHub
	dispatcher   *dispatch.Dispatcher
	payClient    *pay.Client
}

// NewServer constructs Server.
func NewServer(logger dispatch.Logger, cfg dispatch.Config, geoClient *geo.DGISClient, orders *repo.OrdersRepo, offers *repo.OffersRepo, payments *repo.PaymentsRepo, driverHub *ws.DriverHub, passengerHub *ws.PassengerHub, dispatcher *dispatch.Dispatcher, payClient *pay.Client) *Server {
	return &Server{
		logger:       logger,
		cfg:          cfg,
		geoClient:    geoClient,
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
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/route/quote", s.handleRouteQuote)
	mux.HandleFunc("/api/v1/orders", s.handleOrders)
	mux.HandleFunc("/api/v1/orders/", s.handleOrderSubroutes)
	mux.HandleFunc("/api/v1/offers/accept", s.handleOfferAccept)
	mux.HandleFunc("/api/v1/payments/airbapay/webhook", s.handleAirbaPayWebhook)
	mux.HandleFunc("/ws/driver", s.handleDriverWS)
	mux.HandleFunc("/ws/passenger", s.handlePassengerWS)
}

func (s *Server) handleRouteQuote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		FromAddress string `json:"from_address"`
		ToAddress   string `json:"to_address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	fromLon, fromLat, err := s.geoClient.Geocode(ctx, req.FromAddress)
	if err != nil {
		writeError(w, http.StatusBadGateway, "geocode from failed")
		return
	}
	toLon, toLat, err := s.geoClient.Geocode(ctx, req.ToAddress)
	if err != nil {
		writeError(w, http.StatusBadGateway, "geocode to failed")
		return
	}
	fmt.Println("From:", fromLon, fromLat, "To:", toLon, toLat)
	distance, eta, err := s.geoClient.RouteMatrix(ctx, fromLon, fromLat, toLon, toLat)
	if err != nil {
		// ВРЕМЕННО на отладку: отдаём подробности
		writeError(w, http.StatusBadGateway, fmt.Sprintf("route matrix failed: %v", err))
		return
	}
	rec := pricing.Recommended(distance, s.cfg.GetPricePerKM(), s.cfg.GetMinPrice())
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
			Lon float64 `json:"lon"`
			Lat float64 `json:"lat"`
		} `json:"from"`
		To struct {
			Lon float64 `json:"lon"`
			Lat float64 `json:"lat"`
		} `json:"to"`
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

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	distance, eta, err := s.geoClient.RouteMatrix(ctx, req.From.Lon, req.From.Lat, req.To.Lon, req.To.Lat)
	if err != nil {
		writeError(w, http.StatusBadGateway, "route validation failed")
		return
	}
	if !validateInt(distance, req.DistanceM) {
		writeError(w, http.StatusBadRequest, "distance mismatch")
		return
	}
	if !validateInt(eta, req.EtaSeconds) {
		writeError(w, http.StatusBadRequest, "eta mismatch")
		return
	}

	rec := pricing.Recommended(distance, s.cfg.GetPricePerKM(), s.cfg.GetMinPrice())
	order := repo.Order{
		PassengerID:      passengerID,
		FromLon:          req.From.Lon,
		FromLat:          req.From.Lat,
		ToLon:            req.To.Lon,
		ToLat:            req.To.Lat,
		DistanceM:        distance,
		EtaSeconds:       eta,
		RecommendedPrice: rec,
		ClientPrice:      req.ClientPrice,
		PaymentMethod:    req.PaymentMethod,
	}
	if req.Notes != "" {
		order.Notes = sql.NullString{String: req.Notes, Valid: true}
	}

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
