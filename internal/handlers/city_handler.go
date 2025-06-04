package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"naimuBack/internal/models"
	"naimuBack/internal/services"
)

type CityHandler struct {
	Service *services.CityService
}

func (h *CityHandler) CreateCity(w http.ResponseWriter, r *http.Request) {
	var city models.City
	if err := json.NewDecoder(r.Body).Decode(&city); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	newCity, err := h.Service.CreateCity(r.Context(), city)
	if err != nil {
		http.Error(w, "Failed to create", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(newCity)
}

func (h *CityHandler) GetCities(w http.ResponseWriter, r *http.Request) {
	cities, err := h.Service.GetCities(r.Context())
	if err != nil {
		http.Error(w, "Failed to fetch", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(cities)
}

func (h *CityHandler) GetCityByID(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get(":id"))
	city, err := h.Service.GetCityByID(r.Context(), id)
	if err != nil {
		http.Error(w, "City not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(city)
}

func (h *CityHandler) UpdateCity(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get(":id"))
	var city models.City
	json.NewDecoder(r.Body).Decode(&city)
	city.ID = id
	updatedCity, err := h.Service.UpdateCity(r.Context(), city)
	if err != nil {
		http.Error(w, "Failed to update", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(updatedCity)
}

func (h *CityHandler) DeleteCity(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get(":id"))
	err := h.Service.DeleteCity(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to delete", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
