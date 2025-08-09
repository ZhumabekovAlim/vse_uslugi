package repositories

import (
	"context"
	"database/sql"
)

func getUserTotalReviews(ctx context.Context, db *sql.DB, userID int) (int, error) {
	query := `
        SELECT (
            (SELECT COUNT(*) FROM reviews r JOIN service s ON r.service_id = s.id WHERE s.user_id = ?) +
            (SELECT COUNT(*) FROM ad_reviews r JOIN ad a ON r.ad_id = a.id WHERE a.user_id = ?) +
            (SELECT COUNT(*) FROM work_reviews r JOIN work w ON r.work_id = w.id WHERE w.user_id = ?) +
            (SELECT COUNT(*) FROM work_ad_reviews r JOIN work_ad wa ON r.work_ad_id = wa.id WHERE wa.user_id = ?) +
            (SELECT COUNT(*) FROM rent_reviews r JOIN rent rn ON r.rent_id = rn.id WHERE rn.user_id = ?) +
            (SELECT COUNT(*) FROM rent_ad_reviews r JOIN rent_ad ra ON r.rent_ad_id = ra.id WHERE ra.user_id = ?)
        ) AS total_reviews`
	var count int
	err := db.QueryRowContext(ctx, query, userID, userID, userID, userID, userID, userID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
