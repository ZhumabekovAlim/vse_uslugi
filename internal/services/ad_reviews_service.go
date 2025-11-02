package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type AdReviewService struct {
	AdReviewsRepo    *repositories.AdReviewRepository
	ConfirmationRepo *repositories.AdConfirmationRepository
}

func (s *AdReviewService) CreateAdReview(ctx context.Context, review models.AdReviews) (models.AdReviews, error) {
	rev, err := s.AdReviewsRepo.CreateAdReview(ctx, review)
	if err != nil {
		return rev, err
	}
	if s.ConfirmationRepo != nil && review.AdID != 0 {
		if err := s.ConfirmationRepo.Done(ctx, review.AdID); err != nil {
			return rev, err
		}
	}
	return rev, nil
}

func (s *AdReviewService) GetReviewsByAdID(ctx context.Context, adID int) ([]models.AdReviews, error) {
	return s.AdReviewsRepo.GetReviewsByAdID(ctx, adID)
}

func (s *AdReviewService) UpdateAdReview(ctx context.Context, review models.AdReviews) error {
	return s.AdReviewsRepo.UpdateAdReview(ctx, review)
}

func (s *AdReviewService) DeleteAdReview(ctx context.Context, id int) error {
	return s.AdReviewsRepo.DeleteAdReview(ctx, id)
}
