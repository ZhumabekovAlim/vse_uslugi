package models

import (
	"time"

	"github.com/dgrijalva/jwt-go"
)

type User struct {
	ID           int        `json:"id"`
	Name         string     `json:"name"`
	Surname      string     `json:"surname"`
	Middlename   string     `json:"middlename,omitempty"`
	Phone        string     `json:"phone"`
	Email        string     `json:"email"`
	Password     string     `json:"password"`
	CityID       *int       `json:"city_id,omitempty"`
	YearsOfExp   *int       `json:"years_of_exp,omitempty"`
	DocOfProof   *string    `json:"doc_of_proof,omitempty"`
	AvatarPath   *string    `json:"avatar_path,omitempty"`
	ReviewRating float64    `json:"review_rating"`
	Role         string     `json:"role,omitempty"`
	Banned       bool       `json:"banned"`
	Latitude     *string    `json:"latitude,omitempty"`
	Longitude    *string    `json:"longitude,omitempty"`
	IsOnline     bool       `json:"is_online"`
	Skills       string     `json:"skills,omitempty"`
	Categories   []Category `json:"categories,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
	DriverID     *int       `json:"driver_id,omitempty"`
	CourierID    *int       `json:"courier_id,omitempty"`
}

type Claims struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`
	CityID int    `json:"city_id"`
	jwt.StandardClaims
}

type Tokens struct {
	AccessToken  string
	RefreshToken string
}

type Session struct {
	RefreshToken string    `json:"refreshToken" bson:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt" bson:"expiresAt"`
	// Из User
	Name       string `json:"name"`
	Surname    string `json:"surname"`
	Middlename string `json:"middlename"`
	Phone      string `json:"phone"`
	CityID     int    `json:"city_id"`
	UserID     int    `json:"user_id"`
	Role       string `json:"role"`
	// Из City
	City string `json:"city"`
}

type SignInRequest struct {
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type SignUpResponse struct {
	User             User   `json:"user"`
	VerificationCode string `json:"verification_code,omitempty"`
}
type SignUpResponse1 struct {
	User             User   `json:"user"`
	VerificationCode string `json:"verification_code"`
}

type DuplicateCheckRequest struct {
	Phone string `json:"phone"`
	Email string `json:"email"`
}

type VerificationCodeEntry struct {
	ID        int       `json:"id"`
	Phone     string    `json:"phone"`
	Code      string    `json:"code"`
	CreatedAt time.Time `json:"created_at"`
}
