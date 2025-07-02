package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type AdResponseService struct {
	AdResponseRepo *repositories.AdResponseRepository
}

func (s *AdResponseService) CreateAdResponse(ctx context.Context, resp models.AdResponses) (models.AdResponses, error) {
	return s.AdResponseRepo.CreateAdResponse(ctx, resp)
}
