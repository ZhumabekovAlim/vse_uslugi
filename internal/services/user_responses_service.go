package services

import (
	"context"

	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

// UserResponsesService provides business logic for retrieving user responses.
type UserResponsesService struct {
	ResponsesRepo *repositories.UserResponsesRepository
}

// GetResponsesByUserID fetches all responses made by a specific user.
func (s *UserResponsesService) GetResponsesByUserID(ctx context.Context, userID int) (models.UserResponses, error) {
	return s.ResponsesRepo.GetResponsesByUserID(ctx, userID)
}
