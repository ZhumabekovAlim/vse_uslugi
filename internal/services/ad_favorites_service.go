package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type AdFavoriteService struct {
	AdFavoriteRepo *repositories.AdFavoriteRepository
}

func (s *AdFavoriteService) AddAdToFavorites(ctx context.Context, fav models.AdFavorite) error {
	return s.AdFavoriteRepo.AddAdToFavorites(ctx, fav)
}

func (s *AdFavoriteService) RemoveAdFromFavorites(ctx context.Context, userID, adID int) error {
	return s.AdFavoriteRepo.RemoveAdFromFavorites(ctx, userID, adID)
}

func (s *AdFavoriteService) IsAdFavorite(ctx context.Context, userID, adID int) (bool, error) {
	return s.AdFavoriteRepo.IsAdFavorite(ctx, userID, adID)
}

func (s *AdFavoriteService) GetAdFavoritesByUser(ctx context.Context, userID int) ([]models.AdFavorite, error) {
	return s.AdFavoriteRepo.GetAdFavoritesByUser(ctx, userID)
}
