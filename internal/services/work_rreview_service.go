package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type WorkReviewService struct {
	WorkReviewsRepo *repositories.WorkReviewRepository
}

func (s *WorkReviewService) CreateWorkReview(ctx context.Context, review models.WorkReviews) (models.WorkReviews, error) {
	return s.WorkReviewsRepo.CreateWorkReview(ctx, review)
}

func (s *WorkReviewService) GetWorkReviewsByWorkID(ctx context.Context, workID int) ([]models.WorkReviews, error) {
	return s.WorkReviewsRepo.GetWorkReviewsByWorkID(ctx, workID)
}

func (s *WorkReviewService) UpdateWorkReview(ctx context.Context, review models.WorkReviews) error {
	return s.WorkReviewsRepo.UpdateWorkReview(ctx, review)
}

func (s *WorkReviewService) DeleteWorkReview(ctx context.Context, id int) error {
	return s.WorkReviewsRepo.DeleteWorkReview(ctx, id)
}
