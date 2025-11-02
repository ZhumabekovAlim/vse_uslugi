package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

var (
	ErrTaxiIntercityOrderForbidden = errors.New("forbidden")
)

type TaxiIntercityOrderService struct {
	Repo *repositories.TaxiIntercityOrderRepository
}

func (s *TaxiIntercityOrderService) Create(ctx context.Context, clientID int, req models.CreateTaxiIntercityOrderRequest) (models.TaxiIntercityOrder, error) {
	order := models.TaxiIntercityOrder{
		ClientID: clientID,
		FromCity: strings.TrimSpace(req.FromCity),
		ToCity:   strings.TrimSpace(req.ToCity),
		TripType: strings.TrimSpace(req.TripType),
		Comment:  strings.TrimSpace(req.Comment),
		Price:    req.Price,
		Status:   "open",
	}

	departureDate, err := time.Parse("2006-01-02", strings.TrimSpace(req.DepartureDate))
	if err != nil {
		return models.TaxiIntercityOrder{}, err
	}
	order.DepartureDate = departureDate

	return s.Repo.Create(ctx, order)
}

func (s *TaxiIntercityOrderService) GetByID(ctx context.Context, id int) (models.TaxiIntercityOrder, error) {
	return s.Repo.GetByID(ctx, id)
}

func (s *TaxiIntercityOrderService) Search(ctx context.Context, filter models.TaxiIntercityOrderFilter) ([]models.TaxiIntercityOrder, error) {
	if filter.Status == "" {
		filter.Status = "open"
	}
	return s.Repo.Search(ctx, filter)
}

func (s *TaxiIntercityOrderService) ListByClient(ctx context.Context, clientID int, status string) ([]models.TaxiIntercityOrder, error) {
	return s.Repo.ListByClient(ctx, clientID, status)
}

func (s *TaxiIntercityOrderService) Close(ctx context.Context, orderID int, clientID int) error {
	order, err := s.Repo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}
	if order.ClientID != clientID {
		return ErrTaxiIntercityOrderForbidden
	}
	if order.Status == "closed" {
		return nil
	}
	return s.Repo.UpdateStatus(ctx, orderID, "closed")
}
