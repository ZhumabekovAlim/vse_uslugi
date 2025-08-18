package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"naimuBack/internal/models"
	"naimuBack/internal/services"
)

type RentComplaintHandler struct {
	Service *services.RentComplaintService
}

func (h *RentComplaintHandler) CreateRentComplaint(w http.ResponseWriter, r *http.Request) {
	var c models.RentComplaint
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	if err := h.Service.CreateRentComplaint(r.Context(), c); err != nil {
		log.Printf("CreateRentComplaint error: %v", err)
		http.Error(w, "Failed to create complaint", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *RentComplaintHandler) GetComplaintsByRentID(w http.ResponseWriter, r *http.Request) {
	rentID, err := strconv.Atoi(r.URL.Query().Get(":rent_id"))
	if err != nil {
		http.Error(w, "Invalid rent_id", http.StatusBadRequest)
		return
	}
	complaints, err := h.Service.GetComplaintsByRentID(r.Context(), rentID)
	if err != nil {
		log.Printf("GetComplaintsByRentID error: %v", err)
		http.Error(w, "Failed to get complaints", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(complaints)
}

func (h *RentComplaintHandler) DeleteRentComplaintByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get(":id")
	if idStr == "" {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	id, _ := strconv.Atoi(idStr)
	if err := h.Service.DeleteRentComplaintByID(r.Context(), id); err != nil {
		log.Printf("DeleteRentComplaint error: %v", err)
		http.Error(w, "Failed to delete complaint", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RentComplaintHandler) GetAllRentComplaints(w http.ResponseWriter, r *http.Request) {
	complaints, err := h.Service.GetAllRentComplaints(r.Context())
	if err != nil {
		log.Printf("GetAllRentComplaints error: %v", err)
		http.Error(w, "Failed to get complaints", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(complaints)
}
