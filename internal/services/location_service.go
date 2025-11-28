package services

import (
	"context"
	"fmt"

	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

// LocationService coordinates location repository interactions.
type LocationService struct {
	Repo         *repositories.LocationRepository
	BusinessRepo *repositories.BusinessRepository
}

// SetLocation updates coordinates for a user.
func (s *LocationService) SetLocation(ctx context.Context, loc models.Location) error {
	return s.Repo.SetLocation(ctx, loc)
}

// GetLocation returns stored coordinates for a user.
func (s *LocationService) GetLocation(ctx context.Context, userID int) (models.Location, error) {
	return s.Repo.GetLocation(ctx, userID)
}

// GoOffline clears coordinates and marks user offline.
func (s *LocationService) GoOffline(ctx context.Context, userID int) error {
	return s.Repo.ClearLocation(ctx, userID)
}

// GetExecutors returns online executors with active items by filter.
func (s *LocationService) GetExecutors(ctx context.Context, f models.ExecutorLocationFilter) ([]models.ExecutorLocationGroup, error) {
	return s.Repo.GetExecutors(ctx, f)
}

// GetBusinessMarkers returns aggregated markers for all businesses with online workers.
func (s *LocationService) GetBusinessMarkers(ctx context.Context) ([]models.BusinessAggregatedMarker, error) {
	return s.Repo.GetBusinessMarkers(ctx)
}

// UpdateBusinessWorkerLocation stores coordinates for a business worker and returns detailed payloads for broadcasting.
func (s *LocationService) UpdateBusinessWorkerLocation(ctx context.Context, workerUserID int, lat, lon float64) (models.BusinessWorkerLocation, *models.BusinessAggregatedMarker, error) {
	if s.BusinessRepo == nil {
		return models.BusinessWorkerLocation{}, nil, fmt.Errorf("business repository not configured")
	}

	worker, err := s.BusinessRepo.GetWorkerByUserID(ctx, workerUserID)
	if err != nil {
		return models.BusinessWorkerLocation{}, nil, err
	}
	if worker.ID == 0 {
		return models.BusinessWorkerLocation{}, nil, fmt.Errorf("business worker not found")
	}

	if err := s.Repo.SetLocation(ctx, models.Location{UserID: workerUserID, Latitude: &lat, Longitude: &lon}); err != nil {
		return models.BusinessWorkerLocation{}, nil, err
	}

	execs, err := s.Repo.GetExecutors(ctx, models.ExecutorLocationFilter{BusinessUserID: worker.BusinessUserID, WorkerIDs: []int{workerUserID}})
	if err != nil {
		return models.BusinessWorkerLocation{}, nil, err
	}

	var workerPayload models.ExecutorLocationGroup
	if len(execs) > 0 {
		workerPayload = execs[0]
	}

	marker, err := s.buildAggregatedMarker(ctx, worker.BusinessUserID)
	if err != nil {
		return models.BusinessWorkerLocation{}, nil, err
	}

	return models.BusinessWorkerLocation{BusinessUserID: worker.BusinessUserID, Worker: workerPayload}, marker, nil
}

// SetBusinessWorkerOffline clears location for a business worker, marks offline and returns aggregation updates.
func (s *LocationService) SetBusinessWorkerOffline(ctx context.Context, workerUserID int) (int, *models.BusinessAggregatedMarker, error) {
	if s.BusinessRepo == nil {
		if err := s.Repo.ClearLocation(ctx, workerUserID); err != nil {
			return 0, nil, err
		}
		return 0, nil, nil
	}

	worker, err := s.BusinessRepo.GetWorkerByUserID(ctx, workerUserID)
	if err != nil {
		return 0, nil, err
	}

	if err := s.Repo.ClearLocation(ctx, workerUserID); err != nil {
		return worker.BusinessUserID, nil, err
	}

	if worker.ID == 0 {
		return 0, nil, nil
	}

	marker, err := s.buildAggregatedMarker(ctx, worker.BusinessUserID)
	if err != nil {
		return worker.BusinessUserID, nil, err
	}

	return worker.BusinessUserID, marker, nil
}

func (s *LocationService) buildAggregatedMarker(ctx context.Context, businessUserID int) (*models.BusinessAggregatedMarker, error) {
	execs, err := s.Repo.GetExecutors(ctx, models.ExecutorLocationFilter{BusinessUserID: businessUserID})
	if err != nil {
		return nil, err
	}

	var (
		sumLat float64
		sumLon float64
		count  int
	)

	for _, exec := range execs {
		if exec.Latitude == nil || exec.Longitude == nil {
			continue
		}
		sumLat += *exec.Latitude
		sumLon += *exec.Longitude
		count++
	}

	marker := &models.BusinessAggregatedMarker{BusinessUserID: businessUserID, WorkerCount: count}
	if count == 0 {
		return marker, nil
	}

	lat := sumLat / float64(count)
	lon := sumLon / float64(count)
	marker.Latitude = &lat
	marker.Longitude = &lon
	return marker, nil
}
