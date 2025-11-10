package http

import "net/http"

func (s *Server) handleCourierWS(w http.ResponseWriter, r *http.Request) {
	if s.cHub == nil {
		http.Error(w, "courier websocket unavailable", http.StatusServiceUnavailable)
		return
	}
	s.cHub.ServeWS(w, r)
}

func (s *Server) handleSenderWS(w http.ResponseWriter, r *http.Request) {
	if s.sHub == nil {
		http.Error(w, "sender websocket unavailable", http.StatusServiceUnavailable)
		return
	}
	s.sHub.ServeWS(w, r)
}
