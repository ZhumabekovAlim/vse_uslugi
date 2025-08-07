package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type WorkCategoryService struct {
	CategoryRepo *repositories.WorkCategoryRepository
}

func (s *WorkCategoryService) CreateCategory(ctx context.Context, category models.WorkCategory) (models.WorkCategory, error) {
	return s.CategoryRepo.CreateCategory(ctx, category)
}

func (s *WorkCategoryService) GetCategoryByID(ctx context.Context, id int) (models.WorkCategory, error) {
	return s.CategoryRepo.GetCategoryByID(ctx, id)
}

func (s *WorkCategoryService) UpdateCategory(ctx context.Context, category models.WorkCategory) (models.WorkCategory, error) {
	return s.CategoryRepo.UpdateCategory(ctx, category)
}

func (s *WorkCategoryService) DeleteCategory(ctx context.Context, id int) error {
	return s.CategoryRepo.DeleteCategory(ctx, id)
}

func (s *WorkCategoryService) GetAllCategories(ctx context.Context) ([]models.WorkCategory, error) {
	return s.CategoryRepo.GetAllCategories(ctx)
}
