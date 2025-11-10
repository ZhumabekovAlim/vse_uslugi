package http

import (
	"encoding/json"
	"net/http"

	"naimuBack/internal/courier/pricing"
)

func (s *Server) handleQuote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		DistanceM int `json:"distance_m"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.DistanceM <= 0 {
		writeError(w, http.StatusBadRequest, "distance_m must be positive")
		return
	}
	recommended := pricing.Recommended(req.DistanceM, s.cfg.PricePerKM, s.cfg.MinPrice)
	writeJSON(w, http.StatusOK, map[string]int{"recommended_price": recommended})
}
