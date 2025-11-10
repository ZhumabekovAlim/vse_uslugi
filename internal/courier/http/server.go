package http

import (
	"net/http"

	"naimuBack/internal/courier/repo"
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
}

// NewServer constructs a Server instance.
func NewServer(cfg Config, logger Logger, orders *repo.OrdersRepo, offers *repo.OffersRepo) *Server {
	return &Server{cfg: cfg, logger: logger, orders: orders, offers: offers}
}

// Register mounts courier routes on the mux.
func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/courier/orders", s.handleOrders)
	mux.HandleFunc("/api/v1/courier/orders/", s.handleOrderSubroutes)
	mux.HandleFunc("/api/v1/courier/route/quote", s.handleQuote)
	mux.HandleFunc("/api/v1/courier/offers/price", s.handleOfferPrice)
	mux.HandleFunc("/api/v1/courier/offers/accept", s.handleOfferAccept)
	mux.HandleFunc("/api/v1/courier/offers/decline", s.handleOfferDecline)
	mux.HandleFunc("/ws/courier", s.handleCourierWS)
	mux.HandleFunc("/ws/sender", s.handleSenderWS)
}
