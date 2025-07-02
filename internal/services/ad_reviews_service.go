package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type AdReviewService struct {
	AdReviewsRepo *repositories.AdReviewRepository
}

func (s *AdReviewService) CreateAdReview(ctx context.Context, review models.AdReviews) (models.AdReviews, error) {
	return s.AdReviewsRepo.CreateAdReview(ctx, review)
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
