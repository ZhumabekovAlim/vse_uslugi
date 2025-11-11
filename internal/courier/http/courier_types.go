package http

import (
	"time"

	"naimuBack/internal/courier/repo"
)

type courierResponse struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	MiddleName  *string   `json:"middle_name"`
	Photo       string    `json:"courier_photo"`
	IIN         string    `json:"iin"`
	DateOfBirth time.Time `json:"date_of_birth"`
	IDCardFront string    `json:"id_card_front"`
	IDCardBack  string    `json:"id_card_back"`
	Phone       string    `json:"phone"`
	Rating      *float64  `json:"rating,omitempty"`
	Balance     int       `json:"balance"`
	Status      string    `json:"status"`
	Approval    string    `json:"approval_status"`
	IsBanned    bool      `json:"is_banned"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type courierStatsResponse struct {
	Total     int `json:"total_orders"`
	Active    int `json:"active_orders"`
	Completed int `json:"completed_orders"`
	Canceled  int `json:"canceled_orders"`
}

func makeCourierResponse(c repo.Courier) courierResponse {
	var middleName *string
	if c.MiddleName.Valid {
		v := c.MiddleName.String
		middleName = &v
	}
	var rating *float64
	if c.Rating.Valid {
		v := c.Rating.Float64
		rating = &v
	}
	return courierResponse{
		ID:          c.ID,
		UserID:      c.UserID,
		FirstName:   c.FirstName,
		LastName:    c.LastName,
		MiddleName:  middleName,
		Photo:       c.Photo,
		IIN:         c.IIN,
		DateOfBirth: c.BirthDate,
		IDCardFront: c.IDCardFront,
		IDCardBack:  c.IDCardBack,
		Phone:       c.Phone,
		Rating:      rating,
		Balance:     c.Balance,
		Status:      c.Status,
		Approval:    c.ApprovalStatus,
		IsBanned:    c.IsBanned,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

type courierProfileResponse struct {
	Courier courierResponse      `json:"courier"`
	Stats   courierStatsResponse `json:"stats"`
}

type courierReviewResponse struct {
	Rating        *float64      `json:"rating,omitempty"`
	CourierRating *float64      `json:"courier_rating,omitempty"`
	Comment       string        `json:"comment,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	Order         orderResponse `json:"order"`
}

func makeCourierStatsResponse(stats repo.CourierOrderStats) courierStatsResponse {
	return courierStatsResponse{
		Total:     stats.Total,
		Active:    stats.Active,
		Completed: stats.Completed,
		Canceled:  stats.Canceled,
	}
}
