package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"naimuBack/internal/models"
)

// GetAds handles GET /ads requests and returns combined ads list
func (h *AdHandler) GetAds(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := models.AdsFilter{}
	filter.Type = q.Get("type")
	if v := q.Get("category_id"); v != "" {
		if id, err := strconv.Atoi(v); err == nil {
			filter.CategoryID = id
		}
	}
	if v := q.Get("subcategory_id"); v != "" {
		if id, err := strconv.Atoi(v); err == nil {
			filter.SubcategoryID = id
		}
	}
	if v := q.Get("min_price"); v != "" {
		if p, err := strconv.ParseFloat(v, 64); err == nil {
			filter.MinPrice = p
		}
	}
	if v := q.Get("max_price"); v != "" {
		if p, err := strconv.ParseFloat(v, 64); err == nil {
			filter.MaxPrice = p
		}
	}
	filter.Search = q.Get("search")
	if v := q.Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			filter.Page = p
		}
	}
	if v := q.Get("page_size"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			filter.PageSize = p
		}
	}

	result, err := h.Service.ListAds(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
