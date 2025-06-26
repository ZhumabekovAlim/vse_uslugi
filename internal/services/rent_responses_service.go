package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type RentResponseService struct {
	RentResponseRepo *repositories.RentResponseRepository
}

func (s *RentResponseService) CreateRentResponse(ctx context.Context, resp models.RentResponses) (models.RentResponses, error) {
	return s.RentResponseRepo.CreateRentResponse(ctx, resp)
}
