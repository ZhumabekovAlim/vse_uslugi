package http

import (
	"net/http"

	"naimuBack/internal/courier/repo"
	"naimuBack/internal/courier/ws"
)

// Config is the subset of runtime configuration required by the HTTP handlers.
type Config struct {
	PricePerKM int
	MinPrice   int
}

// Logger captures the logging contract required by the server.
type Logger interface {
	Infof(string, ...interface{})
	Errorf(string, ...interface{})
}

// Server provides HTTP handlers for the courier domain.
type Server struct {
        cfg    Config
        logger Logger
        orders *repo.OrdersRepo
        offers *repo.OffersRepo
        couriers *repo.CouriersRepo
        cHub   *ws.CourierHub
        sHub   *ws.SenderHub
}

// NewServer constructs a Server instance.
func NewServer(cfg Config, logger Logger, orders *repo.OrdersRepo, offers *repo.OffersRepo, couriers *repo.CouriersRepo, cHub *ws.CourierHub, sHub *ws.SenderHub) *Server {
        return &Server{cfg: cfg, logger: logger, orders: orders, offers: offers, couriers: couriers, cHub: cHub, sHub: sHub}
}

// Register mounts courier routes on the mux.
func (s *Server) Register(mux *http.ServeMux) {
        mux.HandleFunc("/api/v1/courier/orders", s.handleOrders)
        mux.HandleFunc("/api/v1/courier/orders/", s.handleOrderSubroutes)
        mux.HandleFunc("/api/v1/courier/orders/active", s.handleActiveOrder)
        mux.HandleFunc("/api/v1/courier/my/orders", s.handleCourierOrders)
        mux.HandleFunc("/api/v1/courier/my/orders/active", s.handleCourierActiveOrder)
        mux.HandleFunc("/api/v1/courier/route/quote", s.handleQuote)
        mux.HandleFunc("/api/v1/courier/offers/price", s.handleOfferPrice)
        mux.HandleFunc("/api/v1/courier/offers/accept", s.handleOfferAccept)
        mux.HandleFunc("/api/v1/courier/offers/decline", s.handleOfferDecline)
        mux.HandleFunc("/api/v1/courier/offers/respond", s.handleOfferRespond)
        mux.HandleFunc("/api/v1/courier/balance/deposit", s.handleCourierBalanceDeposit)
        mux.HandleFunc("/api/v1/courier/balance/withdraw", s.handleCourierBalanceWithdraw)
        mux.HandleFunc("/api/v1/courier/orders/stats", s.handleAdminCourierOrdersStats)
        mux.HandleFunc("/api/v1/admin/courier/orders/stats", s.handleAdminCourierOrdersStats)
        mux.HandleFunc("/api/v1/admin/courier/orders", s.handleAdminCourierOrders)
        mux.HandleFunc("/api/v1/admin/courier/couriers", s.handleAdminCouriers)
        mux.HandleFunc("/api/v1/admin/courier/couriers/stats", s.handleAdminCouriersStats)
        mux.HandleFunc("/api/v1/admin/courier/couriers/", s.handleAdminCourierActions)
        mux.HandleFunc("/api/v1/couriers", s.handleCourierUpsert)
        mux.HandleFunc("/api/v1/courier/", s.handleCourierProfileRoutes)
        mux.HandleFunc("/ws/courier", s.handleCourierWS)
        mux.HandleFunc("/ws/sender", s.handleSenderWS)
}
