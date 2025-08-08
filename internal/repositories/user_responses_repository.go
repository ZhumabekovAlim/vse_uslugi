package repositories

import (
	"context"
	"database/sql"

	"naimuBack/internal/models"
)

// UserResponsesRepository retrieves user responses across all entity types.
type UserResponsesRepository struct {
	DB *sql.DB
}

// GetResponsesByUserID returns all responses for a given user grouped by entity type.
func (r *UserResponsesRepository) GetResponsesByUserID(ctx context.Context, userID int) (models.UserResponses, error) {
	var result models.UserResponses

	serviceQuery := `
        SELECT s.id, s.name, s.price, s.description, sr.price, sr.created_at
        FROM service_responses sr
        JOIN service s ON s.id = sr.service_id
        WHERE sr.user_id = ?`
	if err := r.collect(ctx, &result.Service, serviceQuery, userID, "Service"); err != nil {
		return result, err
	}

	adQuery := `
        SELECT a.id, a.name, a.price, a.description, ar.price, ar.created_at
        FROM ad_responses ar
        JOIN ad a ON a.id = ar.ad_id
        WHERE ar.user_id = ?`
	if err := r.collect(ctx, &result.Ad, adQuery, userID, "Ad"); err != nil {
		return result, err
	}

	workQuery := `
        SELECT w.id, w.name, w.price, w.description, wr.price, wr.created_at
        FROM work_responses wr
        JOIN work w ON w.id = wr.work_id
        WHERE wr.user_id = ?`
	if err := r.collect(ctx, &result.Work, workQuery, userID, "Work"); err != nil {
		return result, err
	}

	workAdQuery := `
        SELECT wa.id, wa.name, wa.price, wa.description, war.price, war.created_at
        FROM work_ad_responses war
        JOIN work_ad wa ON wa.id = war.work_ad_id
        WHERE war.user_id = ?`
	if err := r.collect(ctx, &result.WorkAd, workAdQuery, userID, "Work Ad"); err != nil {
		return result, err
	}

	rentQuery := `
        SELECT r.id, r.name, r.price, r.description, rr.price, rr.created_at
        FROM rent_responses rr
        JOIN rent r ON r.id = rr.rent_id
        WHERE rr.user_id = ?`
	if err := r.collect(ctx, &result.Rent, rentQuery, userID, "Rent"); err != nil {
		return result, err
	}

	rentAdQuery := `
        SELECT ra.id, ra.name, ra.price, ra.description, rar.price, rar.created_at
        FROM rent_ad_responses rar
        JOIN rent_ad ra ON ra.id = rar.rent_ad_id
        WHERE rar.user_id = ?`
	if err := r.collect(ctx, &result.RentAd, rentAdQuery, userID, "Rent Ad"); err != nil {
		return result, err
	}

	return result, nil
}

func (r *UserResponsesRepository) collect(ctx context.Context, dest *[]models.UserResponseItem, query string, userID int, typ string) error {
	rows, err := r.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var item models.UserResponseItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Price, &item.Description, &item.ResponsePrice, &item.ResponseDate); err != nil {
			return err
		}
		item.Type = typ
		*dest = append(*dest, item)
	}
	return rows.Err()
}
