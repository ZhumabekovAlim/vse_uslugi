package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"naimuBack/internal/repositories"
	"naimuBack/internal/services"
)

type BusinessHandler struct {
	Service *services.BusinessService
}

func (h *BusinessHandler) respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *BusinessHandler) PurchaseSeats(w http.ResponseWriter, r *http.Request) {
	var req services.PurchaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	userID, _ := r.Context().Value("user_id").(int)
	acc, err := h.Service.PurchaseSeats(r.Context(), userID, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]any{"account": acc})
}

func (h *BusinessHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value("user_id").(int)
	acc, err := h.Service.GetOrCreateAccount(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	seatsLeft := acc.SeatsTotal - acc.SeatsUsed
	h.respondJSON(w, http.StatusOK, map[string]any{"account": acc, "seats_left": seatsLeft})
}

func (h *BusinessHandler) CreateWorker(w http.ResponseWriter, r *http.Request) {
	var req services.CreateWorkerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	userID, _ := r.Context().Value("user_id").(int)
	worker, err := h.Service.CreateWorker(r.Context(), userID, req)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, repositories.ErrBusinessAccountSuspended) {
			status = http.StatusForbidden
		}
		http.Error(w, err.Error(), status)
		return
	}
	h.respondJSON(w, http.StatusCreated, map[string]any{"worker": worker})
}

func (h *BusinessHandler) ListWorkers(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value("user_id").(int)
	workers, err := h.Service.ListWorkers(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]any{"workers": workers})
}

func (h *BusinessHandler) UpdateWorker(w http.ResponseWriter, r *http.Request) {
	var req services.UpdateWorkerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	workerIDStr := r.URL.Query().Get(":id")
	workerID, err := strconv.Atoi(workerIDStr)
	if err != nil {
		http.Error(w, "invalid worker id", http.StatusBadRequest)
		return
	}
	userID, _ := r.Context().Value("user_id").(int)
	worker, err := h.Service.UpdateWorker(r.Context(), userID, workerID, req)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, repositories.ErrBusinessAccountSuspended) {
			status = http.StatusForbidden
		}
		http.Error(w, err.Error(), status)
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]any{"worker": worker})
}

func (h *BusinessHandler) DisableWorker(w http.ResponseWriter, r *http.Request) {
	workerIDStr := r.URL.Query().Get(":id")
	workerID, err := strconv.Atoi(workerIDStr)
	if err != nil {
		http.Error(w, "invalid worker id", http.StatusBadRequest)
		return
	}
	userID, _ := r.Context().Value("user_id").(int)
	if err := h.Service.DisableWorker(r.Context(), userID, workerID); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, repositories.ErrBusinessAccountSuspended) {
			status = http.StatusForbidden
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AttachListing binds an existing listing to a worker.
func (h *BusinessHandler) AttachListing(w http.ResponseWriter, r *http.Request) {
	workerIDStr := r.URL.Query().Get(":id")
	workerID, err := strconv.Atoi(workerIDStr)
	if err != nil {
		http.Error(w, "invalid worker id", http.StatusBadRequest)
		return
	}
	var req services.AttachListingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	businessUserID, _ := r.Context().Value("user_id").(int)
	if err := h.Service.AttachListing(r.Context(), businessUserID, workerID, req); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, repositories.ErrBusinessAccountSuspended) {
			status = http.StatusForbidden
		} else if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DetachListing removes listing binding.
func (h *BusinessHandler) DetachListing(w http.ResponseWriter, r *http.Request) {
	workerIDStr := r.URL.Query().Get(":id")
	workerID, err := strconv.Atoi(workerIDStr)
	if err != nil {
		http.Error(w, "invalid worker id", http.StatusBadRequest)
		return
	}
	var req services.AttachListingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	businessUserID, _ := r.Context().Value("user_id").(int)
	if err := h.Service.DetachListing(r.Context(), businessUserID, workerID, req); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, repositories.ErrBusinessAccountSuspended) {
			status = http.StatusForbidden
		} else if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListWorkerListings returns map of worker user_id to listing attachments.
func (h *BusinessHandler) ListWorkerListings(w http.ResponseWriter, r *http.Request) {
	businessUserID, _ := r.Context().Value("user_id").(int)
	listings, err := h.Service.ListWorkerListings(r.Context(), businessUserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]any{"listings": listings})
}
