package dispatch

import (
    "context"
    "testing"
    "time"

    "naimuBack/internal/taxi/geo"
    "naimuBack/internal/taxi/repo"
    "naimuBack/internal/taxi/ws"
    "github.com/redis/go-redis/v9"
)

type testLogger struct{}

func (testLogger) Infof(string, ...interface{}) {}
func (testLogger) Errorf(string, ...interface{}) {}

type stubOrders struct {
    order repo.Order
}

func (s *stubOrders) Get(ctx context.Context, id int64) (repo.Order, error) {
    return s.order, nil
}

type stubDispatch struct {
    radius int
    next   time.Time
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

func (stubOffers) AlreadyOffered(ctx context.Context, orderID, driverID int64) (bool, error) { return false, nil }
func (stubOffers) CreateOffer(ctx context.Context, orderID, driverID int64, ttl time.Time) error { return nil }

type stubDriverHub struct { sent int }

func (s *stubDriverHub) SendOffer(driverID int64, payload ws.DriverOfferPayload) { s.sent++ }

type stubPassengerHub struct {
    events []ws.PassengerEvent
}

func (s *stubPassengerHub) PushOrderEvent(passengerID int64, event ws.PassengerEvent) {
    s.events = append(s.events, event)
}

func TestDispatcherRadiusExpansion(t *testing.T) {
    locator := geo.NewDriverLocator(redis.NewClient(nil))
    orders := &stubOrders{order: repo.Order{ID: 1, PassengerID: 10, FromLon: 76.9, FromLat: 43.2, Status: "searching"}}
    dispatchRepo := &stubDispatch{}
    offers := &stubOffers{}
    driverHub := &stubDriverHub{}
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
    }

    d := New(orders, dispatchRepo, offers, locator, driverHub, passengerHub, testLogger{}, cfg)

    rec := repo.DispatchRecord{OrderID: 1, RadiusM: cfg.SearchRadiusStart}
    if err := d.processRecord(context.Background(), rec); err != nil {
        t.Fatalf("processRecord error: %v", err)
    }
    if dispatchRepo.radius != cfg.SearchRadiusStart+cfg.SearchRadiusStep {
        t.Fatalf("expected radius to increase, got %d", dispatchRepo.radius)
    }
    if len(passengerHub.events) == 0 || passengerHub.events[0].Type != "search_progress" {
        t.Fatalf("expected search_progress event")
    }
}
