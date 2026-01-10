package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type ServiceFavoriteService struct {
	ServiceFavoriteRepo *repositories.ServiceFavoriteRepository
}

func (s *ServiceFavoriteService) AddToFavorites(ctx context.Context, fav models.ServiceFavorite) error {
	return s.ServiceFavoriteRepo.AddToFavorites(ctx, fav)
}

func (s *ServiceFavoriteService) RemoveFromFavorites(ctx context.Context, userID, serviceID int) error {
	return s.ServiceFavoriteRepo.RemoveFromFavorites(ctx, userID, serviceID)
}

func (s *ServiceFavoriteService) IsFavorite(ctx context.Context, userID, serviceID int) (bool, error) {
	return s.ServiceFavoriteRepo.IsFavorite(ctx, userID, serviceID)
}

func (s *ServiceFavoriteService) GetFavoritesByUser(ctx context.Context, userID, cityID int) ([]models.ServiceFavorite, error) {
	return s.ServiceFavoriteRepo.GetFavoritesByUser(ctx, userID, cityID)
}
