package repositories

import (
	"context"
	"database/sql"

	"naimuBack/internal/models"
)

// UserReviewsRepository retrieves user reviews across all entity types.
type UserReviewsRepository struct {
	DB *sql.DB
}

// GetReviewsByUserID returns all reviews for a given user grouped by entity type.
func (r *UserReviewsRepository) GetReviewsByUserID(ctx context.Context, userID int) (models.UserReviews, error) {
	var result models.UserReviews

	serviceQuery := `
        SELECT s.id, s.name, s.price, s.description, rv.rating, rv.review, rv.created_at
        FROM reviews rv
        JOIN service s ON s.id = rv.service_id
        WHERE rv.user_id = ?`
	if err := r.collect(ctx, &result.Service, serviceQuery, userID, "Service"); err != nil {
		return result, err
	}

	adQuery := `
        SELECT a.id, a.name, a.price, a.description, ar.rating, ar.review, ar.created_at
        FROM ad_reviews ar
        JOIN ad a ON a.id = ar.ad_id
        WHERE ar.user_id = ?`
	if err := r.collect(ctx, &result.Ad, adQuery, userID, "Ad"); err != nil {
		return result, err
	}

	workQuery := `
        SELECT w.id, w.name, w.price, w.description, wr.rating, wr.review, wr.created_at
        FROM work_reviews wr
        JOIN work w ON w.id = wr.work_id
        WHERE wr.user_id = ?`
	if err := r.collect(ctx, &result.Work, workQuery, userID, "Work"); err != nil {
		return result, err
	}

	workAdQuery := `
        SELECT wa.id, wa.name, wa.price, wa.description, war.rating, war.review, war.created_at
        FROM work_ad_reviews war
        JOIN work_ad wa ON wa.id = war.work_ad_id
        WHERE war.user_id = ?`
	if err := r.collect(ctx, &result.WorkAd, workAdQuery, userID, "Work Ad"); err != nil {
		return result, err
	}

	rentQuery := `
        SELECT r.id, r.name, r.price, r.description, rr.rating, rr.review, rr.created_at
        FROM rent_reviews rr
        JOIN rent r ON r.id = rr.rent_id
        WHERE rr.user_id = ?`
	if err := r.collect(ctx, &result.Rent, rentQuery, userID, "Rent"); err != nil {
		return result, err
	}

	rentAdQuery := `
        SELECT ra.id, ra.name, ra.price, ra.description, rar.rating, rar.review, rar.created_at
        FROM rent_ad_reviews rar
        JOIN rent_ad ra ON ra.id = rar.rent_ad_id
        WHERE rar.user_id = ?`
	if err := r.collect(ctx, &result.RentAd, rentAdQuery, userID, "Rent Ad"); err != nil {
		return result, err
	}

	return result, nil
}

func (r *UserReviewsRepository) collect(ctx context.Context, dest *[]models.UserReviewItem, query string, userID int, typ string) error {
	rows, err := r.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var item models.UserReviewItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Price, &item.Description, &item.Rating, &item.Review, &item.ReviewDate); err != nil {
			return err
		}
		item.Type = typ
		*dest = append(*dest, item)
	}
	return rows.Err()
}
