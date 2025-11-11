package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"naimuBack/internal/models"
)

var (
	ErrListingNotFound = errors.New("listing not found")
)

type TopRepository struct {
	DB *sql.DB
}

func NewTopRepository(db *sql.DB) *TopRepository {
	return &TopRepository{DB: db}
}

func (r *TopRepository) UpdateTop(ctx context.Context, listingType string, listingID int, info models.TopInfo) error {
	table, ok := models.ResolveTopTable(listingType)
	if !ok {
		return fmt.Errorf("unsupported listing type: %s", listingType)
	}
	payload, err := info.Marshal()
	if err != nil {
		return err
	}
	query := fmt.Sprintf("UPDATE %s SET top = ?, updated_at = NOW() WHERE id = ?", table)
	result, err := r.DB.ExecContext(ctx, query, payload, listingID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrListingNotFound
	}
	return nil
}

func (r *TopRepository) GetOwnerID(ctx context.Context, listingType string, listingID int) (int, error) {
	table, ok := models.ResolveTopTable(listingType)
	if !ok {
		return 0, fmt.Errorf("unsupported listing type: %s", listingType)
	}
	query := fmt.Sprintf("SELECT user_id FROM %s WHERE id = ?", table)
	var ownerID int
	err := r.DB.QueryRowContext(ctx, query, listingID).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrListingNotFound
	}
	if err != nil {
		return 0, err
	}
	return ownerID, nil
}
