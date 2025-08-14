package repositories

import (
	"context"
	"database/sql"
)

func getUserAverageRating(ctx context.Context, db *sql.DB, userID int) (float64, error) {
	query := `
       SELECT COALESCE(AVG(rating),0) FROM (
           SELECT r.rating FROM reviews r JOIN service s ON r.service_id = s.id WHERE s.user_id = ?
           UNION ALL
           SELECT ar.rating FROM ad_reviews ar JOIN ad a ON ar.ad_id = a.id WHERE a.user_id = ?
           UNION ALL
           SELECT wr.rating FROM work_reviews wr JOIN work w ON wr.work_id = w.id WHERE w.user_id = ?
           UNION ALL
           SELECT war.rating FROM work_ad_reviews war JOIN work_ad wa ON war.work_ad_id = wa.id WHERE wa.user_id = ?
           UNION ALL
           SELECT rr.rating FROM rent_reviews rr JOIN rent rn ON rr.rent_id = rn.id WHERE rn.user_id = ?
           UNION ALL
           SELECT rar.rating FROM rent_ad_reviews rar JOIN rent_ad ra ON rar.rent_ad_id = ra.id WHERE ra.user_id = ?
       ) all_ratings`
	var avg float64
	err := db.QueryRowContext(ctx, query, userID, userID, userID, userID, userID, userID).Scan(&avg)
	if err != nil {
		return 0, err
	}
	return avg, nil
}

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
