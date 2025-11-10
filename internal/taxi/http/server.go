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
	"sort"
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
	logger         dispatch.Logger
	cfg            dispatch.Config
	geoClient      *geo.DGISClient
	driversRepo    *repo.DriversRepo
	ordersRepo     *repo.OrdersRepo
	passengersRepo *repo.PassengersRepo
	intercityRepo  *repo.IntercityOrdersRepo
	offersRepo     *repo.OffersRepo
	paymentsRepo   *repo.PaymentsRepo
	driverHub      *ws.DriverHub
	passengerHub   *ws.PassengerHub
	dispatcher     *dispatch.Dispatcher
	payClient      *pay.Client
}

const (
	lifecycleArrivalRadiusMeters  = 100.0
	lifecycleStartRadiusMeters    = 100.0
	lifecycleWaypointRadiusMeters = 50.0
	lifecycleFinishRadiusMeters   = 100.0
	lifecycleStationarySpeedKPH   = 5.0
	minDriverBalanceTenge         = 1000
	driverCommissionPercent       = 10
)

const lifecycleTelemetryFreshness = 5 * time.Minute

var (
	errOrderStatusConflict = errors.New("order status changed")
	errDriverBanned        = errors.New("driver banned")
	errDriverNotApproved   = errors.New("driver not approved")
	errInvalidLimit        = errors.New("invalid limit")
	errInvalidOffset       = errors.New("invalid offset")
)

func nstr(v sql.NullString) string {
	if v.Valid {
		return v.String
	}
	return ""
}

// NewServer constructs Server.
func NewServer(logger dispatch.Logger, cfg dispatch.Config, geoClient *geo.DGISClient, drivers *repo.DriversRepo, orders *repo.OrdersRepo, passengers *repo.PassengersRepo, intercity *repo.IntercityOrdersRepo, offers *repo.OffersRepo, payments *repo.PaymentsRepo, driverHub *ws.DriverHub, passengerHub *ws.PassengerHub, dispatcher *dispatch.Dispatcher, payClient *pay.Client) *Server {
	return &Server{
		logger:         logger,
		cfg:            cfg,
		geoClient:      geoClient,
		driversRepo:    drivers,
		ordersRepo:     orders,
		passengersRepo: passengers,
		intercityRepo:  intercity,
		offersRepo:     offers,
		paymentsRepo:   payments,
		driverHub:      driverHub,
		passengerHub:   passengerHub,
		dispatcher:     dispatcher,
		payClient:      payClient,
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
	ID             int64     `json:"id"`
	UserID         int64     `json:"user_id"`
	Name           string    `json:"name"`
	Surname        string    `json:"surname"`
	Middlename     string    `json:"middlename,omitempty"`
	Status         string    `json:"status"`
	ApprovalStatus string    `json:"approval_status"`
	IsBanned       bool      `json:"is_banned"`
	CarModel       string    `json:"car_model,omitempty"`
	CarColor       string    `json:"car_color,omitempty"`
	CarNumber      string    `json:"car_number"`
	TechPassport   string    `json:"tech_passport"`
	CarPhotoFront  string    `json:"car_photo_front"`
	CarPhotoBack   string    `json:"car_photo_back"`
	CarPhotoLeft   string    `json:"car_photo_left"`
	CarPhotoRight  string    `json:"car_photo_right"`
	DriverPhoto    string    `json:"driver_photo"`
	Phone          string    `json:"phone"`
	IIN            string    `json:"iin"`
	IDCardFront    string    `json:"id_card_front"`
	IDCardBack     string    `json:"id_card_back"`
	Rating         float64   `json:"rating"`
	Balance        int       `json:"balance"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type driverProfileResponse struct {
	Driver         driverResponse `json:"driver"`
	CompletedTrips int            `json:"completed_trips"`
	Balance        int            `json:"balance"`
}

type driverReviewResponse struct {
	Rating    *float64      `json:"rating,omitempty"`
	Comment   string        `json:"comment,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
	Order     orderResponse `json:"order"`
}

type driverDayStatsResponse struct {
	Date        string          `json:"date"`
	OrdersCount int             `json:"orders_count"`
	TotalAmount int             `json:"total_amount"`
	NetProfit   int             `json:"net_profit"`
	Orders      []orderResponse `json:"orders"`
}

type driverStatsResponse struct {
	TotalOrders int                      `json:"total_orders"`
	TotalAmount int                      `json:"total_amount"`
	NetProfit   int                      `json:"net_profit"`
	Days        []driverDayStatsResponse `json:"days"`
}

func newDriverResponse(d repo.Driver) driverResponse {
	resp := driverResponse{
		ID:             d.ID,
		UserID:         d.UserID,
		Name:           d.Name,
		Surname:        d.Surname,
		Status:         d.Status,
		ApprovalStatus: d.ApprovalStatus,
		IsBanned:       d.IsBanned,
		CarModel:       d.CarModel.String,
		CarColor:       d.CarColor.String,
		CarNumber:      d.CarNumber,
		TechPassport:   d.TechPassport,
		CarPhotoFront:  d.CarPhotoFront,
		CarPhotoBack:   d.CarPhotoBack,
		CarPhotoLeft:   d.CarPhotoLeft,
		CarPhotoRight:  d.CarPhotoRight,
		DriverPhoto:    d.DriverPhoto,
		Phone:          d.Phone,
		IIN:            d.IIN,
		IDCardFront:    d.IDCardFront,
		IDCardBack:     d.IDCardBack,
		Rating:         d.Rating,
		Balance:        d.Balance,
		UpdatedAt:      d.UpdatedAt,
	}
	if d.Middlename.Valid {
		resp.Middlename = d.Middlename.String
	}
	return resp
}

func (s *Server) ensureDriverEligible(ctx context.Context, driverID int64) (repo.Driver, error) {
	driver, err := s.driversRepo.Get(ctx, driverID)
	if err != nil {
		return repo.Driver{}, err
	}
	if driver.IsBanned {
		return driver, errDriverBanned
	}
	if driver.ApprovalStatus != "approved" {
		return driver, errDriverNotApproved
	}
	return driver, nil
}

func (s *Server) getDriverForAction(w http.ResponseWriter, ctx context.Context, driverID int64) (repo.Driver, bool) {
	driver, err := s.ensureDriverEligible(ctx, driverID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			writeError(w, http.StatusUnauthorized, "driver not found")
		case errors.Is(err, errDriverBanned):
			writeError(w, http.StatusForbidden, "driver is banned")
		case errors.Is(err, errDriverNotApproved):
			writeError(w, http.StatusForbidden, "driver not approved")
		default:
			writeError(w, http.StatusInternalServerError, "driver lookup failed")
		}
		return repo.Driver{}, false
	}
	return driver, true
}

func parseLimitOffset(r *http.Request, defaultLimit int) (int, int, error) {
	limit := defaultLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return 0, 0, errInvalidLimit
		}
		limit = n
	}
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return 0, 0, errInvalidOffset
		}
		offset = n
	}
	return limit, offset, nil
}

type passengerResponse struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	Surname      string     `json:"surname"`
	Middlename   string     `json:"middlename,omitempty"`
	Phone        string     `json:"phone"`
	Email        string     `json:"email,omitempty"`
	CityID       *int64     `json:"city_id,omitempty"`
	YearsOfExp   *int64     `json:"years_of_exp,omitempty"`
	DocOfProof   string     `json:"doc_of_proof,omitempty"`
	ReviewRating *float64   `json:"review_rating,omitempty"`
	Role         string     `json:"role,omitempty"`
	Latitude     string     `json:"latitude,omitempty"`
	Longitude    string     `json:"longitude,omitempty"`
	AvatarPath   string     `json:"avatar_path,omitempty"`
	Skills       string     `json:"skills,omitempty"`
	IsOnline     *bool      `json:"is_online,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
}

func newPassengerResponse(p repo.Passenger) passengerResponse {
	resp := passengerResponse{
		ID:        p.ID,
		Name:      p.Name,
		Surname:   p.Surname,
		Phone:     p.Phone,
		Email:     p.Email,
		CreatedAt: p.CreatedAt,
	}
	if p.Middlename.Valid {
		resp.Middlename = p.Middlename.String
	}
	if p.CityID.Valid {
		v := p.CityID.Int64
		resp.CityID = &v
	}
	if p.YearsOfExp.Valid {
		v := p.YearsOfExp.Int64
		resp.YearsOfExp = &v
	}
	if p.DocOfProof.Valid {
		resp.DocOfProof = p.DocOfProof.String
	}
	if p.ReviewRating.Valid {
		v := p.ReviewRating.Float64
		resp.ReviewRating = &v
	}
	if p.Role.Valid {
		resp.Role = p.Role.String
	}
	if p.Latitude.Valid {
		resp.Latitude = p.Latitude.String
	}
	if p.Longitude.Valid {
		resp.Longitude = p.Longitude.String
	}
	if p.AvatarPath.Valid {
		resp.AvatarPath = p.AvatarPath.String
	}
	if p.Skills.Valid {
		resp.Skills = p.Skills.String
	}
	if p.IsOnline.Valid {
		v := p.IsOnline.Bool
		resp.IsOnline = &v
	}
	if p.UpdatedAt.Valid {
		ts := p.UpdatedAt.Time
		resp.UpdatedAt = &ts
	}
	return resp
}

func newPassengerDriver(d repo.Driver) ws.PassengerDriver {
	driver := ws.PassengerDriver{
		ID:          d.ID,
		Status:      d.Status,
		CarNumber:   d.CarNumber,
		DriverPhoto: d.DriverPhoto,
		Phone:       d.Phone,
		Rating:      d.Rating,
	}
	if d.CarModel.Valid {
		driver.CarModel = d.CarModel.String
	}
	if d.CarColor.Valid {
		driver.CarColor = d.CarColor.String
	}
	if d.CarPhotoFront != "" {
		driver.CarPhotoFront = d.CarPhotoFront
	}
	if d.CarPhotoBack != "" {
		driver.CarPhotoBack = d.CarPhotoBack
	}
	if d.CarPhotoLeft != "" {
		driver.CarPhotoLeft = d.CarPhotoLeft
	}
	if d.CarPhotoRight != "" {
		driver.CarPhotoRight = d.CarPhotoRight
	}
	return driver
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
	Passenger        *passengerResponse     `json:"passenger,omitempty"`
}

func newOrderResponse(o repo.Order, driver *repo.Driver, passenger *repo.Passenger) orderResponse {
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
	if passenger != nil {
		p := newPassengerResponse(*passenger)
		resp.Passenger = &p
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

type intercityCancelPayload struct {
	DriverID int64 `json:"driver_id"`
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
	ID            int64                       `json:"id"`
	PassengerID   int64                       `json:"passenger_id"`
	DriverID      *int64                      `json:"driver_id,omitempty"`
	FromLocation  string                      `json:"from"`
	ToLocation    string                      `json:"to"`
	TripType      string                      `json:"trip_type"`
	Comment       string                      `json:"comment,omitempty"`
	Price         int                         `json:"price"`
	ContactPhone  string                      `json:"contact_phone"`
	DepartureDate string                      `json:"departure_date"`
	DepartureTime string                      `json:"departure_time,omitempty"`
	Status        string                      `json:"status"`
	CreatedAt     time.Time                   `json:"created_at"`
	UpdatedAt     time.Time                   `json:"updated_at"`
	ClosedAt      *time.Time                  `json:"closed_at,omitempty"`
	CreatorRole   string                      `json:"creator_role"`
	Driver        *intercityDriverResponse    `json:"driver,omitempty"`
	Passenger     *intercityPassengerResponse `json:"passenger,omitempty"`
}

type intercityDriverResponse struct {
	ID               int64      `json:"id"`
	CarModel         string     `json:"car_model,omitempty"`
	FullName         string     `json:"full_name,omitempty"`
	Rating           *float64   `json:"rating,omitempty"`
	Photo            string     `json:"photo,omitempty"`
	AvatarPath       string     `json:"avatar_path,omitempty"`
	ProfileUpdatedAt *time.Time `json:"profile_updated_at,omitempty"`
}

type intercityPassengerResponse struct {
	ID               int64      `json:"id"`
	FullName         string     `json:"full_name,omitempty"`
	AvatarPath       string     `json:"avatar_path,omitempty"`
	Phone            string     `json:"phone,omitempty"`
	Rating           *float64   `json:"rating,omitempty"`
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
		if strings.EqualFold(o.CreatorRole, "passenger") {
			passenger := intercityPassengerResponse{ID: o.PassengerID.Int64}
			if o.PassengerFullName.Valid {
				passenger.FullName = strings.TrimSpace(o.PassengerFullName.String)
			}
			if o.PassengerAvatar.Valid {
				passenger.AvatarPath = o.PassengerAvatar.String
			}
			if o.PassengerPhone.Valid {
				passenger.Phone = strings.TrimSpace(o.PassengerPhone.String)
			}
			if o.PassengerRating.Valid {
				rating := o.PassengerRating.Float64
				passenger.Rating = &rating
			}
			if o.PassengerProfileStamp.Valid {
				ts := o.PassengerProfileStamp.Time
				passenger.ProfileUpdatedAt = &ts
			}
			resp.Passenger = &passenger
		}
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
		if o.DriverAvatar.Valid {
			driver.AvatarPath = o.DriverAvatar.String
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
	mux.HandleFunc("/api/v1/admin/taxi/drivers", s.handleAdminTaxiDrivers)
	mux.HandleFunc("/api/v1/admin/taxi/drivers/", s.handleAdminTaxiDriver)
	mux.HandleFunc("/api/v1/admin/taxi/orders", s.handleAdminTaxiOrders)
	mux.HandleFunc("/api/v1/admin/taxi/intercity/orders", s.handleAdminTaxiIntercityOrders)

	mux.HandleFunc("/api/v1/drivers", s.handleDrivers)
	mux.HandleFunc("/api/v1/drivers/", s.handleDriver)
	mux.HandleFunc("/api/v1/driver/balance/deposit", s.handleDriverBalanceDeposit)
	mux.HandleFunc("/api/v1/driver/balance/withdraw", s.handleDriverBalanceWithdraw)
	mux.HandleFunc("/api/v1/driver/", s.handleDriverInfoRoutes)

	mux.HandleFunc("/api/v1/route/quote", s.handleRouteQuote)
	mux.HandleFunc("/api/v1/orders", s.handleOrders)
	mux.HandleFunc("/api/v1/orders/active", s.handlePassengerActiveOrder)
	mux.HandleFunc("/api/v1/driver/orders", s.handleDriverOrders)
	mux.HandleFunc("/api/v1/driver/orders/active", s.handleDriverActiveOrder)
	mux.HandleFunc("/api/v1/orders/", s.handleOrderSubroutes)

	mux.HandleFunc("/api/v1/intercity/orders", s.handleIntercityOrders)
	mux.HandleFunc("/api/v1/intercity/orders/list", s.listIntercityOrders)
	mux.HandleFunc("/api/v1/intercity/orders/", s.handleIntercityOrderSubroutes)

	mux.HandleFunc("/api/taxi/orders/", s.handleTaxiLifecycle)
	mux.HandleFunc("/api/v1/offers/accept", s.handleOfferAccept)
	mux.HandleFunc("/api/v1/offers/propose_price", s.handleOfferPrice)
	mux.HandleFunc("/api/v1/offers/respond", s.handleOfferResponse)

	mux.HandleFunc("/api/v1/payments/airbapay/webhook", s.handleAirbaPayWebhook)

	mux.HandleFunc("/ws/driver", s.handleDriverWS)
	mux.HandleFunc("/ws/passenger", s.handlePassengerWS)
}

func (s *Server) handleAdminTaxiDrivers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	limit, offset, err := parseLimitOffset(r, 100)
	if err != nil {
		switch {
		case errors.Is(err, errInvalidLimit):
			writeError(w, http.StatusBadRequest, "invalid limit")
		case errors.Is(err, errInvalidOffset):
			writeError(w, http.StatusBadRequest, "invalid offset")
		default:
			writeError(w, http.StatusBadRequest, "invalid pagination")
		}
		return
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
	writeJSON(w, http.StatusOK, map[string]interface{}{"drivers": resp, "limit": limit, "offset": offset})
}

func (s *Server) handleAdminTaxiDriver(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/taxi/drivers/")
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
	if len(parts) < 2 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	switch parts[1] {
	case "ban":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			Banned bool `json:"banned"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if err := s.driversRepo.SetBanStatus(ctx, id, payload.Banned); err != nil {
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
	case "approval":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		status := strings.ToLower(strings.TrimSpace(payload.Status))
		if status != "approved" && status != "rejected" {
			writeError(w, http.StatusBadRequest, "status must be approved or rejected")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if err := s.driversRepo.UpdateApprovalStatus(ctx, id, status); err != nil {
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
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *Server) handleAdminTaxiOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	limit, offset, err := parseLimitOffset(r, 100)
	if err != nil {
		switch {
		case errors.Is(err, errInvalidLimit):
			writeError(w, http.StatusBadRequest, "invalid limit")
		case errors.Is(err, errInvalidOffset):
			writeError(w, http.StatusBadRequest, "invalid offset")
		default:
			writeError(w, http.StatusBadRequest, "invalid pagination")
		}
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	orders, err := s.ordersRepo.ListAll(ctx, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list orders failed")
		return
	}

	passengerCache := make(map[int64]repo.Passenger)
	driverCache := make(map[int64]repo.Driver)
	resp := make([]orderResponse, 0, len(orders))
	for _, order := range orders {
		var passenger *repo.Passenger
		if cached, ok := passengerCache[order.PassengerID]; ok {
			cachedPassenger := cached
			passenger = &cachedPassenger
		} else {
			if p, err := s.passengersRepo.Get(ctx, order.PassengerID); err == nil {
				passengerCache[order.PassengerID] = p
				passenger = &p
			}
		}

		var driver *repo.Driver
		if order.DriverID.Valid {
			if cached, ok := driverCache[order.DriverID.Int64]; ok {
				cachedDriver := cached
				driver = &cachedDriver
			} else {
				if d, err := s.driversRepo.Get(ctx, order.DriverID.Int64); err == nil {
					driverCache[order.DriverID.Int64] = d
					driver = &d
				}
			}
		}

		resp = append(resp, newOrderResponse(order, driver, passenger))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"orders": resp, "limit": limit, "offset": offset})
}

func (s *Server) handleAdminTaxiIntercityOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	limit, offset, err := parseLimitOffset(r, 100)
	if err != nil {
		switch {
		case errors.Is(err, errInvalidLimit):
			writeError(w, http.StatusBadRequest, "invalid limit")
		case errors.Is(err, errInvalidOffset):
			writeError(w, http.StatusBadRequest, "invalid offset")
		default:
			writeError(w, http.StatusBadRequest, "invalid pagination")
		}
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	orders, err := s.intercityRepo.List(ctx, repo.IntercityOrdersFilter{Limit: limit, Offset: offset})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list orders failed")
		return
	}

	resp := make([]intercityOrderResponse, 0, len(orders))
	for _, order := range orders {
		resp = append(resp, newIntercityOrderResponse(order))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"orders": resp, "limit": limit, "offset": offset})
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

func (s *Server) handleDriverInfoRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/driver/")
	path = strings.TrimSuffix(path, "/")
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
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	switch parts[1] {
	case "profile":
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.getDriverProfile(w, r, id)
	case "reviews":
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.listDriverReviews(w, r, id)
	case "stats":
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.getDriverStats(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *Server) handleDriverBalanceDeposit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	driverID, err := parseAuthID(r, "X-Driver-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing driver id")
		return
	}
	var payload struct {
		Amount int `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if payload.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be positive")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	balance, err := s.driversRepo.Deposit(ctx, driverID, payload.Amount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, "driver not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "deposit failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"balance": balance})
}

func (s *Server) getDriverProfile(w http.ResponseWriter, r *http.Request, id int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	driver, err := s.driversRepo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "driver not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load driver")
		return
	}

	completed, err := s.ordersRepo.CountCompletedByDriver(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to count completed trips")
		return
	}

	resp := driverProfileResponse{
		Driver:         newDriverResponse(driver),
		CompletedTrips: completed,
		Balance:        driver.Balance,
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) listDriverReviews(w http.ResponseWriter, r *http.Request, id int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	driver, err := s.driversRepo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "driver not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load driver")
		return
	}

	reviews, err := s.ordersRepo.ListDriverReviews(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list driver reviews")
		return
	}

	passengerCache := make(map[int64]repo.Passenger)
	resp := make([]driverReviewResponse, 0, len(reviews))
	for _, review := range reviews {
		var passenger *repo.Passenger
		if cached, ok := passengerCache[review.Order.PassengerID]; ok {
			cachedPassenger := cached
			passenger = &cachedPassenger
		} else {
			if p, err := s.passengersRepo.Get(ctx, review.Order.PassengerID); err == nil {
				passengerCache[review.Order.PassengerID] = p
				passenger = &p
			}
		}

		orderResp := newOrderResponse(review.Order, &driver, passenger)
		reviewResp := driverReviewResponse{
			CreatedAt: review.CreatedAt,
			Order:     orderResp,
		}
		if review.Rating.Valid {
			v := review.Rating.Float64
			reviewResp.Rating = &v
		}
		if review.Comment.Valid {
			reviewResp.Comment = review.Comment.String
		}
		resp = append(resp, reviewResp)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"reviews": resp})
}

func (s *Server) getDriverStats(w http.ResponseWriter, r *http.Request, id int64) {
	fromStr := strings.TrimSpace(r.URL.Query().Get("from"))
	toStr := strings.TrimSpace(r.URL.Query().Get("to"))
	if toStr == "" {
		toStr = strings.TrimSpace(r.URL.Query().Get("until"))
	}
	if fromStr == "" || toStr == "" {
		writeError(w, http.StatusBadRequest, "from and to parameters are required")
		return
	}

	fromDate, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid from date")
		return
	}
	toDate, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid to date")
		return
	}
	if toDate.Before(fromDate) {
		writeError(w, http.StatusBadRequest, "to date must be on or after from date")
		return
	}

	// include the entire "to" day by shifting exclusive upper bound by one day
	toExclusive := toDate.AddDate(0, 0, 1)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	driver, err := s.driversRepo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "driver not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load driver")
		return
	}

	orders, err := s.ordersRepo.ListCompletedByDriverBetween(ctx, id, fromDate, toExclusive)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list driver orders")
		return
	}

	passengerCache := make(map[int64]repo.Passenger)
	dayMap := make(map[string]*driverDayStatsResponse)
	stats := driverStatsResponse{}

	for _, order := range orders {
		var passenger *repo.Passenger
		if cached, ok := passengerCache[order.PassengerID]; ok {
			cachedPassenger := cached
			passenger = &cachedPassenger
		} else {
			if p, err := s.passengersRepo.Get(ctx, order.PassengerID); err == nil {
				passengerCache[order.PassengerID] = p
				passenger = &p
			}
		}

		orderResp := newOrderResponse(order, &driver, passenger)

		commission := calculateCommission(order.ClientPrice)
		netProfit := order.ClientPrice - commission
		stats.TotalOrders++
		stats.TotalAmount += order.ClientPrice
		stats.NetProfit += netProfit

		dayKey := order.UpdatedAt.Format("2006-01-02")
		if day, ok := dayMap[dayKey]; ok {
			day.OrdersCount++
			day.TotalAmount += order.ClientPrice
			day.NetProfit += netProfit
			day.Orders = append(day.Orders, orderResp)
		} else {
			dayMap[dayKey] = &driverDayStatsResponse{
				Date:        dayKey,
				OrdersCount: 1,
				TotalAmount: order.ClientPrice,
				NetProfit:   netProfit,
				Orders:      []orderResponse{orderResp},
			}
		}
	}

	if len(dayMap) > 0 {
		keys := make([]string, 0, len(dayMap))
		for k := range dayMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		stats.Days = make([]driverDayStatsResponse, 0, len(keys))
		for _, k := range keys {
			stats.Days = append(stats.Days, *dayMap[k])
		}
	} else {
		stats.Days = []driverDayStatsResponse{}
	}

	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleDriverBalanceWithdraw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	driverID, err := parseAuthID(r, "X-Driver-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing driver id")
		return
	}
	var payload struct {
		Amount int `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if payload.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be positive")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	balance, err := s.driversRepo.Withdraw(ctx, driverID, payload.Amount)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			writeError(w, http.StatusUnauthorized, "driver not found")
			return
		case errors.Is(err, repo.ErrInsufficientBalance):
			writeError(w, http.StatusBadRequest, "insufficient balance")
			return
		default:
			writeError(w, http.StatusInternalServerError, "withdraw failed")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]int{"balance": balance})
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

func (s *Server) handleTaxiLifecycle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/taxi/orders/")
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
	orderID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid order id")
		return
	}
	switch parts[1] {
	case "arrive":
		s.handleLifecycleArrive(w, r, orderID)
	case "waiting":
		if len(parts) == 3 && parts[2] == "advance" {
			s.handleLifecycleWaitingAdvance(w, r, orderID)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	case "start":
		s.handleLifecycleStart(w, r, orderID)
	case "waypoints":
		if len(parts) == 3 && parts[2] == "next" {
			s.handleLifecycleWaypointNext(w, r, orderID)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	case "pause":
		s.handleLifecyclePause(w, r, orderID)
	case "resume":
		s.handleLifecycleResume(w, r, orderID)
	case "finish":
		s.handleLifecycleFinish(w, r, orderID)
	case "confirm-cash":
		s.handleLifecycleConfirmCash(w, r, orderID)
	case "cancel":
		s.handleLifecycleCancel(w, r, orderID)
	case "no-show":
		s.handleLifecycleNoShow(w, r, orderID)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

type telemetryPayload struct {
	Timestamp string `json:"timestamp"`
	Position  struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"position"`
	SpeedKPH float64 `json:"speed_kph"`
}

func (p telemetryPayload) parseTimestamp() (time.Time, error) {
	if strings.TrimSpace(p.Timestamp) == "" {
		return time.Time{}, errors.New("timestamp required")
	}
	return time.Parse(time.RFC3339, p.Timestamp)
}

func (s *Server) handleLifecycleArrive(w http.ResponseWriter, r *http.Request, orderID int64) {
	driverID, err := parseAuthID(r, "X-Driver-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing driver id")
		return
	}
	var payload telemetryPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ts, err := payload.parseTimestamp()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if time.Since(ts) > lifecycleTelemetryFreshness {
		writeError(w, http.StatusBadRequest, "outdated telemetry")
		return
	}
	if payload.SpeedKPH > lifecycleStationarySpeedKPH {
		writeError(w, http.StatusBadRequest, "driver must be stationary")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, ok := s.getDriverForAction(w, ctx, driverID); !ok {
		return
	}

	order, err := s.ordersRepo.Get(ctx, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "fetch order failed")
		return
	}
	if !order.DriverID.Valid || order.DriverID.Int64 != driverID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}
	distance := distanceMeters(order.FromLon, order.FromLat, payload.Position.Lon, payload.Position.Lat)
	if distance > lifecycleArrivalRadiusMeters {
		writeError(w, http.StatusBadRequest, "driver outside pickup radius")
		return
	}

	switch order.Status {
	case fsm.StatusWaitingFree, fsm.StatusWaitingPaid, fsm.StatusInProgress:
		writeJSON(w, http.StatusOK, map[string]string{"status": order.Status})
		return
	}

	sequence := []string{fsm.StatusDriverAtPickup, fsm.StatusWaitingFree}
	if order.Status == fsm.StatusDriverAtPickup {
		sequence = sequence[1:]
	}

	if err := s.applyStatusSequence(ctx, &order, sequence...); err != nil {
		if errors.Is(err, errOrderStatusConflict) {
			writeError(w, http.StatusConflict, "order status changed")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.notifyPassengerStatus(order.PassengerID, order.ID, order.Status)
	writeJSON(w, http.StatusOK, map[string]string{"status": order.Status})
}

func (s *Server) applyStatusSequence(ctx context.Context, order *repo.Order, statuses ...string) error {
	current := order.Status
	for _, target := range statuses {
		if target == "" || current == target {
			continue
		}
		if !fsm.CanTransition(current, target) {
			return fmt.Errorf("invalid transition %s -> %s", current, target)
		}
		if err := s.ordersRepo.UpdateStatusCAS(ctx, order.ID, current, target); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return errOrderStatusConflict
			}
			return err
		}
		current = target
	}
	order.Status = current
	return nil
}

func (s *Server) notifyPassengerStatus(passengerID, orderID int64, status string) {
	if s.passengerHub == nil || passengerID == 0 {
		return
	}
	s.passengerHub.PushOrderEvent(passengerID, ws.PassengerEvent{Type: "order_status", OrderID: orderID, Status: status})
}

func (s *Server) handleLifecycleWaitingAdvance(w http.ResponseWriter, r *http.Request, orderID int64) {
	driverID, err := parseAuthID(r, "X-Driver-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing driver id")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, ok := s.getDriverForAction(w, ctx, driverID); !ok {
		return
	}

	order, err := s.ordersRepo.Get(ctx, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "fetch order failed")
		return
	}
	if !order.DriverID.Valid || order.DriverID.Int64 != driverID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}
	if order.Status == fsm.StatusWaitingPaid || order.Status == fsm.StatusInProgress {
		writeJSON(w, http.StatusOK, map[string]string{"status": order.Status})
		return
	}
	if order.Status != fsm.StatusWaitingFree {
		writeError(w, http.StatusConflict, "order not in waiting state")
		return
	}

	if err := s.applyStatusSequence(ctx, &order, fsm.StatusWaitingPaid); err != nil {
		if errors.Is(err, errOrderStatusConflict) {
			writeError(w, http.StatusConflict, "order status changed")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.notifyPassengerStatus(order.PassengerID, order.ID, order.Status)
	writeJSON(w, http.StatusOK, map[string]string{"status": order.Status})
}

func (s *Server) handleLifecycleStart(w http.ResponseWriter, r *http.Request, orderID int64) {
	driverID, err := parseAuthID(r, "X-Driver-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing driver id")
		return
	}
	var payload struct {
		telemetryPayload
		PinConfirmed bool `json:"pin_confirmed"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ts, err := payload.telemetryPayload.parseTimestamp()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if time.Since(ts) > lifecycleTelemetryFreshness {
		writeError(w, http.StatusBadRequest, "outdated telemetry")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, ok := s.getDriverForAction(w, ctx, driverID); !ok {
		return
	}

	order, err := s.ordersRepo.Get(ctx, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "fetch order failed")
		return
	}
	if !order.DriverID.Valid || order.DriverID.Int64 != driverID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	distance := distanceMeters(order.FromLon, order.FromLat, payload.Position.Lon, payload.Position.Lat)
	if distance > lifecycleStartRadiusMeters {
		writeError(w, http.StatusBadRequest, "driver outside start radius")
		return
	}

	switch order.Status {
	case fsm.StatusInProgress, fsm.StatusAtLastPoint, fsm.StatusCompleted:
		writeJSON(w, http.StatusOK, map[string]string{"status": order.Status})
		return
	case fsm.StatusWaitingFree, fsm.StatusWaitingPaid, fsm.StatusDriverAtPickup, fsm.StatusAccepted, fsm.StatusAssigned:
	default:
		writeError(w, http.StatusConflict, "order cannot be started in current status")
		return
	}

	if err := s.applyStatusSequence(ctx, &order, fsm.StatusInProgress); err != nil {
		if errors.Is(err, errOrderStatusConflict) {
			writeError(w, http.StatusConflict, "order status changed")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.notifyPassengerStatus(order.PassengerID, order.ID, order.Status)
	writeJSON(w, http.StatusOK, map[string]string{"status": order.Status})
}

func (s *Server) handleLifecycleWaypointNext(w http.ResponseWriter, r *http.Request, orderID int64) {
	driverID, err := parseAuthID(r, "X-Driver-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing driver id")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, ok := s.getDriverForAction(w, ctx, driverID); !ok {
		return
	}

	order, err := s.ordersRepo.Get(ctx, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "fetch order failed")
		return
	}
	if !order.DriverID.Valid || order.DriverID.Int64 != driverID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}
	if order.Status != fsm.StatusInProgress {
		writeError(w, http.StatusConflict, "order not in progress")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": order.Status})
}

func (s *Server) handleLifecyclePause(w http.ResponseWriter, r *http.Request, orderID int64) {
	driverID, err := parseAuthID(r, "X-Driver-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing driver id")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, ok := s.getDriverForAction(w, ctx, driverID); !ok {
		return
	}

	order, err := s.ordersRepo.Get(ctx, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "fetch order failed")
		return
	}
	if !order.DriverID.Valid || order.DriverID.Int64 != driverID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}
	if order.Status != fsm.StatusInProgress {
		writeError(w, http.StatusConflict, "order not in progress")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": order.Status})
}

func (s *Server) handleLifecycleResume(w http.ResponseWriter, r *http.Request, orderID int64) {
	driverID, err := parseAuthID(r, "X-Driver-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing driver id")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, ok := s.getDriverForAction(w, ctx, driverID); !ok {
		return
	}

	order, err := s.ordersRepo.Get(ctx, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "fetch order failed")
		return
	}
	if !order.DriverID.Valid || order.DriverID.Int64 != driverID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}
	if order.Status != fsm.StatusInProgress {
		writeError(w, http.StatusConflict, "order not in progress")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": order.Status})
}

func (s *Server) handleLifecycleFinish(w http.ResponseWriter, r *http.Request, orderID int64) {
	driverID, err := parseAuthID(r, "X-Driver-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing driver id")
		return
	}
	var payload telemetryPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ts, err := payload.parseTimestamp()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if time.Since(ts) > lifecycleTelemetryFreshness {
		writeError(w, http.StatusBadRequest, "outdated telemetry")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, ok := s.getDriverForAction(w, ctx, driverID); !ok {
		return
	}

	order, err := s.ordersRepo.Get(ctx, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "fetch order failed")
		return
	}
	if !order.DriverID.Valid || order.DriverID.Int64 != driverID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	distance := distanceMeters(order.ToLon, order.ToLat, payload.Position.Lon, payload.Position.Lat)
	if distance > lifecycleFinishRadiusMeters {
		writeError(w, http.StatusBadRequest, "driver outside finish radius")
		return
	}

	switch order.Status {
	case fsm.StatusAtLastPoint, fsm.StatusCompleted:
		writeJSON(w, http.StatusOK, map[string]string{"status": order.Status})
		return
	case fsm.StatusInProgress:
	default:
		writeError(w, http.StatusConflict, "order not in progress")
		return
	}

	if err := s.applyStatusSequence(ctx, &order, fsm.StatusAtLastPoint); err != nil {
		if errors.Is(err, errOrderStatusConflict) {
			writeError(w, http.StatusConflict, "order status changed")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.notifyPassengerStatus(order.PassengerID, order.ID, order.Status)
	writeJSON(w, http.StatusOK, map[string]string{"status": order.Status})
}

func (s *Server) handleLifecycleConfirmCash(w http.ResponseWriter, r *http.Request, orderID int64) {
	driverID, err := parseAuthID(r, "X-Driver-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing driver id")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, ok := s.getDriverForAction(w, ctx, driverID); !ok {
		return
	}

	order, err := s.ordersRepo.Get(ctx, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "fetch order failed")
		return
	}
	if !order.DriverID.Valid || order.DriverID.Int64 != driverID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}
	if order.Status == fsm.StatusCompleted {
		writeJSON(w, http.StatusOK, map[string]string{"status": order.Status})
		return
	}
	if order.Status != fsm.StatusAtLastPoint {
		writeError(w, http.StatusConflict, "order not ready for completion")
		return
	}

	if err := s.applyStatusSequence(ctx, &order, fsm.StatusCompleted); err != nil {
		if errors.Is(err, errOrderStatusConflict) {
			writeError(w, http.StatusConflict, "order status changed")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.notifyPassengerStatus(order.PassengerID, order.ID, order.Status)
	writeJSON(w, http.StatusOK, map[string]string{"status": order.Status})
}

func (s *Server) handleLifecycleCancel(w http.ResponseWriter, r *http.Request, orderID int64) {
	var payload struct {
		By     string `json:"by"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	by := strings.ToLower(strings.TrimSpace(payload.By))
	switch by {
	case "passenger":
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
		s.handlePassengerCancel(ctx, w, r, order, payload.Reason)
	case "driver":
		driverID, err := parseAuthID(r, "X-Driver-ID")
		if err != nil {
			writeError(w, http.StatusUnauthorized, "missing driver id")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if _, ok := s.getDriverForAction(w, ctx, driverID); !ok {
			return
		}
		order, err := s.ordersRepo.Get(ctx, orderID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusNotFound, "order not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "fetch order failed")
			return
		}
		if !order.DriverID.Valid || order.DriverID.Int64 != driverID {
			writeError(w, http.StatusForbidden, "access denied")
			return
		}
		if err := s.applyStatusSequence(ctx, &order, fsm.StatusCanceledByDriver); err != nil {
			if errors.Is(err, errOrderStatusConflict) {
				writeError(w, http.StatusConflict, "order status changed")
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.notifyPassengerStatus(order.PassengerID, order.ID, order.Status)
		writeJSON(w, http.StatusOK, map[string]string{"status": order.Status})
	default:
		writeError(w, http.StatusBadRequest, "invalid cancel initiator")
	}
}

func (s *Server) handleLifecycleNoShow(w http.ResponseWriter, r *http.Request, orderID int64) {
	driverID, err := parseAuthID(r, "X-Driver-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing driver id")
		return
	}
	var payload telemetryPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ts, err := payload.parseTimestamp()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if time.Since(ts) > lifecycleTelemetryFreshness {
		writeError(w, http.StatusBadRequest, "outdated telemetry")
		return
	}
	if payload.SpeedKPH > lifecycleStationarySpeedKPH {
		writeError(w, http.StatusBadRequest, "driver must be stationary")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, ok := s.getDriverForAction(w, ctx, driverID); !ok {
		return
	}

	order, err := s.ordersRepo.Get(ctx, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "fetch order failed")
		return
	}
	if !order.DriverID.Valid || order.DriverID.Int64 != driverID {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}
	if order.Status == fsm.StatusNoShow {
		writeJSON(w, http.StatusOK, map[string]string{"status": order.Status})
		return
	}
	if order.Status != fsm.StatusWaitingFree && order.Status != fsm.StatusWaitingPaid {
		writeError(w, http.StatusConflict, "order not in waiting state")
		return
	}
	distance := distanceMeters(order.FromLon, order.FromLat, payload.Position.Lon, payload.Position.Lat)
	if distance > lifecycleArrivalRadiusMeters {
		writeError(w, http.StatusBadRequest, "driver outside pickup radius")
		return
	}

	if err := s.applyStatusSequence(ctx, &order, fsm.StatusNoShow); err != nil {
		if errors.Is(err, errOrderStatusConflict) {
			writeError(w, http.StatusConflict, "order status changed")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.notifyPassengerStatus(order.PassengerID, order.ID, order.Status)
	writeJSON(w, http.StatusOK, map[string]string{"status": order.Status})
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
		Status:        "offline",
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

	current, err := s.driversRepo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "driver not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "fetch driver failed")
		return
	}
	if current.IsBanned && payload.Status != "offline" {
		writeError(w, http.StatusForbidden, "driver is banned")
		return
	}
	if current.ApprovalStatus != "approved" && payload.Status != "offline" {
		writeError(w, http.StatusForbidden, "driver not approved")
		return
	}

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

	driver, err = s.driversRepo.Get(ctx, id)
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

func calculateCommission(amount int) int {
	if amount <= 0 || driverCommissionPercent <= 0 {
		return 0
	}
	return (amount*driverCommissionPercent + 99) / 100
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

func (s *Server) handlePassengerActiveOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	passengerID, err := parseAuthID(r, "X-Passenger-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing passenger id")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	orderID, err := s.ordersRepo.GetActiveOrderIDByPassenger(ctx, passengerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeError(w, http.StatusInternalServerError, "fetch active order failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int64{"order_id": orderID})
}

func (s *Server) handleDriverOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListDriverOrders(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleDriverActiveOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	driverID, err := parseAuthID(r, "X-Driver-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing driver id")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	orderID, err := s.ordersRepo.GetActiveOrderIDByDriver(ctx, driverID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeError(w, http.StatusInternalServerError, "fetch active order failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int64{"order_id": orderID})
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

	var passenger *repo.Passenger
	if p, err := s.passengersRepo.Get(ctx, passengerID); err == nil {
		passenger = &p
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
		resp = append(resp, newOrderResponse(order, driver, passenger))
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

	passengerCache := make(map[int64]repo.Passenger)
	resp := make([]orderResponse, 0, len(orders))
	for _, order := range orders {
		var passenger *repo.Passenger
		if cached, ok := passengerCache[order.PassengerID]; ok {
			cachedPassenger := cached
			passenger = &cachedPassenger
		} else {
			if p, err := s.passengersRepo.Get(ctx, order.PassengerID); err == nil {
				passengerCache[order.PassengerID] = p
				passenger = &p
			}
		}
		resp = append(resp, newOrderResponse(order, driver, passenger))
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
	//passengerID, err := parseAuthID(r, "X-Passenger-ID")
	//if err != nil {
	//	writeError(w, http.StatusUnauthorized, "missing passenger id")
	//	return
	//}

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
	//if order.PassengerID != passengerID {
	//	writeError(w, http.StatusForbidden, "access denied")
	//	return
	//}
	var passenger *repo.Passenger
	if p, err := s.passengersRepo.Get(ctx, order.PassengerID); err == nil {
		passenger = &p
	}
	writeJSON(w, http.StatusOK, newOrderResponse(order, driver, passenger))
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

	if len(parts) == 2 && parts[1] == "cancel" {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.cancelIntercityOrder(w, r, id)
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

	commission := 0
	if payload.DriverID > 0 {
		commission = calculateCommission(order.Price)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if commission > 0 {
		if _, err := s.driversRepo.Withdraw(ctx, payload.DriverID, commission); err != nil {
			switch {
			case errors.Is(err, sql.ErrNoRows):
				writeError(w, http.StatusBadRequest, "driver not found")
				return
			case errors.Is(err, repo.ErrInsufficientBalance):
				writeError(w, http.StatusBadRequest, "insufficient balance")
				return
			default:
				writeError(w, http.StatusInternalServerError, "commission charge failed")
				return
			}
		}
	}

	id, err := s.intercityRepo.Create(ctx, order)
	if err != nil {
		if commission > 0 {
			if _, depErr := s.driversRepo.Deposit(ctx, payload.DriverID, commission); depErr != nil {
				s.logger.Errorf("failed to refund intercity commission for driver %d: %v", payload.DriverID, depErr)
			}
		}
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

func (s *Server) cancelIntercityOrder(w http.ResponseWriter, r *http.Request, id int64) {
	var payload intercityCancelPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if payload.DriverID <= 0 {
		writeError(w, http.StatusBadRequest, "driver_id is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := s.intercityRepo.CancelByDriver(ctx, id, payload.DriverID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "cancel failed")
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

func (s *Server) handleOfferPrice(w http.ResponseWriter, r *http.Request) {
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
		Price   int   `json:"price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.OrderID <= 0 {
		writeError(w, http.StatusBadRequest, "order_id is required")
		return
	}
	if req.Price <= 0 {
		writeError(w, http.StatusBadRequest, "price must be positive")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	driver, ok := s.getDriverForAction(w, ctx, driverID)
	if !ok {
		return
	}

	order, err := s.ordersRepo.Get(ctx, req.OrderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "order lookup failed")
		return
	}
	if order.Status != "searching" {
		writeError(w, http.StatusConflict, "order not searching")
		return
	}

	if err := s.offersRepo.SetDriverPrice(ctx, req.OrderID, driverID, req.Price); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "offer not available")
			return
		}
		writeError(w, http.StatusInternalServerError, "set price failed")
		return
	}
	driverInfo := newPassengerDriver(driver)
	ev := ws.PassengerEvent{
		Type:     "offer_price",
		OrderID:  req.OrderID,
		DriverID: driverID,
		Price:    req.Price,
		Driver:   &driverInfo,
	}

	// фактическая отправка в WS
	s.passengerHub.PushOrderEvent(order.PassengerID, ev)

	writeJSON(w, http.StatusOK, map[string]string{"status": "price_proposed"})
}

func (s *Server) handleOfferResponse(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handleOfferResponse called")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	passengerID, err := parseAuthID(r, "X-Passenger-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing passenger id")
		return
	}
	var req struct {
		OrderID  int64  `json:"order_id"`
		DriverID int64  `json:"driver_id"`
		Decision string `json:"decision"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.OrderID <= 0 || req.DriverID <= 0 {
		writeError(w, http.StatusBadRequest, "order_id and driver_id are required")
		return
	}
	decision := strings.ToLower(strings.TrimSpace(req.Decision))
	if decision != "accept" && decision != "decline" {
		writeError(w, http.StatusBadRequest, "decision must be accept or decline")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	order, err := s.ordersRepo.Get(ctx, req.OrderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "order lookup failed")
		return
	}
	if order.PassengerID != passengerID {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if order.Status != "searching" {
		writeError(w, http.StatusConflict, "order not searching")
		return
	}

	driver, err := s.ensureDriverEligible(ctx, req.DriverID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			writeError(w, http.StatusNotFound, "driver not found")
		case errors.Is(err, errDriverBanned), errors.Is(err, errDriverNotApproved):
			writeError(w, http.StatusConflict, "driver not available")
		default:
			writeError(w, http.StatusInternalServerError, "driver lookup failed")
		}
		return
	}
	if driver.Balance < minDriverBalanceTenge {
		writeError(w, http.StatusForbidden, "insufficient driver balance")
		return
	}

	switch decision {
	case "accept":
		closedDrivers, pricePtr, err := s.offersRepo.AcceptOffer(ctx, req.OrderID, req.DriverID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusConflict, "offer not available")
				return
			}
			writeError(w, http.StatusInternalServerError, "accept failed")
			return
		}

		if pricePtr != nil && *pricePtr > 0 && order.ClientPrice != *pricePtr {
			if err := s.ordersRepo.UpdatePrice(ctx, req.OrderID, order.ClientPrice, *pricePtr); err != nil && !errors.Is(err, sql.ErrNoRows) {
				s.logger.Errorf("update price failed: %v", err)
			} else {
				order.ClientPrice = *pricePtr
			}
		}

		if err := s.ordersRepo.AssignDriver(ctx, req.OrderID, req.DriverID); err != nil {
			writeError(w, http.StatusInternalServerError, "assign failed")
			return
		}

		s.driverHub.NotifyPriceResponse(req.DriverID, ws.DriverPriceResponsePayload{OrderID: req.OrderID, Status: "accepted", Price: order.ClientPrice})

		s.passengerHub.PushOrderEvent(order.PassengerID, ws.PassengerEvent{Type: "order_assigned", OrderID: order.ID, Status: "accepted"})

		if len(closedDrivers) > 0 {
			s.driverHub.NotifyOfferClosed(req.OrderID, closedDrivers, "accepted_by_other")
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
	case "decline":
		pricePtr, err := s.offersRepo.DeclineOffer(ctx, req.OrderID, req.DriverID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusConflict, "offer not available")
				return
			}
			writeError(w, http.StatusInternalServerError, "decline failed")
			return
		}

		price := 0
		if pricePtr != nil {
			price = *pricePtr
		}
		s.driverHub.NotifyPriceResponse(req.DriverID, ws.DriverPriceResponsePayload{OrderID: req.OrderID, Status: "declined", Price: price})
		s.passengerHub.PushOrderEvent(order.PassengerID, ws.PassengerEvent{Type: "offer_price_declined", OrderID: order.ID, DriverID: req.DriverID, Price: price})

		writeJSON(w, http.StatusOK, map[string]string{"status": "declined"})
	}
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

	driver, ok := s.getDriverForAction(w, ctx, driverID)
	if !ok {
		return
	}
	if driver.Balance < minDriverBalanceTenge {
		writeError(w, http.StatusForbidden, "insufficient driver balance")
		return
	}

	order, err := s.ordersRepo.Get(ctx, req.OrderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "order lookup failed")
		return
	}

	closedDrivers, pricePtr, err := s.offersRepo.AcceptOffer(ctx, req.OrderID, driverID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusConflict, "offer not available")
			return
		}
		writeError(w, http.StatusInternalServerError, "accept failed")
		return
	}
	if pricePtr != nil && *pricePtr > 0 && order.ClientPrice != *pricePtr {
		if err := s.ordersRepo.UpdatePrice(ctx, req.OrderID, order.ClientPrice, *pricePtr); err != nil && !errors.Is(err, sql.ErrNoRows) {
			s.logger.Errorf("update price failed: %v", err)
		} else {
			order.ClientPrice = *pricePtr
		}
	}
	if err := s.ordersRepo.AssignDriver(ctx, req.OrderID, driverID); err != nil {
		writeError(w, http.StatusInternalServerError, "assign failed")
		return
	}

	s.passengerHub.PushOrderEvent(order.PassengerID, ws.PassengerEvent{Type: "order_assigned", OrderID: order.ID, Status: "accepted"})

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

	if req.Status == "canceled" {
		s.handlePassengerCancel(ctx, w, r, order, "")
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

	var updateErr error
	if req.Status == fsm.StatusCompleted {
		commission := calculateCommission(order.ClientPrice)
		driverID := int64(0)
		if order.DriverID.Valid {
			driverID = order.DriverID.Int64
		}
		updateErr = s.ordersRepo.UpdateStatusWithDriverCharge(ctx, orderID, order.Status, req.Status, driverID, commission)
	} else {
		updateErr = s.ordersRepo.UpdateStatusCAS(ctx, orderID, order.Status, req.Status)
	}
	if updateErr != nil {
		if errors.Is(updateErr, sql.ErrNoRows) {
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

func (s *Server) handlePassengerCancel(ctx context.Context, w http.ResponseWriter, r *http.Request, order repo.Order, note string) {
	const targetStatus = fsm.StatusCanceled

	passengerID, err := parseAuthID(r, "X-Passenger-ID")
	if err != nil {
		msg := "missing passenger id"
		s.pushPassengerError(order.PassengerID, order.ID, msg)
		writeError(w, http.StatusUnauthorized, msg)
		return
	}
	if passengerID != order.PassengerID {
		msg := "access denied"
		s.pushPassengerError(passengerID, order.ID, msg)
		writeError(w, http.StatusForbidden, msg)
		return
	}

	if trimmed := strings.TrimSpace(note); trimmed != "" && s.logger != nil {
		s.logger.Infof("passenger %d canceled order %d: %s", passengerID, order.ID, trimmed)
	}

	switch order.Status {
	case fsm.StatusSearching, fsm.StatusAccepted, fsm.StatusArrived, fsm.StatusWaitingFree, fsm.StatusWaitingPaid:
	default:
		msg := fmt.Sprintf("order cannot be canceled in status %s", order.Status)
		s.pushPassengerError(passengerID, order.ID, msg)
		writeError(w, http.StatusConflict, "cannot cancel in current status")
		return
	}
	if !fsm.CanTransition(order.Status, targetStatus) {
		msg := "invalid transition"
		s.pushPassengerError(passengerID, order.ID, msg)
		writeError(w, http.StatusConflict, msg)
		return
	}

	// === единственный CAS ===
	if err := s.ordersRepo.UpdateStatusCAS(ctx, order.ID, order.Status, targetStatus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			msg := "order status changed"
			s.pushPassengerError(passengerID, order.ID, msg)
			writeError(w, http.StatusConflict, msg)
			return
		}
		msg := "update status failed"
		s.pushPassengerError(passengerID, order.ID, msg)
		writeError(w, http.StatusInternalServerError, msg)
		return
	}

	// 1) совместимость
	if s.passengerHub != nil {
		s.passengerHub.PushOrderEvent(passengerID, ws.PassengerEvent{
			Type:    "order_status",
			OrderID: order.ID,
			Status:  targetStatus,
		})
	}

	// 2) явное событие
	if s.passengerHub != nil {
		s.passengerHub.PushOrderEvent(passengerID, ws.PassengerEvent{
			Type:    "order_canceled",
			OrderID: order.ID,
			Status:  targetStatus,
		})
	}

	// 3) закрыть офферы у драйверов
	if s.driverHub != nil {
		const reason = "canceled_by_passenger"

		recipients := make(map[int64]struct{})
		if s.offersRepo != nil {
			driverIDs, err := s.offersRepo.GetActiveOfferDriverIDs(ctx, order.ID)
			if err != nil {
				s.logger.Errorf("list active offer drivers for order %d failed: %v", order.ID, err)
			} else {
				for _, id := range driverIDs {
					recipients[id] = struct{}{}
				}
			}
		}
		if order.DriverID.Valid {
			recipients[order.DriverID.Int64] = struct{}{}
		}

		if len(recipients) > 0 {
			ids := make([]int64, 0, len(recipients))
			for id := range recipients {
				ids = append(ids, id)
			}
			s.driverHub.NotifyOfferClosed(order.ID, ids, reason)
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": targetStatus})
}

func (s *Server) pushPassengerError(passengerID, orderID int64, message string) {
	if passengerID <= 0 || message == "" {
		return
	}
	s.passengerHub.PushOrderEvent(passengerID, ws.PassengerEvent{Type: "error", OrderID: orderID, Message: message})
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
