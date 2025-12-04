package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type WorkService struct {
	WorkRepo         *repositories.WorkRepository
	SubscriptionRepo *repositories.SubscriptionRepository
}

func (s *WorkService) CreateWork(ctx context.Context, work models.Work) (models.Work, error) {
	has, err := s.SubscriptionRepo.HasActiveSubscription(ctx, work.UserID, models.SubscriptionTypeWork)
	if err != nil {
		return models.Work{}, err
	}
	if !has {
		return models.Work{}, ErrNoActiveSubscriptionWork
	}
	return s.WorkRepo.CreateWork(ctx, work)
}

func (s *WorkService) GetWorkByID(ctx context.Context, id int, userID int) (models.Work, error) {
	return s.WorkRepo.GetWorkByID(ctx, id, userID)
}

func (s *WorkService) UpdateWork(ctx context.Context, work models.Work) (models.Work, error) {
	if work.Status == "active" {
		existing, err := s.WorkRepo.GetWorkByID(ctx, work.ID, 0)
		if err != nil {
			return models.Work{}, err
		}
		if existing.Status != "active" {
			has, err := s.SubscriptionRepo.HasActiveSubscription(ctx, work.UserID, models.SubscriptionTypeWork)
			if err != nil {
				return models.Work{}, err
			}
			if !has {
				return models.Work{}, ErrNoActiveSubscription
			}
		}
	}
	return s.WorkRepo.UpdateWork(ctx, work)
}

func (s *WorkService) DeleteWork(ctx context.Context, id int) error {
	return s.WorkRepo.DeleteWork(ctx, id)
}

func (s *WorkService) ArchiveWork(ctx context.Context, id int, archive bool) error {
	status := "archive"
	if !archive {
		work, err := s.WorkRepo.GetWorkByID(ctx, id, 0)
		if err != nil {
			return err
		}
		has, err := s.SubscriptionRepo.HasActiveSubscription(ctx, work.UserID, models.SubscriptionTypeWork)
		if err != nil {
			return err
		}
		if !has {
			return ErrNoActiveSubscriptionWork
		}
		status = "active"
	}
	return s.WorkRepo.UpdateStatus(ctx, id, status)
}

func (s *WorkService) GetFilteredWorks(ctx context.Context, filter models.WorkFilterRequest, userID int, cityID int) (models.WorkListResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit

	works, minPrice, maxPrice, err := s.WorkRepo.GetWorksWithFilters(
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
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
	if err != nil {
		return models.WorkListResponse{}, err
	}

	return models.WorkListResponse{
		Works:    works,
		MinPrice: minPrice,
		MaxPrice: maxPrice,
	}, nil
}

func (s *WorkService) GetWorksByUserID(ctx context.Context, userID int) ([]models.Work, error) {
	return s.WorkRepo.GetWorksByUserID(ctx, userID)
}

func (s *WorkService) GetFilteredWorksPost(ctx context.Context, req models.FilterWorkRequest) ([]models.FilteredWork, error) {
	return s.WorkRepo.GetFilteredWorksPost(ctx, req)
}

func (s *WorkService) GetWorksByStatusAndUserID(ctx context.Context, userID int, status string) ([]models.Work, error) {
	return s.WorkRepo.FetchByStatusAndUserID(ctx, userID, status)
}

func (s *WorkService) GetFilteredWorksWithLikes(ctx context.Context, req models.FilterWorkRequest, userID int) ([]models.FilteredWork, error) {
	return s.WorkRepo.GetFilteredWorksWithLikes(ctx, req, userID)
}

func (s *WorkService) GetWorkByWorkIDAndUserID(ctx context.Context, workID int, userID int) (models.Work, error) {
	return s.WorkRepo.GetWorkByWorkIDAndUserID(ctx, workID, userID)
}
