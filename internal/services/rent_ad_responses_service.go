package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type RentAdResponseService struct {
	RentAdResponseRepo *repositories.RentAdResponseRepository
}

func (s *RentAdResponseService) CreateRentAdResponse(ctx context.Context, resp models.RentAdResponses) (models.RentAdResponses, error) {
	return s.RentAdResponseRepo.CreateRentAdResponse(ctx, resp)
}
