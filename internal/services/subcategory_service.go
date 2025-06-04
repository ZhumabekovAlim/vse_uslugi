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
