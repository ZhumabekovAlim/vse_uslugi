package repositories

import (
	"context"
	"database/sql"
	"fmt"
)

func getAverageRating(ctx context.Context, db *sql.DB, table, column string, id int) float64 {
	query := fmt.Sprintf("SELECT COALESCE(AVG(rating),0) FROM %s WHERE %s = ?", table, column)
	var avg sql.NullFloat64
	if err := db.QueryRowContext(ctx, query, id).Scan(&avg); err != nil {
		return 0
	}
	if avg.Valid {
		return avg.Float64
	}
	return 0
}
