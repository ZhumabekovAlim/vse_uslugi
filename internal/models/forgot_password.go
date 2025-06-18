package models

type PasswordResetRequest struct {
	Email string `json:"email"`
}

type VerifyResetCodeRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type NewPasswordRequest struct {
	Email       string `json:"email"`
	NewPassword string `json:"new_password"`
}
