package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type WorkAdFavoriteService struct {
	WorkAdFavoriteRepo *repositories.WorkAdFavoriteRepository
}

func (s *WorkAdFavoriteService) AddWorkAdToFavorites(ctx context.Context, fav models.WorkAdFavorite) error {
	return s.WorkAdFavoriteRepo.AddWorkAdToFavorites(ctx, fav)
}

func (s *WorkAdFavoriteService) RemoveWorkAdFromFavorites(ctx context.Context, userID, workAdID int) error {
	return s.WorkAdFavoriteRepo.RemoveWorkAdFromFavorites(ctx, userID, workAdID)
}

func (s *WorkAdFavoriteService) IsWorkAdFavorite(ctx context.Context, userID, workAdID int) (bool, error) {
	return s.WorkAdFavoriteRepo.IsWorkAdFavorite(ctx, userID, workAdID)
}

func (s *WorkAdFavoriteService) GetWorkAdFavoritesByUser(ctx context.Context, userID int) ([]models.WorkAdFavorite, error) {
	return s.WorkAdFavoriteRepo.GetWorkAdFavoritesByUser(ctx, userID)
}
