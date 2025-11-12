package services

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	_ "github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
	"naimuBack/utils"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type tokenClaims struct {
	jwt.StandardClaims
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
	CityID int    `json:"city_id"`
}
type UserService struct {
	UserRepo     *repositories.UserRepository
	TokenManager *utils.Manager
}

var errMailgunNotConfigured = errors.New("mailgun not configured")

const (
	salt       = "sadasdnsadna"
	tokenTTL   = 1000 * time.Minute
	signingKey = "asdadsadadaadsasd"
)

func generateVerificationCode() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%04d", rand.Intn(10000))
}

func SendSMS(apiKey, phone, message string) error {
	endpoint := "https://api.mobizon.kz/service/message/sendsmsmessage"

	data := url.Values{}
	data.Set("apiKey", apiKey)
	data.Set("recipient", phone)
	data.Set("text", message)

	resp, err := http.PostForm(endpoint, data)
	if err != nil {
		return fmt.Errorf("ошибка при отправке запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("не удалось прочитать ответ: %v", err)
	}

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("не удалось распарсить ответ: %v", err)
	}

	if result.Code != 0 {
		return fmt.Errorf("ошибка Mobizon: %s (код %d)", result.Message, result.Code)
	}

	return nil
}

func (s *UserService) sendSMS(apiKey, phone, message string) error {
	endpoint := "https://api.mobizon.kz/service/message/sendsmsmessage"

	data := url.Values{}
	data.Set("apiKey", apiKey)
	data.Set("recipient", phone)
	data.Set("text", message)

	resp, err := http.PostForm(endpoint, data)
	if err != nil {
		return fmt.Errorf("ошибка при отправке запроса: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("не удалось прочитать ответ Mobizon: %v", err)
	}

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("не удалось распарсить ответ Mobizon: %v", err)
	}

	if result.Code != 0 {
		return fmt.Errorf("ошибка Mobizon: %s (код %d)", result.Message, result.Code)
	}

	return nil
}

func (s *UserService) SignUp(ctx context.Context, user models.User, inputCode string) (models.SignUpResponse, error) {
	// 1. Проверяем код подтверждения
	if err := s.CheckVerificationCode(ctx, user.Email, inputCode); err != nil {
		return models.SignUpResponse{}, err
	}

	// 3. Хешируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return models.SignUpResponse{}, err
	}
	user.Password = string(hashedPassword)
	user.Role = "client"

	// 4. Сохраняем пользователя
	newUser, err := s.UserRepo.CreateUser(ctx, user)
	if err != nil {
		return models.SignUpResponse{}, err
	}

	// 5. Можно очистить использованный код, если хочешь
	_ = s.UserRepo.ClearVerificationCodeByEmail(ctx, user.Email)

	return models.SignUpResponse{User: newUser}, nil
}

func (s *UserService) CheckVerificationCode(ctx context.Context, email, inputCode string) error {
	codeFromDB, err := s.UserRepo.GetVerificationCodeByEmail(ctx, email)
	if err != nil {
		return err
	}

	if inputCode != codeFromDB {
		return models.ErrInvalidVerificationCode
	}

	return nil
}

func (s *UserService) SignIn(ctx context.Context, name, phone, email, password string) (models.Tokens, error) {
	user, err := s.UserRepo.GetUserByPhone(ctx, phone)
	if err != nil {
		log.Printf("User not found: %s", phone)
		return models.Tokens{}, errors.New("user not found")
	}

	// Compare the provided password with the hashed password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		log.Printf("Invalid password for user: %s", phone)
		return models.Tokens{}, errors.New("invalid password")
	}

	cityID := 0
	if user.CityID != nil {
		cityID = *user.CityID
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &tokenClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(tokenTTL).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
		UserID: user.ID,
		Role:   user.Role,
		CityID: cityID,
	})

	accessToken, err := token.SignedString([]byte(signingKey))
	if err != nil {
		log.Printf("Error signing token: %v", err)
		return models.Tokens{}, err
	}
	fmt.Println("login token:", accessToken)
	tokens, err := s.CreateSession(ctx, user, accessToken)
	if err != nil {
		log.Printf("Error creating session: %v", err)
		return models.Tokens{}, err
	}

	return tokens, nil
}

func (s *UserService) CreateSession(ctx context.Context, user models.User, accessToken string) (models.Tokens, error) {
	var (
		res models.Tokens
		err error
	)

	userIDStr := strconv.Itoa(user.ID)

	res.AccessToken = accessToken

	// Generate RefreshToken using UUID as a fallback
	res.RefreshToken = uuid.New().String() // Fallback if TokenManager is unavailable
	if s.TokenManager != nil {
		res.RefreshToken, err = s.TokenManager.NewRefreshToken()
		if err != nil {
			return res, err
		}
	}

	// Создание и сохранение сессии с RefreshToken
	session := models.Session{
		RefreshToken: res.RefreshToken,
		ExpiresAt:    time.Now().Add(24 * 30 * 2 * time.Hour),
	}

	err = s.UserRepo.SetSession(ctx, userIDStr, session)
	if err != nil {
		return res, err
	}

	return res, nil
}

func (s *UserService) CreateUser(ctx context.Context, user models.User) (models.User, error) {
	return s.UserRepo.CreateUser(ctx, user)
}

func (s *UserService) GetUserByID(ctx context.Context, id int) (models.User, error) {
	return s.UserRepo.GetUserByID(ctx, id)
}

func (s *UserService) GetUserByToken(ctx context.Context, tokenString string) (models.User, error) {
	claims := &models.Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(signingKey), nil
	})
	if err != nil || !token.Valid {
		return models.User{}, fmt.Errorf("invalid token")
	}

	return s.UserRepo.GetUserByID(ctx, int(claims.UserID))
}

func (s *UserService) UpdateUser(ctx context.Context, user models.User) (models.User, error) {
	existingUser1, err := s.UserRepo.GetUserByEmail(ctx, user.Email)
	if err != nil {
		return models.User{}, err
	}
	if existingUser1.Email != "" && existingUser1.ID != user.ID {
		return models.User{}, errors.New("user with this email already exists")
	}

	existingUser2, err := s.UserRepo.GetUserByPhone1(ctx, user.Phone)
	if err != nil {
		return models.User{}, err
	}
	if existingUser2.Phone != "" && existingUser2.ID != user.ID {
		return models.User{}, errors.New("user with this phone already exists")
	}

	return s.UserRepo.UpdateUser(ctx, user)
}

func (s *UserService) UpdateUserAvatar(ctx context.Context, userID int, avatarPath string) (models.User, error) {
	return s.UserRepo.UpdateUserAvatar(ctx, userID, avatarPath)
}

func (s *UserService) DeleteUserAvatar(ctx context.Context, userID int) error {
	user, err := s.UserRepo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.AvatarPath != nil && *user.AvatarPath != "" {
		avatarFile := filepath.Base(*user.AvatarPath)
		if avatarFile != "" {
			diskPath := filepath.Join("cmd/uploads/avatars", avatarFile)
			if removeErr := os.Remove(diskPath); removeErr != nil && !os.IsNotExist(removeErr) {
				return removeErr
			}
		}
	}

	return s.UserRepo.ClearUserAvatar(ctx, userID)
}

func (s *UserService) DeleteUser(ctx context.Context, id int) (err error) {
	tx, err := s.UserRepo.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("rollback delete user %d: %v", id, rollbackErr)
			}
		}
	}()

	cleanupQueries := []struct {
		query string
		args  []interface{}
	}{
		{query: "DELETE FROM courier_offers WHERE courier_id IN (SELECT id FROM couriers WHERE user_id = ?)", args: []interface{}{id}},
		{query: "DELETE FROM courier_orders WHERE sender_id = ? OR courier_id IN (SELECT id FROM couriers WHERE user_id = ?)", args: []interface{}{id, id}},
		{query: "DELETE FROM couriers WHERE user_id = ?", args: []interface{}{id}},
		{query: "DELETE FROM service WHERE user_id = ?", args: []interface{}{id}},
		{query: "DELETE FROM ad WHERE user_id = ?", args: []interface{}{id}},
		{query: "DELETE FROM rent WHERE user_id = ?", args: []interface{}{id}},
		{query: "DELETE FROM rent_ad WHERE user_id = ?", args: []interface{}{id}},
		{query: "DELETE FROM work WHERE user_id = ?", args: []interface{}{id}},
		{query: "DELETE FROM work_ad WHERE user_id = ?", args: []interface{}{id}},
	}

	for _, cleanup := range cleanupQueries {
		if _, err = tx.ExecContext(ctx, cleanup.query, cleanup.args...); err != nil {
			return err
		}
	}

	result, err := tx.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return repositories.ErrUserNotFound
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (s *UserService) GetUserByPhone(ctx context.Context, phone string) (models.User, error) {
	return s.UserRepo.GetUserByPhone(ctx, phone)
}

func (s *UserService) GetUsersByRole(ctx context.Context, role string) ([]models.User, error) {
	return s.UserRepo.GetUsersByRole(ctx, role)
}

func (s *UserService) GetAllUsers(ctx context.Context) ([]models.User, error) {
	return s.UserRepo.GetAllUsers(ctx)
}

func (s *UserService) UpdatePassword(ctx context.Context, userID int, oldPassword, newPassword string) error {
	user, err := s.UserRepo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return models.ErrInvalidPassword
	}

	// Hash the new password before saving
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.UserRepo.UpdatePassword(ctx, userID, string(hashedPassword))
}

func (s *UserService) UserLogOut(ctx context.Context, UserID int) error {
	return s.UserRepo.UserLogOut(ctx, UserID)
}

func (s *UserService) ChangeNumber(ctx context.Context, number string) (models.SignUpResponse, error) {
	existingUser, err := s.UserRepo.GetUserByPhone1(ctx, number)
	if err != nil {
		return models.SignUpResponse{}, err
	}
	if existingUser.Phone != "" {
		return models.SignUpResponse{}, models.ErrDuplicatePhone
	}

	code := generateVerificationCode()
	message := fmt.Sprintf("Ваш код подтверждения: %s. Код отправлен компанией https://nusacorp.com/", code)
	apiKey := "kzfaad0a91a4b498db593b78414dfdaa2c213b8b8996afa325a223543481efeb11dd11"

	if err := s.sendSMS(apiKey, number, message); err != nil {
		return models.SignUpResponse{}, fmt.Errorf("ошибка при отправке SMS: %v", err)
	}

	return models.SignUpResponse{
		VerificationCode: code,
	}, nil
}

func (s *UserService) sendEmailSMTP(toEmail, subject, body string) error {
	host := os.Getenv("SMTP_HOST") // smtp.gmail.com
	port := os.Getenv("SMTP_PORT") // 465
	username := os.Getenv("SMTP_USER")
	password := os.Getenv("SMTP_PASS")

	addr := host + ":" + port

	// Подключаемся СРАЗУ по TLS (важно для 465)
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp4", addr, &tls.Config{ServerName: host})
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Quit()

	// AUTH только App Password
	if err = c.Auth(smtp.PlainAuth("", username, password, host)); err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	// Заголовки (UTF-8) и пустая строка перед телом
	msg := []byte(
		"From: Barlyq Qyzmet <" + username + ">\r\n" +
			"To: " + toEmail + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"\r\n" + body + "\r\n",
	)

	if err = c.Mail(username); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}
	if err = c.Rcpt(toEmail); err != nil {
		return fmt.Errorf("RCPT TO: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}
	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("write msg: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("close data: %w", err)
	}

	return nil
}

func (s *UserService) SendCodeToEmail(ctx context.Context, email string) error {
	existingUser, err := s.UserRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return err
	}
	if existingUser.Email != "" {
		return models.ErrDuplicateEmail
	}

	code := generateVerificationCode()
	subject := "Код подтверждения регистрации"
	body := fmt.Sprintf("Ваш код подтверждения: %s\n\nОт компании https://nusacorp.com/", code)

	if err := s.sendEmailSMTP(email, subject, body); err != nil {
		return fmt.Errorf("ошибка при отправке email: %v", err)
	}

	if err := s.UserRepo.ClearVerificationCodeByEmail(ctx, email); err != nil {
		return err
	}

	if err := s.UserRepo.SaveEmailVerificationCode(ctx, email, code); err != nil {
		return fmt.Errorf("не удалось сохранить код подтверждения: %v", err)
	}

	return nil
}

func (s *UserService) sendEmailMailgun(toEmail, subject, body string) error {
	apiKey := os.Getenv("MAILGUN_API_KEY")
	domain := os.Getenv("MAILGUN_DOMAIN")
	from := "postmaster@" + domain

	apiURL := fmt.Sprintf("https://api.mailgun.net/v3/%s/messages", domain)

	data := url.Values{}
	data.Set("from", from)
	data.Set("to", toEmail)
	data.Set("subject", subject)
	data.Set("text", body)

	req, err := http.NewRequest("POST", apiURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %v", err)
	}
	req.SetBasicAuth("api", apiKey)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка отправки письма: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("ошибка Mailgun: %s", respBody)
	}

	return nil
}

func (s *UserService) ChangeEmail(ctx context.Context, email string) (models.SignUpResponse, error) {
	existingUser, err := s.UserRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return models.SignUpResponse{}, err
	}
	if existingUser.Email != "" {
		return models.SignUpResponse{}, models.ErrDuplicateEmail
	}

	code := generateVerificationCode()
	subject := "Код подтверждения почты"
	body := fmt.Sprintf("Ваш код подтверждения: %s\n\nОт компании https://nusacorp.com/", code)

	if err := s.sendEmailMailgun(email, subject, body); err != nil {
		return models.SignUpResponse{}, fmt.Errorf("ошибка при отправке email: %v", err)
	}

	return models.SignUpResponse{
		User: models.User{
			Email: email,
		},
		VerificationCode: code,
	}, nil
}

func (s *UserService) ChangeCityForUser(ctx context.Context, userID int, cityID int) error {
	return s.UserRepo.ChangeCityForUser(ctx, userID, cityID)
}

func (s *UserService) UpdateToWorker(ctx context.Context, user models.User) (models.User, error) {
	return s.UserRepo.UpdateWorkerProfile(ctx, user)
}

func (s *UserService) CheckUserDuplicate(ctx context.Context, req models.User) (bool, error) {
	if strings.TrimSpace(req.Email) == "" {
		return false, fmt.Errorf("email is required for verification")
	}

	taken, err := s.UserRepo.IsPhoneOrEmailTaken(ctx, req.Phone, req.Email)
	if err != nil {
		return false, err
	}
	if taken {
		return true, nil
	}

	code := generateVerificationCode()
	subject := "Код подтверждения регистрации"
	body := fmt.Sprintf("Ваш код подтверждения: %s\n\nОт компании https://nusacorp.com/", code)

	if err := s.sendEmailSMTP(req.Email, subject, body); err != nil {
		return false, fmt.Errorf("не удалось отправить email: %v", err)
	}

	if err := s.UserRepo.ClearVerificationCodeByEmail(ctx, req.Email); err != nil {
		return false, err
	}

	if err := s.UserRepo.SaveEmailVerificationCode(ctx, req.Email, code); err != nil {
		return false, fmt.Errorf("не удалось сохранить код подтверждения: %v", err)
	}

	return false, nil
}

func (s *UserService) SendResetCode(ctx context.Context, email string) error {
	code := generateVerificationCode()
	subject := "Восстановление пароля"
	body := fmt.Sprintf("Ваш код подтверждения для сброса пароля: %s", code)
	if err := s.sendMailgunEmail(email, subject, body); err != nil {
		log.Printf("mailgun send failed, falling back to SMTP: %v", err)
		if errSMTP := s.sendEmailSMTP(email, subject, body); errSMTP != nil {
			return fmt.Errorf("не удалось отправить email: %w", errSMTP)
		}
	}
	return s.UserRepo.SaveResetCode(ctx, email, code)
}

func (s *UserService) VerifyResetCode(ctx context.Context, email, code string) (bool, error) {
	return s.UserRepo.VerifyResetCode(ctx, email, code)
}

func (s *UserService) ResetPassword(ctx context.Context, email, newPassword string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.UserRepo.UpdatePasswordEmail(ctx, email, string(hashed))
}

func (s *UserService) sendMailgunEmail(to, subject, body string) error {
	apiKey := os.Getenv("MAILGUN_API_KEY")
	domain := os.Getenv("MAILGUN_DOMAIN")
	if apiKey == "" || domain == "" {
		return errMailgunNotConfigured
	}
	apiUrl := fmt.Sprintf("https://api.mailgun.net/v3/%s/messages", domain)

	data := url.Values{}
	data.Set("from", "Nusa Corp <noreply@nusacorp.com>")
	data.Set("to", to)
	data.Set("subject", subject)
	data.Set("text", body)

	req, err := http.NewRequest("POST", apiUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth("api", apiKey)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mailgun error: %s", string(body))
	}
	return nil
}
