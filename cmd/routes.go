package main

import (
	"fmt"
	"github.com/bmizerany/pat"
	"github.com/justinas/alice"
	"net/http"
	"strings"
	// httpSwagger "github.com/swaggo/http-swagger"
	// _ "naimuBack/docs"
)

func (app *application) JWTMiddlewareWithRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return app.JWTMiddleware(next, requiredRole)
	}
}

// Пробросить user_id из контекста в заголовок для downstream (такси-хендлеров)
func (app *application) withHeaderFromCtx(next http.Handler, header string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v := r.Context().Value("user_id"); v != nil {
			if id, ok := v.(int); ok {
				r = r.Clone(r.Context())
				r.Header.Set(header, fmt.Sprintf("%d", id))
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (app *application) withTaxiRoleHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, _ := r.Context().Value("role").(string)
		id, _ := r.Context().Value("user_id").(int)
		switch role {
		case "worker":
			r = r.Clone(r.Context())
			r.Header.Set("X-Driver-ID", fmt.Sprintf("%d", id))
		case "client":
			r = r.Clone(r.Context())
			r.Header.Set("X-Passenger-ID", fmt.Sprintf("%d", id))
		case "admin":
			// admins can observe without impersonation
		default:
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Для WS: дописать ?passenger_id=... или ?driver_id=... в URL
func (app *application) wsWithQueryUserID(next http.Handler, param string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if v := r.Context().Value("user_id"); v != nil {
			if _, ok := v.(int); ok {
				q := r.URL.Query()
				//q.Set(param, fmt.Sprintf("%d", id))
				r = r.Clone(r.Context())
				r.URL.RawQuery = q.Encode()
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Для WS: прокинуть Authorization токен из query параметров в заголовок.
func (app *application) wsWithAuthFromQuery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			q := r.URL.Query()
			token := q.Get("token")
			if token == "" {
				token = q.Get("authorization")
			}
			if token != "" {
				if !strings.HasPrefix(strings.ToLower(token), "bearer ") {
					token = "Bearer " + token
				}
				r = r.Clone(r.Context())
				r.Header.Set("Authorization", token)
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (app *application) routes() http.Handler {
	standardMiddleware := alice.New(app.recoverPanic, app.logRequest, secureHeaders, makeResponseJSON)
	authMiddleware := standardMiddleware.Append(app.JWTMiddlewareWithRole("user"))
	adminAuthMiddleware := standardMiddleware.Append(app.JWTMiddlewareWithRole("admin"))

	wsMiddleware := alice.New(app.recoverPanic, app.logRequest)

	mux := pat.New()

	clientAuth := standardMiddleware.Append(app.JWTMiddlewareWithRole("client"))
	workerAuth := standardMiddleware.Append(app.JWTMiddlewareWithRole("worker"))

	// mux.Get("/swagger/", httpSwagger.WrapHandler)

	mux.Post("/ai/ask", standardMiddleware.ThenFunc(app.assistantHandler.Ask))

	// Users
	mux.Post("/user", adminAuthMiddleware.ThenFunc(app.userHandler.CreateUser))     //
	mux.Get("/user", authMiddleware.ThenFunc(app.userHandler.GetUsers))             //РАБОТАЕТ
	mux.Get("/user/token", authMiddleware.ThenFunc(app.userHandler.GetUserByToken)) //РАБОТАЕТ
	mux.Get("/user/:id", authMiddleware.ThenFunc(app.userHandler.GetUserByID))      //РАБОТАЕТ
	mux.Put("/user/:id", authMiddleware.ThenFunc(app.userHandler.UpdateUser))       //РАБОТАЕТ
	mux.Del("/user/me", authMiddleware.ThenFunc(app.userHandler.DeleteOwnAccount))
	mux.Del("/user/:id", authMiddleware.ThenFunc(app.userHandler.DeleteUser))                  //ИСПРАВИТЬ
	mux.Post("/user/sign_up", standardMiddleware.ThenFunc(app.userHandler.SignUp))             //РАБОТАЕТ
	mux.Post("/user/sign_in", standardMiddleware.ThenFunc(app.userHandler.SignIn))             //РАБОТАЕТ
	mux.Post("/user/change_number", standardMiddleware.ThenFunc(app.userHandler.ChangeNumber)) //РАБОТАЕТ
	mux.Post("/user/change_email", standardMiddleware.ThenFunc(app.userHandler.ChangeEmail))   //РАБОТАЕТ
	mux.Post("/user/send_email_code", standardMiddleware.ThenFunc(app.userHandler.SendCodeToEmail))
	mux.Post("/user/code_check", standardMiddleware.ThenFunc(app.userHandler.CheckVerificationCode))
	mux.Put("/user/:id/city", authMiddleware.ThenFunc(app.userHandler.ChangeCityForUser))
	mux.Get("/docs/:filename", authMiddleware.ThenFunc(app.userHandler.ServeProofDocument))
	mux.Post("/user/:id/avatar", authMiddleware.ThenFunc(app.userHandler.UploadAvatar))
	mux.Del("/user/:id/avatar", authMiddleware.ThenFunc(app.userHandler.DeleteAvatar))
	mux.Get("/images/avatars/:filename", standardMiddleware.ThenFunc(app.userHandler.ServeAvatar))
	mux.Post("/user/:id/upgrade", authMiddleware.ThenFunc(app.userHandler.UpdateToWorker))
	mux.Post("/users/check_duplicate", standardMiddleware.ThenFunc(app.userHandler.CheckUserDuplicate))
	mux.Post("/user/request_reset", standardMiddleware.ThenFunc(app.userHandler.RequestPasswordReset))
	mux.Post("/user/verify_reset_code", standardMiddleware.ThenFunc(app.userHandler.VerifyResetCode))
	mux.Post("/user/reset_password", standardMiddleware.ThenFunc(app.userHandler.ResetPassword))
	mux.Get("/subscription/:user_id", authMiddleware.ThenFunc(app.subscriptionHandler.GetSubscription))
	mux.Get("/subscriptions", authMiddleware.ThenFunc(app.subscriptionHandler.GetSubscriptions))
	mux.Post("/airbapay/pay", standardMiddleware.ThenFunc(app.airbapayHandler.CreatePayment))
	mux.Post("/airbapay/callback", standardMiddleware.ThenFunc(app.airbapayHandler.Callback))
	mux.Get("/airbapay/history/:user_id", authMiddleware.ThenFunc(app.airbapayHandler.GetHistory))
	mux.Get("/airbapay/success", standardMiddleware.ThenFunc(app.airbapayHandler.SuccessRedirect))
	mux.Get("/airbapay/failure", standardMiddleware.ThenFunc(app.airbapayHandler.FailureRedirect))

	mux.Get("/user/posts/:user_id", authMiddleware.ThenFunc(app.userItemsHandler.GetPostsByUserID))
	mux.Get("/user/ads/:user_id", authMiddleware.ThenFunc(app.userItemsHandler.GetAdsByUserID))
	mux.Get("/user/orders/:user_id", authMiddleware.ThenFunc(app.userItemsHandler.GetOrderHistoryByUserID))
	mux.Get("/user/active_orders/:user_id", authMiddleware.ThenFunc(app.userItemsHandler.GetActiveOrdersByUserID))

	mux.Get("/ads", standardMiddleware.ThenFunc(app.adHandler.GetAds))

	// Service
	mux.Post("/service", authMiddleware.ThenFunc(app.serviceHandler.CreateService))      //РАБОТАЕТ
	mux.Get("/service/get", standardMiddleware.ThenFunc(app.serviceHandler.GetServices)) //РАБОТАЕТ
	mux.Get("/admin/service/get", adminAuthMiddleware.ThenFunc(app.serviceHandler.GetServicesAdmin))
	mux.Get("/service/:id", standardMiddleware.ThenFunc(app.serviceHandler.GetServiceByID)) //РАБОТАЕТ
	mux.Put("/service/:id", authMiddleware.ThenFunc(app.serviceHandler.UpdateService))      //РАБОТАЕТ
	mux.Del("/service/:id", authMiddleware.ThenFunc(app.serviceHandler.DeleteService))      //РАБОТАЕТ
	mux.Post("/service/archive", authMiddleware.ThenFunc(app.serviceHandler.ArchiveService))
	mux.Get("/service/sort/:type/user/:user_id", standardMiddleware.ThenFunc(app.serviceHandler.GetServicesSorted)) //user_id - id пользователя который авторизован
	mux.Get("/service/user/:user_id", standardMiddleware.ThenFunc(app.serviceHandler.GetServiceByUserID))           //РАБОТАЕТ
	mux.Post("/service/filtered", standardMiddleware.ThenFunc(app.serviceHandler.GetFilteredServicesPost))          //РАБОТАЕТ
	mux.Post("/service/status", authMiddleware.ThenFunc(app.serviceHandler.GetServicesByStatusAndUserID))
	mux.Post("/service/confirm", authMiddleware.ThenFunc(app.serviceConfirmationHandler.ConfirmService))
	mux.Post("/service/cancel", authMiddleware.ThenFunc(app.serviceConfirmationHandler.CancelService))
	mux.Post("/service/done", authMiddleware.ThenFunc(app.serviceConfirmationHandler.DoneService))
	mux.Get("/images/services/:filename", http.HandlerFunc(app.serviceHandler.ServeServiceImage))
	mux.Get("/videos/services/:filename", http.HandlerFunc(app.serviceHandler.ServeServiceVideo))
	mux.Post("/service/filtered/:user_id", authMiddleware.ThenFunc(app.serviceHandler.GetFilteredServicesWithLikes))
	mux.Get("/service/service_id/:service_id/user/:user_id", standardMiddleware.ThenFunc(app.serviceHandler.GetServiceByServiceIDAndUserID))

	// Categories
	mux.Post("/category", authMiddleware.ThenFunc(app.categoryHandler.CreateCategory))
	mux.Get("/category", standardMiddleware.ThenFunc(app.categoryHandler.GetAllCategories))
	mux.Get("/category/:id", standardMiddleware.ThenFunc(app.categoryHandler.GetCategoryByID))
	mux.Put("/category/:id", authMiddleware.ThenFunc(app.categoryHandler.UpdateCategory))
	mux.Del("/category/:id", authMiddleware.ThenFunc(app.categoryHandler.DeleteCategory))
	fs := http.StripPrefix("/static/categories/", http.FileServer(http.Dir("./cmd/uploads/categories")))
	mux.Get("/static/categories/", fs)
	mux.Get("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))
	mux.Get("/images/categories/:filename", standardMiddleware.ThenFunc(app.categoryHandler.ServeImage))

	// Rent Categories
	mux.Post("/rent_category", authMiddleware.ThenFunc(app.rentCategoryHandler.CreateCategory))
	mux.Get("/rent_category", standardMiddleware.ThenFunc(app.rentCategoryHandler.GetAllCategories))
	mux.Get("/rent_category/:id", standardMiddleware.ThenFunc(app.rentCategoryHandler.GetCategoryByID))
	mux.Put("/rent_category/:id", authMiddleware.ThenFunc(app.rentCategoryHandler.UpdateCategory))
	mux.Del("/rent_category/:id", authMiddleware.ThenFunc(app.rentCategoryHandler.DeleteCategory))
	mux.Get("/images/rent_categories/:filename", standardMiddleware.ThenFunc(app.rentCategoryHandler.ServeImage))

	// Work Categories
	mux.Post("/work_category", authMiddleware.ThenFunc(app.workCategoryHandler.CreateCategory))
	mux.Get("/work_category", standardMiddleware.ThenFunc(app.workCategoryHandler.GetAllCategories))
	mux.Get("/work_category/:id", standardMiddleware.ThenFunc(app.workCategoryHandler.GetCategoryByID))
	mux.Put("/work_category/:id", authMiddleware.ThenFunc(app.workCategoryHandler.UpdateCategory))
	mux.Del("/work_category/:id", authMiddleware.ThenFunc(app.workCategoryHandler.DeleteCategory))
	mux.Get("/images/work_categories/:filename", standardMiddleware.ThenFunc(app.workCategoryHandler.ServeImage))

	// Reviews
	mux.Post("/review", authMiddleware.ThenFunc(app.reviewsHandler.CreateReview))
	mux.Get("/review/:service_id", standardMiddleware.ThenFunc(app.reviewsHandler.GetReviewsByServiceID))
	mux.Put("/review/:id", authMiddleware.ThenFunc(app.reviewsHandler.UpdateReview))
	mux.Del("/review/:id", authMiddleware.ThenFunc(app.reviewsHandler.DeleteReview))
	mux.Get("/reviews/:user_id", authMiddleware.ThenFunc(app.userReviewsHandler.GetReviewsByUserID))

	// Service Favorites
	mux.Post("/favorites", authMiddleware.ThenFunc(app.serviceFavorite.AddToFavorites))
	mux.Del("/favorites/user/:user_id/service/:service_id", authMiddleware.ThenFunc(app.serviceFavorite.RemoveFromFavorites))
	mux.Get("/favorites/check/user/:user_id/service/:service_id", standardMiddleware.ThenFunc(app.serviceFavorite.IsFavorite))
	mux.Get("/favorites/:user_id", standardMiddleware.ThenFunc(app.serviceFavorite.GetFavoritesByUser))

	// Subcategories
	mux.Post("/subcategory", authMiddleware.ThenFunc(app.subcategoryHandler.CreateSubcategory))
	mux.Get("/subcategory", standardMiddleware.ThenFunc(app.subcategoryHandler.GetAllSubcategories))
	mux.Get("/subcategory/cat/:category_id", standardMiddleware.ThenFunc(app.subcategoryHandler.GetByCategory))
	mux.Get("/subcategory/:id", standardMiddleware.ThenFunc(app.subcategoryHandler.GetSubcategoryByID))
	mux.Put("/subcategory/:id", authMiddleware.ThenFunc(app.subcategoryHandler.UpdateSubcategoryByID))
	mux.Del("/subcategory/:id", authMiddleware.ThenFunc(app.subcategoryHandler.DeleteSubcategoryByID))

	// Rent Subcategories
	mux.Post("/rent_subcategory", authMiddleware.ThenFunc(app.rentSubcategoryHandler.CreateSubcategory))
	mux.Get("/rent_subcategory", standardMiddleware.ThenFunc(app.rentSubcategoryHandler.GetAllSubcategories))
	mux.Get("/rent_subcategory/cat/:category_id", standardMiddleware.ThenFunc(app.rentSubcategoryHandler.GetByCategory))
	mux.Get("/rent_subcategory/:id", standardMiddleware.ThenFunc(app.rentSubcategoryHandler.GetSubcategoryByID))
	mux.Put("/rent_subcategory/:id", authMiddleware.ThenFunc(app.rentSubcategoryHandler.UpdateSubcategoryByID))
	mux.Del("/rent_subcategory/:id", authMiddleware.ThenFunc(app.rentSubcategoryHandler.DeleteSubcategoryByID))

	// Work Subcategories
	mux.Post("/work_subcategory", authMiddleware.ThenFunc(app.workSubcategoryHandler.CreateSubcategory))
	mux.Get("/work_subcategory", standardMiddleware.ThenFunc(app.workSubcategoryHandler.GetAllSubcategories))
	mux.Get("/work_subcategory/cat/:category_id", standardMiddleware.ThenFunc(app.workSubcategoryHandler.GetByCategory))
	mux.Get("/work_subcategory/:id", standardMiddleware.ThenFunc(app.workSubcategoryHandler.GetSubcategoryByID))
	mux.Put("/work_subcategory/:id", authMiddleware.ThenFunc(app.workSubcategoryHandler.UpdateSubcategoryByID))
	mux.Del("/work_subcategory/:id", authMiddleware.ThenFunc(app.workSubcategoryHandler.DeleteSubcategoryByID))

	// City
	mux.Post("/city", authMiddleware.ThenFunc(app.cityHandler.CreateCity))
	mux.Get("/city", standardMiddleware.ThenFunc(app.cityHandler.GetCities))
	mux.Get("/city/:id", standardMiddleware.ThenFunc(app.cityHandler.GetCityByID))
	mux.Put("/city/:id", authMiddleware.ThenFunc(app.cityHandler.UpdateCity))
	mux.Del("/city/:id", authMiddleware.ThenFunc(app.cityHandler.DeleteCity))

	// Chat
	mux.Get("/ws", wsMiddleware.ThenFunc(app.WebSocketHandler))
	mux.Get("/ws/location", wsMiddleware.ThenFunc(app.LocationWebSocketHandler))

	// === Taxi API & WS ===
	mux.Get("/api/v1/admin/taxi/drivers", adminAuthMiddleware.Then(app.taxiMux))
	mux.Post("/api/v1/admin/taxi/drivers/:driver_id/ban", adminAuthMiddleware.Then(app.taxiMux))
	mux.Post("/api/v1/admin/taxi/drivers/:driver_id/approval", adminAuthMiddleware.Then(app.taxiMux))
	mux.Get("/api/v1/admin/taxi/orders", adminAuthMiddleware.Then(app.taxiMux))
	mux.Get("/api/v1/admin/taxi/intercity/orders", adminAuthMiddleware.Then(app.taxiMux))

	mux.Post("/api/v1/route/quote", standardMiddleware.Then(app.taxiMux))
	mux.Get("/api/v1/orders", clientAuth.Then(app.withHeaderFromCtx(app.taxiMux, "X-Passenger-ID")))
	mux.Get("/api/v1/orders/active", clientAuth.Then(app.withHeaderFromCtx(app.taxiMux, "X-Passenger-ID")))
	mux.Post("/api/v1/orders", clientAuth.Then(app.withHeaderFromCtx(app.taxiMux, "X-Passenger-ID")))
	mux.Get("/api/v1/orders/:id", standardMiddleware.Then(app.withHeaderFromCtx(app.taxiMux, "X-Passenger-ID")))
	mux.Get("/api/v1/orders/:id", authMiddleware.Then(app.taxiMux))
	mux.Post("/api/v1/orders/:id/reprice", clientAuth.Then(app.withHeaderFromCtx(app.taxiMux, "X-Passenger-ID")))
	mux.Post("/api/v1/orders/:id/status", clientAuth.Then(app.withHeaderFromCtx(app.taxiMux, "X-Passenger-ID")))
	// Taxi: driver profile extras.
	mux.Post("/api/v1/drivers", authMiddleware.Then(app.taxiMux))           // Возвращает {"driver": {...}, "completed_trips": int, "balance": int}
	mux.Get("/api/v1/driver/:id/profile", authMiddleware.Then(app.taxiMux)) // Возвращает {"driver": {...}, "completed_trips": int, "balance": int}
	mux.Get("/api/v1/driver/:id/reviews", authMiddleware.Then(app.taxiMux)) // Возвращает {"reviews": [{"rating": number|null, "comment": string, "created_at": string, "order": {...}}]}
	mux.Get("/api/v1/driver/:id/stats", authMiddleware.Then(app.taxiMux))   // Возвращает {"total_orders": int, "total_amount": int, "net_profit": int, "days": [{"date": "YYYY-MM-DD", "orders_count": int, "total_amount": int, "net_profit": int, "orders": [...]}]}

	mux.Get("/api/v1/driver/orders", workerAuth.Then(app.withHeaderFromCtx(app.taxiMux, "X-Driver-ID")))
	mux.Get("/api/v1/driver/orders/active", workerAuth.Then(app.withHeaderFromCtx(app.taxiMux, "X-Driver-ID")))
	mux.Post("/api/v1/driver/balance/deposit", workerAuth.Then(app.withHeaderFromCtx(app.taxiMux, "X-Driver-ID")))
	mux.Post("/api/v1/driver/balance/withdraw", workerAuth.Then(app.withHeaderFromCtx(app.taxiMux, "X-Driver-ID")))
	mux.Post("/api/v1/offers/accept", clientAuth.Then(app.withHeaderFromCtx(app.taxiMux, "X-Driver-ID")))
	mux.Post("/api/v1/offers/propose_price", workerAuth.Then(app.withHeaderFromCtx(app.taxiMux, "X-Driver-ID")))
	mux.Post("/api/v1/offers/respond", clientAuth.Then(app.withHeaderFromCtx(app.taxiMux, "X-Passenger-ID")))
	mux.Post("/api/v1/payments/airbapay/webhook", standardMiddleware.Then(app.taxiMux))
	mux.Post("/api/v1/intercity/orders", standardMiddleware.Then(app.taxiMux))
	mux.Post("/api/v1/intercity/orders/list", standardMiddleware.Then(app.taxiMux))
	mux.Get("/api/v1/intercity/orders/:id", standardMiddleware.Then(app.taxiMux))
	mux.Post("/api/v1/intercity/orders/:id/close", standardMiddleware.Then(app.taxiMux))
	mux.Post("/api/v1/intercity/orders/:id/cancel", standardMiddleware.Then(app.taxiMux))
	mux.Post("/api/taxi/orders/:id/arrive", workerAuth.Then(app.withHeaderFromCtx(app.taxiMux, "X-Driver-ID")))
	mux.Post("/api/taxi/orders/:id/waiting/advance", workerAuth.Then(app.withHeaderFromCtx(app.taxiMux, "X-Driver-ID")))
	mux.Post("/api/taxi/orders/:id/start", workerAuth.Then(app.withHeaderFromCtx(app.taxiMux, "X-Driver-ID")))
	mux.Post("/api/taxi/orders/:id/waypoints/next", workerAuth.Then(app.withHeaderFromCtx(app.taxiMux, "X-Driver-ID")))
	mux.Post("/api/taxi/orders/:id/pause", workerAuth.Then(app.withHeaderFromCtx(app.taxiMux, "X-Driver-ID")))
	mux.Post("/api/taxi/orders/:id/resume", workerAuth.Then(app.withHeaderFromCtx(app.taxiMux, "X-Driver-ID")))
	mux.Post("/api/taxi/orders/:id/finish", workerAuth.Then(app.withHeaderFromCtx(app.taxiMux, "X-Driver-ID")))
	mux.Post("/api/taxi/orders/:id/confirm-cash", workerAuth.Then(app.withHeaderFromCtx(app.taxiMux, "X-Driver-ID")))
	mux.Post("/api/taxi/orders/:id/cancel", authMiddleware.Append(app.withTaxiRoleHeaders).Then(app.taxiMux))
	mux.Post("/api/taxi/orders/:id/no-show", workerAuth.Then(app.withHeaderFromCtx(app.taxiMux, "X-Driver-ID")))
	mux.Get("/ws/passenger", wsMiddleware.Append(app.wsWithAuthFromQuery).Append(app.JWTMiddlewareWithRole("client")).Then(app.wsWithQueryUserID(app.taxiMux, "passenger_id")))
	mux.Get("/ws/driver", wsMiddleware.Append(app.wsWithAuthFromQuery).Append(app.JWTMiddlewareWithRole("worker")).Then(app.wsWithQueryUserID(app.taxiMux, "driver_id")))

	mux.Post("/location", authMiddleware.ThenFunc(app.locationHandler.UpdateLocation))
	mux.Post("/location/offline", authMiddleware.ThenFunc(app.locationHandler.GoOffline))
	mux.Get("/location/:user_id", authMiddleware.ThenFunc(app.locationHandler.GetLocation))
	mux.Post("/executors/location/:type", standardMiddleware.ThenFunc(app.locationHandler.GetExecutors))
	mux.Post("/executors/location", standardMiddleware.ThenFunc(app.locationHandler.GetExecutors))

	mux.Post("/api/chats", authMiddleware.ThenFunc(app.chatHandler.CreateChat))
	mux.Get("/api/chats/:id", authMiddleware.ThenFunc(app.chatHandler.GetChatByID))
	mux.Get("/api/chats", authMiddleware.ThenFunc(app.chatHandler.GetAllChats))
	mux.Get("/api/chats/user/:user_id", authMiddleware.ThenFunc(app.chatHandler.GetChatsByUserID))
	mux.Del("/api/chats/:id", authMiddleware.ThenFunc(app.chatHandler.DeleteChat))

	mux.Post("/api/messages", authMiddleware.ThenFunc(app.messageHandler.CreateMessage))
	mux.Get("/api/messages/:chatId", authMiddleware.ThenFunc(app.messageHandler.GetMessagesForChat))
	mux.Del("/api/messages/:messageId", authMiddleware.ThenFunc(app.messageHandler.DeleteMessage))

	mux.Get("/api/users/messages", authMiddleware.ThenFunc(app.messageHandler.GetMessagesByUserIDs))

	// Complaints
	mux.Post("/complaints", authMiddleware.ThenFunc(app.complaintHandler.CreateComplaint))
	mux.Get("/complaints/service/:service_id", standardMiddleware.ThenFunc(app.complaintHandler.GetComplaintsByServiceID))
	mux.Del("/complaints/:id", authMiddleware.ThenFunc(app.complaintHandler.DeleteComplaintByID))
	mux.Get("/complaints", standardMiddleware.ThenFunc(app.complaintHandler.GetAllComplaints))

	// Ad Complaints
	mux.Post("/ad_complaint", authMiddleware.ThenFunc(app.adComplaintHandler.CreateAdComplaint))
	mux.Get("/ad_complaint/:ad_id", standardMiddleware.ThenFunc(app.adComplaintHandler.GetComplaintsByAdID))
	mux.Del("/ad_complaint/:id", authMiddleware.ThenFunc(app.adComplaintHandler.DeleteAdComplaintByID))
	mux.Get("/ad_complaints", standardMiddleware.ThenFunc(app.adComplaintHandler.GetAllAdComplaints))

	// Work Complaints
	mux.Post("/work_complaint", authMiddleware.ThenFunc(app.workComplaintHandler.CreateWorkComplaint))
	mux.Get("/work_complaint/:work_id", standardMiddleware.ThenFunc(app.workComplaintHandler.GetComplaintsByWorkID))
	mux.Del("/work_complaint/:id", authMiddleware.ThenFunc(app.workComplaintHandler.DeleteWorkComplaintByID))
	mux.Get("/work_complaints", standardMiddleware.ThenFunc(app.workComplaintHandler.GetAllWorkComplaints))

	// Work Ad Complaints
	mux.Post("/work_ad_complaint", authMiddleware.ThenFunc(app.workAdComplaintHandler.CreateWorkAdComplaint))
	mux.Get("/work_ad_complaint/:work_ad_id", standardMiddleware.ThenFunc(app.workAdComplaintHandler.GetComplaintsByWorkAdID))
	mux.Del("/work_ad_complaint/:id", authMiddleware.ThenFunc(app.workAdComplaintHandler.DeleteWorkAdComplaintByID))
	mux.Get("/work_ad_complaints", standardMiddleware.ThenFunc(app.workAdComplaintHandler.GetAllWorkAdComplaints))

	// Rent Complaints
	mux.Post("/rent_complaint", authMiddleware.ThenFunc(app.rentComplaintHandler.CreateRentComplaint))
	mux.Get("/rent_complaint/:rent_id", standardMiddleware.ThenFunc(app.rentComplaintHandler.GetComplaintsByRentID))
	mux.Del("/rent_complaint/:id", authMiddleware.ThenFunc(app.rentComplaintHandler.DeleteRentComplaintByID))
	mux.Get("/rent_complaints", standardMiddleware.ThenFunc(app.rentComplaintHandler.GetAllRentComplaints))

	// Rent Ad Complaints
	mux.Post("/rent_ad_complaint", authMiddleware.ThenFunc(app.rentAdComplaintHandler.CreateRentAdComplaint))
	mux.Get("/rent_ad_complaint/:rent_ad_id", standardMiddleware.ThenFunc(app.rentAdComplaintHandler.GetComplaintsByRentAdID))
	mux.Del("/rent_ad_complaint/:id", authMiddleware.ThenFunc(app.rentAdComplaintHandler.DeleteRentAdComplaintByID))
	mux.Get("/rent_ad_complaints", standardMiddleware.ThenFunc(app.rentAdComplaintHandler.GetAllRentAdComplaints))

	// Service Response
	mux.Post("/responses", authMiddleware.ThenFunc(app.serviceResponseHandler.CreateServiceResponse))
	mux.Del("/responses/:id", authMiddleware.ThenFunc(app.serviceResponseHandler.CancelServiceResponse))
	mux.Get("/responses/:user_id", authMiddleware.ThenFunc(app.userResponsesHandler.GetResponsesByUserID))
	mux.Get("/responses/item/:type/:item_id", authMiddleware.ThenFunc(app.responseUsersHandler.GetUsersByItemID))

	// Work
	mux.Post("/work", authMiddleware.ThenFunc(app.workHandler.CreateWork))
	mux.Get("/work/get", standardMiddleware.ThenFunc(app.workHandler.GetWorks))
	mux.Get("/admin/work/get", adminAuthMiddleware.ThenFunc(app.workHandler.GetWorksAdmin))
	mux.Get("/work/:id", standardMiddleware.ThenFunc(app.workHandler.GetWorkByID))
	mux.Put("/work/:id", authMiddleware.ThenFunc(app.workHandler.UpdateWork))
	mux.Del("/work/:id", authMiddleware.ThenFunc(app.workHandler.DeleteWork))
	mux.Post("/work/archive", authMiddleware.ThenFunc(app.workHandler.ArchiveWork))
	mux.Get("/work/user/:user_id", authMiddleware.ThenFunc(app.workHandler.GetWorksByUserID))
	mux.Post("/work/filtered", standardMiddleware.ThenFunc(app.workHandler.GetFilteredWorksPost))
	mux.Post("/work/status", authMiddleware.ThenFunc(app.workHandler.GetWorksByStatusAndUserID))
	mux.Post("/work/confirm", authMiddleware.ThenFunc(app.workConfirmationHandler.ConfirmWork))
	mux.Post("/work/cancel", authMiddleware.ThenFunc(app.workConfirmationHandler.CancelWork))
	mux.Post("/work/done", authMiddleware.ThenFunc(app.workConfirmationHandler.DoneWork))
	mux.Get("/images/works/:filename", http.HandlerFunc(app.workHandler.ServeWorkImage))
	mux.Get("/videos/works/:filename", http.HandlerFunc(app.workHandler.ServeWorkVideo))
	mux.Post("/work/filtered/:user_id", authMiddleware.ThenFunc(app.workHandler.GetFilteredWorksWithLikes))
	mux.Get("/work/work_id/:work_id/user/:user_id", standardMiddleware.ThenFunc(app.workHandler.GetWorkByWorkIDAndUserID))

	// Work Reviews
	mux.Post("/work_review", authMiddleware.ThenFunc(app.workReviewHandler.CreateWorkReview))
	mux.Get("/work_review/:work_id", standardMiddleware.ThenFunc(app.workReviewHandler.GetWorkReviewsByWorkID))
	mux.Put("/work_review/:id", authMiddleware.ThenFunc(app.workReviewHandler.UpdateWorkReview))
	mux.Del("/work_review/:id", authMiddleware.ThenFunc(app.workReviewHandler.DeleteWorkReview))

	// Work Response
	mux.Post("/work_responses", authMiddleware.ThenFunc(app.workResponseHandler.CreateWorkResponse))
	mux.Del("/work_responses/:id", authMiddleware.ThenFunc(app.workResponseHandler.CancelWorkResponse))

	// Work Favorites
	mux.Post("/work_favorites", authMiddleware.ThenFunc(app.workFavoriteHandler.AddWorkToFavorites))
	mux.Del("/work_favorites/user/:user_id/work/:work_id", authMiddleware.ThenFunc(app.workFavoriteHandler.RemoveWorkFromFavorites))
	mux.Get("/work_favorites/check/user/:user_id/work/:work_id", standardMiddleware.ThenFunc(app.workFavoriteHandler.IsWorkFavorite))
	mux.Get("/work_favorites/:user_id", standardMiddleware.ThenFunc(app.workFavoriteHandler.GetWorkFavoritesByUser))

	// Rent
	mux.Post("/rent", authMiddleware.ThenFunc(app.rentHandler.CreateRent))
	mux.Get("/rent/get", standardMiddleware.ThenFunc(app.rentHandler.GetRents))
	mux.Get("/admin/rent/get", adminAuthMiddleware.ThenFunc(app.rentHandler.GetRentsAdmin))
	mux.Get("/rent/:id", standardMiddleware.ThenFunc(app.rentHandler.GetRentByID))
	mux.Put("/rent/:id", authMiddleware.ThenFunc(app.rentHandler.UpdateRent))
	mux.Del("/rent/:id", authMiddleware.ThenFunc(app.rentHandler.DeleteRent))
	mux.Post("/rent/archive", authMiddleware.ThenFunc(app.rentHandler.ArchiveRent))
	mux.Get("/rent/user/:user_id", authMiddleware.ThenFunc(app.rentHandler.GetRentsByUserID))
	mux.Post("/rent/filtered", standardMiddleware.ThenFunc(app.rentHandler.GetFilteredRentsPost))
	mux.Post("/rent/status", authMiddleware.ThenFunc(app.rentHandler.GetRentsByStatusAndUserID))
	mux.Post("/rent/confirm", authMiddleware.ThenFunc(app.rentConfirmationHandler.ConfirmRent))
	mux.Post("/rent/cancel", authMiddleware.ThenFunc(app.rentConfirmationHandler.CancelRent))
	mux.Post("/rent/done", authMiddleware.ThenFunc(app.rentConfirmationHandler.DoneRent))
	mux.Get("/images/rents/:filename", http.HandlerFunc(app.rentHandler.ServeRentsImage))
	mux.Get("/videos/rents/:filename", http.HandlerFunc(app.rentHandler.ServeRentVideo))
	mux.Post("/rent/filtered/:user_id", authMiddleware.ThenFunc(app.rentHandler.GetFilteredRentsWithLikes))
	mux.Get("/rent/rent_id/:rent_id/user/:user_id", standardMiddleware.ThenFunc(app.rentHandler.GetRentByRentIDAndUserID))

	// Rent Reviews
	mux.Post("/rent_review", authMiddleware.ThenFunc(app.rentReviewHandler.CreateRentReview))
	mux.Get("/rent_review/:rent_id", standardMiddleware.ThenFunc(app.rentReviewHandler.GetRentReviewsByRentID))
	mux.Put("/rent_review/:id", authMiddleware.ThenFunc(app.rentReviewHandler.UpdateRentReview))
	mux.Del("/rent_review/:id", authMiddleware.ThenFunc(app.rentReviewHandler.DeleteRentReview))

	// Reent Response
	mux.Post("/rent_responses", authMiddleware.ThenFunc(app.rentResponseHandler.CreateRentResponse))
	mux.Del("/rent_responses/:id", authMiddleware.ThenFunc(app.rentResponseHandler.CancelRentResponse))

	// Rent Favorites
	mux.Post("/rent_favorites", authMiddleware.ThenFunc(app.rentFavoriteHandler.AddRentToFavorites))
	mux.Del("/rent_favorites/user/:user_id/rent/:rent_id", authMiddleware.ThenFunc(app.rentFavoriteHandler.RemoveRentFromFavorites))
	mux.Get("/rent_favorites/check/user/:user_id/rent/:rent_id", standardMiddleware.ThenFunc(app.rentFavoriteHandler.IsRentFavorite))
	mux.Get("/rent_favorites/:user_id", standardMiddleware.ThenFunc(app.rentFavoriteHandler.GetRentFavoritesByUser))

	// Ad
	mux.Post("/ad", authMiddleware.ThenFunc(app.adHandler.CreateAd))
	mux.Get("/ad/get", standardMiddleware.ThenFunc(app.adHandler.GetAd))
	mux.Get("/admin/ad/get", adminAuthMiddleware.ThenFunc(app.adHandler.GetAdAdmin))
	mux.Get("/ad/:id", standardMiddleware.ThenFunc(app.adHandler.GetAdByID))
	mux.Put("/ad/:id", authMiddleware.ThenFunc(app.adHandler.UpdateAd))
	mux.Del("/ad/:id", authMiddleware.ThenFunc(app.adHandler.DeleteAd))
	mux.Post("/ad/archive", authMiddleware.ThenFunc(app.adHandler.ArchiveAd))
	mux.Get("/ad/user/:user_id", authMiddleware.ThenFunc(app.adHandler.GetAdByUserID))
	mux.Post("/ad/filtered", standardMiddleware.ThenFunc(app.adHandler.GetFilteredAdPost))
	mux.Post("/ad/status", authMiddleware.ThenFunc(app.adHandler.GetAdByStatusAndUserID))
	mux.Post("/ad/confirm", authMiddleware.ThenFunc(app.adConfirmationHandler.ConfirmAd))
	mux.Post("/ad/cancel", authMiddleware.ThenFunc(app.adConfirmationHandler.CancelAd))
	mux.Post("/ad/done", authMiddleware.ThenFunc(app.adConfirmationHandler.DoneAd))
	mux.Get("/images/ad/:filename", http.HandlerFunc(app.adHandler.ServeAdImage))
	mux.Get("/videos/ad/:filename", http.HandlerFunc(app.adHandler.ServeAdVideo))
	mux.Post("/ad/filtered/:user_id", authMiddleware.ThenFunc(app.adHandler.GetFilteredAdWithLikes))
	mux.Get("/ad/ad_id/:ad_id/user/:user_id", standardMiddleware.ThenFunc(app.adHandler.GetAdByAdIDAndUserID))

	// Ad Reviews
	mux.Post("/ad_review", authMiddleware.ThenFunc(app.adReviewHandler.CreateAdReview))
	mux.Get("/ad_review/:ad_id", standardMiddleware.ThenFunc(app.adReviewHandler.GetReviewsByAdID))
	mux.Put("/ad_review/:id", authMiddleware.ThenFunc(app.adReviewHandler.UpdateAdReview))
	mux.Del("/ad_review/:id", authMiddleware.ThenFunc(app.adReviewHandler.DeleteAdReview))

	// Ad Response
	mux.Post("/ad_responses", authMiddleware.ThenFunc(app.adResponseHandler.CreateAdResponse))
	mux.Del("/ad_responses/:id", authMiddleware.ThenFunc(app.adResponseHandler.CancelAdResponse))

	// Ad Favorites
	mux.Post("/ad_favorites", authMiddleware.ThenFunc(app.adFavoriteHandler.AddAdToFavorites))
	mux.Del("/ad_favorites/user/:user_id/ad/:ad_id", authMiddleware.ThenFunc(app.adFavoriteHandler.RemoveAdFromFavorites))
	mux.Get("/ad_favorites/check/user/:user_id/ad/:ad_id", standardMiddleware.ThenFunc(app.adFavoriteHandler.IsAdFavorite))
	mux.Get("/ad_favorites/:user_id", standardMiddleware.ThenFunc(app.adFavoriteHandler.GetAdFavoritesByUser))

	// Work Ad
	mux.Post("/work_ad", authMiddleware.ThenFunc(app.workAdHandler.CreateWorkAd))
	mux.Get("/work_ad/get", standardMiddleware.ThenFunc(app.workAdHandler.GetWorksAd))
	mux.Get("/admin/work_ad/get", adminAuthMiddleware.ThenFunc(app.workAdHandler.GetWorksAdAdmin))
	mux.Get("/work_ad/:id", standardMiddleware.ThenFunc(app.workAdHandler.GetWorkAdByID))
	mux.Put("/work_ad/:id", authMiddleware.ThenFunc(app.workAdHandler.UpdateWorkAd))
	mux.Del("/work_ad/:id", authMiddleware.ThenFunc(app.workAdHandler.DeleteWorkAd))
	mux.Post("/work_ad/archive", authMiddleware.ThenFunc(app.workAdHandler.ArchiveWorkAd))
	mux.Get("/work_ad/user/:user_id", authMiddleware.ThenFunc(app.workAdHandler.GetWorksAdByUserID))
	mux.Post("/work_ad/filtered", standardMiddleware.ThenFunc(app.workAdHandler.GetFilteredWorksAdPost))
	mux.Post("/work_ad/status", authMiddleware.ThenFunc(app.workAdHandler.GetWorksAdByStatusAndUserID))
	mux.Post("/work_ad/confirm", authMiddleware.ThenFunc(app.workAdConfirmationHandler.ConfirmWorkAd))
	mux.Post("/work_ad/cancel", authMiddleware.ThenFunc(app.workAdConfirmationHandler.CancelWorkAd))
	mux.Post("/work_ad/done", authMiddleware.ThenFunc(app.workAdConfirmationHandler.DoneWorkAd))
	mux.Get("/images/work_ad/:filename", http.HandlerFunc(app.workAdHandler.ServeWorkAdImage))
	mux.Get("/videos/work_ad/:filename", http.HandlerFunc(app.workAdHandler.ServeWorkAdVideo))
	mux.Post("/work_ad/filtered/:user_id", authMiddleware.ThenFunc(app.workAdHandler.GetFilteredWorksAdWithLikes))
	mux.Get("/work_ad/work_ad_id/:work_ad_id/user/:user_id", standardMiddleware.ThenFunc(app.workAdHandler.GetWorkAdByWorkIDAndUserID))

	// Work Ad Reviews
	mux.Post("/work_ad_review", authMiddleware.ThenFunc(app.workAdReviewHandler.CreateWorkAdReview))
	mux.Get("/work_ad_review/:work_ad_id", standardMiddleware.ThenFunc(app.workAdReviewHandler.GetWorkAdReviewsByWorkID))
	mux.Put("/work_ad_review/:id", authMiddleware.ThenFunc(app.workAdReviewHandler.UpdateWorkAdReview))
	mux.Del("/work_ad_review/:id", authMiddleware.ThenFunc(app.workAdReviewHandler.DeleteWorkAdReview))

	// Work Response
	mux.Post("/work_ad_responses", authMiddleware.ThenFunc(app.workAdResponseHandler.CreateWorkAdResponse))
	mux.Del("/work_ad_responses/:id", authMiddleware.ThenFunc(app.workAdResponseHandler.CancelWorkAdResponse))

	// Work Favorites
	mux.Post("/work_ad_favorites", authMiddleware.ThenFunc(app.workAdFavoriteHandler.AddWorkAdToFavorites))
	mux.Del("/work_ad_favorites/user/:user_id/work/:work_ad_id", authMiddleware.ThenFunc(app.workAdFavoriteHandler.RemoveWorkAdFromFavorites))
	mux.Get("/work_ad_favorites/check/user/:user_id/work/:work_ad_id", standardMiddleware.ThenFunc(app.workAdFavoriteHandler.IsWorkAdFavorite))
	mux.Get("/work_ad_favorites/:user_id", standardMiddleware.ThenFunc(app.workAdFavoriteHandler.GetWorkAdFavoritesByUser))

	// Rent Ad
	mux.Post("/rent_ad", authMiddleware.ThenFunc(app.rentAdHandler.CreateRentAd))
	mux.Get("/rent_ad/get", standardMiddleware.ThenFunc(app.rentAdHandler.GetRentsAd))
	mux.Get("/admin/rent_ad/get", adminAuthMiddleware.ThenFunc(app.rentAdHandler.GetRentsAdAdmin))
	mux.Get("/rent_ad/:id", standardMiddleware.ThenFunc(app.rentAdHandler.GetRentAdByID))
	mux.Put("/rent_ad/:id", authMiddleware.ThenFunc(app.rentAdHandler.UpdateRentAd))
	mux.Del("/rent_ad/:id", authMiddleware.ThenFunc(app.rentAdHandler.DeleteRentAd))
	mux.Post("/rent_ad/archive", authMiddleware.ThenFunc(app.rentAdHandler.ArchiveRentAd))
	mux.Get("/rent_ad/user/:user_id", authMiddleware.ThenFunc(app.rentAdHandler.GetRentsAdByUserID))
	mux.Post("/rent_ad/filtered", standardMiddleware.ThenFunc(app.rentAdHandler.GetFilteredRentsAdPost))
	mux.Post("/rent_ad/status", authMiddleware.ThenFunc(app.rentAdHandler.GetRentsAdByStatusAndUserID))
	mux.Post("/rent_ad/confirm", authMiddleware.ThenFunc(app.rentAdConfirmationHandler.ConfirmRentAd))
	mux.Post("/rent_ad/cancel", authMiddleware.ThenFunc(app.rentAdConfirmationHandler.CancelRentAd))
	mux.Post("/rent_ad/done", authMiddleware.ThenFunc(app.rentAdConfirmationHandler.DoneRentAd))
	mux.Get("/images/rents/:filename", http.HandlerFunc(app.rentAdHandler.ServeRentsAdImage))
	mux.Get("/videos/rent_ad/:filename", http.HandlerFunc(app.rentAdHandler.ServeRentAdVideo))
	mux.Post("/rent_ad/filtered/:user_id", authMiddleware.ThenFunc(app.rentAdHandler.GetFilteredRentsAdWithLikes))
	mux.Get("/rent_ad/rent_ad_id/:rent_ad_id/user/:user_id", standardMiddleware.ThenFunc(app.rentAdHandler.GetRentAdByRentIDAndUserID))

	// Rent Ad Reviews
	mux.Post("/rent_ad_review", authMiddleware.ThenFunc(app.rentAdReviewHandler.CreateRentAdReview))
	mux.Get("/rent_ad_review/:rent_ad_id", standardMiddleware.ThenFunc(app.rentAdReviewHandler.GetRentAdReviewsByRentID))
	mux.Put("/rent_ad_review/:id", authMiddleware.ThenFunc(app.rentAdReviewHandler.UpdateRentAdReview))
	mux.Del("/rent_ad_review/:id", authMiddleware.ThenFunc(app.rentAdReviewHandler.DeleteRentAdReview))

	// Reent Response
	mux.Post("/rent_ad_responses", authMiddleware.ThenFunc(app.rentAdResponseHandler.CreateRentAdResponse))
	mux.Del("/rent_ad_responses/:id", authMiddleware.ThenFunc(app.rentAdResponseHandler.CancelRentAdResponse))

	// Rent Ad Favorites
	mux.Post("/rent_ad_favorites", authMiddleware.ThenFunc(app.rentAdFavoriteHandler.AddRentAdToFavorites))
	mux.Del("/rent_ad_favorites/user/:user_id/rent_ad/:rent_ad_id", authMiddleware.ThenFunc(app.rentAdFavoriteHandler.RemoveRentAdFromFavorites))
	mux.Get("/rent_ad_favorites/check/user/:user_id/rent/:rent_ad_id", standardMiddleware.ThenFunc(app.rentAdFavoriteHandler.IsRentAdFavorite))
	mux.Get("/rent_ad_favorites/:user_id", standardMiddleware.ThenFunc(app.rentAdFavoriteHandler.GetRentAdFavoritesByUser))

	return standardMiddleware.Then(mux)
}
