package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type WorkFavoriteService struct {
	WorkFavoriteRepo *repositories.WorkFavoriteRepository
}

func (s *WorkFavoriteService) AddWorkToFavorites(ctx context.Context, fav models.WorkFavorite) error {
	return s.WorkFavoriteRepo.AddWorkToFavorites(ctx, fav)
}

func (s *WorkFavoriteService) RemoveWorkFromFavorites(ctx context.Context, userID, workID int) error {
	return s.WorkFavoriteRepo.RemoveWorkFromFavorites(ctx, userID, workID)
}

func (s *WorkFavoriteService) IsWorkFavorite(ctx context.Context, userID, workID int) (bool, error) {
	return s.WorkFavoriteRepo.IsWorkFavorite(ctx, userID, workID)
}

func (s *WorkFavoriteService) GetWorkFavoritesByUser(ctx context.Context, userID, cityID int) ([]models.WorkFavorite, error) {
	return s.WorkFavoriteRepo.GetWorkFavoritesByUser(ctx, userID, cityID)
}
