package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type SubcategoryService struct {
	SubcategoryRepo *repositories.SubcategoryRepository
}

func (s *SubcategoryService) CreateSubcategory(ctx context.Context, sub models.Subcategory) (models.Subcategory, error) {
	return s.SubcategoryRepo.CreateSubcategory(ctx, sub)
}

func (s *SubcategoryService) GetAllSubcategories(ctx context.Context) ([]models.Subcategory, error) {
	return s.SubcategoryRepo.GetAllSubcategories(ctx)
}

func (s *SubcategoryService) GetByCategoryID(ctx context.Context, catID int) ([]models.Subcategory, error) {
	return s.SubcategoryRepo.GetByCategoryID(ctx, catID)
}

func (s *SubcategoryService) GetSubcategoryByID(ctx context.Context, id int) (models.Subcategory, error) {
	return s.SubcategoryRepo.GetSubcategoryByID(ctx, id)
}

func (s *SubcategoryService) UpdateSubcategoryByID(ctx context.Context, sub models.Subcategory) (models.Subcategory, error) {
	return s.SubcategoryRepo.UpdateSubcategoryByID(ctx, sub)
}

func (s *SubcategoryService) DeleteSubcategoryByID(ctx context.Context, id int) error {
	return s.SubcategoryRepo.DeleteSubcategoryByID(ctx, id)
}
