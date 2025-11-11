package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"naimuBack/internal/models"
	"time"
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

func (r *TopRepository) ClearExpiredTop(ctx context.Context, now time.Time) (int, error) {
	if r == nil || r.DB == nil {
		return 0, nil
	}
	now = now.UTC()
	tables := models.AllowedTopTypes()
	seen := make(map[string]struct{}, len(tables))
	totalCleared := 0
	for _, table := range tables {
		if _, processed := seen[table]; processed {
			continue
		}
		cleared, err := r.clearExpiredTopForTable(ctx, table, now)
		if err != nil {
			return totalCleared, err
		}
		totalCleared += cleared
		seen[table] = struct{}{}
	}
	return totalCleared, nil
}

func (r *TopRepository) clearExpiredTopForTable(ctx context.Context, table string, now time.Time) (int, error) {
	query := fmt.Sprintf("SELECT id, top FROM %s WHERE top IS NOT NULL AND top <> ''", table)
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	cleared := 0
	for rows.Next() {
		var (
			id  int
			top sql.NullString
		)
		if err := rows.Scan(&id, &top); err != nil {
			return cleared, err
		}
		if !top.Valid {
			continue
		}
		info, err := models.ParseTopInfo(top.String)
		if err != nil || info == nil {
			continue
		}
		if info.IsActive(now) {
			continue
		}
		updateQuery := fmt.Sprintf("UPDATE %s SET top = NULL, updated_at = NOW() WHERE id = ?", table)
		if _, err := r.DB.ExecContext(ctx, updateQuery, id); err != nil {
			return cleared, err
		}
		cleared++
	}
	if err := rows.Err(); err != nil {
		return cleared, err
	}
	return cleared, nil
}
