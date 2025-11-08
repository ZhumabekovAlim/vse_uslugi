package dispatch

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"naimuBack/internal/taxi/geo"
	"naimuBack/internal/taxi/pricing"
	"naimuBack/internal/taxi/repo"
	"naimuBack/internal/taxi/timeutil"
	"naimuBack/internal/taxi/ws"
)

// Logger is a minimal logger interface required by dispatcher.
type Logger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// Config holds required configuration subset.
type Config interface {
	GetPricePerKM() int
	GetMinPrice() int
	GetSearchRadiusStart() int
	GetSearchRadiusStep() int
	GetSearchRadiusMax() int
	GetDispatchTick() time.Duration
	GetOfferTTL() time.Duration
	GetRegionID() string
	GetSearchTimeout() time.Duration
}

// Dispatcher performs periodic matching between orders and drivers.
type OrdersRepository interface {
	Get(ctx context.Context, id int64) (repo.Order, error)
	UpdateStatusCAS(ctx context.Context, orderID int64, fromStatus, toStatus string) error
}

type DispatchRepository interface {
	ListDue(ctx context.Context, now time.Time) ([]repo.DispatchRecord, error)
	UpdateRadius(ctx context.Context, orderID int64, radius int, next time.Time) error
	Finish(ctx context.Context, orderID int64) error
}

type OffersRepository interface {
	AlreadyOffered(ctx context.Context, orderID, driverID int64) (bool, error)
	CreateOffer(ctx context.Context, orderID, driverID int64, ttl time.Time) error
}

type DriverNotifier interface {
	SendOffer(driverID int64, payload ws.DriverOfferPayload)
}

type PassengerNotifier interface {
	PushOrderEvent(passengerID int64, event ws.PassengerEvent)
}

type PassengerRepository interface {
	Get(ctx context.Context, id int64) (repo.Passenger, error)
}

type driverLocator interface {
	Nearby(ctx context.Context, lon, lat float64, radiusMeters float64, limit int, city string) ([]geo.NearbyDriver, error)
}

type Dispatcher struct {
	orders      OrdersRepository
	dispatch    DispatchRepository
	offers      OffersRepository
	passengers  PassengerRepository
	locator     driverLocator
	driverWS    DriverNotifier
	passengerWS PassengerNotifier
	logger      Logger
	cfg         Config
}

// New creates a dispatcher instance.
func New(orders OrdersRepository, dispatch DispatchRepository, offers OffersRepository, passengers PassengerRepository, locator driverLocator, driverWS DriverNotifier, passengerWS PassengerNotifier, logger Logger, cfg Config) *Dispatcher {
	return &Dispatcher{orders: orders, dispatch: dispatch, offers: offers, passengers: passengers, locator: locator, driverWS: driverWS, passengerWS: passengerWS, logger: logger, cfg: cfg}
}

// Run starts the dispatcher loop.
func (d *Dispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(d.cfg.GetDispatchTick())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.tick(ctx)
		}
	}
}

func (d *Dispatcher) tick(ctx context.Context) {
	loc, _ := time.LoadLocation("Asia/Almaty")
	now := time.Now().In(loc)
	records, err := d.dispatch.ListDue(ctx, now)
	if err != nil {
		d.logger.Errorf("dispatch: list due failed: %v", err)
		return
	}
	for _, rec := range records {
		if err := d.processRecord(ctx, rec, now); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			d.logger.Errorf("dispatch: process order %d failed: %v", rec.OrderID, err)
		}
	}
}

func (d *Dispatcher) processRecord(ctx context.Context, rec repo.DispatchRecord, now time.Time) error {

	order, err := d.orders.Get(ctx, rec.OrderID)
	if err != nil {
		d.logger.Errorf("dispatch: load order %d failed: %v", rec.OrderID, err)
		return err
	}
	if order.Status != "searching" {
		d.logger.Infof("dispatch: order %d not searching (status=%s) → finish", rec.OrderID, order.Status)
		return d.dispatch.Finish(ctx, rec.OrderID)
	}

	if timeout := d.cfg.GetSearchTimeout(); timeout > 0 && now.Sub(rec.CreatedAt) >= timeout {
		d.logger.Infof("dispatch: order %d timed out after %s → mark not_found", rec.OrderID, timeout)
		if err := d.orders.UpdateStatusCAS(ctx, order.ID, "searching", "not_found"); err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
		} else {
			d.passengerWS.PushOrderEvent(order.PassengerID, ws.PassengerEvent{Type: "order_status", OrderID: order.ID, Status: "not_found"})
		}
		return d.dispatch.Finish(ctx, rec.OrderID)
	}

	cityKey := d.cfg.GetRegionID()
	if strings.TrimSpace(cityKey) == "" {
		cityKey = "astana" // fallback на время отладки
	}
	drivers, err := d.locator.Nearby(ctx, order.FromLon, order.FromLat, float64(rec.RadiusM), 20, cityKey)
	if err != nil {
		d.logger.Errorf("dispatch: Nearby failed: %v", err)
		return err
	}

	ttl := now.Add(d.cfg.GetOfferTTL())
	sentOffers := 0
	skippedExisting := 0

	var passengerPayload *ws.DriverPassenger
	if d.passengers != nil {
		passenger, err := d.passengers.Get(ctx, order.PassengerID)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				d.logger.Errorf("dispatch: load passenger %d failed: %v", order.PassengerID, err)
			}
		} else {
			payload := mapPassengerToWS(passenger)
			passengerPayload = &payload
		}
	}

	for _, driver := range drivers {
		offered, err := d.offers.AlreadyOffered(ctx, order.ID, driver.ID)
		if err != nil {
			d.logger.Errorf("dispatch: AlreadyOffered(order=%d,driver=%d) failed: %v", order.ID, driver.ID, err)
			// НЕ прерываем — продолжаем со следующими
			continue
		}
		if offered {
			skippedExisting++
			continue
		}

		if err := d.offers.CreateOffer(ctx, order.ID, driver.ID, ttl); err != nil {
			d.logger.Errorf("dispatch: CreateOffer(order=%d,driver=%d) failed: %v", order.ID, driver.ID, err)
			// НЕ прерываем — продолжаем со следующими
			continue
		}

		payload := ws.DriverOfferPayload{
			OrderID:      order.ID,
			FromLon:      order.FromLon,
			FromLat:      order.FromLat,
			ToLon:        order.ToLon,
			ToLat:        order.ToLat,
			ClientPrice:  order.ClientPrice,
			DistanceM:    order.DistanceM,
			EtaSeconds:   order.EtaSeconds,
			ExpiresInSec: int(d.cfg.GetOfferTTL().Seconds()),
			Passenger:    passengerPayload,
		}
		if len(order.Addresses) > 0 {
			route := make([]ws.DriverRoutePoint, 0, len(order.Addresses))
			for _, addr := range order.Addresses {
				point := ws.DriverRoutePoint{Lon: addr.Lon, Lat: addr.Lat}
				if addr.Address.Valid {
					point.Address = addr.Address.String
				}
				route = append(route, point)
			}
			payload.Route = route
		}
		d.driverWS.SendOffer(driver.ID, payload)
		sentOffers++
		d.logger.Infof("✅ dispatch: offer created & sent order=%d → driver=%d (ttl=%s)", order.ID, driver.ID, ttl.Format(time.RFC3339))
	}

	// Планирование следующего тика
	switch {
	case len(drivers) == 0:
		// никого не нашли — расширяем радиус
		newRadius := rec.RadiusM + d.cfg.GetSearchRadiusStep()
		if newRadius > d.cfg.GetSearchRadiusMax() {
			newRadius = d.cfg.GetSearchRadiusMax()
		}
		next := now.Add(d.cfg.GetDispatchTick())
		if err := d.dispatch.UpdateRadius(ctx, rec.OrderID, newRadius, next); err != nil {
			return err
		}
		d.logger.Infof("dispatch: no drivers; radius ↑ to %d; next_tick=%s", newRadius, next.Format(time.RFC3339))
		d.passengerWS.PushOrderEvent(order.PassengerID, ws.PassengerEvent{Type: "search_progress", OrderID: order.ID, Radius: newRadius})

	case sentOffers == 0 && skippedExisting > 0:
		// все найденные водители уже получили оффер — расширяем радиус, чтобы найти новых
		newRadius := rec.RadiusM + d.cfg.GetSearchRadiusStep()
		if newRadius > d.cfg.GetSearchRadiusMax() {
			newRadius = d.cfg.GetSearchRadiusMax()
		}
		next := now.Add(d.cfg.GetDispatchTick() / 2)
		if next.Before(now.Add(1 * time.Second)) {
			next = now.Add(1 * time.Second)
		}
		if err := d.dispatch.UpdateRadius(ctx, rec.OrderID, newRadius, next); err != nil {
			return err
		}
		if newRadius > rec.RadiusM {

			d.passengerWS.PushOrderEvent(order.PassengerID, ws.PassengerEvent{Type: "search_progress", OrderID: order.ID, Radius: newRadius})
		} else {
			d.passengerWS.PushOrderEvent(order.PassengerID, ws.PassengerEvent{Type: "searching", OrderID: order.ID, Radius: rec.RadiusM})
		}

	default:
		// офферы отправлены — оставляем радиус, ставим обычный next_tick
		next := now.Add(d.cfg.GetDispatchTick())
		if err := d.dispatch.UpdateRadius(ctx, rec.OrderID, rec.RadiusM, next); err != nil {
			return err
		}
		d.passengerWS.PushOrderEvent(order.PassengerID, ws.PassengerEvent{Type: "searching", OrderID: order.ID, Radius: rec.RadiusM})
	}

	return nil
}

func mapPassengerToWS(p repo.Passenger) ws.DriverPassenger {
	result := ws.DriverPassenger{
		ID:        p.ID,
		Name:      p.Name,
		Surname:   p.Surname,
		Phone:     p.Phone,
		Email:     p.Email,
		CreatedAt: p.CreatedAt,
	}
	if p.Middlename.Valid {
		result.Middlename = p.Middlename.String
	}
	if p.CityID.Valid {
		id := p.CityID.Int64
		result.CityID = &id
	}
	if p.YearsOfExp.Valid {
		exp := p.YearsOfExp.Int64
		result.YearsOfExp = &exp
	}
	if p.DocOfProof.Valid {
		result.DocOfProof = p.DocOfProof.String
	}
	if p.ReviewRating.Valid {
		rating := p.ReviewRating.Float64
		result.ReviewRating = &rating
	}
	if p.Role.Valid {
		result.Role = p.Role.String
	}
	if p.Latitude.Valid {
		result.Latitude = p.Latitude.String
	}
	if p.Longitude.Valid {
		result.Longitude = p.Longitude.String
	}
	if p.AvatarPath.Valid {
		result.AvatarPath = p.AvatarPath.String
	}
	if p.Skills.Valid {
		result.Skills = p.Skills.String
	}
	if p.IsOnline.Valid {
		online := p.IsOnline.Bool
		result.IsOnline = &online
	}
	if p.UpdatedAt.Valid {
		updated := p.UpdatedAt.Time
		result.UpdatedAt = &updated
	}
	return result
}

// TriggerImmediate schedules an order for immediate dispatch tick.
func (d *Dispatcher) TriggerImmediate(ctx context.Context, orderID int64) error {
	return d.dispatch.UpdateRadius(ctx, orderID, d.cfg.GetSearchRadiusStart(), timeutil.Now())
}

// ConfigAdapter allows TaxiConfig to satisfy Config interface.
type ConfigAdapter struct {
	PricePerKM        int
	MinPrice          int
	SearchRadiusStart int
	SearchRadiusStep  int
	SearchRadiusMax   int
	DispatchTick      time.Duration
	OfferTTL          time.Duration
	RegionID          string
	SearchTimeout     time.Duration
}

func (c ConfigAdapter) GetPricePerKM() int              { return c.PricePerKM }
func (c ConfigAdapter) GetMinPrice() int                { return c.MinPrice }
func (c ConfigAdapter) GetSearchRadiusStart() int       { return c.SearchRadiusStart }
func (c ConfigAdapter) GetSearchRadiusStep() int        { return c.SearchRadiusStep }
func (c ConfigAdapter) GetSearchRadiusMax() int         { return c.SearchRadiusMax }
func (c ConfigAdapter) GetDispatchTick() time.Duration  { return c.DispatchTick }
func (c ConfigAdapter) GetOfferTTL() time.Duration      { return c.OfferTTL }
func (c ConfigAdapter) GetRegionID() string             { return c.RegionID }
func (c ConfigAdapter) GetSearchTimeout() time.Duration { return c.SearchTimeout }

// RecalculateRecommendedPrice recalculates price based on distance.
func RecalculateRecommendedPrice(distanceM int, cfg Config) int {
	return pricing.Recommended(distanceM, cfg.GetPricePerKM(), cfg.GetMinPrice())
}
