package http

import (
	"time"

	"naimuBack/internal/courier/repo"
)

type orderPointInput struct {
	Address  string  `json:"address"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Entrance *string `json:"entrance"`
	Apt      *string `json:"apt"`
	Floor    *string `json:"floor"`
	Intercom *string `json:"intercom"`
	Phone    *string `json:"phone"`
	Comment  *string `json:"comment"`
}

type orderPointResponse struct {
	Address  string  `json:"address"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Entrance *string `json:"entrance"`
	Apt      *string `json:"apt"`
	Floor    *string `json:"floor"`
	Intercom *string `json:"intercom"`
	Phone    *string `json:"phone"`
	Comment  *string `json:"comment"`
	Seq      int     `json:"seq"`
}

type orderResponse struct {
	ID               int64                `json:"id"`
	SenderID         int64                `json:"sender_id"`
	CourierID        *int64               `json:"courier_id"`
	DistanceM        int                  `json:"distance_m"`
	EtaSeconds       int                  `json:"eta_s"`
	RecommendedPrice int                  `json:"recommended_price"`
	ClientPrice      int                  `json:"client_price"`
	PaymentMethod    string               `json:"payment_method"`
	Status           string               `json:"status"`
	Comment          *string              `json:"comment"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
	RoutePoints      []orderPointResponse `json:"route_points"`
	Sender           *userResponse        `json:"sender,omitempty"`
	Courier          *courierResponse     `json:"courier,omitempty"`
}

func makeOrderResponse(o repo.Order) orderResponse {
	var courierID *int64
	if o.CourierID.Valid {
		v := o.CourierID.Int64
		courierID = &v
	}
	var comment *string
	if o.Comment.Valid {
		v := o.Comment.String
		comment = &v
	}
	points := make([]orderPointResponse, 0, len(o.Points))
	for _, p := range o.Points {
		points = append(points, orderPointResponse{
			Address:  p.Address,
			Lat:      p.Lat,
			Lon:      p.Lon,
			Entrance: nullToPtr(p.Entrance),
			Apt:      nullToPtr(p.Apt),
			Floor:    nullToPtr(p.Floor),
			Intercom: nullToPtr(p.Intercom),
			Phone:    nullToPtr(p.Phone),
			Comment:  nullToPtr(p.Comment),
			Seq:      p.Seq,
		})
	}
	var sender *userResponse
	if o.Sender.ID != 0 {
		s := makeUserResponse(o.Sender)
		sender = &s
	}
	var courier *courierResponse
	if o.Courier != nil {
		c := makeCourierResponse(*o.Courier)
		courier = &c
	}
	return orderResponse{
		ID:               o.ID,
		SenderID:         o.SenderID,
		CourierID:        courierID,
		DistanceM:        o.DistanceM,
		EtaSeconds:       o.EtaSeconds,
		RecommendedPrice: o.RecommendedPrice,
		ClientPrice:      o.ClientPrice,
		PaymentMethod:    o.PaymentMethod,
		Status:           o.Status,
		Comment:          comment,
		CreatedAt:        o.CreatedAt,
		UpdatedAt:        o.UpdatedAt,
		RoutePoints:      points,
		Sender:           sender,
		Courier:          courier,
	}
}
