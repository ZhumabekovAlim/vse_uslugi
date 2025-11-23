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

// GetItemResponses fetches item details alongside users who responded to it.
func (s *ResponseUsersService) GetItemResponses(ctx context.Context, itemType string, itemID int) (models.ItemResponse, error) {
	users, err := s.Repo.GetUsersByItemID(ctx, itemType, itemID)
	if err != nil {
		return models.ItemResponse{}, err
	}

	itemInfo, err := s.Repo.GetItemInfo(ctx, itemType, itemID)
	if err != nil {
		return models.ItemResponse{}, err
	}

	itemInfo.Users = users
	return itemInfo, nil
}
