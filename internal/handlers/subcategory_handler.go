package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"naimuBack/internal/models"
	"naimuBack/internal/services"
)

type SubcategoryHandler struct {
	Service *services.SubcategoryService
}

func (h *SubcategoryHandler) CreateSubcategory(w http.ResponseWriter, r *http.Request) {
	var s models.Subcategory
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	sub, err := h.Service.CreateSubcategory(r.Context(), s)
	if err != nil {
		http.Error(w, "Failed to create", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(sub)
}

func (h *SubcategoryHandler) GetAllSubcategories(w http.ResponseWriter, r *http.Request) {
	subs, err := h.Service.GetAllSubcategories(r.Context())
	if err != nil {
		http.Error(w, "Failed to fetch", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(subs)
}

func (h *SubcategoryHandler) GetByCategory(w http.ResponseWriter, r *http.Request) {
	catID, err := strconv.Atoi(r.URL.Query().Get(":category_id"))
	if err != nil {
		http.Error(w, "Invalid category_id", http.StatusBadRequest)
		return
	}
	subs, err := h.Service.GetByCategoryID(r.Context(), catID)
	if err != nil {
		http.Error(w, "Failed to fetch", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(subs)
}
