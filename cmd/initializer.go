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
	errorLog                   *log.Logger
	infoLog                    *log.Logger
	userHandler                *handlers.UserHandler
	userRepo                   *repositories.UserRepository
	serviceHandler             *handlers.ServiceHandler
	serviceRepo                *repositories.ServiceRepository
	categoryHandler            *handlers.CategoryHandler
	categoryRepo               *repositories.CategoryRepository
	rentCategoryHandler        *handlers.RentCategoryHandler
	rentCategoryRepo           *repositories.RentCategoryRepository
	workCategoryHandler        *handlers.WorkCategoryHandler
	workCategoryRepo           *repositories.WorkCategoryRepository
	reviewsHandler             *handlers.ReviewHandler
	reviewsRepo                *repositories.ReviewRepository
	serviceFavorite            *handlers.ServiceFavoriteHandler
	serviceFavoriteRepo        *repositories.ServiceFavoriteRepository
	subcategoryHandler         handlers.SubcategoryHandler
	subcategoryRepo            repositories.SubcategoryRepository
	rentSubcategoryHandler     handlers.RentSubcategoryHandler
	rentSubcategoryRepo        repositories.RentSubcategoryRepository
	workSubcategoryHandler     handlers.WorkSubcategoryHandler
	workSubcategoryRepo        repositories.WorkSubcategoryRepository
	cityHandler                handlers.CityHandler
	cityRepo                   repositories.CityRepository
	wsManager                  *WebSocketManager
	chatHandler                *handlers.ChatHandler
	messageHandler             *handlers.MessageHandler
	db                         *sql.DB
	complaintHandler           *handlers.ComplaintHandler
	complaintRepo              *repositories.ComplaintRepository
	serviceResponseHandler     *handlers.ServiceResponseHandler
	serviceResponseRepo        *repositories.ServiceResponseRepository
	serviceConfirmationHandler *handlers.ServiceConfirmationHandler
	serviceConfirmationRepo    *repositories.ServiceConfirmationRepository
	userResponsesHandler       *handlers.UserResponsesHandler
	userResponsesRepo          *repositories.UserResponsesRepository
	userReviewsHandler         *handlers.UserReviewsHandler
	userReviewsRepo            *repositories.UserReviewsRepository
	workHandler                *handlers.WorkHandler
	workRepo                   *repositories.WorkRepository
	rentHandler                *handlers.RentHandler
	rentRepo                   *repositories.RentRepository
	workReviewHandler          *handlers.WorkReviewHandler
	workReviewRepo             *repositories.WorkReviewRepository
	workResponseHandler        *handlers.WorkResponseHandler
	workResponseRepo           *repositories.WorkResponseRepository
	workFavoriteHandler        *handlers.WorkFavoriteHandler
	workFavoriteRepo           *repositories.WorkFavoriteRepository
	rentReviewHandler          *handlers.RentReviewHandler
	rentReviewRepo             *repositories.RentReviewRepository
	rentResponseHandler        *handlers.RentResponseHandler
	rentResponseRepo           *repositories.RentResponseRepository
	rentFavoriteHandler        *handlers.RentFavoriteHandler
	rentFavoriteRepo           *repositories.RentFavoriteRepository
	adHandler                  *handlers.AdHandler
	adRepo                     *repositories.AdRepository
	adReviewHandler            *handlers.AdReviewHandler
	adReviewRepo               *repositories.AdReviewRepository
	adResponseHandler          *handlers.AdResponseHandler
	adResponseRepo             *repositories.AdResponseRepository
	adFavoriteHandler          *handlers.AdFavoriteHandler
	adFavoriteRepo             *repositories.AdFavoriteRepository
	workAdHandler              *handlers.WorkAdHandler
	workAdRepo                 *repositories.WorkAdRepository
	workAdReviewHandler        *handlers.WorkAdReviewHandler
	workAdReviewRepo           *repositories.WorkAdReviewRepository
	workAdResponseHandler      *handlers.WorkAdResponseHandler
	workAdResponseRepo         *repositories.WorkAdResponseRepository
	workAdFavoriteHandler      *handlers.WorkAdFavoriteHandler
	workAdFavoriteRepo         *repositories.WorkAdFavoriteRepository
	rentAdHandler              *handlers.RentAdHandler
	rentAdRepo                 *repositories.AdRepository
	rentAdReviewHandler        *handlers.RentAdReviewHandler
	rentAdReviewRepo           *repositories.AdReviewRepository
	rentAdResponseHandler      *handlers.RentAdResponseHandler
	rentAdResponseRepo         *repositories.AdResponseRepository
	rentAdFavoriteHandler      *handlers.RentAdFavoriteHandler
	rentAdFavoriteRepo         *repositories.AdFavoriteRepository

	// authService *services/*/.AuthService
}

func initializeApp(db *sql.DB, errorLog, infoLog *log.Logger) *application {
	// Repositories\
	userRepo := repositories.UserRepository{DB: db}
	serviceRepo := repositories.ServiceRepository{DB: db}
	categoryRepo := repositories.CategoryRepository{DB: db}
	rentCategoryRepo := repositories.RentCategoryRepository{DB: db}
	workCategoryRepo := repositories.WorkCategoryRepository{DB: db}
	reviewsRepo := repositories.ReviewRepository{DB: db}
	serviceFavoriteRepo := repositories.ServiceFavoriteRepository{DB: db}
	subcategoryRepo := repositories.SubcategoryRepository{DB: db}
	rentSubcategoryRepo := repositories.RentSubcategoryRepository{DB: db}
	workSubcategoryRepo := repositories.WorkSubcategoryRepository{DB: db}
	cityRepo := repositories.CityRepository{DB: db}
	complaintRepo := repositories.ComplaintRepository{DB: db}
	chatRepo := repositories.ChatRepository{Db: db}
	serviceResponseRepo := repositories.ServiceResponseRepository{DB: db}
	serviceConfirmationRepo := repositories.ServiceConfirmationRepository{DB: db}
	userResponsesRepo := repositories.UserResponsesRepository{DB: db}
	userReviewsRepo := repositories.UserReviewsRepository{DB: db}
	workRepo := repositories.WorkRepository{DB: db}
	rentRepo := repositories.RentRepository{DB: db}
	workReviewRepo := repositories.WorkReviewRepository{DB: db}
	workResponseRepo := repositories.WorkResponseRepository{DB: db}
	workFavoriteRepo := repositories.WorkFavoriteRepository{DB: db}
	rentReviewRepo := repositories.RentReviewRepository{DB: db}
	rentResponseRepo := repositories.RentResponseRepository{DB: db}
	rentFavoriteRepo := repositories.RentFavoriteRepository{DB: db}
	adRepo := repositories.AdRepository{DB: db}
	adReviewRepo := repositories.AdReviewRepository{DB: db}
	adResponseRepo := repositories.AdResponseRepository{DB: db}
	adFavoriteRepo := repositories.AdFavoriteRepository{DB: db}
	workAdRepo := repositories.WorkAdRepository{DB: db}
	workAdReviewRepo := repositories.WorkAdReviewRepository{DB: db}
	workAdResponseRepo := repositories.WorkAdResponseRepository{DB: db}
	workAdFavoriteRepo := repositories.WorkAdFavoriteRepository{DB: db}
	rentAdRepo := repositories.RentAdRepository{DB: db}
	rentAdReviewRepo := repositories.RentAdReviewRepository{DB: db}
	rentAdResponseRepo := repositories.RentAdResponseRepository{DB: db}
	rentAdFavoriteRepo := repositories.RentAdFavoriteRepository{DB: db}
	// Services
	userService := &services.UserService{UserRepo: &userRepo}
	serviceService := &services.ServiceService{ServiceRepo: &serviceRepo}
	categoryService := &services.CategoryService{CategoryRepo: &categoryRepo}
	rentCategoryService := &services.RentCategoryService{CategoryRepo: &rentCategoryRepo}
	workCategoryService := &services.WorkCategoryService{CategoryRepo: &workCategoryRepo}
	reviewsService := &services.ReviewService{ReviewsRepo: &reviewsRepo}
	serviceFavoritesService := &services.ServiceFavoriteService{ServiceFavoriteRepo: &serviceFavoriteRepo}
	subcategoryService := services.SubcategoryService{SubcategoryRepo: &subcategoryRepo}
	rentSubcategoryService := services.RentSubcategoryService{SubcategoryRepo: &rentSubcategoryRepo}
	workSubcategoryService := services.WorkSubcategoryService{SubcategoryRepo: &workSubcategoryRepo}
	cityService := services.CityService{CityRepo: &cityRepo}
	complaintService := services.ComplaintService{ComplaintRepo: &complaintRepo}
	serviceResponseService := &services.ServiceResponseService{ServiceResponseRepo: &serviceResponseRepo, ServiceRepo: &serviceRepo, ChatRepo: &chatRepo, ConfirmationRepo: &serviceConfirmationRepo}
	serviceConfirmationService := &services.ServiceConfirmationService{ConfirmationRepo: &serviceConfirmationRepo}
	userResponsesService := &services.UserResponsesService{ResponsesRepo: &userResponsesRepo}
	userReviewsService := &services.UserReviewsService{ReviewsRepo: &userReviewsRepo}
	workService := &services.WorkService{WorkRepo: &workRepo}
	rentService := &services.RentService{RentRepo: &rentRepo}
	workReviewService := &services.WorkReviewService{WorkReviewsRepo: &workReviewRepo}
	workResponseService := &services.WorkResponseService{WorkResponseRepo: &workResponseRepo}
	workFavoriteService := &services.WorkFavoriteService{WorkFavoriteRepo: &workFavoriteRepo}
	rentReviewService := &services.RentReviewService{RentReviewsRepo: &rentReviewRepo}
	rentResponseService := &services.RentResponseService{RentResponseRepo: &rentResponseRepo}
	rentFavoriteService := &services.RentFavoriteService{RentFavoriteRepo: &rentFavoriteRepo}
	adService := &services.AdService{AdRepo: &adRepo}
	adReviewService := &services.AdReviewService{AdReviewsRepo: &adReviewRepo}
	adResponseService := &services.AdResponseService{AdResponseRepo: &adResponseRepo}
	adFavoriteService := &services.AdFavoriteService{AdFavoriteRepo: &adFavoriteRepo}
	workAdService := &services.WorkAdService{WorkAdRepo: &workAdRepo}
	workAdReviewService := &services.WorkAdReviewService{WorkAdReviewsRepo: &workAdReviewRepo}
	workAdResponseService := &services.WorkAdResponseService{WorkAdResponseRepo: &workAdResponseRepo}
	workAdFavoriteService := &services.WorkAdFavoriteService{WorkAdFavoriteRepo: &workAdFavoriteRepo}
	rentAdService := &services.RentAdService{RentAdRepo: &rentAdRepo}
	rentAdReviewService := &services.RentAdReviewService{RentAdReviewsRepo: &rentAdReviewRepo}
	rentAdResponseService := &services.RentAdResponseService{RentAdResponseRepo: &rentAdResponseRepo}
	rentAdFavoriteService := &services.RentAdFavoriteService{RentAdFavoriteRepo: &rentAdFavoriteRepo}
	// authService := &services.AuthService{DB: db}

	// Handlers
	userHandler := &handlers.UserHandler{Service: userService}
	serviceHandler := &handlers.ServiceHandler{Service: serviceService}
	categoryHandler := &handlers.CategoryHandler{Service: categoryService}
	rentCategoryHandler := &handlers.RentCategoryHandler{Service: rentCategoryService}
	workCategoryHandler := &handlers.WorkCategoryHandler{Service: workCategoryService}
	reviewHandler := &handlers.ReviewHandler{Service: reviewsService}
	serviceFavoriteHandler := &handlers.ServiceFavoriteHandler{Service: serviceFavoritesService}
	subcategoryHandler := handlers.SubcategoryHandler{Service: &subcategoryService}
	rentSubcategoryHandler := handlers.RentSubcategoryHandler{Service: &rentSubcategoryService}
	workSubcategoryHandler := handlers.WorkSubcategoryHandler{Service: &workSubcategoryService}
	cityHandler := handlers.CityHandler{Service: &cityService}
	complaintHandler := &handlers.ComplaintHandler{Service: &complaintService}
	serviceResponseHandler := &handlers.ServiceResponseHandler{Service: serviceResponseService}
	serviceConfirmationHandler := &handlers.ServiceConfirmationHandler{Service: serviceConfirmationService}
	userResponsesHandler := &handlers.UserResponsesHandler{Service: userResponsesService}
	userReviewsHandler := &handlers.UserReviewsHandler{Service: userReviewsService}
	workHandler := &handlers.WorkHandler{Service: workService}
	rentHandler := &handlers.RentHandler{Service: rentService}
	workReviewHandler := &handlers.WorkReviewHandler{Service: workReviewService}
	workResponseHandler := &handlers.WorkResponseHandler{Service: workResponseService}
	workFavoriteHandler := &handlers.WorkFavoriteHandler{Service: workFavoriteService}
	rentReviewHandler := &handlers.RentReviewHandler{Service: rentReviewService}
	rentResponseHandler := &handlers.RentResponseHandler{Service: rentResponseService}
	rentFavoriteHandler := &handlers.RentFavoriteHandler{Service: rentFavoriteService}
	adHandler := &handlers.AdHandler{Service: adService}
	adReviewHandler := &handlers.AdReviewHandler{Service: adReviewService}
	adResponseHandler := &handlers.AdResponseHandler{Service: adResponseService}
	adFavoriteHandler := &handlers.AdFavoriteHandler{Service: adFavoriteService}
	workAdHandler := &handlers.WorkAdHandler{Service: workAdService}
	workAdReviewHandler := &handlers.WorkAdReviewHandler{Service: workAdReviewService}
	workAdResponseHandler := &handlers.WorkAdResponseHandler{Service: workAdResponseService}
	workAdFavoriteHandler := &handlers.WorkAdFavoriteHandler{Service: workAdFavoriteService}
	rentAdHandler := &handlers.RentAdHandler{Service: rentAdService}
	rentADReviewHandler := &handlers.RentAdReviewHandler{Service: rentAdReviewService}
	rentAdResponseHandler := &handlers.RentAdResponseHandler{Service: rentAdResponseService}
	rentAdFavoriteHandler := &handlers.RentAdFavoriteHandler{Service: rentAdFavoriteService}

	// Chat
	wsManager := NewWebSocketManager()
	// Создание сервиса и обработчика для чатов
	chatService := &services.ChatService{ChatRepo: &chatRepo}
	chatHandler := &handlers.ChatHandler{ChatService: chatService}

	// Создание репозитория, сервиса и обработчика для сообщений
	messageRepo := &repositories.MessageRepository{Db: db}
	messageService := &services.MessageService{MessageRepo: messageRepo}
	messageHandler := &handlers.MessageHandler{MessageService: messageService}

	return &application{
		errorLog:                   errorLog,
		infoLog:                    infoLog,
		userHandler:                userHandler,
		serviceHandler:             serviceHandler,
		categoryHandler:            categoryHandler,
		rentCategoryHandler:        rentCategoryHandler,
		workCategoryHandler:        workCategoryHandler,
		reviewsHandler:             reviewHandler,
		serviceFavorite:            serviceFavoriteHandler,
		subcategoryHandler:         subcategoryHandler,
		rentSubcategoryHandler:     rentSubcategoryHandler,
		workSubcategoryHandler:     workSubcategoryHandler,
		cityHandler:                cityHandler,
		chatHandler:                chatHandler,
		messageHandler:             messageHandler,
		wsManager:                  wsManager,
		db:                         db,
		complaintHandler:           complaintHandler,
		serviceResponseHandler:     serviceResponseHandler,
		serviceConfirmationHandler: serviceConfirmationHandler,
		userResponsesHandler:       userResponsesHandler,
		userReviewsHandler:         userReviewsHandler,
		workHandler:                workHandler,
		rentHandler:                rentHandler,
		workReviewHandler:          workReviewHandler,
		workResponseHandler:        workResponseHandler,
		workFavoriteHandler:        workFavoriteHandler,
		rentReviewHandler:          rentReviewHandler,
		rentResponseHandler:        rentResponseHandler,
		rentFavoriteHandler:        rentFavoriteHandler,
		adHandler:                  adHandler,
		adReviewHandler:            adReviewHandler,
		adResponseHandler:          adResponseHandler,
		adFavoriteHandler:          adFavoriteHandler,
		workAdHandler:              workAdHandler,
		workAdReviewHandler:        workAdReviewHandler,
		workAdResponseHandler:      workAdResponseHandler,
		workAdFavoriteHandler:      workAdFavoriteHandler,
		rentAdHandler:              rentAdHandler,
		rentAdReviewHandler:        rentADReviewHandler,
		rentAdResponseHandler:      rentAdResponseHandler,
		rentAdFavoriteHandler:      rentAdFavoriteHandler,
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
