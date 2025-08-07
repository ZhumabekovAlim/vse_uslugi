package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type RentSubcategoryService struct {
	SubcategoryRepo *repositories.RentSubcategoryRepository
}

func (s *RentSubcategoryService) CreateSubcategory(ctx context.Context, sub models.RentSubcategory) (models.RentSubcategory, error) {
	return s.SubcategoryRepo.CreateSubcategory(ctx, sub)
}

func (s *RentSubcategoryService) GetAllSubcategories(ctx context.Context) ([]models.RentSubcategory, error) {
	return s.SubcategoryRepo.GetAllSubcategories(ctx)
}

func (s *RentSubcategoryService) GetByCategoryID(ctx context.Context, catID int) ([]models.RentSubcategory, error) {
	return s.SubcategoryRepo.GetByCategoryID(ctx, catID)
}

func (s *RentSubcategoryService) GetSubcategoryByID(ctx context.Context, id int) (models.RentSubcategory, error) {
	return s.SubcategoryRepo.GetSubcategoryByID(ctx, id)
}

func (s *RentSubcategoryService) UpdateSubcategoryByID(ctx context.Context, sub models.RentSubcategory) (models.RentSubcategory, error) {
	return s.SubcategoryRepo.UpdateSubcategoryByID(ctx, sub)
}

func (s *RentSubcategoryService) DeleteSubcategoryByID(ctx context.Context, id int) error {
	return s.SubcategoryRepo.DeleteSubcategoryByID(ctx, id)
}
