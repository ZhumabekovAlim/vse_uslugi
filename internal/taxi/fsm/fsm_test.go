package fsm

import "testing"

func TestCanTransition(t *testing.T) {
	if !CanTransition(StatusCreated, StatusSearching) {
		t.Fatal("expected created -> searching to be allowed")
	}
	if CanTransition(StatusCreated, StatusCompleted) {
		t.Fatal("unexpected transition allowed")
	}
	if !CanTransition(StatusAccepted, StatusAssigned) {
		t.Fatal("expected accepted -> assigned to be allowed")
	}
	if !CanTransition(StatusWaitingFree, StatusWaitingPaid) {
		t.Fatal("expected waiting_free -> waiting_paid to be allowed")
	}
	if !CanTransition(StatusInProgress, StatusAtLastPoint) {
		t.Fatal("expected in_progress -> at_last_point to be allowed")
	}
	if !CanTransition(StatusAtLastPoint, StatusCompleted) {
		t.Fatal("expected at_last_point -> completed to be allowed")
	}
	if !CanTransition(StatusCanceledByPassenger, StatusClosed) {
		t.Fatal("expected canceled_by_passenger -> closed to be allowed")
	}
}
