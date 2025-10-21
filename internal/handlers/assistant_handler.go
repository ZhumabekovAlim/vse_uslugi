package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"naimuBack/internal/services"
)

type AssistantHandler struct {
	Service *services.AssistantService
}

type AskRequest struct {
	Question string `json:"question"`
	Locale   string `json:"locale,omitempty"`
	Screen   string `json:"screen,omitempty"`
	Role     string `json:"role,omitempty"`
	UseLLM   *bool  `json:"use_llm,omitempty"`
	MaxKB    *int   `json:"max_kb,omitempty"`
}

func NewAssistantHandler(service *services.AssistantService) *AssistantHandler {
	return &AssistantHandler{Service: service}
}

func (h *AssistantHandler) Ask(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		http.Error(w, "assistant service unavailable", http.StatusServiceUnavailable)
		return
	}

	var req AskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	question := strings.TrimSpace(req.Question)
	if question == "" {
		http.Error(w, "question is required", http.StatusBadRequest)
		return
	}

	locale := strings.TrimSpace(req.Locale)
	if locale == "" {
		locale = "ru"
	}

	role := strings.TrimSpace(req.Role)
	if role == "" {
		role = "user"
	}

	maxKB := 3
	if req.MaxKB != nil {
		maxKB = clamp(*req.MaxKB, 1, 5)
	}

	useLLM := true
	if req.UseLLM != nil {
		useLLM = *req.UseLLM
	}

	params := services.AskParams{
		Question: question,
		Locale:   locale,
		Screen:   strings.TrimSpace(req.Screen),
		Role:     role,
		UseLLM:   useLLM,
		MaxKB:    maxKB,
	}

	result, err := h.Service.Ask(r.Context(), params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	payload, err := json.Marshal(result)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
