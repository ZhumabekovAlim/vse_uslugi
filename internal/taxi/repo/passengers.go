package repo

import (
	"context"
	"database/sql"
	"time"
)

// Passenger represents a user acting as a taxi passenger.
type Passenger struct {
	ID           int64
	Name         string
	Surname      string
	Middlename   sql.NullString
	Phone        string
	Email        string
	CityID       sql.NullInt64
	YearsOfExp   sql.NullInt64
	DocOfProof   sql.NullString
	ReviewRating sql.NullFloat64
	Role         sql.NullString
	Latitude     sql.NullString
	Longitude    sql.NullString
	AvatarPath   sql.NullString
	Skills       sql.NullString
	IsOnline     sql.NullBool
	CreatedAt    time.Time
	UpdatedAt    sql.NullTime
}

// PassengersRepo provides access to passenger data backed by the users table.
type PassengersRepo struct {
	db *sql.DB
}

// NewPassengersRepo constructs a PassengersRepo instance.
func NewPassengersRepo(db *sql.DB) *PassengersRepo {
	return &PassengersRepo{db: db}
}

// Get retrieves a passenger by user identifier.
func (r *PassengersRepo) Get(ctx context.Context, id int64) (Passenger, error) {
	var p Passenger
	row := r.db.QueryRowContext(ctx, `SELECT
        id,
        name,
        surname,
        middlename,
        phone,
        email,
        city_id,
        years_of_exp,
        doc_of_proof,
        review_rating,
        role,
        latitude,
        longitude,
        avatar_path,
        skills,
        is_online,
        created_at,
        updated_at
    FROM users
    WHERE id = ?`, id)
	err := row.Scan(
		&p.ID,
		&p.Name,
		&p.Surname,
		&p.Middlename,
		&p.Phone,
		&p.Email,
		&p.CityID,
		&p.YearsOfExp,
		&p.DocOfProof,
		&p.ReviewRating,
		&p.Role,
		&p.Latitude,
		&p.Longitude,
		&p.AvatarPath,
		&p.Skills,
		&p.IsOnline,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		return Passenger{}, err
	}
	return p, nil
}
