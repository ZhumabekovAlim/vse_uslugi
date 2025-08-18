package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"naimuBack/internal/models"
	"naimuBack/internal/services"
)

type RentAdComplaintHandler struct {
	Service *services.RentAdComplaintService
}

func (h *RentAdComplaintHandler) CreateRentAdComplaint(w http.ResponseWriter, r *http.Request) {
	var c models.RentAdComplaint
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	if err := h.Service.CreateRentAdComplaint(r.Context(), c); err != nil {
		log.Printf("CreateRentAdComplaint error: %v", err)
		http.Error(w, "Failed to create complaint", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *RentAdComplaintHandler) GetComplaintsByRentAdID(w http.ResponseWriter, r *http.Request) {
	rentAdID, err := strconv.Atoi(r.URL.Query().Get(":rent_ad_id"))
	if err != nil {
		http.Error(w, "Invalid rent_ad_id", http.StatusBadRequest)
		return
	}
	complaints, err := h.Service.GetComplaintsByRentAdID(r.Context(), rentAdID)
	if err != nil {
		log.Printf("GetComplaintsByRentAdID error: %v", err)
		http.Error(w, "Failed to get complaints", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(complaints)
}

func (h *RentAdComplaintHandler) DeleteRentAdComplaintByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get(":id")
	if idStr == "" {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	id, _ := strconv.Atoi(idStr)
	if err := h.Service.DeleteRentAdComplaintByID(r.Context(), id); err != nil {
		log.Printf("DeleteRentAdComplaint error: %v", err)
		http.Error(w, "Failed to delete complaint", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RentAdComplaintHandler) GetAllRentAdComplaints(w http.ResponseWriter, r *http.Request) {
	complaints, err := h.Service.GetAllRentAdComplaints(r.Context())
	if err != nil {
		log.Printf("GetAllRentAdComplaints error: %v", err)
		http.Error(w, "Failed to get complaints", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(complaints)
}
