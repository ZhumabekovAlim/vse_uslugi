package services

import (
	"context"

	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type RentComplaintService struct {
	ComplaintRepo *repositories.RentComplaintRepository
}

func (s *RentComplaintService) CreateRentComplaint(ctx context.Context, c models.RentComplaint) error {
	return s.ComplaintRepo.CreateRentComplaint(ctx, c)
}

func (s *RentComplaintService) GetComplaintsByRentID(ctx context.Context, rentID int) ([]models.RentComplaint, error) {
	return s.ComplaintRepo.GetComplaintsByRentID(ctx, rentID)
}

func (s *RentComplaintService) DeleteRentComplaintByID(ctx context.Context, id int) error {
	return s.ComplaintRepo.DeleteRentComplaintByID(ctx, id)
}

func (s *RentComplaintService) GetAllRentComplaints(ctx context.Context) ([]models.RentComplaint, error) {
	return s.ComplaintRepo.GetAllRentComplaints(ctx)
}
