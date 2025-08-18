package services

import (
	"context"

	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type WorkAdComplaintService struct {
	ComplaintRepo *repositories.WorkAdComplaintRepository
}

func (s *WorkAdComplaintService) CreateWorkAdComplaint(ctx context.Context, c models.WorkAdComplaint) error {
	return s.ComplaintRepo.CreateWorkAdComplaint(ctx, c)
}

func (s *WorkAdComplaintService) GetComplaintsByWorkAdID(ctx context.Context, workAdID int) ([]models.WorkAdComplaint, error) {
	return s.ComplaintRepo.GetComplaintsByWorkAdID(ctx, workAdID)
}

func (s *WorkAdComplaintService) DeleteWorkAdComplaintByID(ctx context.Context, id int) error {
	return s.ComplaintRepo.DeleteWorkAdComplaintByID(ctx, id)
}

func (s *WorkAdComplaintService) GetAllWorkAdComplaints(ctx context.Context) ([]models.WorkAdComplaint, error) {
	return s.ComplaintRepo.GetAllWorkAdComplaints(ctx)
}
