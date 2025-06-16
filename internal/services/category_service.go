package services

import (
	"context"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
)

type CategoryService struct {
	CategoryRepo *repositories.CategoryRepository
}

func (s *CategoryService) CreateCategory(ctx context.Context, category models.Category) (models.Category, error) {
	return s.CategoryRepo.CreateCategory(ctx, category, category.SubcategoryIDs)
}

func (s *CategoryService) GetCategoryByID(ctx context.Context, id int) (models.Category, error) {
	return s.CategoryRepo.GetCategoryByID(ctx, id)
}

func (s *CategoryService) UpdateCategory(ctx context.Context, category models.Category) (models.Category, error) {
	return s.CategoryRepo.UpdateCategory(ctx, category)
}

func (s *CategoryService) DeleteCategory(ctx context.Context, id int) error {
	return s.CategoryRepo.DeleteCategory(ctx, id)
}

func (s *CategoryService) GetAllCategories(ctx context.Context) ([]models.Category, error) {
	return s.CategoryRepo.GetAllCategories(ctx)
}
