package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	_ "github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"io/ioutil"
	"log"
	"math/rand"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
	"naimuBack/utils"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type tokenClaims struct {
	jwt.StandardClaims
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
}
type UserService struct {
	UserRepo     *repositories.UserRepository
	TokenManager *utils.Manager
}

const (
	salt       = "sadasdnsadna"
	tokenTTL   = 120 * time.Minute
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

func (s *UserService) SignUp(ctx context.Context, user models.User) (models.SignUpResponse, error) {
	existingUser1, err := s.UserRepo.GetUserByEmail(ctx, user.Email)
	if err != nil {
		return models.SignUpResponse{}, err
	}
	if existingUser1.Email != "" {
		return models.SignUpResponse{}, models.ErrDuplicateEmail
	}

	existingUser2, err := s.UserRepo.GetUserByPhone1(ctx, user.Phone)
	if err != nil {
		return models.SignUpResponse{}, err
	}
	if existingUser2.Phone != "" {
		return models.SignUpResponse{}, models.ErrDuplicatePhone
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return models.SignUpResponse{}, err
	}
	user.Password = string(hashedPassword)

	userID, err := s.UserRepo.CreateUser(ctx, user)
	if err != nil {
		return models.SignUpResponse{}, err
	}
	user.ID = userID.ID

	// ✅ Генерация и отправка SMS
	code := generateVerificationCode()
	message := fmt.Sprintf("Ваш код подтверждения: %s", code)
	apiKey := "kzfaad0a91a4b498db593b78414dfdaa2c213b8b8996afa325a223543481efeb11dd11" // Вынеси в конфиг позже

	if err := s.sendSMS(apiKey, user.Phone, message); err != nil {
		return models.SignUpResponse{}, fmt.Errorf("ошибка при отправке SMS: %v", err)
	}

	return models.SignUpResponse{
		User:             user,
		VerificationCode: code,
	}, nil
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

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &tokenClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(tokenTTL).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
		UserID: user.ID,
		Role:   user.Role,
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

func (s *UserService) UpdateUser(ctx context.Context, user models.User) (models.User, error) {
	existingUser1, err := s.UserRepo.GetUserByEmail(ctx, user.Email)
	if err != nil {
		return models.User{}, err
	}
	if existingUser1.Email != "" {
		return models.User{}, errors.New("user with this email already exists")
	}

	existingUser2, err := s.UserRepo.GetUserByPhone1(ctx, user.Phone)
	if err != nil {
		return models.User{}, err
	}
	if existingUser2.Phone != "" {
		return models.User{}, errors.New("user with this phone already exists")
	}
	return s.UserRepo.UpdateUser(ctx, user)
}

func (s *UserService) DeleteUser(ctx context.Context, id int) error {
	return s.UserRepo.DeleteUser(ctx, id)
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
