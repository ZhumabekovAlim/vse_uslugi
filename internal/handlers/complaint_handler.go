package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"naimuBack/internal/models"
	"naimuBack/internal/services"
)

type ComplaintHandler struct {
	Service *services.ComplaintService
}

func (h *ComplaintHandler) CreateComplaint(w http.ResponseWriter, r *http.Request) {
	var c models.Complaint
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	if err := h.Service.CreateComplaint(r.Context(), c); err != nil {
		log.Printf("CreateComplaint error: %v", err)
		http.Error(w, "Failed to create complaint", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *ComplaintHandler) GetComplaintsByServiceID(w http.ResponseWriter, r *http.Request) {
	serviceID, err := strconv.Atoi(r.URL.Query().Get(":service_id"))
	if err != nil {
		http.Error(w, "Invalid service_id", http.StatusBadRequest)
		return
	}
	complaints, err := h.Service.GetComplaintsByServiceID(r.Context(), serviceID)
	if err != nil {
		log.Printf("GetComplaintByService error: %v", err)
		http.Error(w, "Failed to get complaints", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(complaints)
}

func (h *ComplaintHandler) DeleteComplaintByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get(":id")
	if idStr == "" {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	id, _ := strconv.Atoi(idStr)
	if err := h.Service.DeleteComplaintByID(r.Context(), id); err != nil {
		log.Printf("DeleteComplaint error: %v", err)
		http.Error(w, "Failed to delete complaint", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ComplaintHandler) GetAllComplaints(w http.ResponseWriter, r *http.Request) {
	complaints, err := h.Service.GetAllComplaints(r.Context())
	if err != nil {
		log.Printf("GetAllComplaints error: %v", err)
		http.Error(w, "Failed to get complaints", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(complaints)
}
