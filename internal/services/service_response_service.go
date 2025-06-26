package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type ServiceResponseService struct {
	ServiceResponseRepo *repositories.ServiceResponseRepository
}

func (s *ServiceResponseService) CreateServiceResponse(ctx context.Context, resp models.ServiceResponses) (models.ServiceResponses, error) {
	return s.ServiceResponseRepo.CreateResponse(ctx, resp)
}
