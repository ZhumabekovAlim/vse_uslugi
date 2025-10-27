package fsm

import "testing"

func TestCanTransition(t *testing.T) {
    if !CanTransition("created", "searching") {
        t.Fatal("expected created -> searching to be allowed")
    }
    if CanTransition("created", "completed") {
        t.Fatal("unexpected transition allowed")
    }
}
