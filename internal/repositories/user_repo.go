package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
        INSERT INTO users (name, surname, middlename, phone, email, password, city_id, review_rating, role, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `
	user.CreatedAt = time.Now()
	user.UpdatedAt = &user.CreatedAt
	result, err := r.DB.ExecContext(ctx, query,
		user.Name, user.Surname, user.Middlename, user.Phone, user.Email, user.Password, user.CityID, user.ReviewRating, user.Role,
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

func (r *UserRepository) GetUserByID(ctx context.Context, userID int) (models.User, error) {
	var user models.User

	// Получаем основную информацию о пользователе
	query := `SELECT id, name, middlename, surname, phone, email, password, city_id, role,
               years_of_exp, skills, doc_of_proof, avatar_path, created_at, updated_at
               FROM users WHERE id = ?`

	err := r.DB.QueryRowContext(ctx, query, userID).Scan(
		&user.ID, &user.Name, &user.Middlename, &user.Surname, &user.Phone, &user.Email, &user.Password,
		&user.CityID, &user.Role, &user.YearsOfExp, &user.Skills, &user.DocOfProof, &user.AvatarPath,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return models.User{}, err
	}

	// 🆕 Получаем список категорий с названиями
	catQuery := `
		SELECT c.id, c.name
		FROM user_categories uc
		JOIN categories c ON uc.category_id = c.id
		WHERE uc.user_id = ?`
	rows, err := r.DB.QueryContext(ctx, catQuery, userID)
	if err != nil {
		return models.User{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var cat models.Category
		if err := rows.Scan(&cat.ID, &cat.Name); err != nil {
			return models.User{}, err
		}
		user.Categories = append(user.Categories, cat)
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
	if user.AvatarPath != nil {
		setParts = append(setParts, "avatar_path = ?")
		args = append(args, user.AvatarPath)
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
       SELECT id, name, surname, middlename, phone, email, password, city_id, years_of_exp, doc_of_proof, avatar_path, review_rating, role, latitude, longitude, created_at, updated_at
        FROM users
        WHERE phone = ?
    `
	err := r.DB.QueryRowContext(ctx, query, phone).Scan(
		&user.ID, &user.Name, &user.Surname, &user.Middlename, &user.Phone, &user.Email, &user.Password, &user.CityID,
		&user.YearsOfExp, &user.DocOfProof, &user.AvatarPath, &user.ReviewRating, &user.Role,
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
       SELECT id, name, surname, middlename, phone, email, password, city_id, years_of_exp, doc_of_proof, avatar_path, review_rating, role, latitude, longitude, created_at, updated_at
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
			&user.YearsOfExp, &user.DocOfProof, &user.AvatarPath, &user.ReviewRating, &user.Role,
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
       SELECT id, name, surname, middlename, phone, email, password, city_id, years_of_exp, doc_of_proof, avatar_path, review_rating, role, latitude, longitude, created_at, updated_at
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
			&user.YearsOfExp, &user.DocOfProof, &user.AvatarPath, &user.ReviewRating, &user.Role,
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

func (r *UserRepository) GetSessionByToken(ctx context.Context, token string) (models.Session, error) {
	query := `
		SELECT id, role, refresh_token, expires_at
		FROM users
		WHERE refresh_token = ?
	`

	var session models.Session
	err := r.DB.QueryRowContext(ctx, query, token).Scan(
		&session.UserID,
		&session.Role,
		&session.RefreshToken,
		&session.ExpiresAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return session, errors.New("no session found for the token")
		}
		return session, err
	}

	return session, nil
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	var user models.User
	query := `
       SELECT id, name, surname, middlename, phone, email, password, city_id, years_of_exp, doc_of_proof, avatar_path, review_rating, role, latitude, longitude, created_at, updated_at
        FROM users
        WHERE email = ?
    `
	err := r.DB.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Name, &user.Surname, &user.Middlename, &user.Phone, &user.Email, &user.Password, &user.CityID,
		&user.YearsOfExp, &user.DocOfProof, &user.AvatarPath, &user.ReviewRating, &user.Role,
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

func (r *UserRepository) GetUserByPhone1(ctx context.Context, phone string) (models.User, error) {
	var user models.User
	query := `
       SELECT id, name, surname, middlename, phone, email, password, city_id, years_of_exp, doc_of_proof, avatar_path, review_rating, role, latitude, longitude, created_at, updated_at
        FROM users
        WHERE phone = ?
    `
	err := r.DB.QueryRowContext(ctx, query, phone).Scan(
		&user.ID, &user.Name, &user.Surname, &user.Middlename, &user.Phone, &user.Email, &user.Password, &user.CityID,
		&user.YearsOfExp, &user.DocOfProof, &user.AvatarPath, &user.ReviewRating, &user.Role,
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

func (r *UserRepository) UpdateUserAvatar(ctx context.Context, userID int, avatarPath string) (models.User, error) {
	query := `UPDATE users SET avatar_path = ?, updated_at = NOW() WHERE id = ?`
	if _, err := r.DB.ExecContext(ctx, query, avatarPath, userID); err != nil {
		return models.User{}, err
	}
	return r.GetUserByID(ctx, userID)
}

func (r *UserRepository) ChangeCityForUser(ctx context.Context, userID int, cityID int) error {
	query := `UPDATE users SET city_id = ?, updated_at = NOW() WHERE id = ?`

	result, err := r.DB.ExecContext(ctx, query, cityID, userID)
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

func (r *UserRepository) UpdateWorkerProfile(ctx context.Context, user models.User) (models.User, error) {
	// Обновляем роль и профиль
	query := `
		UPDATE users
		SET role = ?, years_of_exp = ?, skills = ?, doc_of_proof = ?, updated_at = NOW()
		WHERE id = ?`
	_, err := r.DB.ExecContext(ctx, query, user.Role, user.YearsOfExp, user.Skills, user.DocOfProof, user.ID)
	if err != nil {
		return models.User{}, err
	}

	// Удалим старые связи
	_, _ = r.DB.ExecContext(ctx, `DELETE FROM user_categories WHERE user_id = ?`, user.ID)
	fmt.Println("CATEGORIES TO INSERT:")
	for _, category := range user.Categories {
		fmt.Println(" -", category.ID)
		_, insertErr := r.DB.ExecContext(ctx,
			`INSERT INTO user_categories (user_id, category_id) VALUES (?, ?)`,
			user.ID, category.ID)
		if insertErr != nil {
			return models.User{}, insertErr
		}
	}

	return r.GetUserByID(ctx, user.ID)
}

func (r *UserRepository) IsPhoneOrEmailTaken(ctx context.Context, phone, email string) (bool, error) {
	var count int

	// Проверка email
	err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE email = ?`, email).Scan(&count)
	if err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}

	// Проверка телефона
	err = r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE phone = ?`, phone).Scan(&count)
	if err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	return false, nil
}

func (r *UserRepository) SaveVerificationCode(ctx context.Context, phone, code string) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO verification_codes (phone, code, created_at) VALUES (?, ?, NOW())`,
		phone, code,
	)
	return err
}

func (r *UserRepository) GetVerificationCodeByPhone(ctx context.Context, phone string) (string, error) {
	var code string
	err := r.DB.QueryRowContext(ctx, `SELECT code FROM verification_codes WHERE phone = ?`, phone).Scan(&code)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", models.ErrInvalidVerificationCode
		}
		return "", err
	}
	return code, nil
}

func (r *UserRepository) ClearVerificationCode(ctx context.Context, phone string) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM verification_codes WHERE phone = ?`, phone)
	return err
}

func (r *UserRepository) SaveEmailVerificationCode(ctx context.Context, email, code string) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO verification_codes (email, code, created_at) VALUES (?, ?, NOW())`,
		email, code,
	)
	return err
}

func (r *UserRepository) GetVerificationCodeByEmail(ctx context.Context, email string) (string, error) {
	var code string
	err := r.DB.QueryRowContext(ctx, `SELECT code FROM verification_codes WHERE email = ?`, email).Scan(&code)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", models.ErrInvalidVerificationCode
		}
		return "", err
	}
	return code, nil
}

func (r *UserRepository) ClearVerificationCodeByEmail(ctx context.Context, email string) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM verification_codes WHERE email = ?`, email)
	return err
}

func (r *UserRepository) SaveResetCode(ctx context.Context, email, code string) error {
	query := `UPDATE users SET reset_code = ? WHERE email = ?`
	_, err := r.DB.ExecContext(ctx, query, code, email)
	return err
}

func (r *UserRepository) VerifyResetCode(ctx context.Context, email, code string) (bool, error) {
	var storedCode string
	err := r.DB.QueryRowContext(ctx, `SELECT reset_code FROM users WHERE email = ?`, email).Scan(&storedCode)
	if err != nil {
		return false, err
	}
	return storedCode == code, nil
}

func (r *UserRepository) UpdatePasswordEmail(ctx context.Context, email, hashedPassword string) error {
	query := `UPDATE users SET password = ?, reset_code = NULL WHERE email = ?`
	_, err := r.DB.ExecContext(ctx, query, hashedPassword, email)
	return err
}
