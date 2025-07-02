package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type AdService struct {
	AdRepo *repositories.AdRepository
}

func (s *AdService) CreateAd(ctx context.Context, ad models.Ad) (models.Ad, error) {
	return s.AdRepo.CreateAd(ctx, ad)
}

func (s *AdService) GetAdByID(ctx context.Context, id int) (models.Ad, error) {
	return s.AdRepo.GetAdByID(ctx, id)
}

func (s *AdService) UpdateAd(ctx context.Context, service models.Ad) (models.Ad, error) {
	return s.AdRepo.UpdateAd(ctx, service)
}

func (s *AdService) DeleteAd(ctx context.Context, id int) error {
	return s.AdRepo.DeleteAd(ctx, id)
}

func (s *AdService) GetFilteredAd(ctx context.Context, filter models.AdFilterRequest, userID int) (models.AdListResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit

	ads, minPrice, maxPrice, err := s.AdRepo.GetAdWithFilters(
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
		return models.AdListResponse{}, err
	}

	return models.AdListResponse{
		Ads:      ads,
		MinPrice: minPrice,
		MaxPrice: maxPrice,
	}, nil
}

func (s *AdService) GetAdByUserID(ctx context.Context, userID int) ([]models.Ad, error) {
	return s.AdRepo.GetAdByUserID(ctx, userID)
}

func (s *AdService) GetFilteredAdPost(ctx context.Context, req models.FilterAdRequest) ([]models.FilteredAd, error) {
	return s.AdRepo.GetFilteredAdPost(ctx, req)
}

func (s *AdService) GetAdByStatusAndUserID(ctx context.Context, userID int, status string) ([]models.Ad, error) {
	return s.AdRepo.FetchAdByStatusAndUserID(ctx, userID, status)
}

func (s *AdService) GetFilteredAdWithLikes(ctx context.Context, req models.FilterAdRequest, userID int) ([]models.FilteredAd, error) {
	return s.AdRepo.GetFilteredAdWithLikes(ctx, req, userID)
}

func (s *AdService) GetAdByAdIDAndUserID(ctx context.Context, adID int, userID int) (models.Ad, error) {
	return s.AdRepo.GetAdByAdIDAndUserID(ctx, adID, userID)
}
