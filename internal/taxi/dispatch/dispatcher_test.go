package dispatch

import (
	"context"
	"testing"
	"time"

	"naimuBack/internal/taxi/geo"
	"naimuBack/internal/taxi/repo"
	"naimuBack/internal/taxi/ws"
)

type testLogger struct{}

func (testLogger) Infof(string, ...interface{})  {}
func (testLogger) Errorf(string, ...interface{}) {}

type stubOrders struct {
	order repo.Order
	from  string
	to    string
}

func (s *stubOrders) Get(ctx context.Context, id int64) (repo.Order, error) {
	return s.order, nil
}

func (s *stubOrders) UpdateStatusCAS(ctx context.Context, orderID int64, fromStatus, toStatus string) error {
	s.from = fromStatus
	s.to = toStatus
	s.order.Status = toStatus
	return nil
}

type stubDispatch struct {
	radius   int
	next     time.Time
	finished bool
}

func (s *stubDispatch) ListDue(ctx context.Context, now time.Time) ([]repo.DispatchRecord, error) {
	return nil, nil
}

func (s *stubDispatch) UpdateRadius(ctx context.Context, orderID int64, radius int, next time.Time) error {
	s.radius = radius
	s.next = next
	return nil
}

func (s *stubDispatch) Finish(ctx context.Context, orderID int64) error {
	s.finished = true
	return nil
}

type stubOffers struct{}

func (stubOffers) AlreadyOffered(ctx context.Context, orderID, driverID int64) (bool, error) {
	return false, nil
}
func (stubOffers) CreateOffer(ctx context.Context, orderID, driverID int64, ttl time.Time) error {
	return nil
}

type stubDriverHub struct{ sent int }

func (s *stubDriverHub) SendOffer(driverID int64, payload ws.DriverOfferPayload) { s.sent++ }

type stubPassengers struct {
	passenger repo.Passenger
	err       error
}

func (s *stubPassengers) Get(ctx context.Context, id int64) (repo.Passenger, error) {
	if s.err != nil {
		return repo.Passenger{}, s.err
	}
	if s.passenger.ID == 0 {
		s.passenger.ID = id
	}
	return s.passenger, nil
}

type stubPassengerHub struct {
	events []ws.PassengerEvent
}

func (s *stubPassengerHub) PushOrderEvent(passengerID int64, event ws.PassengerEvent) {
	s.events = append(s.events, event)
}

type stubLocator struct {
	drivers []geo.NearbyDriver
}

func (s *stubLocator) Nearby(ctx context.Context, lon, lat float64, radiusMeters float64, limit int, city string) ([]geo.NearbyDriver, error) {
	return s.drivers, nil
}

func TestDispatcherRadiusExpansion(t *testing.T) {
	locator := &stubLocator{}
	orders := &stubOrders{order: repo.Order{ID: 1, PassengerID: 10, FromLon: 76.9, FromLat: 43.2, Status: "searching"}}
	dispatchRepo := &stubDispatch{}
	offers := &stubOffers{}
	driverHub := &stubDriverHub{}
	passengers := &stubPassengers{}
	passengerHub := &stubPassengerHub{}

	cfg := ConfigAdapter{
		PricePerKM:        300,
		MinPrice:          1200,
		SearchRadiusStart: 800,
		SearchRadiusStep:  400,
		SearchRadiusMax:   3000,
		DispatchTick:      time.Minute,
		OfferTTL:          20 * time.Second,
		RegionID:          "test",
		SearchTimeout:     time.Hour,
	}

	d := New(orders, dispatchRepo, offers, passengers, locator, driverHub, passengerHub, testLogger{}, cfg)

	now := time.Now()
	rec := repo.DispatchRecord{OrderID: 1, RadiusM: cfg.SearchRadiusStart, NextTickAt: now, CreatedAt: now}
	if err := d.processRecord(context.Background(), rec, now); err != nil {
		t.Fatalf("processRecord error: %v", err)
	}
	if dispatchRepo.radius != cfg.SearchRadiusStart+cfg.SearchRadiusStep {
		t.Fatalf("expected radius to increase, got %d", dispatchRepo.radius)
	}
	expectedNext := now.Add(cfg.DispatchTick)
	if dispatchRepo.next.Before(expectedNext.Add(-time.Second)) || dispatchRepo.next.After(expectedNext.Add(time.Second)) {
		t.Fatalf("expected next tick around %v, got %v", expectedNext, dispatchRepo.next)
	}
	if len(passengerHub.events) == 0 || passengerHub.events[0].Type != "search_progress" {
		t.Fatalf("expected search_progress event")
	}
}

func TestDispatcherTimeoutCancelsOrder(t *testing.T) {
	timeout := 10 * time.Minute
	locator := &stubLocator{}
	orders := &stubOrders{order: repo.Order{ID: 1, PassengerID: 42, Status: "searching", CreatedAt: time.Now().Add(-timeout - time.Minute)}}
	dispatchRepo := &stubDispatch{}
	offers := &stubOffers{}
	driverHub := &stubDriverHub{}
	passengers := &stubPassengers{}
	passengerHub := &stubPassengerHub{}

	cfg := ConfigAdapter{
		PricePerKM:        300,
		MinPrice:          1200,
		SearchRadiusStart: 800,
		SearchRadiusStep:  400,
		SearchRadiusMax:   3000,
		DispatchTick:      time.Minute,
		OfferTTL:          20 * time.Second,
		RegionID:          "test",
		SearchTimeout:     timeout,
	}

	d := New(orders, dispatchRepo, offers, passengers, locator, driverHub, passengerHub, testLogger{}, cfg)

	now := time.Now()
	rec := repo.DispatchRecord{OrderID: 1, RadiusM: cfg.SearchRadiusStart, NextTickAt: now, CreatedAt: now.Add(-timeout - time.Minute)}
	if err := d.processRecord(context.Background(), rec, now); err != nil {
		t.Fatalf("processRecord error: %v", err)
	}
	if !dispatchRepo.finished {
		t.Fatalf("expected dispatch to finish after timeout")
	}
	if orders.to != "not_found" {
		t.Fatalf("expected order status to change to not_found, got %q", orders.to)
	}
	if len(passengerHub.events) == 0 {
		t.Fatalf("expected passenger event to be sent")
	}
	if passengerHub.events[0].Type != "order_status" || passengerHub.events[0].Status != "not_found" {
		t.Fatalf("unexpected passenger event: %+v", passengerHub.events[0])
	}
}
