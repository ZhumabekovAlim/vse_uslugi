package lifecycle

import (
	"fmt"
	"time"

	"naimuBack/internal/taxi/fsm"
)

// Service encapsulates business operations for taxi order lifecycle.
type Service struct {
	cfg Config
}

// NewService constructs a Service instance.
func NewService(cfg Config) *Service {
	if cfg.ButtonPolicies == nil {
		cfg.ButtonPolicies = make(map[Action]ButtonPolicy)
	}
	return &Service{cfg: cfg}
}

// Config returns copy of the service configuration.
func (s *Service) Config() Config {
	return s.cfg
}

func (s *Service) validateTelemetry(now time.Time, telemetry Telemetry) error {
	if telemetry.Timestamp.IsZero() {
		return ErrOutdatedTelemetry
	}
	if now.Sub(telemetry.Timestamp) > s.cfg.CoordinateFreshness {
		return ErrOutdatedTelemetry
	}
	return nil
}

func (s *Service) ensureActionAllowed(order *Order, action Action, now time.Time) error {
	policy, ok := s.cfg.ButtonPolicies[action]
	if !ok {
		return nil
	}
	state := order.buttonStateFor(action)
	if policy.TTL > 0 {
		if state.expiresAt.IsZero() || now.After(state.expiresAt) {
			state.count = 0
			state.expiresAt = now.Add(policy.TTL)
		}
	}
	if policy.MaxPresses > 0 && state.count >= policy.MaxPresses {
		return fmt.Errorf("action %s exceeded retry limit", action)
	}
	if !state.lastPress.IsZero() && now.Sub(state.lastPress) < policy.Cooldown {
		return fmt.Errorf("action %s pressed too frequently", action)
	}
	state.lastPress = now
	state.count++
	return nil
}

// MarkDriverAtPickup handles "I'm on site" button.
func (s *Service) MarkDriverAtPickup(order *Order, now time.Time, telemetry Telemetry) error {
	if order.Status == fsm.StatusWaitingFree || order.Status == fsm.StatusWaitingPaid || order.Status == fsm.StatusInProgress || order.Status == fsm.StatusAtLastPoint || order.Status == fsm.StatusCompleted {
		// already progressed, idempotent
		return nil
	}
	if order.Status != fsm.StatusAssigned && order.Status != fsm.StatusDriverAtPickup {
		return ErrInvalidOperation
	}
	if err := s.ensureActionAllowed(order, ActionArrive, now); err != nil {
		return err
	}
	if err := s.validateTelemetry(now, telemetry); err != nil {
		return err
	}
	pickup := order.Waypoints[0]
	if telemetry.SpeedKPH > s.cfg.StationarySpeedKPH {
		return ErrGeoConstraintViolation
	}
	if telemetry.Position.DistanceTo(pickup.Point) > s.cfg.ArrivalRadiusMeters {
		return ErrGeoConstraintViolation
	}
	fmt.Println("Max distance:", s.cfg.ArrivalRadiusMeters)
	order.recordTelemetry(telemetry)
	order.ensureWaypointReached(0, now)
	order.ArrivedAt = &now
	if order.Status != fsm.StatusDriverAtPickup {
		if !fsm.CanTransition(order.Status, fsm.StatusDriverAtPickup) {
			return ErrInvalidOperation
		}
		order.appendStatus(fsm.StatusDriverAtPickup, now, "driver arrived at pickup")
	}
	if order.Status != fsm.StatusWaitingFree {
		if !fsm.CanTransition(order.Status, fsm.StatusWaitingFree) {
			return ErrInvalidOperation
		}
		order.startWaitingSession(WaitSessionFree, now)
		order.appendStatus(fsm.StatusWaitingFree, now, "free waiting started")
	}
	return nil
}

// AdvanceWaiting transitions free waiting to paid when the complimentary window expires.
func (s *Service) AdvanceWaiting(order *Order, now time.Time) (bool, error) {
	if order.Status != fsm.StatusWaitingFree {
		return false, nil
	}
	session := order.activeWaitingSession()
	if session == nil || session.Type != WaitSessionFree {
		return false, nil
	}
	switchAt := session.Started.Add(s.cfg.FreeWaitingWindow)
	if now.Before(switchAt) {
		return false, nil
	}
	order.closeActiveWaiting(switchAt, s.cfg)
	if !fsm.CanTransition(order.Status, fsm.StatusWaitingPaid) {
		return false, ErrInvalidOperation
	}
	order.appendStatus(fsm.StatusWaitingPaid, switchAt, "paid waiting started")
	order.startWaitingSession(WaitSessionPaid, switchAt)
	return true, nil
}

// StartTrip starts the ride once passenger is onboard.
func (s *Service) StartTrip(order *Order, now time.Time, telemetry Telemetry, pinConfirmed bool) error {
	if order.Status == fsm.StatusInProgress || order.Status == fsm.StatusAtLastPoint || order.Status == fsm.StatusCompleted {
		return nil
	}
	if order.Status != fsm.StatusWaitingFree && order.Status != fsm.StatusWaitingPaid && order.Status != fsm.StatusDriverAtPickup && order.Status != fsm.StatusArrived && order.Status != fsm.StatusPickedUp {
		return ErrInvalidOperation
	}
	if err := s.ensureActionAllowed(order, ActionStart, now); err != nil {
		return err
	}
	if s.cfg.RequireBoardingPIN && !pinConfirmed {
		return ErrPinRequired
	}
	if err := s.validateTelemetry(now, telemetry); err != nil {
		return err
	}
	pickup := order.Waypoints[0]
	if telemetry.Position.DistanceTo(pickup.Point) > s.cfg.StartRadiusMeters {
		return ErrGeoConstraintViolation
	}
	order.recordTelemetry(telemetry)
	order.ensureWaypointReached(0, now)
	order.closeActiveWaiting(now, s.cfg)
	if !fsm.CanTransition(order.Status, fsm.StatusInProgress) {
		return ErrInvalidOperation
	}
	order.appendStatus(fsm.StatusInProgress, now, "trip started")
	order.StartedAt = &now
	if order.Fare.BaseAmount == 0 {
		order.Fare.BaseAmount = order.BaseFare
	}
	return nil
}

// ReachWaypoint records reaching an intermediate stop.
func (s *Service) ReachWaypoint(order *Order, now time.Time, telemetry Telemetry) error {
	if order.Status != fsm.StatusInProgress {
		return ErrInvalidOperation
	}
	if err := s.validateTelemetry(now, telemetry); err != nil {
		return err
	}
	wp, index, ok := order.activeWaypoint()
	if !ok {
		return ErrInvalidOperation
	}
	if wp.Kind == WaypointFinish {
		return nil
	}
	radius := wp.Radius
	if radius == 0 {
		radius = s.cfg.WaypointRadiusMeters
	}
	if telemetry.Position.DistanceTo(wp.Point) > radius {
		return ErrGeoConstraintViolation
	}
	order.recordTelemetry(telemetry)
	order.ensureWaypointReached(index, now)
	return nil
}

// StartPause opens an in-trip waiting session.
func (s *Service) StartPause(order *Order, now time.Time) error {
	if order.Status != fsm.StatusInProgress {
		return ErrInvalidOperation
	}
	if err := s.ensureActionAllowed(order, ActionPause, now); err != nil {
		return err
	}
	active := order.activeWaitingSession()
	if active != nil && active.Type == WaitSessionPause {
		return nil
	}
	order.startWaitingSession(WaitSessionPause, now)
	return nil
}

// EndPause closes the active in-trip waiting session.
func (s *Service) EndPause(order *Order, now time.Time) error {
	if order.Status != fsm.StatusInProgress {
		return ErrInvalidOperation
	}
	if err := s.ensureActionAllowed(order, ActionResume, now); err != nil {
		return err
	}
	active := order.activeWaitingSession()
	if active == nil || active.Type != WaitSessionPause {
		return nil
	}
	order.closeActiveWaiting(now, s.cfg)
	return nil
}

// FinishTrip handles the "Finish" button at the final destination.
func (s *Service) FinishTrip(order *Order, now time.Time, telemetry Telemetry) error {
	if order.Status == fsm.StatusAtLastPoint || order.Status == fsm.StatusCompleted {
		return nil
	}
	if order.Status != fsm.StatusInProgress {
		return ErrInvalidOperation
	}
	if err := s.ensureActionAllowed(order, ActionFinish, now); err != nil {
		return err
	}
	if err := s.validateTelemetry(now, telemetry); err != nil {
		return err
	}
	wp, index, ok := order.activeWaypoint()
	if !ok {
		return ErrInvalidOperation
	}
	if wp.Kind != WaypointFinish {
		// allow finishing even if intermediate skipped, but still check distance to final
		wp = &order.Waypoints[len(order.Waypoints)-1]
		index = len(order.Waypoints) - 1
	}
	radius := wp.Radius
	if radius == 0 {
		radius = s.cfg.FinishRadiusMeters
	}
	if telemetry.Position.DistanceTo(wp.Point) > radius {
		return ErrGeoConstraintViolation
	}
	order.recordTelemetry(telemetry)
	order.closeActiveWaiting(now, s.cfg)
	order.ensureWaypointReached(index, now)
	if !fsm.CanTransition(order.Status, fsm.StatusAtLastPoint) {
		return ErrInvalidOperation
	}
	order.appendStatus(fsm.StatusAtLastPoint, now, "arrived at final point")
	order.FinishedAt = &now
	return nil
}

// ConfirmCashPayment finalises the order after receiving cash.
func (s *Service) ConfirmCashPayment(order *Order, now time.Time) error {
	if order.Status == fsm.StatusCompleted {
		return nil
	}
	if order.Status != fsm.StatusAtLastPoint {
		return ErrInvalidOperation
	}
	order.closeActiveWaiting(now, s.cfg)
	if !fsm.CanTransition(order.Status, fsm.StatusCompleted) {
		return ErrInvalidOperation
	}
	order.appendStatus(fsm.StatusCompleted, now, "cash payment confirmed")
	order.PaymentConfirmed = &now
	return nil
}

// CancelByPassenger cancels the order initiated by passenger.
func (s *Service) CancelByPassenger(order *Order, now time.Time, reason string) error {
	if order.Status == fsm.StatusCompleted || order.Status == fsm.StatusClosed {
		return ErrInvalidOperation
	}
	if order.Status == fsm.StatusCanceledByPassenger {
		return nil
	}
	if err := s.ensureActionAllowed(order, ActionCancel, now); err != nil {
		return err
	}
	order.closeActiveWaiting(now, s.cfg)
	if !fsm.CanTransition(order.Status, fsm.StatusCanceledByPassenger) {
		return ErrInvalidOperation
	}
	note := "canceled by passenger"
	if reason != "" {
		note = fmt.Sprintf("%s: %s", note, reason)
	}
	order.appendStatus(fsm.StatusCanceledByPassenger, now, note)
	return nil
}

// CancelByDriver cancels the order initiated by driver.
func (s *Service) CancelByDriver(order *Order, now time.Time, reason string) error {
	if order.Status == fsm.StatusCompleted || order.Status == fsm.StatusClosed {
		return ErrInvalidOperation
	}
	if order.Status == fsm.StatusCanceledByDriver {
		return nil
	}
	if err := s.ensureActionAllowed(order, ActionCancel, now); err != nil {
		return err
	}
	order.closeActiveWaiting(now, s.cfg)
	if !fsm.CanTransition(order.Status, fsm.StatusCanceledByDriver) {
		return ErrInvalidOperation
	}
	note := "canceled by driver"
	if reason != "" {
		note = fmt.Sprintf("%s: %s", note, reason)
	}
	order.appendStatus(fsm.StatusCanceledByDriver, now, note)
	return nil
}

// MarkNoShow marks passenger no-show event.
func (s *Service) MarkNoShow(order *Order, now time.Time, telemetry Telemetry) error {
	if order.Status == fsm.StatusNoShow {
		return nil
	}
	if order.Status != fsm.StatusWaitingFree && order.Status != fsm.StatusWaitingPaid {
		return ErrInvalidOperation
	}
	if err := s.ensureActionAllowed(order, ActionNoShow, now); err != nil {
		return err
	}
	if err := s.validateTelemetry(now, telemetry); err != nil {
		return err
	}
	pickup := order.Waypoints[0]
	if telemetry.Position.DistanceTo(pickup.Point) > s.cfg.ArrivalRadiusMeters {
		return ErrGeoConstraintViolation
	}
	order.recordTelemetry(telemetry)
	order.closeActiveWaiting(now, s.cfg)
	if !fsm.CanTransition(order.Status, fsm.StatusNoShow) {
		return ErrInvalidOperation
	}
	order.appendStatus(fsm.StatusNoShow, now, "passenger no-show recorded")
	return nil
}
