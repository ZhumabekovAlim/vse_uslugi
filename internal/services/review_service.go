package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type ReviewService struct {
	ReviewsRepo      *repositories.ReviewRepository
	ConfirmationRepo *repositories.ServiceConfirmationRepository
}

func (s *ReviewService) CreateReview(ctx context.Context, review models.Reviews) (models.Reviews, error) {
	rev, err := s.ReviewsRepo.CreateReview(ctx, review)
	if err != nil {
		return rev, err
	}
	if s.ConfirmationRepo != nil && review.ServiceID != 0 {
		if err := s.ConfirmationRepo.Done(ctx, review.ServiceID); err != nil {
			return rev, err
		}
	}
	return rev, nil
}

func (s *ReviewService) GetReviewsByServiceID(ctx context.Context, serviceID int) ([]models.Reviews, error) {
	return s.ReviewsRepo.GetReviewsByServiceID(ctx, serviceID)
}

func (s *ReviewService) UpdateReview(ctx context.Context, review models.Reviews) error {
	return s.ReviewsRepo.UpdateReview(ctx, review)
}

func (s *ReviewService) DeleteReview(ctx context.Context, id int) error {
	return s.ReviewsRepo.DeleteReview(ctx, id)
}
