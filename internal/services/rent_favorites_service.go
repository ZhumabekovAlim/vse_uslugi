package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type RentFavoriteService struct {
	RentFavoriteRepo *repositories.RentFavoriteRepository
}

func (s *RentFavoriteService) AddRentToFavorites(ctx context.Context, fav models.RentFavorite) error {
	return s.RentFavoriteRepo.AddRentToFavorites(ctx, fav)
}

func (s *RentFavoriteService) RemoveRentFromFavorites(ctx context.Context, userID, rentID int) error {
	return s.RentFavoriteRepo.RemoveRentFromFavorites(ctx, userID, rentID)
}

func (s *RentFavoriteService) IsRentFavorite(ctx context.Context, userID, rentID int) (bool, error) {
	return s.RentFavoriteRepo.IsRentFavorite(ctx, userID, rentID)
}

func (s *RentFavoriteService) GetRentFavoritesByUser(ctx context.Context, userID, cityID int) ([]models.RentFavorite, error) {
	return s.RentFavoriteRepo.GetRentFavoritesByUser(ctx, userID, cityID)
}
