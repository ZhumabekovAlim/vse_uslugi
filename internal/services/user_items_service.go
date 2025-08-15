package services

import (
	"context"

	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

// UserItemsService provides business logic for retrieving user items.
type UserItemsService struct {
	ItemsRepo *repositories.UserItemsRepository
}

// GetServiceWorkRentByUserID fetches service, work and rent items for the user.
func (s *UserItemsService) GetServiceWorkRentByUserID(ctx context.Context, userID int) ([]models.UserItem, error) {
	return s.ItemsRepo.GetServiceWorkRentByUserID(ctx, userID)
}

// GetAdWorkAdRentAdByUserID fetches ad, work_ad and rent_ad items for the user.
func (s *UserItemsService) GetAdWorkAdRentAdByUserID(ctx context.Context, userID int) ([]models.UserItem, error) {
	return s.ItemsRepo.GetAdWorkAdRentAdByUserID(ctx, userID)
}
