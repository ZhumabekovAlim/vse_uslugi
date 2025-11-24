package services

import (
	"context"

	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

// ResponseUsersService provides business logic for retrieving users who responded to items.
type ResponseUsersService struct {
	Repo *repositories.ResponseUsersRepository
}

// GetUsersByItemID fetches users who responded to the given item type and ID.
func (s *ResponseUsersService) GetUsersByItemID(ctx context.Context, itemType string, itemID int) ([]models.ResponseUser, error) {
	return s.Repo.GetUsersByItemID(ctx, itemType, itemID)
}
