package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type RentAdReviewService struct {
	RentAdReviewsRepo *repositories.RentAdReviewRepository
}

func (s *RentAdReviewService) CreateRentAdReview(ctx context.Context, review models.RentAdReviews) (models.RentAdReviews, error) {
	return s.RentAdReviewsRepo.CreateRentAdReview(ctx, review)
}

func (s *RentAdReviewService) GetRentAdReviewsByRentID(ctx context.Context, rentAdID int) ([]models.RentAdReviews, error) {
	return s.RentAdReviewsRepo.GetRentAdReviewsByRentID(ctx, rentAdID)
}

func (s *RentAdReviewService) UpdateRentAdReview(ctx context.Context, review models.RentAdReviews) error {
	return s.RentAdReviewsRepo.UpdateRentAdReview(ctx, review)
}

func (s *RentAdReviewService) DeleteRentAdReview(ctx context.Context, id int) error {
	return s.RentAdReviewsRepo.DeleteRentAdReview(ctx, id)
}
