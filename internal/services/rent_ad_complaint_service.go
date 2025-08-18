package services

import (
	"context"

	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type RentAdComplaintService struct {
	ComplaintRepo *repositories.RentAdComplaintRepository
}

func (s *RentAdComplaintService) CreateRentAdComplaint(ctx context.Context, c models.RentAdComplaint) error {
	return s.ComplaintRepo.CreateRentAdComplaint(ctx, c)
}

func (s *RentAdComplaintService) GetComplaintsByRentAdID(ctx context.Context, rentAdID int) ([]models.RentAdComplaint, error) {
	return s.ComplaintRepo.GetComplaintsByRentAdID(ctx, rentAdID)
}

func (s *RentAdComplaintService) DeleteRentAdComplaintByID(ctx context.Context, id int) error {
	return s.ComplaintRepo.DeleteRentAdComplaintByID(ctx, id)
}

func (s *RentAdComplaintService) GetAllRentAdComplaints(ctx context.Context) ([]models.RentAdComplaint, error) {
	return s.ComplaintRepo.GetAllRentAdComplaints(ctx)
}
