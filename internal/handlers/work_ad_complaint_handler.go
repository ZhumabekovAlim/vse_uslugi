package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"naimuBack/internal/models"
	"naimuBack/internal/services"
)

type WorkAdComplaintHandler struct {
	Service *services.WorkAdComplaintService
}

func (h *WorkAdComplaintHandler) CreateWorkAdComplaint(w http.ResponseWriter, r *http.Request) {
	var c models.WorkAdComplaint
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	if err := h.Service.CreateWorkAdComplaint(r.Context(), c); err != nil {
		log.Printf("CreateWorkAdComplaint error: %v", err)
		http.Error(w, "Failed to create complaint", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *WorkAdComplaintHandler) GetComplaintsByWorkAdID(w http.ResponseWriter, r *http.Request) {
	workAdID, err := strconv.Atoi(r.URL.Query().Get(":work_ad_id"))
	if err != nil {
		http.Error(w, "Invalid work_ad_id", http.StatusBadRequest)
		return
	}
	complaints, err := h.Service.GetComplaintsByWorkAdID(r.Context(), workAdID)
	if err != nil {
		log.Printf("GetComplaintsByWorkAdID error: %v", err)
		http.Error(w, "Failed to get complaints", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(complaints)
}

func (h *WorkAdComplaintHandler) DeleteWorkAdComplaintByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get(":id")
	if idStr == "" {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	id, _ := strconv.Atoi(idStr)
	if err := h.Service.DeleteWorkAdComplaintByID(r.Context(), id); err != nil {
		log.Printf("DeleteWorkAdComplaint error: %v", err)
		http.Error(w, "Failed to delete complaint", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *WorkAdComplaintHandler) GetAllWorkAdComplaints(w http.ResponseWriter, r *http.Request) {
	complaints, err := h.Service.GetAllWorkAdComplaints(r.Context())
	if err != nil {
		log.Printf("GetAllWorkAdComplaints error: %v", err)
		http.Error(w, "Failed to get complaints", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(complaints)
}
