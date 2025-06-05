package repositories

import (
	"context"
	"database/sql"
	"errors"
	_ "fmt"
	"naimuBack/internal/models"
	"strings"
	"time"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

type UserRepository struct {
	DB *sql.DB
}

type Session struct {
	ID     string `json:"id"`
	Expiry string `json:"expiry"`
}

func (r *UserRepository) CreateUser(ctx context.Context, user models.User) (models.User, error) {
	query := `
        INSERT INTO users (name, surname, middlename, phone, email, password, review_rating, role, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `
	user.CreatedAt = time.Now()
	user.UpdatedAt = &user.CreatedAt
	result, err := r.DB.ExecContext(ctx, query,
		user.Name, user.Surname, user.Middlename, user.Phone, user.Email, user.Password, user.ReviewRating, user.Role,
		user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return models.User{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return models.User{}, err
	}
	user.ID = int(id)
	return user, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, id int) (models.User, error) {
	var user models.User
	query := `
        SELECT id, name, surname, middlename, phone, email, password, city_id, years_of_exp, doc_of_proof, review_rating, role, latitude, longitude, created_at, updated_at
        FROM users
        WHERE id = ?
    `
	err := r.DB.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Name, &user.Surname, &user.Middlename, &user.Phone, &user.Email, &user.Password, &user.CityID,
		&user.YearsOfExp, &user.DocOfProof, &user.ReviewRating, &user.Role,
		&user.Latitude, &user.Longitude, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return models.User{}, ErrUserNotFound
	}
	if err != nil {
		return models.User{}, err
	}
	return user, nil
}

func (r *UserRepository) UpdateUser(ctx context.Context, user models.User) (models.User, error) {
	query := `UPDATE users SET `
	args := []interface{}{}
	setParts := []string{}

	updatedAt := time.Now()
	user.UpdatedAt = &updatedAt
	setParts = append(setParts, "updated_at = ?")
	args = append(args, updatedAt)

	if user.Name != "" {
		setParts = append(setParts, "name = ?")
		args = append(args, user.Name)
	}
	if user.Surname != "" {
		setParts = append(setParts, "surname = ?")
		args = append(args, user.Surname)
	}
	if user.Middlename != "" {
		setParts = append(setParts, "middlename = ?")
		args = append(args, user.Middlename)
	}
	if user.Phone != "" {
		setParts = append(setParts, "phone = ?")
		args = append(args, user.Phone)
	}
	if user.Email != "" {
		setParts = append(setParts, "email = ?")
		args = append(args, user.Email)
	}
	if user.Password != "" {
		setParts = append(setParts, "password = ?")
		args = append(args, user.Password)
	}
	if user.CityID != nil {
		setParts = append(setParts, "city_id = ?")
		args = append(args, user.CityID)
	}
	if user.YearsOfExp != nil {
		setParts = append(setParts, "years_of_exp = ?")
		args = append(args, user.YearsOfExp)
	}
	if user.DocOfProof != nil {
		setParts = append(setParts, "doc_of_proof = ?")
		args = append(args, user.DocOfProof)
	}
	if user.ReviewRating != 0 {
		setParts = append(setParts, "review_rating = ?")
		args = append(args, user.ReviewRating)
	}
	if user.Role != "" {
		setParts = append(setParts, "role = ?")
		args = append(args, user.Role)
	}
	if user.Latitude != nil {
		setParts = append(setParts, "latitude = ?")
		args = append(args, user.Latitude)
	}
	if user.Longitude != nil {
		setParts = append(setParts, "longitude = ?")
		args = append(args, user.Longitude)
	}

	// Если ничего не обновляется кроме updated_at
	if len(setParts) == 1 {
		return models.User{}, errors.New("no fields to update")
	}

	// Добавляем WHERE
	query += strings.Join(setParts, ", ") + " WHERE id = ?"
	args = append(args, user.ID)

	// Выполняем
	result, err := r.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return models.User{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return models.User{}, err
	}
	if rowsAffected == 0 {
		return models.User{}, ErrUserNotFound
	}

	return r.GetUserByID(ctx, user.ID)
}

func (r *UserRepository) DeleteUser(ctx context.Context, id int) error {
	query := `DELETE FROM users WHERE id = ?`
	result, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *UserRepository) GetUserByPhone(ctx context.Context, phone string) (models.User, error) {
	var user models.User
	query := `
        SELECT id, name, surname, middlename, phone, email, password, city_id, years_of_exp, doc_of_proof, review_rating, role, latitude, longitude, created_at, updated_at
        FROM users
        WHERE phone = ?
    `
	err := r.DB.QueryRowContext(ctx, query, phone).Scan(
		&user.ID, &user.Name, &user.Surname, &user.Middlename, &user.Phone, &user.Email, &user.Password, &user.CityID,
		&user.YearsOfExp, &user.DocOfProof, &user.ReviewRating, &user.Role,
		&user.Latitude, &user.Longitude, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return models.User{}, ErrUserNotFound
	}
	if err != nil {
		return models.User{}, err
	}
	return user, nil
}

func (r *UserRepository) GetUsersByRole(ctx context.Context, role string) ([]models.User, error) {
	query := `
        SELECT id, name, surname, middlename, phone, email, password, city_id, years_of_exp, doc_of_proof, review_rating, role, latitude, longitude, created_at, updated_at
        FROM users
        WHERE role = ?
    `
	rows, err := r.DB.QueryContext(ctx, query, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID, &user.Name, &user.Surname, &user.Middlename, &user.Phone, &user.Email, &user.Password, &user.CityID,
			&user.YearsOfExp, &user.DocOfProof, &user.ReviewRating, &user.Role,
			&user.Latitude, &user.Longitude, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, ErrUserNotFound
	}
	return users, nil
}

func (r *UserRepository) GetAllUsers(ctx context.Context) ([]models.User, error) {
	query := `
        SELECT id, name, surname, middlename, phone, email, password, city_id, years_of_exp, doc_of_proof, review_rating, role, latitude, longitude, created_at, updated_at
        FROM users
    `
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID, &user.Name, &user.Surname, &user.Middlename, &user.Phone, &user.Email, &user.Password, &user.CityID,
			&user.YearsOfExp, &user.DocOfProof, &user.ReviewRating, &user.Role,
			&user.Latitude, &user.Longitude, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) SetSession(ctx context.Context, id string, session models.Session) error {

	query := `
		UPDATE users 
		SET refresh_token = ?, expires_at = ? 
		WHERE id = ?
	`

	result, err := r.DB.ExecContext(ctx, query, session.RefreshToken, session.ExpiresAt, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("no rows updated")
	}

	return nil
}
func (r *UserRepository) GetSession(ctx context.Context, id string) (models.Session, error) {
	query := `
		SELECT refresh_token, expires_at 
		FROM users 
		WHERE id = ?
	`

	var session models.Session
	err := r.DB.QueryRowContext(ctx, query, id).Scan(&session.RefreshToken, &session.ExpiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return session, errors.New("no session found for the user")
		}
		return session, err
	}

	return session, nil
}

func (r *UserRepository) UserLogOut(ctx context.Context, userID int) error {

	query := "UPDATE users SET refresh_token = ? WHERE id = ? "
	rows, err := r.DB.QueryContext(ctx, query, "", userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	return nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID int, newPassword string) error {
	query := `UPDATE users SET password = ? WHERE id = ?`
	_, err := r.DB.ExecContext(ctx, query, newPassword, userID)
	return err
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	var user models.User
	query := `
        SELECT id, name, surname, middlename, phone, email, password, city_id, years_of_exp, doc_of_proof, review_rating, role, latitude, longitude, created_at, updated_at
        FROM users
        WHERE email = ?
    `
	err := r.DB.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Name, &user.Surname, &user.Middlename, &user.Phone, &user.Email, &user.Password, &user.CityID,
		&user.YearsOfExp, &user.DocOfProof, &user.ReviewRating, &user.Role,
		&user.Latitude, &user.Longitude, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.User{}, nil
		}
		return models.User{}, err
	}
	return user, nil
}

func (r *UserRepository) GetUserByPhone1(ctx context.Context, phone string) (models.User, error) {
	var user models.User
	query := `
        SELECT id, name, surname, middlename, phone, email, password, city_id, years_of_exp, doc_of_proof, review_rating, role, latitude, longitude, created_at, updated_at
        FROM users
        WHERE phone = ?
    `
	err := r.DB.QueryRowContext(ctx, query, phone).Scan(
		&user.ID, &user.Name, &user.Surname, &user.Middlename, &user.Phone, &user.Email, &user.Password, &user.CityID,
		&user.YearsOfExp, &user.DocOfProof, &user.ReviewRating, &user.Role,
		&user.Latitude, &user.Longitude, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.User{}, nil
		}
		return models.User{}, err
	}
	return user, nil
}
