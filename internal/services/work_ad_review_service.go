package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type WorkAdReviewService struct {
	WorkAdReviewsRepo *repositories.WorkAdReviewRepository
}

func (s *WorkAdReviewService) CreateWorkAdReview(ctx context.Context, review models.WorkAdReviews) (models.WorkAdReviews, error) {
	return s.WorkAdReviewsRepo.CreateWorkAdReview(ctx, review)
}

func (s *WorkAdReviewService) GetWorkAdReviewsByWorkID(ctx context.Context, workAdID int) ([]models.WorkAdReviews, error) {
	return s.WorkAdReviewsRepo.GetWorkAdReviewsByWorkID(ctx, workAdID)
}

func (s *WorkAdReviewService) UpdateWorkAdReview(ctx context.Context, review models.WorkAdReviews) error {
	return s.WorkAdReviewsRepo.UpdateWorkAdReview(ctx, review)
}

func (s *WorkAdReviewService) DeleteWorkAdReview(ctx context.Context, id int) error {
	return s.WorkAdReviewsRepo.DeleteWorkAdReview(ctx, id)
}
