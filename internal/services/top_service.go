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
	listingType, err := s.ensureActivationAllowed(ctx, userID, req)
	if err != nil {
		return models.TopInfo{}, err
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

// EnsureActivationAllowed validates the request and checks whether the user can manage the listing.
func (s *TopService) EnsureActivationAllowed(ctx context.Context, userID int, req models.TopActivationRequest) error {
	_, err := s.ensureActivationAllowed(ctx, userID, req)
	return err
}

func (s *TopService) ensureActivationAllowed(ctx context.Context, userID int, req models.TopActivationRequest) (string, error) {
	if err := req.Validate(); err != nil {
		return "", err
	}

	listingType := strings.TrimSpace(req.Type)
	ownerID, err := s.Repo.GetOwnerID(ctx, listingType, req.ID)
	if err != nil {
		return "", err
	}
	if ownerID != userID {
		return "", ErrTopForbidden
	}
	return listingType, nil
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
