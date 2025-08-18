package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"naimuBack/internal/models"
	"naimuBack/internal/services"
)

type WorkComplaintHandler struct {
	Service *services.WorkComplaintService
}

func (h *WorkComplaintHandler) CreateWorkComplaint(w http.ResponseWriter, r *http.Request) {
	var c models.WorkComplaint
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	if err := h.Service.CreateWorkComplaint(r.Context(), c); err != nil {
		log.Printf("CreateWorkComplaint error: %v", err)
		http.Error(w, "Failed to create complaint", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *WorkComplaintHandler) GetComplaintsByWorkID(w http.ResponseWriter, r *http.Request) {
	workID, err := strconv.Atoi(r.URL.Query().Get(":work_id"))
	if err != nil {
		http.Error(w, "Invalid work_id", http.StatusBadRequest)
		return
	}
	complaints, err := h.Service.GetComplaintsByWorkID(r.Context(), workID)
	if err != nil {
		log.Printf("GetComplaintsByWorkID error: %v", err)
		http.Error(w, "Failed to get complaints", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(complaints)
}

func (h *WorkComplaintHandler) DeleteWorkComplaintByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get(":id")
	if idStr == "" {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	id, _ := strconv.Atoi(idStr)
	if err := h.Service.DeleteWorkComplaintByID(r.Context(), id); err != nil {
		log.Printf("DeleteWorkComplaint error: %v", err)
		http.Error(w, "Failed to delete complaint", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *WorkComplaintHandler) GetAllWorkComplaints(w http.ResponseWriter, r *http.Request) {
	complaints, err := h.Service.GetAllWorkComplaints(r.Context())
	if err != nil {
		log.Printf("GetAllWorkComplaints error: %v", err)
		http.Error(w, "Failed to get complaints", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(complaints)
}
