package services

import (
	"context"

	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

// UserReviewsService provides business logic for retrieving user reviews.
type UserReviewsService struct {
	ReviewsRepo *repositories.UserReviewsRepository
}

// GetReviewsByUserID fetches all reviews made by a specific user.
func (s *UserReviewsService) GetReviewsByUserID(ctx context.Context, userID int) (models.UserReviews, error) {
	return s.ReviewsRepo.GetReviewsByUserID(ctx, userID)
}
