package dispatch

import (
	"context"
	"database/sql"
	"errors"
	"strings"
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
	// 1) Грузим заказ
	order, err := d.orders.Get(ctx, rec.OrderID)
	if err != nil {
		d.logger.Errorf("courier dispatch: load order %d failed: %v", rec.OrderID, err)
		return err
	}

	// Прерываем, если уже не в поиске (у тебя это repo.StatusNew)
	if order.Status != repo.StatusNew {
		d.logger.Infof("courier dispatch: order %d not searching (status=%s) → finish", rec.OrderID, order.Status)
		return d.dispatch.Finish(ctx, rec.OrderID)
	}

	// Проверяем маршрут
	if len(order.Points) == 0 {
		d.logger.Errorf("courier dispatch: order %d has no route points → finish", rec.OrderID)
		return d.dispatch.Finish(ctx, rec.OrderID)
	}

	// 2) Таймаут поиска — как в такси (CAS → "not_found"), но без senderWS
	if timeout := d.cfg.GetSearchTimeout(); timeout > 0 && now.Sub(rec.CreatedAt) >= timeout {
		d.logger.Infof("courier dispatch: order %d timed out after %s → mark not_found", rec.OrderID, timeout)
		// если у тебя есть UpdateStatusCAS как в такси — используем его
		if err := d.orders.UpdateStatusCAS(ctx, order.ID, repo.StatusNew, "not_found"); err != nil {
			// если CAS не прошёл из-за гонки — игнорим, иначе отдаём ошибку
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
		}
		// завершаем диспатч
		return d.dispatch.Finish(ctx, rec.OrderID)
	}

	// 3) Поиск ближайших курьеров
	origin := order.Points[0]
	cityKey := strings.TrimSpace(d.cfg.GetRegionKey())
	if cityKey == "" {
		cityKey = "astana" // как в такси — fallback на отладку
	}

	drivers, err := d.locator.Nearby(ctx, origin.Lon, origin.Lat, float64(rec.RadiusM), 20, cityKey)
	if err != nil {
		d.logger.Errorf("courier dispatch: Nearby failed: %v", err)
		return err
	}

	// TTL только в payload (как у такси), без изменения репозитория
	ttlSeconds := int(d.cfg.GetOfferTTL().Seconds())
	sentOffers := 0
	skippedExisting := 0

	// Базовый payload оффера курьеру
	payload := ws.CourierOfferPayload{
		OrderID:          order.ID,
		ClientPrice:      order.ClientPrice,
		RecommendedPrice: order.RecommendedPrice,
		DistanceM:        order.DistanceM,
		EtaSeconds:       order.EtaSeconds,
		ExpiresInSec:     ttlSeconds,
		Points:           makeRoutePoints(order.Points),
	}

	// 4) Создание и отправка офферов (как в такси), но через твой имеющийся CreateOffer
	for _, driver := range drivers {
		offered, err := d.offers.AlreadyOffered(ctx, order.ID, driver.ID)
		if err != nil {
			d.logger.Errorf("courier dispatch: AlreadyOffered(order=%d,courier=%d) failed: %v", order.ID, driver.ID, err)
			continue
		}
		if offered {
			skippedExisting++
			continue
		}

		// у тебя уже есть CreateOffer с ценой — используем его
		if err := d.offers.CreateOffer(ctx, order.ID, driver.ID, order.ClientPrice); err != nil {
			d.logger.Errorf("courier dispatch: CreateOffer(order=%d,courier=%d) failed: %v", order.ID, driver.ID, err)
			continue
		}

		d.courierWS.SendOffer(driver.ID, payload)
		sentOffers++
		d.logger.Infof("✅ courier dispatch: offer created & sent order=%d → courier=%d", order.ID, driver.ID)
	}

	// 5) Планирование следующего тика — логика как в такси
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
		d.logger.Infof("courier dispatch: no couriers; radius ↑ to %d; next_tick=%s", newRadius, next.Format(time.RFC3339))

	case sentOffers == 0 && skippedExisting > 0:
		// все найденные уже получали оффер — расширяем радиус быстрее
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
		d.logger.Infof("courier dispatch: only previously-offered couriers; radius ↑ to %d; next_tick=%s", newRadius, next.Format(time.RFC3339))

	default:
		// офферы отправили — оставляем радиус и ставим обычный next_tick
		next := now.Add(d.cfg.GetDispatchTick())
		if err := d.dispatch.UpdateRadius(ctx, rec.OrderID, rec.RadiusM, next); err != nil {
			return err
		}
		d.logger.Infof("courier dispatch: offers sent; keep radius=%d; next_tick=%s", rec.RadiusM, next.Format(time.RFC3339))
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
