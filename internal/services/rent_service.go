package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type RentService struct {
	RentRepo         *repositories.RentRepository
	SubscriptionRepo *repositories.SubscriptionRepository
}

func (s *RentService) CreateRent(ctx context.Context, work models.Rent) (models.Rent, error) {
	has, err := s.SubscriptionRepo.HasActiveSubscription(ctx, work.UserID, models.SubscriptionTypeRent)
	if err != nil {
		return models.Rent{}, err
	}
	if !has {
		return models.Rent{}, ErrNoActiveSubscriptionRent
	}
	return s.RentRepo.CreateRent(ctx, work)
}

func (s *RentService) GetRentByID(ctx context.Context, id int, userID int) (models.Rent, error) {
	return s.RentRepo.GetRentByID(ctx, id, userID)
}

func (s *RentService) UpdateRent(ctx context.Context, work models.Rent) (models.Rent, error) {
	if work.Status == "active" {
		existing, err := s.RentRepo.GetRentByID(ctx, work.ID, 0)
		if err != nil {
			return models.Rent{}, err
		}
		if existing.Status != "active" {
			has, err := s.SubscriptionRepo.HasActiveSubscription(ctx, work.UserID, models.SubscriptionTypeRent)
			if err != nil {
				return models.Rent{}, err
			}
			if !has {
				return models.Rent{}, ErrNoActiveSubscription
			}
		}
	}
	return s.RentRepo.UpdateRent(ctx, work)
}

func (s *RentService) DeleteRent(ctx context.Context, id int) error {
	return s.RentRepo.DeleteRent(ctx, id)
}

func (s *RentService) ArchiveRent(ctx context.Context, id int, archive bool) error {
	status := "archive"
	if !archive {
		rent, err := s.RentRepo.GetRentByID(ctx, id, 0)
		if err != nil {
			return err
		}
		has, err := s.SubscriptionRepo.HasActiveSubscription(ctx, rent.UserID, models.SubscriptionTypeRent)
		if err != nil {
			return err
		}
		if !has {
			return ErrNoActiveSubscriptionRent
		}
		status = "active"
	}
	return s.RentRepo.UpdateStatus(ctx, id, status)
}

func (s *RentService) GetFilteredRents(ctx context.Context, filter models.RentFilterRequest, userID int, cityID int) (models.RentListResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit

	rents, minPrice, maxPrice, err := s.RentRepo.GetRentsWithFilters(
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
	)
	if err != nil {
		return models.RentListResponse{}, err
	}

	return models.RentListResponse{
		Rents:    rents,
		MinPrice: minPrice,
		MaxPrice: maxPrice,
	}, nil
}

func (s *RentService) GetRentsByUserID(ctx context.Context, userID int) ([]models.Rent, error) {
	return s.RentRepo.GetRentsByUserID(ctx, userID)
}

func (s *RentService) GetFilteredRentsPost(ctx context.Context, req models.FilterRentRequest) ([]models.FilteredRent, error) {
	return s.RentRepo.GetFilteredRentsPost(ctx, req)
}

func (s *RentService) GetRentsByStatusAndUserID(ctx context.Context, userID int, status string) ([]models.Rent, error) {
	return s.RentRepo.FetchByStatusAndUserID(ctx, userID, status)
}

func (s *RentService) GetFilteredRentsWithLikes(ctx context.Context, req models.FilterRentRequest, userID int) ([]models.FilteredRent, error) {
	return s.RentRepo.GetFilteredRentsWithLikes(ctx, req, userID)
}

func (s *RentService) GetRentByRentIDAndUserID(ctx context.Context, rentID int, userID int) (models.Rent, error) {
	return s.RentRepo.GetRentByRentIDAndUserID(ctx, rentID, userID)
}
