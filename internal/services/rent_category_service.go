package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type RentCategoryService struct {
	CategoryRepo *repositories.RentCategoryRepository
}

func (s *RentCategoryService) CreateCategory(ctx context.Context, category models.RentCategory) (models.RentCategory, error) {
	return s.CategoryRepo.CreateCategory(ctx, category)
}

func (s *RentCategoryService) GetCategoryByID(ctx context.Context, id int) (models.RentCategory, error) {
	return s.CategoryRepo.GetCategoryByID(ctx, id)
}

func (s *RentCategoryService) UpdateCategory(ctx context.Context, category models.RentCategory) (models.RentCategory, error) {
	return s.CategoryRepo.UpdateCategory(ctx, category)
}

func (s *RentCategoryService) DeleteCategory(ctx context.Context, id int) error {
	return s.CategoryRepo.DeleteCategory(ctx, id)
}

func (s *RentCategoryService) GetAllCategories(ctx context.Context) ([]models.RentCategory, error) {
	return s.CategoryRepo.GetAllCategories(ctx)
}
