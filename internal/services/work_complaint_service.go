package services

import (
	"context"

	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type WorkComplaintService struct {
	ComplaintRepo *repositories.WorkComplaintRepository
}

func (s *WorkComplaintService) CreateWorkComplaint(ctx context.Context, c models.WorkComplaint) error {
	return s.ComplaintRepo.CreateWorkComplaint(ctx, c)
}

func (s *WorkComplaintService) GetComplaintsByWorkID(ctx context.Context, workID int) ([]models.WorkComplaint, error) {
	return s.ComplaintRepo.GetComplaintsByWorkID(ctx, workID)
}

func (s *WorkComplaintService) DeleteWorkComplaintByID(ctx context.Context, id int) error {
	return s.ComplaintRepo.DeleteWorkComplaintByID(ctx, id)
}

func (s *WorkComplaintService) GetAllWorkComplaints(ctx context.Context) ([]models.WorkComplaint, error) {
	return s.ComplaintRepo.GetAllWorkComplaints(ctx)
}
