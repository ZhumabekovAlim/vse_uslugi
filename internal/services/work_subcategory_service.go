package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type WorkSubcategoryService struct {
	SubcategoryRepo *repositories.WorkSubcategoryRepository
}

func (s *WorkSubcategoryService) CreateSubcategory(ctx context.Context, sub models.WorkSubcategory) (models.WorkSubcategory, error) {
	return s.SubcategoryRepo.CreateSubcategory(ctx, sub)
}

func (s *WorkSubcategoryService) GetAllSubcategories(ctx context.Context) ([]models.WorkSubcategory, error) {
	return s.SubcategoryRepo.GetAllSubcategories(ctx)
}

func (s *WorkSubcategoryService) GetByCategoryID(ctx context.Context, catID int) ([]models.WorkSubcategory, error) {
	return s.SubcategoryRepo.GetByCategoryID(ctx, catID)
}

func (s *WorkSubcategoryService) GetSubcategoryByID(ctx context.Context, id int) (models.WorkSubcategory, error) {
	return s.SubcategoryRepo.GetSubcategoryByID(ctx, id)
}

func (s *WorkSubcategoryService) UpdateSubcategoryByID(ctx context.Context, sub models.WorkSubcategory) (models.WorkSubcategory, error) {
	return s.SubcategoryRepo.UpdateSubcategoryByID(ctx, sub)
}

func (s *WorkSubcategoryService) DeleteSubcategoryByID(ctx context.Context, id int) error {
	return s.SubcategoryRepo.DeleteSubcategoryByID(ctx, id)
}
