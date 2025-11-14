package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type WorkAdService struct {
	WorkAdRepo       *repositories.WorkAdRepository
	SubscriptionRepo *repositories.SubscriptionRepository
}

func (s *WorkAdService) CreateWorkAd(ctx context.Context, work models.WorkAd) (models.WorkAd, error) {
	return s.WorkAdRepo.CreateWorkAd(ctx, work)
}

func (s *WorkAdService) GetWorkAdByID(ctx context.Context, id int, userID int) (models.WorkAd, error) {
	return s.WorkAdRepo.GetWorkAdByID(ctx, id, userID)
}

func (s *WorkAdService) UpdateWorkAd(ctx context.Context, work models.WorkAd) (models.WorkAd, error) {
	if work.Status == "active" {
		existing, err := s.WorkAdRepo.GetWorkAdByID(ctx, work.ID, 0)
		if err != nil {
			return models.WorkAd{}, err
		}
		if existing.Status != "active" {
			has, err := s.SubscriptionRepo.HasActiveSubscription(ctx, work.UserID, models.SubscriptionTypeWork)
			if err != nil {
				return models.WorkAd{}, err
			}
			if !has {
				return models.WorkAd{}, ErrNoActiveSubscription
			}
		}
	}
	return s.WorkAdRepo.UpdateWorkAd(ctx, work)
}

func (s *WorkAdService) DeleteWorkAd(ctx context.Context, id int) error {
	return s.WorkAdRepo.DeleteWorkAd(ctx, id)
}

func (s *WorkAdService) ArchiveWorkAd(ctx context.Context, id int, archive bool) error {
	status := "archive"
	if !archive {
		status = "active"
	}
	return s.WorkAdRepo.UpdateStatus(ctx, id, status)
}

func (s *WorkAdService) GetFilteredWorksAd(ctx context.Context, filter models.WorkAdFilterRequest, userID int, cityID int) (models.WorkAdListResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit

	worksad, minPrice, maxPrice, err := s.WorkAdRepo.GetWorksAdWithFilters(
		ctx,
		userID,
		cityID,
		filter.Categories,
		filter.Subcategories,
		filter.PriceFrom,
		filter.PriceTo,
		filter.Ratings,
		filter.SortOption,
		filter.Limit,
		offset,
	)
	if err != nil {
		return models.WorkAdListResponse{}, err
	}

	return models.WorkAdListResponse{
		WorksAd:  worksad,
		MinPrice: minPrice,
		MaxPrice: maxPrice,
	}, nil
}

func (s *WorkAdService) GetWorksAdByUserID(ctx context.Context, userID int) ([]models.WorkAd, error) {
	return s.WorkAdRepo.GetWorksAdByUserID(ctx, userID)
}

func (s *WorkAdService) GetFilteredWorksAdPost(ctx context.Context, req models.FilterWorkAdRequest) ([]models.FilteredWorkAd, error) {
	return s.WorkAdRepo.GetFilteredWorksAdPost(ctx, req)
}

func (s *WorkAdService) GetWorksAdByStatusAndUserID(ctx context.Context, userID int, status string) ([]models.WorkAd, error) {
	return s.WorkAdRepo.FetchByStatusAndUserID(ctx, userID, status)
}

func (s *WorkAdService) GetFilteredWorksAdWithLikes(ctx context.Context, req models.FilterWorkAdRequest, userID int) ([]models.FilteredWorkAd, error) {
	return s.WorkAdRepo.GetFilteredWorksAdWithLikes(ctx, req, userID)
}

func (s *WorkAdService) GetWorkAdByWorkIDAndUserID(ctx context.Context, workadID int, userID int) (models.WorkAd, error) {
	return s.WorkAdRepo.GetWorkAdByWorkIDAndUserID(ctx, workadID, userID)
}
