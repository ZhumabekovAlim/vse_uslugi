package http

import "net/http"

func (s *Server) handleCourierWS(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "courier websocket is not implemented yet", http.StatusNotImplemented)
}

func (s *Server) handleSenderWS(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "sender websocket is not implemented yet", http.StatusNotImplemented)
}
