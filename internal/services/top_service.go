package services

import (
	"context"
	"errors"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
	"strings"
	"time"
)

var (
	ErrTopForbidden = errors.New("user is not allowed to manage this listing")
)

type TopService struct {
	Repo *repositories.TopRepository
}

func NewTopService(repo *repositories.TopRepository) *TopService {
	return &TopService{Repo: repo}
}

func (s *TopService) ActivateTop(ctx context.Context, userID int, req models.TopActivationRequest) (models.TopInfo, error) {
	if err := req.Validate(); err != nil {
		return models.TopInfo{}, err
	}

	listingType := strings.TrimSpace(req.Type)

	ownerID, err := s.Repo.GetOwnerID(ctx, listingType, req.ID)
	if err != nil {
		return models.TopInfo{}, err
	}
	if ownerID != userID {
		return models.TopInfo{}, ErrTopForbidden
	}

	now := time.Now().UTC()
	info, err := models.NewTopInfo(now, req.DurationDays)
	if err != nil {
		return models.TopInfo{}, err
	}
	if err := s.Repo.UpdateTop(ctx, listingType, req.ID, info); err != nil {
		return models.TopInfo{}, err
	}
	return info, nil
}

func (s *TopService) ClearExpiredTop(ctx context.Context, now time.Time) (int, error) {
	if s == nil || s.Repo == nil {
		return 0, nil
	}
	if now.IsZero() {
		now = time.Now()
	}
	return s.Repo.ClearExpiredTop(ctx, now.UTC())
}
