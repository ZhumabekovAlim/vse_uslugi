package handlers

import (
	"errors"
	"net/http"
	"testing"

	"naimuBack/internal/services"
)

func TestAirbapayErrorStatus(t *testing.T) {
	t.Run("propagates 4xx", func(t *testing.T) {
		status := airbapayErrorStatus(&services.AirbapayError{StatusCode: http.StatusNotFound})
		if status != http.StatusNotFound {
			t.Fatalf("expected %d, got %d", http.StatusNotFound, status)
		}
	})

	t.Run("defaults otherwise", func(t *testing.T) {
		err := errors.New("generic error")
		status := airbapayErrorStatus(err)
		if status != http.StatusBadGateway {
			t.Fatalf("expected %d, got %d", http.StatusBadGateway, status)
		}

		status = airbapayErrorStatus(&services.AirbapayError{StatusCode: http.StatusInternalServerError})
		if status != http.StatusBadGateway {
			t.Fatalf("expected %d, got %d", http.StatusBadGateway, status)
		}
	})
}
