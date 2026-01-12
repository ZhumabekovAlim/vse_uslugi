package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type RentAdFavoriteService struct {
	RentAdFavoriteRepo *repositories.RentAdFavoriteRepository
}

func (s *RentAdFavoriteService) AddRentAdToFavorites(ctx context.Context, fav models.RentAdFavorite) error {
	return s.RentAdFavoriteRepo.AddRentAdToFavorites(ctx, fav)
}

func (s *RentAdFavoriteService) RemoveRentAdFromFavorites(ctx context.Context, userID, rentAdID int) error {
	return s.RentAdFavoriteRepo.RemoveRentAdFromFavorites(ctx, userID, rentAdID)
}

func (s *RentAdFavoriteService) IsRentAdFavorite(ctx context.Context, userID, rentAdID int) (bool, error) {
	return s.RentAdFavoriteRepo.IsRentAdFavorite(ctx, userID, rentAdID)
}

func (s *RentAdFavoriteService) GetRentAdFavoritesByUser(ctx context.Context, userID int) ([]models.RentAdFavorite, error) {
	return s.RentAdFavoriteRepo.GetRentAdFavoritesByUser(ctx, userID)
}
