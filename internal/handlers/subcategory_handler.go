package handlers

import (
	"encoding/json"
	"errors"
	"naimuBack/internal/repositories"
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

func (h *SubcategoryHandler) GetSubcategoryByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get(":id")
	if idStr == "" {
		http.Error(w, "Missing subcategory ID", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid subcategory ID", http.StatusBadRequest)
		return
	}

	subcategory, err := h.Service.GetSubcategoryByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repositories.ErrSubcategoryNotFound) {
			http.Error(w, "Subcategory not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to fetch subcategory", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subcategory)
}

func (h *SubcategoryHandler) UpdateSubcategoryByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get(":id")
	if idStr == "" {
		http.Error(w, "Missing subcategory ID", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid subcategory ID", http.StatusBadRequest)
		return
	}

	var sub models.Subcategory
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	sub.ID = id

	updatedSub, err := h.Service.UpdateSubcategoryByID(r.Context(), sub)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedSub)
}

func (h *SubcategoryHandler) DeleteSubcategoryByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get(":id")
	if idStr == "" {
		http.Error(w, "Missing subcategory ID", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid subcategory ID", http.StatusBadRequest)
		return
	}

	err = h.Service.DeleteSubcategoryByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrSubcategoryNotFound) {
			http.Error(w, "Subcategory not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete subcategory", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
