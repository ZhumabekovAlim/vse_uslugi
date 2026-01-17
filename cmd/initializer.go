package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"naimuBack/internal/ai"
	"naimuBack/internal/courier"
	"naimuBack/internal/handlers"
	"naimuBack/internal/models"
	"naimuBack/internal/repositories"
	services "naimuBack/internal/services"
	"naimuBack/internal/taxi"
	_ "naimuBack/utils"
	"net/http"
	"os"
	"strings"

	firebase "firebase.google.com/go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/google/uuid"
	_ "github.com/joho/godotenv"
	"google.golang.org/api/option"
	_ "google.golang.org/api/option"
)

type application struct {
	errorLog                   *log.Logger
	infoLog                    *log.Logger
	userHandler                *handlers.UserHandler
	userRepo                   *repositories.UserRepository
	serviceHandler             *handlers.ServiceHandler
	globalSearchHandler        *handlers.GlobalSearchHandler
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
	locationManager            *LocationManager
	locationHandler            *handlers.LocationHandler
	locationRepo               *repositories.LocationRepository
	locationService            *services.LocationService
	chatHandler                *handlers.ChatHandler
	messageHandler             *handlers.MessageHandler
	db                         *sql.DB
	complaintHandler           *handlers.ComplaintHandler
	complaintRepo              *repositories.ComplaintRepository
	adComplaintHandler         *handlers.AdComplaintHandler
	adComplaintRepo            *repositories.AdComplaintRepository
	workComplaintHandler       *handlers.WorkComplaintHandler
	workComplaintRepo          *repositories.WorkComplaintRepository
	workAdComplaintHandler     *handlers.WorkAdComplaintHandler
	workAdComplaintRepo        *repositories.WorkAdComplaintRepository
	rentComplaintHandler       *handlers.RentComplaintHandler
	rentComplaintRepo          *repositories.RentComplaintRepository
	rentAdComplaintHandler     *handlers.RentAdComplaintHandler
	rentAdComplaintRepo        *repositories.RentAdComplaintRepository
	serviceResponseHandler     *handlers.ServiceResponseHandler
	serviceResponseRepo        *repositories.ServiceResponseRepository
	serviceConfirmationHandler *handlers.ServiceConfirmationHandler
	serviceConfirmationRepo    *repositories.ServiceConfirmationRepository
	userResponsesHandler       *handlers.UserResponsesHandler
	userResponsesRepo          *repositories.UserResponsesRepository
	responseUsersHandler       *handlers.ResponseUsersHandler
	responseUsersRepo          *repositories.ResponseUsersRepository
	userReviewsHandler         *handlers.UserReviewsHandler
	userReviewsRepo            *repositories.UserReviewsRepository
	userItemsHandler           *handlers.UserItemsHandler
	userItemsRepo              *repositories.UserItemsRepository
	businessHandler            *handlers.BusinessHandler
	businessRepo               *repositories.BusinessRepository
	businessService            *services.BusinessService
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
	adConfirmationHandler      *handlers.AdConfirmationHandler
	adConfirmationRepo         *repositories.AdConfirmationRepository

	workAdHandler             *handlers.WorkAdHandler
	workAdRepo                *repositories.WorkAdRepository
	workAdReviewHandler       *handlers.WorkAdReviewHandler
	workAdReviewRepo          *repositories.WorkAdReviewRepository
	workAdResponseHandler     *handlers.WorkAdResponseHandler
	workAdResponseRepo        *repositories.WorkAdResponseRepository
	workAdFavoriteHandler     *handlers.WorkAdFavoriteHandler
	workAdFavoriteRepo        *repositories.WorkAdFavoriteRepository
	workAdConfirmationHandler *handlers.WorkAdConfirmationHandler
	workAdConfirmationRepo    *repositories.WorkAdConfirmationRepository

	rentAdHandler             *handlers.RentAdHandler
	rentAdRepo                *repositories.AdRepository
	rentAdReviewHandler       *handlers.RentAdReviewHandler
	rentAdReviewRepo          *repositories.AdReviewRepository
	rentAdResponseHandler     *handlers.RentAdResponseHandler
	rentAdResponseRepo        *repositories.AdResponseRepository
	rentAdFavoriteHandler     *handlers.RentAdFavoriteHandler
	rentAdFavoriteRepo        *repositories.AdFavoriteRepository
	rentAdConfirmationHandler *handlers.RentAdConfirmationHandler
	rentAdConfirmationRepo    *repositories.RentAdConfirmationRepository
	workConfirmationHandler   *handlers.WorkConfirmationHandler
	workConfirmationRepo      *repositories.WorkConfirmationRepository
	rentConfirmationHandler   *handlers.RentConfirmationHandler
	rentConfirmationRepo      *repositories.RentConfirmationRepository
	subscriptionHandler       *handlers.SubscriptionHandler
	subscriptionRepo          *repositories.SubscriptionRepository
	airbapayHandler           *handlers.AirbapayHandler
	invoiceRepo               *repositories.InvoiceRepo
	topHandler                *handlers.TopHandler
	topService                *services.TopService

	assistantHandler *handlers.AssistantHandler

	iapHandler       *handlers.IAPHandler
	googleIapHandler *handlers.GoogleIAPHandler
	// authService *services/*/.AuthService
	taxiMux     http.Handler
	taxiDeps    *taxi.TaxiDeps
	courierMux  http.Handler
	courierDeps *courier.Deps

	fcmHandler *handlers.FCMHandler
}

func initializeApp(db *sql.DB, errorLog, infoLog *log.Logger) *application {

	ctx := context.Background()
	sa := option.WithCredentialsFile("/root/NaimuBack/vse_uslugi/cmd/serviceAccountKey.json")

	firebaseApp, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: "vse-uslugi-dc6ed"}, sa)
	if err != nil {
		errorLog.Fatalf("Ошибка в нахождении приложения: %v\n", err)
	}

	fcmClient, err := firebaseApp.Messaging(ctx)
	if err != nil {
		errorLog.Fatalf("Ошибка при неверном ID устройства: %v\n", err)
	}
	// FCM Handler
	fcmHandler := handlers.NewFCMHandler(fcmClient, db)

	// Repositories\
	invoiceRepo := repositories.NewInvoiceRepo(db)
	iapRepo := repositories.NewIAPRepository(db)
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
	adComplaintRepo := repositories.AdComplaintRepository{DB: db}
	workComplaintRepo := repositories.WorkComplaintRepository{DB: db}
	workAdComplaintRepo := repositories.WorkAdComplaintRepository{DB: db}
	rentComplaintRepo := repositories.RentComplaintRepository{DB: db}
	rentAdComplaintRepo := repositories.RentAdComplaintRepository{DB: db}
	chatRepo := repositories.ChatRepository{Db: db}
	messageRepo := repositories.MessageRepository{Db: db}
	locationRepo := repositories.LocationRepository{DB: db}
	serviceResponseRepo := repositories.ServiceResponseRepository{DB: db}
	serviceConfirmationRepo := repositories.ServiceConfirmationRepository{DB: db}
	userResponsesRepo := repositories.UserResponsesRepository{DB: db}
	userReviewsRepo := repositories.UserReviewsRepository{DB: db}
	userItemsRepo := repositories.UserItemsRepository{DB: db}
	responseUsersRepo := repositories.ResponseUsersRepository{DB: db}
	businessRepo := repositories.BusinessRepository{DB: db}
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
	adConfirmationRepo := repositories.AdConfirmationRepository{DB: db}
	workConfirmationRepo := repositories.WorkConfirmationRepository{DB: db}
	workAdConfirmationRepo := repositories.WorkAdConfirmationRepository{DB: db}
	rentConfirmationRepo := repositories.RentConfirmationRepository{DB: db}
	rentAdConfirmationRepo := repositories.RentAdConfirmationRepository{DB: db}
	subscriptionRepo := repositories.SubscriptionRepository{DB: db}
	topRepo := repositories.NewTopRepository(db)

	// Services
	userService := &services.UserService{UserRepo: &userRepo, ServiceRepo: &serviceRepo, AdRepo: &adRepo, RentRepo: &rentRepo, WorkRepo: &workRepo, RentAdRepo: &rentAdRepo, WorkAdRepo: &workAdRepo}
	serviceService := &services.ServiceService{ServiceRepo: &serviceRepo, SubscriptionRepo: &subscriptionRepo}
	categoryService := &services.CategoryService{CategoryRepo: &categoryRepo}
	rentCategoryService := &services.RentCategoryService{CategoryRepo: &rentCategoryRepo}
	workCategoryService := &services.WorkCategoryService{CategoryRepo: &workCategoryRepo}
	reviewsService := &services.ReviewService{ReviewsRepo: &reviewsRepo, ConfirmationRepo: &serviceConfirmationRepo}
	serviceFavoritesService := &services.ServiceFavoriteService{ServiceFavoriteRepo: &serviceFavoriteRepo}
	subcategoryService := services.SubcategoryService{SubcategoryRepo: &subcategoryRepo}
	rentSubcategoryService := services.RentSubcategoryService{SubcategoryRepo: &rentSubcategoryRepo}
	workSubcategoryService := services.WorkSubcategoryService{SubcategoryRepo: &workSubcategoryRepo}
	cityService := services.CityService{CityRepo: &cityRepo}
	complaintService := services.ComplaintService{ComplaintRepo: &complaintRepo}
	adComplaintService := services.AdComplaintService{ComplaintRepo: &adComplaintRepo}
	workComplaintService := services.WorkComplaintService{ComplaintRepo: &workComplaintRepo}
	workAdComplaintService := services.WorkAdComplaintService{ComplaintRepo: &workAdComplaintRepo}
	rentComplaintService := services.RentComplaintService{ComplaintRepo: &rentComplaintRepo}
	rentAdComplaintService := services.RentAdComplaintService{ComplaintRepo: &rentAdComplaintRepo}
	serviceResponseService := &services.ServiceResponseService{ServiceResponseRepo: &serviceResponseRepo, ServiceRepo: &serviceRepo, ChatRepo: &chatRepo, ConfirmationRepo: &serviceConfirmationRepo, MessageRepo: &messageRepo, SubscriptionRepo: &subscriptionRepo, UserRepo: &userRepo, BusinessRepo: &businessRepo}
	serviceConfirmationService := &services.ServiceConfirmationService{ConfirmationRepo: &serviceConfirmationRepo}
	userResponsesService := &services.UserResponsesService{ResponsesRepo: &userResponsesRepo}
	userReviewsService := &services.UserReviewsService{ReviewsRepo: &userReviewsRepo}
	userItemsService := &services.UserItemsService{ItemsRepo: &userItemsRepo}
	businessService := &services.BusinessService{
		BusinessRepo: &businessRepo,
		UserRepo:     &userRepo,
		ChatRepo:     &chatRepo,
		ServiceRepo:  &serviceRepo,
		WorkRepo:     &workRepo,
		RentRepo:     &rentRepo,
		AdRepo:       &adRepo,
		WorkAdRepo:   &workAdRepo,
		RentAdRepo:   &rentAdRepo,
	}
	responseUsersService := &services.ResponseUsersService{Repo: &responseUsersRepo}
	workService := &services.WorkService{WorkRepo: &workRepo, SubscriptionRepo: &subscriptionRepo}
	rentService := &services.RentService{RentRepo: &rentRepo, SubscriptionRepo: &subscriptionRepo}
	workReviewService := &services.WorkReviewService{WorkReviewsRepo: &workReviewRepo, ConfirmationRepo: &workConfirmationRepo}
	workResponseService := &services.WorkResponseService{WorkResponseRepo: &workResponseRepo, WorkRepo: &workRepo, ChatRepo: &chatRepo, ConfirmationRepo: &workConfirmationRepo, MessageRepo: &messageRepo, SubscriptionRepo: &subscriptionRepo, UserRepo: &userRepo, BusinessRepo: &businessRepo}
	workFavoriteService := &services.WorkFavoriteService{WorkFavoriteRepo: &workFavoriteRepo}
	rentReviewService := &services.RentReviewService{RentReviewsRepo: &rentReviewRepo, ConfirmationRepo: &rentConfirmationRepo}
	rentResponseService := &services.RentResponseService{RentResponseRepo: &rentResponseRepo, RentRepo: &rentRepo, ChatRepo: &chatRepo, ConfirmationRepo: &rentConfirmationRepo, MessageRepo: &messageRepo, SubscriptionRepo: &subscriptionRepo, UserRepo: &userRepo, BusinessRepo: &businessRepo}
	rentFavoriteService := &services.RentFavoriteService{RentFavoriteRepo: &rentFavoriteRepo}
	adService := &services.AdService{AdRepo: &adRepo, SubscriptionRepo: &subscriptionRepo}
	adReviewService := &services.AdReviewService{AdReviewsRepo: &adReviewRepo, ConfirmationRepo: &adConfirmationRepo}
	adResponseService := &services.AdResponseService{AdResponseRepo: &adResponseRepo, AdRepo: &adRepo, ChatRepo: &chatRepo, ConfirmationRepo: &adConfirmationRepo, MessageRepo: &messageRepo, SubscriptionRepo: &subscriptionRepo, UserRepo: &userRepo, BusinessRepo: &businessRepo}
	adFavoriteService := &services.AdFavoriteService{AdFavoriteRepo: &adFavoriteRepo}
	subscriptionService := &services.SubscriptionService{Repo: &subscriptionRepo, BusinessRepo: &businessRepo}
	locationService := &services.LocationService{Repo: &locationRepo, BusinessRepo: &businessRepo}
	topService := services.NewTopService(topRepo)

	globalSearchService := &services.GlobalSearchService{
		ServiceRepo: &serviceRepo,
		AdRepo:      &adRepo,
		WorkRepo:    &workRepo,
		WorkAdRepo:  &workAdRepo,
		RentRepo:    &rentRepo,
		RentAdRepo:  &rentAdRepo,
	}

	kb, err := ai.LoadKnowledgeBase("/root/NaimuBack/vse_uslugi/kb_base/kb.json")
	if err != nil {
		errorLog.Fatalf("load knowledge base: %v", err)
	}

	var chatClient services.ChatCompletionClient
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey != "" {
		chatClient = services.NewOpenAIClient(nil, apiKey)
	}

	assistantService := services.NewAssistantService(kb, chatClient)

	airbapayCfg := services.AirbapayConfig{
		Username:   getEnv("AIRBAPAY_USERNAME", "VSEUSLUGI"),
		Password:   getEnv("AIRBAPAY_PASSWORD", "v(A3Z!_zua%V&%a"),
		TerminalID: getEnv("AIRBAPAY_TERMINAL_ID", "68e73c28a36bcb28994f2061"),
		BaseURL:    getEnv("AIRBAPAY_BASE_URL", "https://ps.airbapay.kz/acquiring-api"),

		// Куда вернуть ПОЛЬЗОВАТЕЛЯ после оплаты (фронт):
		SuccessBackURL: getEnv("AIRBAPAY_SUCCESS_BACK_URL", "https://vse-uslugi-website.vercel.app/pay/success"),
		FailureBackURL: getEnv("AIRBAPAY_FAILURE_BACK_URL", "https://vse-uslugi-website.vercel.app/pay/failure"),

		// Куда AirbaPay шлёт ВЕБХУК (бэкенд, должен быть HTTPS и доступен извне):
		CallbackURL: getEnv("AIRBAPAY_CALLBACK_URL", "https://api.barlyqqyzmet.kz/airbapay/callback"),

		// Необязательно: будут подставляться в тело create-платежа
		DefaultEmail:     getEnv("AIRBAPAY_DEFAULT_EMAIL", ""),
		DefaultPhone:     getEnv("AIRBAPAY_DEFAULT_PHONE", ""), // строго 11 цифр, без '+'
		DefaultAccountID: getEnv("AIRBAPAY_DEFAULT_ACCOUNT_ID", ""),
	}

	airbapayService, err := services.NewAirbapayService(airbapayCfg)
	if err != nil {
		errorLog.Fatalf("airbapay service init: %v", err)
	}

	workAdService := &services.WorkAdService{WorkAdRepo: &workAdRepo, SubscriptionRepo: &subscriptionRepo}
	workAdReviewService := &services.WorkAdReviewService{WorkAdReviewsRepo: &workAdReviewRepo, ConfirmationRepo: &workAdConfirmationRepo}
	workAdResponseService := &services.WorkAdResponseService{WorkAdResponseRepo: &workAdResponseRepo, WorkAdRepo: &workAdRepo, ChatRepo: &chatRepo, ConfirmationRepo: &workAdConfirmationRepo, MessageRepo: &messageRepo, SubscriptionRepo: &subscriptionRepo, UserRepo: &userRepo, BusinessRepo: &businessRepo}
	workAdFavoriteService := &services.WorkAdFavoriteService{WorkAdFavoriteRepo: &workAdFavoriteRepo}
	rentAdService := &services.RentAdService{RentAdRepo: &rentAdRepo, SubscriptionRepo: &subscriptionRepo}
	rentAdReviewService := &services.RentAdReviewService{RentAdReviewsRepo: &rentAdReviewRepo, ConfirmationRepo: &rentAdConfirmationRepo}
	rentAdResponseService := &services.RentAdResponseService{RentAdResponseRepo: &rentAdResponseRepo, RentAdRepo: &rentAdRepo, ChatRepo: &chatRepo, ConfirmationRepo: &rentAdConfirmationRepo, MessageRepo: &messageRepo, SubscriptionRepo: &subscriptionRepo, UserRepo: &userRepo, BusinessRepo: &businessRepo}
	rentAdFavoriteService := &services.RentAdFavoriteService{RentAdFavoriteRepo: &rentAdFavoriteRepo}
	adConfirmationService := &services.AdConfirmationService{ConfirmationRepo: &adConfirmationRepo, AdRepo: &adRepo, SubscriptionRepo: &subscriptionRepo}
	workConfirmationService := &services.WorkConfirmationService{ConfirmationRepo: &workConfirmationRepo}
	workAdConfirmationService := &services.WorkAdConfirmationService{ConfirmationRepo: &workAdConfirmationRepo, WorkAdRepo: &workAdRepo, SubscriptionRepo: &subscriptionRepo}
	rentConfirmationService := &services.RentConfirmationService{ConfirmationRepo: &rentConfirmationRepo}
	rentAdConfirmationService := &services.RentAdConfirmationService{ConfirmationRepo: &rentAdConfirmationRepo, RentAdRepo: &rentAdRepo, SubscriptionRepo: &subscriptionRepo}
	chatService := &services.ChatService{ChatRepo: &chatRepo, BusinessRepo: &businessRepo}
	// authService := &services.AuthService{DB: db}

	// Handlers
	userHandler := &handlers.UserHandler{Service: userService}
	serviceHandler := &handlers.ServiceHandler{Service: serviceService, ChatService: chatService}
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
	adComplaintHandler := &handlers.AdComplaintHandler{Service: &adComplaintService}
	workComplaintHandler := &handlers.WorkComplaintHandler{Service: &workComplaintService}
	workAdComplaintHandler := &handlers.WorkAdComplaintHandler{Service: &workAdComplaintService}
	rentComplaintHandler := &handlers.RentComplaintHandler{Service: &rentComplaintService}
	rentAdComplaintHandler := &handlers.RentAdComplaintHandler{Service: &rentAdComplaintService}
	serviceResponseHandler := &handlers.ServiceResponseHandler{Service: serviceResponseService}
	serviceConfirmationHandler := &handlers.ServiceConfirmationHandler{Service: serviceConfirmationService}
	userResponsesHandler := &handlers.UserResponsesHandler{Service: userResponsesService}
	userReviewsHandler := &handlers.UserReviewsHandler{Service: userReviewsService}
	userItemsHandler := &handlers.UserItemsHandler{Service: userItemsService}
	businessHandler := &handlers.BusinessHandler{Service: businessService}
	responseUsersHandler := &handlers.ResponseUsersHandler{Service: responseUsersService, ChatService: chatService}
	workHandler := &handlers.WorkHandler{Service: workService, ChatService: chatService}
	rentHandler := &handlers.RentHandler{Service: rentService, ChatService: chatService}
	workReviewHandler := &handlers.WorkReviewHandler{Service: workReviewService}
	workResponseHandler := &handlers.WorkResponseHandler{Service: workResponseService}
	workFavoriteHandler := &handlers.WorkFavoriteHandler{Service: workFavoriteService}
	rentReviewHandler := &handlers.RentReviewHandler{Service: rentReviewService}
	rentResponseHandler := &handlers.RentResponseHandler{Service: rentResponseService}
	rentFavoriteHandler := &handlers.RentFavoriteHandler{Service: rentFavoriteService}
	adHandler := &handlers.AdHandler{Service: adService, ChatService: chatService}
	adReviewHandler := &handlers.AdReviewHandler{Service: adReviewService}
	adResponseHandler := &handlers.AdResponseHandler{Service: adResponseService}
	adFavoriteHandler := &handlers.AdFavoriteHandler{Service: adFavoriteService}
	subscriptionHandler := &handlers.SubscriptionHandler{Service: subscriptionService}
	airbapayHandler := handlers.NewAirbapayHandler(airbapayService, invoiceRepo, &subscriptionRepo)
	airbapayHandler.TopService = topService
	airbapayHandler.BusinessService = businessService
	locationHandler := &handlers.LocationHandler{Service: locationService}
	assistantHandler := handlers.NewAssistantHandler(assistantService)
	topHandler := &handlers.TopHandler{Service: topService, InvoiceRepo: invoiceRepo, PaymentService: airbapayService}
	globalSearchHandler := &handlers.GlobalSearchHandler{Service: globalSearchService}

	adConfirmationHandler := &handlers.AdConfirmationHandler{Service: adConfirmationService}
	workAdHandler := &handlers.WorkAdHandler{Service: workAdService, ChatService: chatService}
	workAdReviewHandler := &handlers.WorkAdReviewHandler{Service: workAdReviewService}
	workAdResponseHandler := &handlers.WorkAdResponseHandler{Service: workAdResponseService}
	workAdFavoriteHandler := &handlers.WorkAdFavoriteHandler{Service: workAdFavoriteService}
	workAdConfirmationHandler := &handlers.WorkAdConfirmationHandler{Service: workAdConfirmationService}
	rentAdHandler := &handlers.RentAdHandler{Service: rentAdService, ChatService: chatService}
	rentADReviewHandler := &handlers.RentAdReviewHandler{Service: rentAdReviewService}
	rentAdResponseHandler := &handlers.RentAdResponseHandler{Service: rentAdResponseService}
	rentAdFavoriteHandler := &handlers.RentAdFavoriteHandler{Service: rentAdFavoriteService}
	rentAdConfirmationHandler := &handlers.RentAdConfirmationHandler{Service: rentAdConfirmationService}
	workConfirmationHandler := &handlers.WorkConfirmationHandler{Service: workConfirmationService}
	rentConfirmationHandler := &handlers.RentConfirmationHandler{Service: rentConfirmationService}

	// Chat
	wsManager := NewWebSocketManager()
	// Создание сервиса и обработчика для чатов
	chatHandler := &handlers.ChatHandler{ChatService: chatService}

	// Создание репозитория, сервиса и обработчика для сообщений
	messageService := &services.MessageService{MessageRepo: &messageRepo, UserRepo: &userRepo}
	messageHandler := &handlers.MessageHandler{MessageService: messageService}

	// Apple IAP
	var iapService *services.AppleIAPService
	iapPrivateKey := strings.TrimSpace(os.Getenv("APPLE_IAP_PRIVATE_KEY"))
	iapPrivateKey = strings.ReplaceAll(iapPrivateKey, `\n`, "\n")
	iapPrivateKey = strings.ReplaceAll(iapPrivateKey, "\r\n", "\n")
	iapPrivateKey = strings.ReplaceAll(iapPrivateKey, "\r", "\n")
	iapProductTargets, err := parseIAPProductTargets(os.Getenv("APPLE_IAP_PRODUCTS"))
	if err != nil {
		errorLog.Printf("apple iap "+
			"products: %v", err)
	}
	if iapPrivateKey != "" {
		iapCfg := services.AppleIAPConfig{
			IssuerID:    getEnv("APPLE_IAP_ISSUER_ID", ""),
			BundleID:    getEnv("APPLE_IAP_BUNDLE_ID", ""),
			KeyID:       getEnv("APPLE_IAP_KEY_ID", ""),
			PrivateKey:  iapPrivateKey,
			Environment: getEnv("APPLE_IAP_ENVIRONMENT", "production"),
		}

		// 🔎 DEBUG: проверить, что реально загрузилось
		log.Println("[APPLE IAP] issuer:", iapCfg.IssuerID)
		log.Println("[APPLE IAP] keyID:", iapCfg.KeyID)
		log.Println("[APPLE IAP] bundle:", iapCfg.BundleID)
		log.Println("[APPLE IAP] env:", iapCfg.Environment)
		log.Println("[APPLE IAP] privateKey len:", len(iapCfg.PrivateKey))

		if s, err := services.NewAppleIAPService(iapCfg); err != nil {
			errorLog.Printf("apple iap init: %v", err)
		} else {
			iapService = s
		}
	} else {
		infoLog.Println("Apple IAP: disabled (no APPLE_IAP_PRIVATE_KEY)")
	}
	iapHandler := handlers.NewIAPHandler(iapService, iapRepo, &subscriptionRepo, subscriptionService, topService, businessService, iapProductTargets)

	// GOOGLE PLAY IAP
	googleTargets, err := parseIAPProductTargets(os.Getenv("GOOGLE_IAP_PRODUCTS"))
	if err != nil {
		errorLog.Printf("google iap products: %v", err)
	}

	var googleSvc *services.GooglePlayService
	pkg := strings.TrimSpace(os.Getenv("GOOGLE_PLAY_PACKAGE_NAME"))
	credsJSON := strings.TrimSpace(os.Getenv("GOOGLE_PLAY_SERVICE_ACCOUNT_JSON"))

	// если creds положили как одну строку с \n — восстановим переносы
	credsJSON = strings.ReplaceAll(credsJSON, `\n`, "\n")

	if pkg != "" && credsJSON != "" {
		gcfg := services.GooglePlayConfig{
			PackageName:        pkg,
			ServiceAccountJSON: credsJSON,
		}
		if s, err := services.NewGooglePlayService(gcfg); err != nil {
			errorLog.Printf("google play init: %v", err)
		} else {
			googleSvc = s
			infoLog.Printf("Google Play IAP: enabled (package=%s, products=%d)", pkg, len(googleTargets))
		}
	} else {
		infoLog.Println("Google Play IAP: disabled (missing GOOGLE_PLAY_PACKAGE_NAME/GOOGLE_PLAY_SERVICE_ACCOUNT_JSON)")
	}

	googleIapRepo := repositories.NewGoogleIAPRepository(db)

	// важно: если ты дальше роуты вешаешь через app.googleIapHandler — сохрани в app
	googleIapHandler := handlers.NewGoogleIAPHandler(
		googleSvc,
		googleIapRepo,
		&subscriptionRepo,
		subscriptionService,
		topService,
		businessService,
		googleTargets,
	)

	return &application{
		errorLog: errorLog,
		infoLog:  infoLog,

		// DB и WebSocket
		db:        db,
		wsManager: wsManager,

		// Репозитории (pointer vs value строго как в вашей структуре)
		userRepo:                &userRepo,
		serviceRepo:             &serviceRepo,
		categoryRepo:            &categoryRepo,
		rentCategoryRepo:        &rentCategoryRepo,
		workCategoryRepo:        &workCategoryRepo,
		reviewsRepo:             &reviewsRepo,
		serviceFavoriteRepo:     &serviceFavoriteRepo,
		subcategoryRepo:         subcategoryRepo,     // value
		rentSubcategoryRepo:     rentSubcategoryRepo, // value
		workSubcategoryRepo:     workSubcategoryRepo, // value
		cityRepo:                cityRepo,            // value
		complaintRepo:           &complaintRepo,
		adComplaintRepo:         &adComplaintRepo,
		workComplaintRepo:       &workComplaintRepo,
		workAdComplaintRepo:     &workAdComplaintRepo,
		rentComplaintRepo:       &rentComplaintRepo,
		rentAdComplaintRepo:     &rentAdComplaintRepo,
		serviceResponseRepo:     &serviceResponseRepo,
		serviceConfirmationRepo: &serviceConfirmationRepo,
		userResponsesRepo:       &userResponsesRepo,
		responseUsersRepo:       &responseUsersRepo,
		userReviewsRepo:         &userReviewsRepo,
		userItemsRepo:           &userItemsRepo,
		businessRepo:            &businessRepo,
		businessService:         businessService,
		workRepo:                &workRepo,
		rentRepo:                &rentRepo,
		workReviewRepo:          &workReviewRepo,
		workResponseRepo:        &workResponseRepo,
		workFavoriteRepo:        &workFavoriteRepo,
		rentReviewRepo:          &rentReviewRepo,
		rentResponseRepo:        &rentResponseRepo,
		rentFavoriteRepo:        &rentFavoriteRepo,
		adRepo:                  &adRepo,
		adReviewRepo:            &adReviewRepo,
		adResponseRepo:          &adResponseRepo,
		adFavoriteRepo:          &adFavoriteRepo,
		adConfirmationRepo:      &adConfirmationRepo,
		workConfirmationRepo:    &workConfirmationRepo,
		rentConfirmationRepo:    &rentConfirmationRepo,
		subscriptionRepo:        &subscriptionRepo,
		locationRepo:            &locationRepo,
		locationService:         locationService,

		// WorkAd блок
		workAdRepo:             &workAdRepo,
		workAdReviewRepo:       &workAdReviewRepo,
		workAdResponseRepo:     &workAdResponseRepo,
		workAdFavoriteRepo:     &workAdFavoriteRepo,
		workAdConfirmationRepo: &workAdConfirmationRepo,

		// RentAd блок
		rentAdRepo:             (*repositories.AdRepository)(&rentAdRepo),
		rentAdReviewRepo:       (*repositories.AdReviewRepository)(&rentAdReviewRepo),
		rentAdResponseRepo:     (*repositories.AdResponseRepository)(&rentAdResponseRepo),
		rentAdFavoriteRepo:     (*repositories.AdFavoriteRepository)(&rentAdFavoriteRepo),
		rentAdConfirmationRepo: &rentAdConfirmationRepo,

		// Хендлеры
		userHandler:                userHandler,
		serviceHandler:             serviceHandler,
		globalSearchHandler:        globalSearchHandler,
		categoryHandler:            categoryHandler,
		rentCategoryHandler:        rentCategoryHandler,
		workCategoryHandler:        workCategoryHandler,
		reviewsHandler:             reviewHandler,
		serviceFavorite:            serviceFavoriteHandler,
		subcategoryHandler:         subcategoryHandler,
		rentSubcategoryHandler:     rentSubcategoryHandler,
		workSubcategoryHandler:     workSubcategoryHandler,
		cityHandler:                cityHandler,
		complaintHandler:           complaintHandler,
		adComplaintHandler:         adComplaintHandler,
		workComplaintHandler:       workComplaintHandler,
		workAdComplaintHandler:     workAdComplaintHandler,
		rentComplaintHandler:       rentComplaintHandler,
		rentAdComplaintHandler:     rentAdComplaintHandler,
		serviceResponseHandler:     serviceResponseHandler,
		serviceConfirmationHandler: serviceConfirmationHandler,
		userResponsesHandler:       userResponsesHandler,
		responseUsersHandler:       responseUsersHandler,
		userReviewsHandler:         userReviewsHandler,
		userItemsHandler:           userItemsHandler,
		businessHandler:            businessHandler,
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
		adConfirmationHandler:      adConfirmationHandler,
		subscriptionHandler:        subscriptionHandler,
		airbapayHandler:            airbapayHandler,
		assistantHandler:           assistantHandler,
		topHandler:                 topHandler,
		topService:                 topService,
		iapHandler:                 iapHandler,
		googleIapHandler:           googleIapHandler,
		workAdHandler:              workAdHandler,
		workAdReviewHandler:        workAdReviewHandler,
		workAdResponseHandler:      workAdResponseHandler,
		workAdFavoriteHandler:      workAdFavoriteHandler,
		workAdConfirmationHandler:  workAdConfirmationHandler,

		rentAdHandler:             rentAdHandler,
		rentAdReviewHandler:       rentADReviewHandler,
		rentAdResponseHandler:     rentAdResponseHandler,
		rentAdFavoriteHandler:     rentAdFavoriteHandler,
		rentAdConfirmationHandler: rentAdConfirmationHandler,

		workConfirmationHandler: workConfirmationHandler,
		rentConfirmationHandler: rentConfirmationHandler,

		// Чаты/сообщения
		chatHandler:    chatHandler,
		messageHandler: messageHandler,

		locationHandler: locationHandler,

		invoiceRepo: invoiceRepo,

		fcmHandler: fcmHandler,
	}
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func parseIAPProductTargets(raw string) (map[string]models.IAPTarget, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var targets map[string]models.IAPTarget
	if err := json.Unmarshal([]byte(raw), &targets); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	for productID, target := range targets {
		if err := target.Validate(); err != nil {
			return nil, fmt.Errorf("product %s: %w", productID, err)
		}
	}
	return targets, nil
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
		if strings.HasPrefix(r.URL.Path, "/ws") {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}
