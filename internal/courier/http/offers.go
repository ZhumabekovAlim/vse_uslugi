package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"naimuBack/internal/courier/lifecycle"
	"naimuBack/internal/courier/repo"
)

// КУРЬЕР ПРЕДЛОЖИЛ ЦЕНУ
func (s *Server) handleOfferPrice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	courierID, err := parseAuthID(r, "X-Courier-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing courier id")
		return
	}
	var req struct {
		OrderID int64 `json:"order_id"`
		Price   int   `json:"price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.OrderID == 0 {
		writeError(w, http.StatusBadRequest, "order_id is required")
		return
	}
	if req.Price < s.cfg.MinPrice {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("price must be >= %d", s.cfg.MinPrice))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	ctx = withCourierActor(ctx, courierID)

	if err := s.offers.Upsert(ctx, req.OrderID, courierID, req.Price); err != nil {
		if s.logger != nil {
			s.logger.Errorf("courier: upsert offer failed: %v", err)
		}
		writeError(w, http.StatusInternalServerError, "failed to store offer")
		return
	}

	price := req.Price
	// Пытаемся подгрузить заказ для детального WS
	if order, err := s.orders.Get(ctx, req.OrderID); err != nil {
		if s.logger != nil {
			s.logger.Errorf("courier: load order %d after offer price failed: %v", req.OrderID, err)
		}
		// Fallback: эмитим события по id
		detached := withCourierActor(context.WithoutCancel(ctx), courierID)
		s.emitOfferEvent(detached, req.OrderID, courierID, repo.OfferStatusProposed, &price, originCourier)
		s.emitOrderEvent(detached, req.OrderID, orderEventTypeUpdated, originCourier)
	} else {
		// Детальные события
		s.emitOffer(ctx, order, courierID, repo.OfferStatusProposed, &price, originCourier)
		s.emitOrder(ctx, order, orderEventTypeUpdated, originCourier)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": repo.StatusNew})
}

// ЗАКАЗЧИК ЯВНО ПРИНИМАЕТ КОНКРЕТНОГО КУРЬЕРА
func (s *Server) handleOfferAccept(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	senderID, err := parseAuthID(r, "X-Sender-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing sender id")
		return
	}
	var req struct {
		OrderID   int64 `json:"order_id"`
		CourierID int64 `json:"courier_id"`
		Price     int   `json:"price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.OrderID == 0 || req.CourierID == 0 {
		writeError(w, http.StatusBadRequest, "order_id and courier_id are required")
		return
	}
	if req.Price < s.cfg.MinPrice {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("price must be >= %d", s.cfg.MinPrice))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	ctx = withSenderActor(ctx, senderID)

	// 1) Обновляем оффер
	if err := s.offers.UpdateStatus(ctx, req.OrderID, req.CourierID, repo.OfferStatusAccepted); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "offer not found")
			return
		}
		if s.logger != nil {
			s.logger.Errorf("courier: accept offer failed: %v", err)
		}
		writeError(w, http.StatusInternalServerError, "failed to accept offer")
		return
	}

	// 2) Назначаем курьера заказу
	if err := s.orders.AssignCourier(ctx, req.OrderID, req.CourierID, lifecycle.StatusAccepted); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "order not found")
			return
		}
		if s.logger != nil {
			s.logger.Errorf("courier: assign courier failed: %v", err)
		}
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	price := req.Price
	// 3) Эмиты
	if order, err := s.orders.Get(ctx, req.OrderID); err != nil {
		if s.logger != nil {
			s.logger.Errorf("courier: load order %d after offer accept failed: %v", req.OrderID, err)
		}
		detached := withSenderActor(context.WithoutCancel(ctx), senderID)
		s.emitOfferEvent(detached, req.OrderID, req.CourierID, repo.OfferStatusAccepted, &price, originSender)
		s.emitOrderEvent(detached, req.OrderID, orderEventTypeUpdated, originSender)
		// Дополнительно сообщим всем курьерам, что заказ уже назначен (чтобы убрали карточку)
		if s.cHub != nil {
			s.cHub.Broadcast(map[string]any{
				"type":       "order_assigned",
				"order_id":   req.OrderID,
				"courier_id": req.CourierID,
			})
		}
	} else {
		s.emitOffer(ctx, order, req.CourierID, repo.OfferStatusAccepted, &price, originSender)
		s.emitOrder(ctx, order, orderEventTypeUpdated, originSender)
		// Аналогичный broadcast всем курьерам
		if s.cHub != nil {
			s.cHub.Broadcast(map[string]any{
				"type":       "order_assigned",
				"order_id":   order.ID,
				"courier_id": req.CourierID,
			})
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": repo.StatusAccepted})
}

// ЗАКАЗЧИК ОТКЛОНЯЕТ ОФФЕР КУРЬЕРА
func (s *Server) handleOfferDecline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	senderID, err := parseAuthID(r, "X-Sender-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing sender id")
		return
	}
	var req struct {
		OrderID   int64 `json:"order_id"`
		CourierID int64 `json:"courier_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.OrderID == 0 || req.CourierID == 0 {
		writeError(w, http.StatusBadRequest, "order_id and courier_id are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	ctx = withSenderActor(ctx, senderID)

	if err := s.offers.UpdateStatus(ctx, req.OrderID, req.CourierID, repo.OfferStatusDeclined); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "offer not found")
			return
		}
		if s.logger != nil {
			s.logger.Errorf("courier: decline offer failed: %v", err)
		}
		writeError(w, http.StatusInternalServerError, "failed to decline offer")
		return
	}

	// Эмиты
	if order, err := s.orders.Get(ctx, req.OrderID); err != nil {
		if s.logger != nil {
			s.logger.Errorf("courier: load order %d after offer decline failed: %v", req.OrderID, err)
		}
		detached := withSenderActor(context.WithoutCancel(ctx), senderID)
		s.emitOfferEvent(detached, req.OrderID, req.CourierID, repo.OfferStatusDeclined, nil, originSender)
	} else {
		s.emitOffer(ctx, order, req.CourierID, repo.OfferStatusDeclined, nil, originSender)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": repo.OfferStatusDeclined})
}

// ОБОБЩЁННЫЙ РЕСПОНС ЗАКАЗЧИКА НА ОФФЕР: accept / decline
func (s *Server) handleOfferRespond(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	senderID, err := parseAuthID(r, "X-Sender-ID")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing sender id")
		return
	}
	var req struct {
		OrderID   int64  `json:"order_id"`
		CourierID int64  `json:"courier_id"`
		Decision  string `json:"decision"` // "accept" | "decline"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.OrderID == 0 || req.CourierID == 0 {
		writeError(w, http.StatusBadRequest, "order_id and courier_id are required")
		return
	}
	decision := strings.ToLower(strings.TrimSpace(req.Decision))
	if decision != "accept" && decision != "decline" {
		writeError(w, http.StatusBadRequest, "decision must be accept or decline")
		return
	}

	ctx, cancel := contextWithTimeout(r)
	defer cancel()
	ctx = withSenderActor(ctx, senderID)

	var respStatus string

	if decision == "accept" {
		// 1) Приняли оффер
		if err := s.offers.UpdateStatus(ctx, req.OrderID, req.CourierID, repo.OfferStatusAccepted); err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				writeError(w, http.StatusNotFound, "offer not found")
				return
			}
			if s.logger != nil {
				s.logger.Errorf("courier: respond offer accept failed: %v", err)
			}
			writeError(w, http.StatusInternalServerError, "failed to update offer")
			return
		}
		// 2) Назначили курьера заказу
		if err := s.orders.AssignCourier(ctx, req.OrderID, req.CourierID, lifecycle.StatusAccepted); err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				writeError(w, http.StatusNotFound, "order not found")
				return
			}
			if s.logger != nil {
				s.logger.Errorf("courier: assign courier on respond failed: %v", err)
			}
			writeError(w, http.StatusConflict, err.Error())
			return
		}

		// 3) Эмиты
		if order, err := s.orders.Get(ctx, req.OrderID); err == nil {
			s.emitOrder(ctx, order, orderEventTypeUpdated, originSender)
			s.emitOffer(ctx, order, req.CourierID, repo.OfferStatusAccepted, nil, originSender)
			// Сообщим всем курьерам, что заказ назначен
			if s.cHub != nil {
				s.cHub.Broadcast(map[string]any{
					"type":       "order_assigned",
					"order_id":   order.ID,
					"courier_id": req.CourierID,
				})
			}
		} else {
			detached := withSenderActor(context.WithoutCancel(ctx), senderID)
			s.emitOrderEvent(detached, req.OrderID, orderEventTypeUpdated, originSender)
			s.emitOfferEvent(detached, req.OrderID, req.CourierID, repo.OfferStatusAccepted, nil, originSender)
			if s.cHub != nil {
				s.cHub.Broadcast(map[string]any{
					"type":       "order_assigned",
					"order_id":   req.OrderID,
					"courier_id": req.CourierID,
				})
			}
		}

		respStatus = string(repo.StatusAccepted)
	} else {
		// decline
		if err := s.offers.UpdateStatus(ctx, req.OrderID, req.CourierID, repo.OfferStatusDeclined); err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				writeError(w, http.StatusNotFound, "offer not found")
				return
			}
			if s.logger != nil {
				s.logger.Errorf("courier: respond offer decline failed: %v", err)
			}
			writeError(w, http.StatusInternalServerError, "failed to update offer")
			return
		}
		if order, err := s.orders.Get(ctx, req.OrderID); err == nil {
			s.emitOffer(ctx, order, req.CourierID, repo.OfferStatusDeclined, nil, originSender)
		} else {
			detached := withSenderActor(context.WithoutCancel(ctx), senderID)
			s.emitOfferEvent(detached, req.OrderID, req.CourierID, repo.OfferStatusDeclined, nil, originSender)
		}
		respStatus = "declined"
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": respStatus})
}
