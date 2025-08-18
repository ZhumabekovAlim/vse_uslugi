package services

import (
	"context"

	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type AdComplaintService struct {
	ComplaintRepo *repositories.AdComplaintRepository
}

func (s *AdComplaintService) CreateAdComplaint(ctx context.Context, c models.AdComplaint) error {
	return s.ComplaintRepo.CreateAdComplaint(ctx, c)
}

func (s *AdComplaintService) GetComplaintsByAdID(ctx context.Context, adID int) ([]models.AdComplaint, error) {
	return s.ComplaintRepo.GetComplaintsByAdID(ctx, adID)
}

func (s *AdComplaintService) DeleteAdComplaintByID(ctx context.Context, id int) error {
	return s.ComplaintRepo.DeleteAdComplaintByID(ctx, id)
}

func (s *AdComplaintService) GetAllAdComplaints(ctx context.Context) ([]models.AdComplaint, error) {
	return s.ComplaintRepo.GetAllAdComplaints(ctx)
}
