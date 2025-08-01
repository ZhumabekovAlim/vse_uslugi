package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type RentAdService struct {
	RentAdRepo *repositories.RentAdRepository
}

func (s *RentAdService) CreateRentAd(ctx context.Context, work models.RentAd) (models.RentAd, error) {
	return s.RentAdRepo.CreateRentAd(ctx, work)
}

func (s *RentAdService) GetRentAdByID(ctx context.Context, id int) (models.RentAd, error) {
	return s.RentAdRepo.GetRentAdByID(ctx, id)
}

func (s *RentAdService) UpdateRentAd(ctx context.Context, work models.RentAd) (models.RentAd, error) {
	return s.RentAdRepo.UpdateRentAd(ctx, work)
}

func (s *RentAdService) DeleteRentAd(ctx context.Context, id int) error {
	return s.RentAdRepo.DeleteRentAd(ctx, id)
}

func (s *RentAdService) GetFilteredRentsAd(ctx context.Context, filter models.RentAdFilterRequest, userID int) (models.RentAdListResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit

	rents_ad, minPrice, maxPrice, err := s.RentAdRepo.GetRentsAdWithFilters(
		ctx,
		userID,
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
		return models.RentAdListResponse{}, err
	}

	return models.RentAdListResponse{
		RentsAd:  rents_ad,
		MinPrice: minPrice,
		MaxPrice: maxPrice,
	}, nil
}

func (s *RentAdService) GetRentsAdByUserID(ctx context.Context, userID int) ([]models.RentAd, error) {
	return s.RentAdRepo.GetRentsAdByUserID(ctx, userID)
}

func (s *RentAdService) GetFilteredRentsAdPost(ctx context.Context, req models.FilterRentAdRequest) ([]models.FilteredRentAd, error) {
	return s.RentAdRepo.GetFilteredRentsAdPost(ctx, req)
}

func (s *RentAdService) GetRentsAdByStatusAndUserID(ctx context.Context, userID int, status string) ([]models.RentAd, error) {
	return s.RentAdRepo.FetchByStatusAndUserID(ctx, userID, status)
}

func (s *RentAdService) GetFilteredRentsAdWithLikes(ctx context.Context, req models.FilterRentAdRequest, userID int) ([]models.FilteredRentAd, error) {
	return s.RentAdRepo.GetFilteredRentsAdWithLikes(ctx, req, userID)
}

func (s *RentAdService) GetRentAdByRentIDAndUserID(ctx context.Context, rentAdID int, userID int) (models.RentAd, error) {
	return s.RentAdRepo.GetRentAdByRentIDAndUserID(ctx, rentAdID, userID)
}
