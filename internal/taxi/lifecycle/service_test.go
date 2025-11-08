package lifecycle

import (
	"testing"
	"time"

	"naimuBack/internal/taxi/fsm"
)

func TestLifecycleHappyPath(t *testing.T) {
	cfg := Config{
		ArrivalRadiusMeters:      50,
		StartRadiusMeters:        50,
		WaypointRadiusMeters:     30,
		FinishRadiusMeters:       50,
		StationarySpeedKPH:       5,
		CoordinateFreshness:      time.Minute,
		FreeWaitingWindow:        3 * time.Minute,
		PaidWaitingRatePerMinute: 100,
		PauseRatePerMinute:       200,
		OfferTTL:                 15 * time.Minute,
		ButtonPolicies:           map[Action]ButtonPolicy{},
	}
	svc := NewService(cfg)

	route := []WaypointTarget{
		{Kind: WaypointPickup, Name: "A", Point: GeoPoint{Lon: 76.9000, Lat: 43.2500}},
		{Kind: WaypointStop, Name: "P1", Point: GeoPoint{Lon: 76.9050, Lat: 43.2550}},
		{Kind: WaypointFinish, Name: "B", Point: GeoPoint{Lon: 76.9100, Lat: 43.2600}},
	}
	created := time.Date(2023, 10, 5, 10, 0, 0, 0, time.UTC)
	order, err := NewOrder(1, 100, 200, 1500, "KZT", created, cfg.OfferTTL, route)
	if err != nil {
		t.Fatalf("NewOrder: %v", err)
	}
	if order.Status != fsm.StatusAssigned {
		t.Fatalf("expected initial status assigned, got %s", order.Status)
	}

	arriveTime := created.Add(10 * time.Minute)
	arriveTelemetry := Telemetry{Position: route[0].Point, SpeedKPH: 2, Timestamp: arriveTime}
	if err := svc.MarkDriverAtPickup(order, arriveTime, arriveTelemetry); err != nil {
		t.Fatalf("MarkDriverAtPickup: %v", err)
	}
	if order.Status != fsm.StatusWaitingFree {
		t.Fatalf("expected waiting_free status, got %s", order.Status)
	}
	if order.ArrivedAt == nil || !order.ArrivedAt.Equal(arriveTime) {
		t.Fatalf("arrival timestamp missing")
	}

	// Complimentary waiting should switch to paid after 3 minutes
	switched, err := svc.AdvanceWaiting(order, arriveTime.Add(4*time.Minute))
	if err != nil {
		t.Fatalf("AdvanceWaiting: %v", err)
	}
	if !switched {
		t.Fatalf("expected switch to paid waiting")
	}
	if order.Status != fsm.StatusWaitingPaid {
		t.Fatalf("expected waiting_paid status, got %s", order.Status)
	}

	startTime := arriveTime.Add(5 * time.Minute)
	startTelemetry := Telemetry{Position: route[0].Point, SpeedKPH: 0.5, Timestamp: startTime}
	if err := svc.StartTrip(order, startTime, startTelemetry, true); err != nil {
		t.Fatalf("StartTrip: %v", err)
	}
	if order.Status != fsm.StatusInProgress {
		t.Fatalf("expected in_progress status, got %s", order.Status)
	}
	if order.StartedAt == nil || !order.StartedAt.Equal(startTime) {
		t.Fatalf("start timestamp mismatch")
	}
	if got := order.Fare.PaidWaitingMinutes; got != 2 {
		t.Fatalf("expected 2 minutes of paid waiting, got %d", got)
	}
	if got := order.Fare.WaitingPaidAmount; got != 200 {
		t.Fatalf("expected paid waiting amount 200, got %d", got)
	}

	// reach intermediate point
	wpTime := startTime.Add(10 * time.Minute)
	wpTelemetry := Telemetry{Position: route[1].Point, SpeedKPH: 10, Timestamp: wpTime}
	if err := svc.ReachWaypoint(order, wpTime, wpTelemetry); err != nil {
		t.Fatalf("ReachWaypoint: %v", err)
	}
	if len(order.WaypointLog) != 2 { // pickup + P1
		t.Fatalf("expected 2 waypoint events, got %d", len(order.WaypointLog))
	}

	// pause and resume
	pauseStart := startTime.Add(12 * time.Minute)
	if err := svc.StartPause(order, pauseStart); err != nil {
		t.Fatalf("StartPause: %v", err)
	}
	pauseEnd := pauseStart.Add(2 * time.Minute)
	if err := svc.EndPause(order, pauseEnd); err != nil {
		t.Fatalf("EndPause: %v", err)
	}
	if got := order.Fare.PauseWaitingMinutes; got != 2 {
		t.Fatalf("expected 2 pause minutes, got %d", got)
	}
	if got := order.Fare.WaitingPauseAmount; got != 400 {
		t.Fatalf("expected pause amount 400, got %d", got)
	}

	finishTime := startTime.Add(20 * time.Minute)
	finishTelemetry := Telemetry{Position: route[2].Point, SpeedKPH: 1, Timestamp: finishTime}
	if err := svc.FinishTrip(order, finishTime, finishTelemetry); err != nil {
		t.Fatalf("FinishTrip: %v", err)
	}
	if order.Status != fsm.StatusAtLastPoint {
		t.Fatalf("expected at_last_point, got %s", order.Status)
	}
	if order.FinishedAt == nil || !order.FinishedAt.Equal(finishTime) {
		t.Fatalf("finish timestamp mismatch")
	}

	confirmTime := finishTime.Add(1 * time.Minute)
	if err := svc.ConfirmCashPayment(order, confirmTime); err != nil {
		t.Fatalf("ConfirmCashPayment: %v", err)
	}
	if order.Status != fsm.StatusCompleted {
		t.Fatalf("expected completed, got %s", order.Status)
	}
	if order.PaymentConfirmed == nil || !order.PaymentConfirmed.Equal(confirmTime) {
		t.Fatalf("payment confirmation timestamp mismatch")
	}
	if got := order.Fare.Total(); got != 1500+200+400 {
		t.Fatalf("unexpected fare total %d", got)
	}
}

func TestNoShowAndCancel(t *testing.T) {
	cfg := Config{
		ArrivalRadiusMeters:      40,
		StartRadiusMeters:        40,
		WaypointRadiusMeters:     25,
		FinishRadiusMeters:       40,
		StationarySpeedKPH:       5,
		CoordinateFreshness:      time.Minute,
		FreeWaitingWindow:        time.Minute,
		PaidWaitingRatePerMinute: 150,
		PauseRatePerMinute:       0,
		OfferTTL:                 10 * time.Minute,
		ButtonPolicies:           map[Action]ButtonPolicy{},
	}
	svc := NewService(cfg)
	route := []WaypointTarget{
		{Kind: WaypointPickup, Name: "A", Point: GeoPoint{Lon: 10, Lat: 10}},
		{Kind: WaypointFinish, Name: "B", Point: GeoPoint{Lon: 10.01, Lat: 10.01}},
	}
	created := time.Unix(0, 0)
	order, err := NewOrder(2, 10, 20, 1000, "KZT", created, cfg.OfferTTL, route)
	if err != nil {
		t.Fatalf("NewOrder: %v", err)
	}
	arriveTime := created.Add(2 * time.Minute)
	telemetry := Telemetry{Position: route[0].Point, SpeedKPH: 0.1, Timestamp: arriveTime}
	if err := svc.MarkDriverAtPickup(order, arriveTime, telemetry); err != nil {
		t.Fatalf("MarkDriverAtPickup: %v", err)
	}
	// No show allowed after waiting window
	if _, err := svc.AdvanceWaiting(order, arriveTime.Add(2*time.Minute)); err != nil {
		t.Fatalf("AdvanceWaiting: %v", err)
	}
	noShowTelemetry := Telemetry{Position: route[0].Point, SpeedKPH: 0.1, Timestamp: arriveTime.Add(3 * time.Minute)}
	if err := svc.MarkNoShow(order, arriveTime.Add(3*time.Minute), noShowTelemetry); err != nil {
		t.Fatalf("MarkNoShow: %v", err)
	}
	if order.Status != fsm.StatusNoShow {
		t.Fatalf("expected no_show status, got %s", order.Status)
	}

	// cancel by driver idempotent
	if err := svc.CancelByDriver(order, arriveTime.Add(4*time.Minute), "passenger absent"); err == nil {
		t.Fatalf("cancel after no_show should fail")
	}

	// new order for passenger cancel
	order2, err := NewOrder(3, 11, 21, 900, "KZT", created, cfg.OfferTTL, route)
	if err != nil {
		t.Fatalf("NewOrder2: %v", err)
	}
	if err := svc.MarkDriverAtPickup(order2, arriveTime, telemetry); err != nil {
		t.Fatalf("MarkDriverAtPickup order2: %v", err)
	}
	cancelTime := arriveTime.Add(30 * time.Second)
	if err := svc.CancelByPassenger(order2, cancelTime, "changed plans"); err != nil {
		t.Fatalf("CancelByPassenger: %v", err)
	}
	if order2.Status != fsm.StatusCanceledByPassenger {
		t.Fatalf("expected canceled_by_passenger, got %s", order2.Status)
	}
	if err := svc.CancelByPassenger(order2, cancelTime.Add(10*time.Second), "changed plans"); err != nil {
		t.Fatalf("CancelByPassenger second call should be idempotent: %v", err)
	}
}
