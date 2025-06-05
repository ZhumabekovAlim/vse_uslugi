package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type CityService struct {
	CityRepo *repositories.CityRepository
}

func (s *CityService) CreateCity(ctx context.Context, city models.Subcategory) (models.City, error) {
	return s.CityRepo.CreateCity(ctx, city)
}

func (s *CityService) GetCities(ctx context.Context) ([]models.City, error) {
	return s.CityRepo.GetCities(ctx)
}

func (s *CityService) GetCityByID(ctx context.Context, id int) (models.City, error) {
	return s.CityRepo.GetCityByID(ctx, id)
}

func (s *CityService) UpdateCity(ctx context.Context, city models.City) (models.City, error) {
	return s.CityRepo.UpdateCity(ctx, city)
}

func (s *CityService) DeleteCity(ctx context.Context, id int) error {
	return s.CityRepo.DeleteCity(ctx, id)
}
