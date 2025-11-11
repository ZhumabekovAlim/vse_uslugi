package repo

import (
	"context"
	"database/sql"
	"time"
)

// User represents a generic profile stored in the users table.
type User struct {
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

// UsersRepo provides access to records stored in the users table.
type UsersRepo struct {
	db *sql.DB
}

// NewUsersRepo constructs a UsersRepo instance.
func NewUsersRepo(db *sql.DB) *UsersRepo {
	return &UsersRepo{db: db}
}

// Get retrieves a user by identifier.
func (r *UsersRepo) Get(ctx context.Context, id int64) (User, error) {
	var u User
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
		&u.ID,
		&u.Name,
		&u.Surname,
		&u.Middlename,
		&u.Phone,
		&u.Email,
		&u.CityID,
		&u.YearsOfExp,
		&u.DocOfProof,
		&u.ReviewRating,
		&u.Role,
		&u.Latitude,
		&u.Longitude,
		&u.AvatarPath,
		&u.Skills,
		&u.IsOnline,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return User{}, err
	}
	return u, nil
}
