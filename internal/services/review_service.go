package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type ReviewService struct {
	ReviewsRepo *repositories.ReviewRepository
}

func (s *ReviewService) CreateReview(ctx context.Context, review models.Reviews) (models.Reviews, error) {
	return s.ReviewsRepo.CreateReview(ctx, review)
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
