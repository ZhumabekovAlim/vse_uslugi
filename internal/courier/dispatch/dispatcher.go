package dispatch

import (
	"context"
	"errors"
	"time"

	"naimuBack/internal/courier/geo"
	"naimuBack/internal/courier/repo"
	"naimuBack/internal/courier/ws"
)

// Logger provides minimal logging for courier dispatcher.
type Logger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// Config exposes dispatcher configuration knobs.
type Config interface {
	GetPricePerKM() int
	GetMinPrice() int
	GetSearchRadiusStart() int
	GetSearchRadiusStep() int
	GetSearchRadiusMax() int
	GetDispatchTick() time.Duration
	GetOfferTTL() time.Duration
	GetSearchTimeout() time.Duration
	GetRegionKey() string
}

// OrdersRepository covers minimal order operations required by dispatcher.
type OrdersRepository interface {
	Get(ctx context.Context, id int64) (repo.Order, error)
	UpdateStatusCAS(ctx context.Context, orderID int64, fromStatus, toStatus string) error
}

// DispatchRepository encapsulates courier dispatch state persistence.
type DispatchRepository interface {
	ListDue(ctx context.Context, now time.Time) ([]repo.DispatchRecord, error)
	UpdateRadius(ctx context.Context, orderID int64, radius int, next time.Time) error
	Finish(ctx context.Context, orderID int64) error
	TriggerImmediate(ctx context.Context, orderID int64, next time.Time) error
}

// OffersRepository stores lightweight offer stubs to avoid duplicates.
type OffersRepository interface {
	AlreadyOffered(ctx context.Context, orderID, courierID int64) (bool, error)
	CreateOffer(ctx context.Context, orderID, courierID int64, price int) error
}

// CourierNotifier dispatches offers to couriers over WebSocket.
type CourierNotifier interface {
	SendOffer(courierID int64, payload ws.CourierOfferPayload)
}

type courierLocator interface {
	Nearby(ctx context.Context, lon, lat float64, radiusMeters float64, limit int, city string) ([]geo.NearbyCourier, error)
}

// Dispatcher implements periodic courier matching against nearby executors.
type Dispatcher struct {
	orders    OrdersRepository
	dispatch  DispatchRepository
	offers    OffersRepository
	locator   courierLocator
	courierWS CourierNotifier
	logger    Logger
	cfg       Config
}

// New constructs a dispatcher instance.
func New(orders OrdersRepository, dispatch DispatchRepository, offers OffersRepository, locator courierLocator, courierWS CourierNotifier, logger Logger, cfg Config) *Dispatcher {
	return &Dispatcher{orders: orders, dispatch: dispatch, offers: offers, locator: locator, courierWS: courierWS, logger: logger, cfg: cfg}
}

// Run launches the dispatcher loop until the context is cancelled.
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
	now := time.Now()
	records, err := d.dispatch.ListDue(ctx, now)
	if err != nil {
		d.logger.Errorf("courier dispatch: list due failed: %v", err)
		return
	}
	for _, rec := range records {
		if err := d.processRecord(ctx, rec, now); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			d.logger.Errorf("courier dispatch: process order %d failed: %v", rec.OrderID, err)
		}
	}
}

func (d *Dispatcher) processRecord(ctx context.Context, rec repo.DispatchRecord, now time.Time) error {
	order, err := d.orders.Get(ctx, rec.OrderID)
	if err != nil {
		return err
	}
	if order.Status != repo.StatusNew {
		if err := d.dispatch.Finish(ctx, rec.OrderID); err != nil {
			d.logger.Errorf("courier dispatch: finish order %d failed: %v", rec.OrderID, err)
			return err
		}
		return nil
	}
	if len(order.Points) == 0 {
		d.logger.Errorf("courier dispatch: order %d has no route points", rec.OrderID)
		return d.dispatch.Finish(ctx, rec.OrderID)
	}

	timeout := d.cfg.GetSearchTimeout()
	if timeout > 0 && now.Sub(rec.CreatedAt) >= timeout {
		d.logger.Infof("courier dispatch: order %d timed out after %s", rec.OrderID, timeout)
		if err := d.dispatch.Finish(ctx, rec.OrderID); err != nil {
			d.logger.Errorf("courier dispatch: finish timed out order %d failed: %v", rec.OrderID, err)
			return err
		}
		return nil
	}

	origin := order.Points[0]
	city := d.cfg.GetRegionKey()
	drivers, err := d.locator.Nearby(ctx, origin.Lon, origin.Lat, float64(rec.RadiusM), 20, city)
	if err != nil {
		return err
	}

	offersCreated := 0
	ttlSeconds := int(d.cfg.GetOfferTTL().Seconds())
	payload := ws.CourierOfferPayload{
		OrderID:          order.ID,
		ClientPrice:      order.ClientPrice,
		RecommendedPrice: order.RecommendedPrice,
		DistanceM:        order.DistanceM,
		EtaSeconds:       order.EtaSeconds,
		ExpiresInSec:     ttlSeconds,
		Points:           makeRoutePoints(order.Points),
	}

	for _, driver := range drivers {
		offered, err := d.offers.AlreadyOffered(ctx, order.ID, driver.ID)
		if err != nil {
			d.logger.Errorf("courier dispatch: AlreadyOffered order=%d courier=%d failed: %v", order.ID, driver.ID, err)
			continue
		}
		if offered {
			continue
		}
		if err := d.offers.CreateOffer(ctx, order.ID, driver.ID, order.ClientPrice); err != nil {
			d.logger.Errorf("courier dispatch: CreateOffer order=%d courier=%d failed: %v", order.ID, driver.ID, err)
			continue
		}
		offersCreated++
		d.courierWS.SendOffer(driver.ID, payload)
	}

	nextRadius := rec.RadiusM
	if offersCreated == 0 && rec.RadiusM < d.cfg.GetSearchRadiusMax() {
		nextRadius = rec.RadiusM + d.cfg.GetSearchRadiusStep()
		if nextRadius > d.cfg.GetSearchRadiusMax() {
			nextRadius = d.cfg.GetSearchRadiusMax()
		}
		d.logger.Infof("courier dispatch: expanding radius for order %d to %d m", rec.OrderID, nextRadius)
	}
	next := now.Add(d.cfg.GetDispatchTick())
	if err := d.dispatch.UpdateRadius(ctx, rec.OrderID, nextRadius, next); err != nil {
		d.logger.Errorf("courier dispatch: update radius order=%d failed: %v", rec.OrderID, err)
		return err
	}
	return nil
}

// TriggerImmediate schedules an order for immediate processing.
func (d *Dispatcher) TriggerImmediate(ctx context.Context, orderID int64) error {
	return d.dispatch.TriggerImmediate(ctx, orderID, time.Now())
}

func makeRoutePoints(points []repo.OrderPoint) []ws.CourierRoutePoint {
	res := make([]ws.CourierRoutePoint, 0, len(points))
	for _, p := range points {
		res = append(res, ws.CourierRoutePoint{Seq: p.Seq, Address: p.Address, Lon: p.Lon, Lat: p.Lat})
	}
	return res
}
