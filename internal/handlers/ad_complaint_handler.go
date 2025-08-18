package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"naimuBack/internal/models"
	"naimuBack/internal/services"
)

type AdComplaintHandler struct {
	Service *services.AdComplaintService
}

func (h *AdComplaintHandler) CreateAdComplaint(w http.ResponseWriter, r *http.Request) {
	var c models.AdComplaint
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	if err := h.Service.CreateAdComplaint(r.Context(), c); err != nil {
		log.Printf("CreateAdComplaint error: %v", err)
		http.Error(w, "Failed to create complaint", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *AdComplaintHandler) GetComplaintsByAdID(w http.ResponseWriter, r *http.Request) {
	adID, err := strconv.Atoi(r.URL.Query().Get(":ad_id"))
	if err != nil {
		http.Error(w, "Invalid ad_id", http.StatusBadRequest)
		return
	}
	complaints, err := h.Service.GetComplaintsByAdID(r.Context(), adID)
	if err != nil {
		log.Printf("GetComplaintsByAdID error: %v", err)
		http.Error(w, "Failed to get complaints", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(complaints)
}

func (h *AdComplaintHandler) DeleteAdComplaintByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get(":id")
	if idStr == "" {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	id, _ := strconv.Atoi(idStr)
	if err := h.Service.DeleteAdComplaintByID(r.Context(), id); err != nil {
		log.Printf("DeleteAdComplaint error: %v", err)
		http.Error(w, "Failed to delete complaint", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdComplaintHandler) GetAllAdComplaints(w http.ResponseWriter, r *http.Request) {
	complaints, err := h.Service.GetAllAdComplaints(r.Context())
	if err != nil {
		log.Printf("GetAllAdComplaints error: %v", err)
		http.Error(w, "Failed to get complaints", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(complaints)
}
