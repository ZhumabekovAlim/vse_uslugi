package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type WorkResponseService struct {
	WorkResponseRepo *repositories.WorkResponseRepository
}

func (s *WorkResponseService) CreateWorkResponse(ctx context.Context, resp models.WorkResponses) (models.WorkResponses, error) {
	return s.WorkResponseRepo.CreateWorkResponse(ctx, resp)
}
