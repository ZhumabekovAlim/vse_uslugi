package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type ComplaintService struct {
	ComplaintRepo *repositories.ComplaintRepository
}

func (s *ComplaintService) CreateComplaint(ctx context.Context, c models.Complaint) error {
	return s.ComplaintRepo.CreateComplaint(ctx, c)
}

func (s *ComplaintService) GetComplaintsByServiceID(ctx context.Context, serviceID int) ([]models.Complaint, error) {
	return s.ComplaintRepo.GetComplaintsByServiceID(ctx, serviceID)
}

func (s *ComplaintService) DeleteComplaintByID(ctx context.Context, id int) error {
	return s.ComplaintRepo.DeleteComplaintByID(ctx, id)
}

func (s *ComplaintService) GetAllComplaints(ctx context.Context) ([]models.Complaint, error) {
	return s.ComplaintRepo.GetAllComplaints(ctx)
}
