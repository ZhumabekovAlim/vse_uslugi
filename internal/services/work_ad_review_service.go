package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type WorkAdReviewService struct {
	WorkAdReviewsRepo *repositories.WorkAdReviewRepository
	ConfirmationRepo  *repositories.WorkAdConfirmationRepository
}

func (s *WorkAdReviewService) CreateWorkAdReview(ctx context.Context, review models.WorkAdReviews) (models.WorkAdReviews, error) {
	rev, err := s.WorkAdReviewsRepo.CreateWorkAdReview(ctx, review)
	if err != nil {
		return rev, err
	}
	if s.ConfirmationRepo != nil && review.WorkAdID != 0 {
		if err := s.ConfirmationRepo.Done(ctx, review.WorkAdID); err != nil {
			return rev, err
		}
	}
	return rev, nil
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
