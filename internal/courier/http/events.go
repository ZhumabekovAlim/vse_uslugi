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

type ctxKey string

const (
	ctxCourierActorKey ctxKey = "courier_actor_id"
	ctxSenderActorKey  ctxKey = "sender_actor_id"
)

type orderEvent struct {
	Type    string            `json:"type"`
	Order   orderResponse     `json:"order"`
	Courier *courierEventInfo `json:"courier,omitempty"`
	Sender  *userResponse     `json:"sender,omitempty"`
}

type courierEventInfo struct {
	Profile courierResponse `json:"profile"`
	User    userResponse    `json:"user"`
}

type offerEvent struct {
	Type      string            `json:"type"`
	OrderID   int64             `json:"order_id"`
	CourierID int64             `json:"courier_id"`
	Status    string            `json:"status"`
	Price     *int              `json:"price,omitempty"`
	Courier   *courierEventInfo `json:"courier,omitempty"`
	Sender    *userResponse     `json:"sender,omitempty"`
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
	s.emitOrder(ctx, order, eventType, origin)
}

func (s *Server) emitOrder(ctx context.Context, order repo.Order, eventType string, origin eventOrigin) {
	if s.sHub == nil && s.cHub == nil {
		return
	}
	resp := makeOrderResponse(order)
	evt := orderEvent{Type: resolveOrderEventType(eventType, order), Order: resp}

	switch origin {
	case originCourier:
		courierID := courierActorFromContext(ctx)
		if courierID == 0 && order.CourierID.Valid {
			courierID = order.CourierID.Int64
		}
		if info, err := s.buildCourierEventInfo(ctx, courierID); err == nil {
			evt.Courier = info
		} else if err != nil && s.logger != nil {
			s.logger.Errorf("courier: load courier info %d failed: %v", courierID, err)
		}
	case originSender:
		senderID := senderActorFromContext(ctx)
		if senderID == 0 {
			senderID = order.SenderID
		}
		if info, err := s.buildUserEventInfo(ctx, senderID); err == nil {
			evt.Sender = info
		} else if err != nil && s.logger != nil {
			s.logger.Errorf("courier: load sender info %d failed: %v", senderID, err)
		}
	}

	notifySender := origin != originSender
	notifyCourier := origin != originCourier
	if s.sHub != nil && notifySender {
		s.sHub.Push(order.SenderID, evt)
	}
	if s.cHub != nil && notifyCourier {
		if order.CourierID.Valid {
			s.cHub.Push(order.CourierID.Int64, evt)
		} else if evt.Type == orderEventTypeCreated {
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
	s.emitOffer(ctx, order, courierID, status, price, origin)
}

func (s *Server) emitOffer(ctx context.Context, order repo.Order, courierID int64, status string, price *int, origin eventOrigin) {
	if s.sHub == nil && s.cHub == nil {
		return
	}
	evt := offerEvent{Type: offerEventTypeUpdated, OrderID: order.ID, CourierID: courierID, Status: status, Price: price}

	switch origin {
	case originCourier:
		actorID := courierActorFromContext(ctx)
		if actorID == 0 {
			actorID = courierID
		}
		if info, err := s.buildCourierEventInfo(ctx, actorID); err == nil {
			evt.Courier = info
		} else if err != nil && s.logger != nil {
			s.logger.Errorf("courier: load courier info %d failed: %v", actorID, err)
		}
	case originSender:
		senderID := senderActorFromContext(ctx)
		if senderID == 0 {
			senderID = order.SenderID
		}
		if info, err := s.buildUserEventInfo(ctx, senderID); err == nil {
			evt.Sender = info
		} else if err != nil && s.logger != nil {
			s.logger.Errorf("courier: load sender info %d failed: %v", senderID, err)
		}
	}

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

func withCourierActor(ctx context.Context, courierID int64) context.Context {
	if courierID == 0 {
		return ctx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxCourierActorKey, courierID)
}

func withSenderActor(ctx context.Context, senderID int64) context.Context {
	if senderID == 0 {
		return ctx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxSenderActorKey, senderID)
}

func courierActorFromContext(ctx context.Context) int64 {
	if ctx == nil {
		return 0
	}
	if v, ok := ctx.Value(ctxCourierActorKey).(int64); ok {
		return v
	}
	return 0
}

func senderActorFromContext(ctx context.Context) int64 {
	if ctx == nil {
		return 0
	}
	if v, ok := ctx.Value(ctxSenderActorKey).(int64); ok {
		return v
	}
	return 0
}

func (s *Server) buildCourierEventInfo(ctx context.Context, courierID int64) (*courierEventInfo, error) {
	if courierID == 0 || s.couriers == nil || s.users == nil {
		return nil, nil
	}
	courier, err := s.couriers.Get(ctx, courierID)
	if err != nil {
		return nil, err
	}
	user, err := s.users.Get(ctx, courier.UserID)
	if err != nil {
		return nil, err
	}
	payload := courierEventInfo{Profile: makeCourierResponse(courier), User: makeUserResponse(user)}
	return &payload, nil
}

func (s *Server) buildUserEventInfo(ctx context.Context, userID int64) (*userResponse, error) {
	if userID == 0 || s.users == nil {
		return nil, nil
	}
	user, err := s.users.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	resp := makeUserResponse(user)
	return &resp, nil
}

func resolveOrderEventType(eventType string, order repo.Order) string {
	if eventType == orderEventTypeCreated {
		return orderEventTypeCreated
	}
	switch order.Status {
	case repo.StatusCanceledBySender, repo.StatusCanceledByCourier, repo.StatusCanceledNoShow:
		return "order_canceled"
	case repo.StatusAccepted:
		return "order_assigned"
	case repo.StatusCompleted, repo.StatusClosed:
		return "order_completed"
	default:
		return "order_status"
	}
}
