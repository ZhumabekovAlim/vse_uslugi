package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type ServiceService struct {
	ServiceRepo      *repositories.ServiceRepository
	SubscriptionRepo *repositories.SubscriptionRepository
}

func (s *ServiceService) CreateService(ctx context.Context, service models.Service) (models.Service, error) {
	has, err := s.SubscriptionRepo.HasActiveSubscription(ctx, service.UserID, models.SubscriptionTypeService)
	if err != nil {
		return models.Service{}, err
	}
	if !has {
		return models.Service{}, ErrNoActiveSubscription
	}
	return s.ServiceRepo.CreateService(ctx, service)
}

func (s *ServiceService) GetServiceByID(ctx context.Context, id int, userID int) (models.Service, error) {
	return s.ServiceRepo.GetServiceByID(ctx, id, userID)
}

func (s *ServiceService) GetServiceByIDWithCity(ctx context.Context, id int, userID int, cityID int) (models.Service, error) {
	return s.ServiceRepo.GetServiceByIDWithCity(ctx, id, userID, cityID)
}

func (s *ServiceService) UpdateService(ctx context.Context, service models.Service) (models.Service, error) {
	if service.Status == "active" {
		existing, err := s.ServiceRepo.GetServiceByID(ctx, service.ID, 0)
		if err != nil {
			return models.Service{}, err
		}
		if existing.Status != "active" {
			has, err := s.SubscriptionRepo.HasActiveSubscription(ctx, service.UserID, models.SubscriptionTypeService)
			if err != nil {
				return models.Service{}, err
			}
			if !has {
				return models.Service{}, ErrNoActiveSubscription
			}
		}
	}
	return s.ServiceRepo.UpdateService(ctx, service)
}

func (s *ServiceService) DeleteService(ctx context.Context, id int) error {
	return s.ServiceRepo.DeleteService(ctx, id)
}

func (s *ServiceService) ArchiveService(ctx context.Context, id int, archive bool) error {
	status := "archive"
	if !archive {
		service, err := s.ServiceRepo.GetServiceByID(ctx, id, 0)
		if err != nil {
			return err
		}
		has, err := s.SubscriptionRepo.HasActiveSubscription(ctx, service.UserID, models.SubscriptionTypeService)
		if err != nil {
			return err
		}
		if !has {
			return ErrNoActiveSubscription
		}
		status = "active"
	}
	return s.ServiceRepo.UpdateStatus(ctx, id, status)
}

func (s *ServiceService) GetFilteredServices(ctx context.Context, filter models.ServiceFilterRequest, userID int, cityID int) (models.ServiceListResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}
	offset := (filter.Page - 1) * filter.Limit

	services, minPrice, maxPrice, err := s.ServiceRepo.GetServicesWithFilters(
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
	)
	if err != nil {
		return models.ServiceListResponse{}, err
	}

	return models.ServiceListResponse{
		Services: services,
		MinPrice: minPrice,
		MaxPrice: maxPrice,
	}, nil
}

func (s *ServiceService) GetServicesByUserID(ctx context.Context, userID int) ([]models.Service, error) {
	return s.ServiceRepo.GetServicesByUserID(ctx, userID)
}

func (s *ServiceService) GetFilteredServicesPost(ctx context.Context, req models.FilterServicesRequest) ([]models.FilteredService, error) {
	return s.ServiceRepo.GetFilteredServicesPost(ctx, req)
}

func (s *ServiceService) GetServicesByStatusAndUserID(ctx context.Context, userID int, status string) ([]models.Service, error) {
	return s.ServiceRepo.FetchByStatusAndUserID(ctx, userID, status)
}

func (s *ServiceService) GetFilteredServicesWithLikes(ctx context.Context, req models.FilterServicesRequest, userID int) ([]models.FilteredService, error) {
	return s.ServiceRepo.GetFilteredServicesWithLikes(ctx, req, userID)
}

func (s *ServiceService) GetServiceByServiceIDAndUserID(ctx context.Context, serviceID int, userID int) (models.Service, error) {
	return s.ServiceRepo.GetServiceByServiceIDAndUserID(ctx, serviceID, userID)
}
