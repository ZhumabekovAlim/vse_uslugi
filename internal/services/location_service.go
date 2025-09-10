package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

// LocationService coordinates location repository interactions.
type LocationService struct {
	Repo *repositories.LocationRepository
}

// SetLocation updates coordinates for a user.
func (s *LocationService) SetLocation(ctx context.Context, loc models.Location) error {
	return s.Repo.SetLocation(ctx, loc)
}

// GetLocation returns stored coordinates for a user.
func (s *LocationService) GetLocation(ctx context.Context, userID int) (models.Location, error) {
	return s.Repo.GetLocation(ctx, userID)
}

// GetExecutors returns online executors with active items by filter.
func (s *LocationService) GetExecutors(ctx context.Context, f models.ExecutorLocationFilter) ([]models.ExecutorLocation, error) {
	return s.Repo.GetExecutors(ctx, f)
}
