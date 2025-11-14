package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"naimuBack/internal/services"
)

const (
	assistantWSMessageTypeAsk    = "ask"
	assistantWSMessageTypeAnswer = "answer"
	assistantWSMessageTypeError  = "error"
)

type assistantWSMessage struct {
	Type      string `json:"type,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	Question  string `json:"question"`
	Locale    string `json:"locale,omitempty"`
	Role      string `json:"role,omitempty"`
	UseLLM    *bool  `json:"use_llm,omitempty"`
	MaxKB     *int   `json:"max_kb,omitempty"`
}

type assistantWSResponse struct {
	Type      string              `json:"type"`
	RequestID string              `json:"request_id,omitempty"`
	Error     string              `json:"error,omitempty"`
	Result    *services.AskResult `json:"result,omitempty"`
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

func (app *application) AssistantWebSocketHandler(w http.ResponseWriter, r *http.Request) {
	if app.assistantHandler == nil || app.assistantHandler.Service == nil {
		http.Error(w, "assistant service unavailable", http.StatusServiceUnavailable)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("assistant WS upgrade error:", err)
		return
	}
	defer conn.Close()

	conn.SetReadLimit(readLimit)
	conn.SetReadDeadline(time.Now().Add(readDeadline))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(readDeadline))
		return nil
	})

	stop := make(chan struct{})
	defer close(stop)
	go assistantPingLoop(conn, stop)

	for {
		var msg assistantWSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			log.Println("assistant ws read error:", err)
			_ = writeClose(conn, websocket.CloseNormalClosure, "read error")
			return
		}
		conn.SetReadDeadline(time.Now().Add(readDeadline))

		if strings.TrimSpace(msg.Type) != "" && strings.TrimSpace(msg.Type) != assistantWSMessageTypeAsk {
			app.sendAssistantWSError(conn, msg.RequestID, "unknown message type")
			continue
		}

		question := strings.TrimSpace(msg.Question)
		if question == "" {
			app.sendAssistantWSError(conn, msg.RequestID, "question is required")
			continue
		}

		locale := strings.TrimSpace(msg.Locale)
		if locale == "" {
			locale = "ru"
		}
		role := strings.TrimSpace(msg.Role)
		if role == "" {
			role = "user"
		}
		maxKB := 5
		if msg.MaxKB != nil {
			maxKB = clamp(*msg.MaxKB, 1, 20)
		}
		useLLM := true
		if msg.UseLLM != nil {
			useLLM = *msg.UseLLM
		}

		params := services.AskParams{
			Question: question,
			Locale:   locale,
			Role:     role,
			UseLLM:   useLLM,
			MaxKB:    maxKB,
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		result, err := app.assistantHandler.Service.Ask(ctx, params)
		cancel()
		if err != nil {
			log.Println("assistant ws ask error:", err)
			app.sendAssistantWSError(conn, msg.RequestID, "failed to get answer")
			continue
		}

		resp := assistantWSResponse{Type: assistantWSMessageTypeAnswer, RequestID: msg.RequestID, Result: &result}
		if err := app.writeAssistantWSResponse(conn, resp); err != nil {
			log.Println("assistant ws write error:", err)
			return
		}
	}
}

func assistantPingLoop(conn *websocket.Conn, stop <-chan struct{}) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(writeDeadline))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				_ = writeClose(conn, websocket.CloseGoingAway, "ping error")
				return
			}
		case <-stop:
			return
		}
	}
}

func (app *application) sendAssistantWSError(conn *websocket.Conn, requestID, message string) {
	resp := assistantWSResponse{Type: assistantWSMessageTypeError, RequestID: requestID, Error: message}
	if err := app.writeAssistantWSResponse(conn, resp); err != nil {
		log.Println("assistant ws send error failed:", err)
	}
}

func (app *application) writeAssistantWSResponse(conn *websocket.Conn, resp assistantWSResponse) error {
	_ = conn.SetWriteDeadline(time.Now().Add(writeDeadline))
	return conn.WriteJSON(resp)
}
