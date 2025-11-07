package repositories

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"naimuBack/internal/models"
)

var ErrTaxiIntercityOrderNotFound = errors.New("taxi intercity order not found")

type TaxiIntercityOrderRepository struct {
	DB *sql.DB
}

func (r *TaxiIntercityOrderRepository) Create(ctx context.Context, order models.TaxiIntercityOrder) (models.TaxiIntercityOrder, error) {
	query := `
                INSERT INTO taxi_intercity_orders (client_id, from_city, to_city, trip_type, comment, price, departure_date, status)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?)
        `

	comment := sql.NullString{String: strings.TrimSpace(order.Comment), Valid: strings.TrimSpace(order.Comment) != ""}

	res, err := r.DB.ExecContext(ctx, query,
		order.ClientID,
		order.FromCity,
		order.ToCity,
		order.TripType,
		comment,
		order.Price,
		order.DepartureDate,
		order.Status,
	)
	if err != nil {
		return models.TaxiIntercityOrder{}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return models.TaxiIntercityOrder{}, err
	}

	return r.GetByID(ctx, int(id))
}

func (r *TaxiIntercityOrderRepository) GetByID(ctx context.Context, id int) (models.TaxiIntercityOrder, error) {
	query := `
                SELECT o.id, o.client_id, u.name, u.phone, o.from_city, o.to_city, o.trip_type,
                       o.comment, o.price, o.departure_date, o.status,
                       o.created_at, o.updated_at, o.closed_at
                FROM taxi_intercity_orders o
                JOIN users u ON o.client_id = u.id
                WHERE o.id = ?
        `

	var (
		order     models.TaxiIntercityOrder
		comment   sql.NullString
		updatedAt sql.NullTime
		closedAt  sql.NullTime
	)

	err := r.DB.QueryRowContext(ctx, query, id).Scan(
		&order.ID,
		&order.ClientID,
		&order.ClientName,
		&order.ClientPhone,
		&order.FromCity,
		&order.ToCity,
		&order.TripType,
		&comment,
		&order.Price,
		&order.DepartureDate,
		&order.Status,
		&order.CreatedAt,
		&updatedAt,
		&closedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.TaxiIntercityOrder{}, ErrTaxiIntercityOrderNotFound
	}
	if err != nil {
		return models.TaxiIntercityOrder{}, err
	}

	if comment.Valid {
		order.Comment = comment.String
	}
	if updatedAt.Valid {
		t := updatedAt.Time
		order.UpdatedAt = &t
	}
	if closedAt.Valid {
		t := closedAt.Time
		order.ClosedAt = &t
	}

	return order, nil
}

func (r *TaxiIntercityOrderRepository) Search(ctx context.Context, filter models.TaxiIntercityOrderFilter) ([]models.TaxiIntercityOrder, error) {
	baseQuery := `
                SELECT o.id, o.client_id, u.name, u.phone, o.from_city, o.to_city, o.trip_type,
                       o.comment, o.price, o.departure_date, o.status,
                       o.created_at, o.updated_at, o.closed_at
                FROM taxi_intercity_orders o
                JOIN users u ON o.client_id = u.id
                WHERE 1=1
        `

	var (
		args  []interface{}
		parts []string
	)

	if filter.Status != "" {
		parts = append(parts, "AND o.status = ?")
		args = append(args, filter.Status)
	}
	if filter.FromCity != "" {
		parts = append(parts, "AND o.from_city LIKE ?")
		args = append(args, "%"+filter.FromCity+"%")
	}
	if filter.ToCity != "" {
		parts = append(parts, "AND o.to_city LIKE ?")
		args = append(args, "%"+filter.ToCity+"%")
	}
	if filter.DepartureDate != nil {
		parts = append(parts, "AND o.departure_date = ?")
		args = append(args, filter.DepartureDate.Format("2006-01-02"))
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := baseQuery + " " + strings.Join(parts, " ") + " ORDER BY o.departure_date ASC, o.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.TaxiIntercityOrder
	for rows.Next() {
		var (
			order     models.TaxiIntercityOrder
			comment   sql.NullString
			updatedAt sql.NullTime
			closedAt  sql.NullTime
		)
		if err := rows.Scan(
			&order.ID,
			&order.ClientID,
			&order.ClientName,
			&order.ClientPhone,
			&order.FromCity,
			&order.ToCity,
			&order.TripType,
			&comment,
			&order.Price,
			&order.DepartureDate,
			&order.Status,
			&order.CreatedAt,
			&updatedAt,
			&closedAt,
		); err != nil {
			return nil, err
		}

		if comment.Valid {
			order.Comment = comment.String
		}
		if updatedAt.Valid {
			t := updatedAt.Time
			order.UpdatedAt = &t
		}
		if closedAt.Valid {
			t := closedAt.Time
			order.ClosedAt = &t
		}

		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *TaxiIntercityOrderRepository) ListByClient(ctx context.Context, clientID int, status string) ([]models.TaxiIntercityOrder, error) {
	baseQuery := `
                SELECT o.id, o.client_id, u.name, u.phone, o.from_city, o.to_city, o.trip_type,
                       o.comment, o.price, o.departure_date, o.status,
                       o.created_at, o.updated_at, o.closed_at
                FROM taxi_intercity_orders o
                JOIN users u ON o.client_id = u.id
                WHERE o.client_id = ?
        `

	args := []interface{}{clientID}
	if status != "" {
		baseQuery += " AND o.status = ?"
		args = append(args, status)
	}
	baseQuery += " ORDER BY o.created_at DESC"

	rows, err := r.DB.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.TaxiIntercityOrder
	for rows.Next() {
		var (
			order     models.TaxiIntercityOrder
			comment   sql.NullString
			updatedAt sql.NullTime
			closedAt  sql.NullTime
		)
		if err := rows.Scan(
			&order.ID,
			&order.ClientID,
			&order.ClientName,
			&order.ClientPhone,
			&order.FromCity,
			&order.ToCity,
			&order.TripType,
			&comment,
			&order.Price,
			&order.DepartureDate,
			&order.Status,
			&order.CreatedAt,
			&updatedAt,
			&closedAt,
		); err != nil {
			return nil, err
		}
		if comment.Valid {
			order.Comment = comment.String
		}
		if updatedAt.Valid {
			t := updatedAt.Time
			order.UpdatedAt = &t
		}
		if closedAt.Valid {
			t := closedAt.Time
			order.ClosedAt = &t
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *TaxiIntercityOrderRepository) UpdateStatus(ctx context.Context, id int, status string) error {
	query := `
                UPDATE taxi_intercity_orders
                SET status = ?,
                    updated_at = CURRENT_TIMESTAMP,
                    closed_at = CASE WHEN ? = 'closed' THEN CURRENT_TIMESTAMP ELSE closed_at END
                WHERE id = ?
        `

	res, err := r.DB.ExecContext(ctx, query, status, status, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrTaxiIntercityOrderNotFound
	}
	return nil
}
