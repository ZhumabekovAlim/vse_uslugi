package main

import (
	_ "context"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/google/uuid"
	_ "github.com/joho/godotenv"
	_ "google.golang.org/api/option"
	"log"
	"naimuBack/internal/handlers"
	_ "naimuBack/internal/models"
	"naimuBack/internal/repositories"
	services "naimuBack/internal/services"
	_ "naimuBack/utils"
	"net/http"
)

type application struct {
	errorLog            *log.Logger
	infoLog             *log.Logger
	userHandler         *handlers.UserHandler
	userRepo            *repositories.UserRepository
	serviceHandler      *handlers.ServiceHandler
	serviceRepo         *repositories.ServiceRepository
	categoryHandler     *handlers.CategoryHandler
	categoryRepo        *repositories.CategoryRepository
	reviewsHandler      *handlers.ReviewHandler
	reviewsRepo         *repositories.ReviewRepository
	serviceFavorite     *handlers.ServiceFavoriteHandler
	serviceFavoriteRepo *repositories.ServiceFavoriteRepository
	subcategoryHandler  handlers.SubcategoryHandler
	subcategoryRepo     repositories.SubcategoryRepository
	cityHandler         handlers.CityHandler
	cityRepo            repositories.CityRepository
	wsManager           *WebSocketManager
	chatHandler         *handlers.ChatHandler
	messageHandler      *handlers.MessageHandler
	db                  *sql.DB

	// authService *services/*/.AuthService
}

func initializeApp(db *sql.DB, errorLog, infoLog *log.Logger) *application {
	// Repositories\
	userRepo := repositories.UserRepository{DB: db}
	serviceRepo := repositories.ServiceRepository{DB: db}
	categoryRepo := repositories.CategoryRepository{DB: db}
	reviewsRepo := repositories.ReviewRepository{DB: db}
	serviceFavoriteRepo := repositories.ServiceFavoriteRepository{DB: db}
	subcategoryRepo := repositories.SubcategoryRepository{DB: db}
	cityRepo := repositories.CityRepository{DB: db}
	// Services
	userService := &services.UserService{UserRepo: &userRepo}
	serviceService := &services.ServiceService{ServiceRepo: &serviceRepo}
	categoryService := &services.CategoryService{CategoryRepo: &categoryRepo}
	reviewsService := &services.ReviewService{ReviewsRepo: &reviewsRepo}
	serviceFavoritesService := &services.ServiceFavoriteService{ServiceFavoriteRepo: &serviceFavoriteRepo}
	subcategoryService := services.SubcategoryService{SubcategoryRepo: &subcategoryRepo}
	cityService := services.CityService{CityRepo: &cityRepo}
	// authService := &services.AuthService{DB: db}

	// Handlers
	userHandler := &handlers.UserHandler{Service: userService}
	serviceHandler := &handlers.ServiceHandler{Service: serviceService}
	categoryHandler := &handlers.CategoryHandler{Service: categoryService}
	reviewHandler := &handlers.ReviewHandler{Service: reviewsService}
	serviceFavoriteHandler := &handlers.ServiceFavoriteHandler{Service: serviceFavoritesService}
	subcategoryHandler := handlers.SubcategoryHandler{Service: &subcategoryService}
	cityHandler := handlers.CityHandler{Service: &cityService}

	// Chat
	wsManager := NewWebSocketManager()
	// Создание репозитория, сервиса и обработчика для чатов
	chatRepo := &repositories.ChatRepository{Db: db}
	chatService := &services.ChatService{ChatRepo: chatRepo}
	chatHandler := &handlers.ChatHandler{ChatService: chatService}

	// Создание репозитория, сервиса и обработчика для сообщений
	messageRepo := &repositories.MessageRepository{Db: db}
	messageService := &services.MessageService{MessageRepo: messageRepo}
	messageHandler := &handlers.MessageHandler{MessageService: messageService}

	return &application{
		errorLog:           errorLog,
		infoLog:            infoLog,
		userHandler:        userHandler,
		serviceHandler:     serviceHandler,
		categoryHandler:    categoryHandler,
		reviewsHandler:     reviewHandler,
		serviceFavorite:    serviceFavoriteHandler,
		subcategoryHandler: subcategoryHandler,
		cityHandler:        cityHandler,
		chatHandler:        chatHandler,
		messageHandler:     messageHandler,
		wsManager:          wsManager,
		db:                 db,
		//authService:    authService,
	}
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Printf("Failed to open DB: %v", err)
		return nil, err
	}
	if err = db.Ping(); err != nil {
		log.Printf("Failed to ping DB: %v", err)
		return nil, err
	}
	db.SetMaxIdleConns(35)
	log.Println("Successfully connected to database")
	return db, nil
}

func addSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}
