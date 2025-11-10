package http

import (
	"context"

	"naimuBack/internal/courier/repo"
)

const (
	orderEventTypeCreated = "order_created"
	orderEventTypeUpdated = "order_updated"
	offerEventTypeUpdated = "offer_updated"
)

type orderEvent struct {
	Type  string        `json:"type"`
	Order orderResponse `json:"order"`
}

type offerEvent struct {
	Type      string `json:"type"`
	OrderID   int64  `json:"order_id"`
	CourierID int64  `json:"courier_id"`
	Status    string `json:"status"`
	Price     *int   `json:"price,omitempty"`
}

type eventOrigin int

const (
	originUnknown eventOrigin = iota
	originSender
	originCourier
)

func (s *Server) emitOrderEvent(ctx context.Context, orderID int64, eventType string, origin eventOrigin) {
	if s.sHub == nil && s.cHub == nil {
		return
	}
	ctx = detachContext(ctx)
	order, err := s.orders.Get(ctx, orderID)
	if err != nil {
		if s.logger != nil {
			s.logger.Errorf("courier: load order %d for ws failed: %v", orderID, err)
		}
		return
	}
	s.emitOrder(order, eventType, origin)
}

func (s *Server) emitOrder(order repo.Order, eventType string, origin eventOrigin) {
	if s.sHub == nil && s.cHub == nil {
		return
	}
	resp := makeOrderResponse(order)
	evt := orderEvent{Type: eventType, Order: resp}
	notifySender := origin != originSender
	notifyCourier := origin != originCourier
	if s.sHub != nil && notifySender {
		s.sHub.Push(order.SenderID, evt)
	}
	if s.cHub != nil && notifyCourier {
		if order.CourierID.Valid {
			s.cHub.Push(order.CourierID.Int64, evt)
		} else if eventType == orderEventTypeCreated {
			s.cHub.Broadcast(evt)
		}
	}
}

func (s *Server) emitOfferEvent(ctx context.Context, orderID, courierID int64, status string, price *int, origin eventOrigin) {
	if s.sHub == nil && s.cHub == nil {
		return
	}
	ctx = detachContext(ctx)
	order, err := s.orders.Get(ctx, orderID)
	if err != nil {
		if s.logger != nil {
			s.logger.Errorf("courier: load order %d for offer event failed: %v", orderID, err)
		}
		return
	}
	s.emitOffer(order, courierID, status, price, origin)
}

func (s *Server) emitOffer(order repo.Order, courierID int64, status string, price *int, origin eventOrigin) {
	if s.sHub == nil && s.cHub == nil {
		return
	}
	evt := offerEvent{Type: offerEventTypeUpdated, OrderID: order.ID, CourierID: courierID, Status: status, Price: price}
	notifySender := origin != originSender
	notifyCourier := origin != originCourier
	if s.sHub != nil && notifySender {
		s.sHub.Push(order.SenderID, evt)
	}
	if s.cHub != nil && notifyCourier {
		s.cHub.Push(courierID, evt)
	}
}

func detachContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return context.WithoutCancel(ctx)
}
