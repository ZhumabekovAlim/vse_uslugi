package http

import (
	"time"

	"naimuBack/internal/courier/repo"
)

type userResponse struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	Surname      string     `json:"surname"`
	Middlename   *string    `json:"middlename,omitempty"`
	Phone        string     `json:"phone"`
	Email        string     `json:"email"`
	CityID       *int64     `json:"city_id,omitempty"`
	YearsOfExp   *int64     `json:"years_of_exp,omitempty"`
	DocOfProof   *string    `json:"doc_of_proof,omitempty"`
	ReviewRating *float64   `json:"review_rating,omitempty"`
	Role         *string    `json:"role,omitempty"`
	Latitude     *string    `json:"latitude,omitempty"`
	Longitude    *string    `json:"longitude,omitempty"`
	AvatarPath   *string    `json:"avatar_path,omitempty"`
	Skills       *string    `json:"skills,omitempty"`
	IsOnline     *bool      `json:"is_online,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
}

func makeUserResponse(u repo.User) userResponse {
	return userResponse{
		ID:           u.ID,
		Name:         u.Name,
		Surname:      u.Surname,
		Middlename:   nullToPtr(u.Middlename),
		Phone:        u.Phone,
		Email:        u.Email,
		CityID:       nullInt64ToPtr(u.CityID),
		YearsOfExp:   nullInt64ToPtr(u.YearsOfExp),
		DocOfProof:   nullToPtr(u.DocOfProof),
		ReviewRating: nullFloat64ToPtr(u.ReviewRating),
		Role:         nullToPtr(u.Role),
		Latitude:     nullToPtr(u.Latitude),
		Longitude:    nullToPtr(u.Longitude),
		AvatarPath:   nullToPtr(u.AvatarPath),
		Skills:       nullToPtr(u.Skills),
		IsOnline:     nullBoolToPtr(u.IsOnline),
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    nullTimeToPtr(u.UpdatedAt),
	}
}
