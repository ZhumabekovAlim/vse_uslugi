package http

import "net/http"

func (s *Server) handleCourierBalanceDeposit(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "courier balance operations are not supported yet")
}

func (s *Server) handleCourierBalanceWithdraw(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "courier balance operations are not supported yet")
}
