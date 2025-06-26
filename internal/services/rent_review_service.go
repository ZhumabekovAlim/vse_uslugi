package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type RentReviewService struct {
	RentReviewsRepo *repositories.RentReviewRepository
}

func (s *RentReviewService) CreateRentReview(ctx context.Context, review models.RentReviews) (models.RentReviews, error) {
	return s.RentReviewsRepo.CreateRentReview(ctx, review)
}

func (s *RentReviewService) GetRentReviewsByRentID(ctx context.Context, rentID int) ([]models.RentReviews, error) {
	return s.RentReviewsRepo.GetRentReviewsByRentID(ctx, rentID)
}

func (s *RentReviewService) UpdateRentReview(ctx context.Context, review models.RentReviews) error {
	return s.RentReviewsRepo.UpdateRentReview(ctx, review)
}

func (s *RentReviewService) DeleteRentReview(ctx context.Context, id int) error {
	return s.RentReviewsRepo.DeleteRentReview(ctx, id)
}
