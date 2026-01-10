package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
)

type cityRequest struct {
	CityID int `json:"city_id"`
}

func decodeCityID(r *http.Request) (int, error) {
	var req cityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return 0, err
	}
	if req.CityID == 0 {
		return 0, errors.New("city_id is required")
	}
	return req.CityID, nil
}
