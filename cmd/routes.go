package main

import (
	"github.com/bmizerany/pat"
	"github.com/justinas/alice"
	"net/http"
	// httpSwagger "github.com/swaggo/http-swagger"
	// _ "naimuBack/docs"
)

func (app *application) JWTMiddlewareWithRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return app.JWTMiddleware(next, requiredRole)
	}
}

func (app *application) routes() http.Handler {
	standardMiddleware := alice.New(app.recoverPanic, app.logRequest, secureHeaders, makeResponseJSON)
	authMiddleware := standardMiddleware.Append(app.JWTMiddlewareWithRole("user"))
	adminAuthMiddleware := standardMiddleware.Append(app.JWTMiddlewareWithRole("admin"))

	mux := pat.New()

	// mux.Get("/swagger/", httpSwagger.WrapHandler)

	// Users
	mux.Post("/user", adminAuthMiddleware.ThenFunc(app.userHandler.CreateUser))                //
	mux.Get("/user", authMiddleware.ThenFunc(app.userHandler.GetUsers))                        //РАБОТАЕТ
	mux.Get("/user/token", authMiddleware.ThenFunc(app.userHandler.GetUserByToken))            //РАБОТАЕТ
	mux.Get("/user/:id", authMiddleware.ThenFunc(app.userHandler.GetUserByID))                 //РАБОТАЕТ
	mux.Put("/user/:id", authMiddleware.ThenFunc(app.userHandler.UpdateUser))                  //РАБОТАЕТ
	mux.Del("/user/:id", authMiddleware.ThenFunc(app.userHandler.DeleteUser))                  //ИСПРАВИТЬ
	mux.Post("/user/sign_up", standardMiddleware.ThenFunc(app.userHandler.SignUp))             //РАБОТАЕТ
	mux.Post("/user/sign_in", standardMiddleware.ThenFunc(app.userHandler.SignIn))             //РАБОТАЕТ
	mux.Post("/user/change_number", standardMiddleware.ThenFunc(app.userHandler.ChangeNumber)) //РАБОТАЕТ
	mux.Post("/user/change_email", standardMiddleware.ThenFunc(app.userHandler.ChangeEmail))   //РАБОТАЕТ
	mux.Put("/user/:id/city", authMiddleware.ThenFunc(app.userHandler.ChangeCityForUser))
	mux.Get("/docs/:filename", authMiddleware.ThenFunc(app.userHandler.ServeProofDocument))
	mux.Post("/user/:id/avatar", authMiddleware.ThenFunc(app.userHandler.UploadAvatar))
	mux.Get("/images/avatars/:filename", standardMiddleware.ThenFunc(app.userHandler.ServeAvatar))
	mux.Post("/user/:id/upgrade", authMiddleware.ThenFunc(app.userHandler.UpdateToWorker))
	mux.Post("/users/check_duplicate", standardMiddleware.ThenFunc(app.userHandler.CheckUserDuplicate))
	mux.Post("/user/request_reset", authMiddleware.ThenFunc(app.userHandler.RequestPasswordReset))
	mux.Post("/user/verify_reset_code", authMiddleware.ThenFunc(app.userHandler.VerifyResetCode))
	mux.Post("/user/reset_password", authMiddleware.ThenFunc(app.userHandler.ResetPassword))
	mux.Get("/subscription/:user_id", authMiddleware.ThenFunc(app.subscriptionHandler.GetSubscription))
	mux.Post("/robokassa/pay", standardMiddleware.ThenFunc(app.robokassaHandler.CreatePayment))
	mux.Post("/robokassa/result", standardMiddleware.ThenFunc(app.robokassaHandler.Result))

	mux.Get("/user/posts/:user_id", authMiddleware.ThenFunc(app.userItemsHandler.GetPostsByUserID))
	mux.Get("/user/ads/:user_id", authMiddleware.ThenFunc(app.userItemsHandler.GetAdsByUserID))
	mux.Get("/user/orders/:user_id", authMiddleware.ThenFunc(app.userItemsHandler.GetOrderHistoryByUserID))

	mux.Get("/ads", standardMiddleware.ThenFunc(app.adHandler.GetAds))

	// Service
	mux.Post("/service", authMiddleware.ThenFunc(app.serviceHandler.CreateService))                                 //РАБОТАЕТ
	mux.Get("/service/get", standardMiddleware.ThenFunc(app.serviceHandler.GetServices))                            //РАБОТАЕТ
	mux.Get("/service/:id", standardMiddleware.ThenFunc(app.serviceHandler.GetServiceByID))                         //РАБОТАЕТ
	mux.Put("/service/:id", authMiddleware.ThenFunc(app.serviceHandler.UpdateService))                              //РАБОТАЕТ
	mux.Del("/service/:id", authMiddleware.ThenFunc(app.serviceHandler.DeleteService))                              //РАБОТАЕТ
	mux.Get("/service/sort/:type/user/:user_id", standardMiddleware.ThenFunc(app.serviceHandler.GetServicesSorted)) //user_id - id пользователя который авторизован
	mux.Get("/service/user/:user_id", standardMiddleware.ThenFunc(app.serviceHandler.GetServiceByUserID))           //РАБОТАЕТ
	mux.Post("/service/filtered", standardMiddleware.ThenFunc(app.serviceHandler.GetFilteredServicesPost))          //РАБОТАЕТ
	mux.Post("/service/status", authMiddleware.ThenFunc(app.serviceHandler.GetServicesByStatusAndUserID))
	mux.Post("/service/confirm", authMiddleware.ThenFunc(app.serviceConfirmationHandler.ConfirmService))
	mux.Post("/service/cancel", authMiddleware.ThenFunc(app.serviceConfirmationHandler.CancelService))
	mux.Post("/service/done", authMiddleware.ThenFunc(app.serviceConfirmationHandler.DoneService))
	mux.Get("/images/services/:filename", http.HandlerFunc(app.serviceHandler.ServeServiceImage))
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
	mux.Get("/ws", standardMiddleware.ThenFunc(app.WebSocketHandler))

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

	// Service Response
	mux.Post("/responses", authMiddleware.ThenFunc(app.serviceResponseHandler.CreateServiceResponse))
	mux.Get("/responses/:user_id", authMiddleware.ThenFunc(app.userResponsesHandler.GetResponsesByUserID))

	// Work
	mux.Post("/work", authMiddleware.ThenFunc(app.workHandler.CreateWork))
	mux.Get("/work/get", standardMiddleware.ThenFunc(app.workHandler.GetWorks))
	mux.Get("/work/:id", standardMiddleware.ThenFunc(app.workHandler.GetWorkByID))
	mux.Put("/work/:id", authMiddleware.ThenFunc(app.workHandler.UpdateWork))
	mux.Del("/work/:id", authMiddleware.ThenFunc(app.workHandler.DeleteWork))
	mux.Get("/work/user/:user_id", authMiddleware.ThenFunc(app.workHandler.GetWorksByUserID))
	mux.Post("/work/filtered", standardMiddleware.ThenFunc(app.workHandler.GetFilteredWorksPost))
	mux.Post("/work/status", authMiddleware.ThenFunc(app.workHandler.GetWorksByStatusAndUserID))
	mux.Post("/work/confirm", authMiddleware.ThenFunc(app.workConfirmationHandler.ConfirmWork))
	mux.Post("/work/cancel", authMiddleware.ThenFunc(app.workConfirmationHandler.CancelWork))
	mux.Post("/work/done", authMiddleware.ThenFunc(app.workConfirmationHandler.DoneWork))
	mux.Get("/images/works/:filename", http.HandlerFunc(app.workHandler.ServeWorkImage))
	mux.Post("/work/filtered/:user_id", authMiddleware.ThenFunc(app.workHandler.GetFilteredWorksWithLikes))
	mux.Get("/work/work_id/:work_id/user/:user_id", standardMiddleware.ThenFunc(app.workHandler.GetWorkByWorkIDAndUserID))

	// Work Reviews
	mux.Post("/work_review", authMiddleware.ThenFunc(app.workReviewHandler.CreateWorkReview))
	mux.Get("/work_review/:work_id", standardMiddleware.ThenFunc(app.workReviewHandler.GetWorkReviewsByWorkID))
	mux.Put("/work_review/:id", authMiddleware.ThenFunc(app.workReviewHandler.UpdateWorkReview))
	mux.Del("/work_review/:id", authMiddleware.ThenFunc(app.workReviewHandler.DeleteWorkReview))

	// Work Response
	mux.Post("/work_responses", authMiddleware.ThenFunc(app.workResponseHandler.CreateWorkResponse))

	// Work Favorites
	mux.Post("/work_favorites", authMiddleware.ThenFunc(app.workFavoriteHandler.AddWorkToFavorites))
	mux.Del("/work_favorites/user/:user_id/work/:work_id", authMiddleware.ThenFunc(app.workFavoriteHandler.RemoveWorkFromFavorites))
	mux.Get("/work_favorites/check/user/:user_id/work/:work_id", standardMiddleware.ThenFunc(app.workFavoriteHandler.IsWorkFavorite))
	mux.Get("/work_favorites/:user_id", standardMiddleware.ThenFunc(app.workFavoriteHandler.GetWorkFavoritesByUser))

	// Rent
	mux.Post("/rent", authMiddleware.ThenFunc(app.rentHandler.CreateRent))
	mux.Get("/rent/get", standardMiddleware.ThenFunc(app.rentHandler.GetRents))
	mux.Get("/rent/:id", standardMiddleware.ThenFunc(app.rentHandler.GetRentByID))
	mux.Put("/rent/:id", authMiddleware.ThenFunc(app.rentHandler.UpdateRent))
	mux.Del("/rent/:id", authMiddleware.ThenFunc(app.rentHandler.DeleteRent))
	mux.Get("/rent/user/:user_id", authMiddleware.ThenFunc(app.rentHandler.GetRentsByUserID))
	mux.Post("/rent/filtered", standardMiddleware.ThenFunc(app.rentHandler.GetFilteredRentsPost))
	mux.Post("/rent/status", authMiddleware.ThenFunc(app.rentHandler.GetRentsByStatusAndUserID))
	mux.Post("/rent/confirm", authMiddleware.ThenFunc(app.rentConfirmationHandler.ConfirmRent))
	mux.Post("/rent/cancel", authMiddleware.ThenFunc(app.rentConfirmationHandler.CancelRent))
	mux.Post("/rent/done", authMiddleware.ThenFunc(app.rentConfirmationHandler.DoneRent))
	mux.Get("/images/rents/:filename", http.HandlerFunc(app.rentHandler.ServeRentsImage))
	mux.Post("/rent/filtered/:user_id", authMiddleware.ThenFunc(app.rentHandler.GetFilteredRentsWithLikes))
	mux.Get("/rent/rent_id/:rent_id/user/:user_id", standardMiddleware.ThenFunc(app.rentHandler.GetRentByRentIDAndUserID))

	// Rent Reviews
	mux.Post("/rent_review", authMiddleware.ThenFunc(app.rentReviewHandler.CreateRentReview))
	mux.Get("/rent_review/:rent_id", standardMiddleware.ThenFunc(app.rentReviewHandler.GetRentReviewsByRentID))
	mux.Put("/rent_review/:id", authMiddleware.ThenFunc(app.rentReviewHandler.UpdateRentReview))
	mux.Del("/rent_review/:id", authMiddleware.ThenFunc(app.rentReviewHandler.DeleteRentReview))

	// Reent Response
	mux.Post("/rent_responses", authMiddleware.ThenFunc(app.rentResponseHandler.CreateRentResponse))

	// Rent Favorites
	mux.Post("/rent_favorites", authMiddleware.ThenFunc(app.rentFavoriteHandler.AddRentToFavorites))
	mux.Del("/rent_favorites/user/:user_id/rent/:rent_id", authMiddleware.ThenFunc(app.rentFavoriteHandler.RemoveRentFromFavorites))
	mux.Get("/rent_favorites/check/user/:user_id/rent/:rent_id", standardMiddleware.ThenFunc(app.rentFavoriteHandler.IsRentFavorite))
	mux.Get("/rent_favorites/:user_id", standardMiddleware.ThenFunc(app.rentFavoriteHandler.GetRentFavoritesByUser))

	// Ad
	mux.Post("/ad", authMiddleware.ThenFunc(app.adHandler.CreateAd))
	mux.Get("/ad/get", standardMiddleware.ThenFunc(app.adHandler.GetAd))
	mux.Get("/ad/:id", standardMiddleware.ThenFunc(app.adHandler.GetAdByID))
	mux.Put("/ad/:id", authMiddleware.ThenFunc(app.adHandler.UpdateAd))
	mux.Del("/ad/:id", authMiddleware.ThenFunc(app.adHandler.DeleteAd))
	mux.Get("/ad/user/:user_id", authMiddleware.ThenFunc(app.adHandler.GetAdByUserID))
	mux.Post("/ad/filtered", standardMiddleware.ThenFunc(app.adHandler.GetFilteredAdPost))
	mux.Post("/ad/status", authMiddleware.ThenFunc(app.adHandler.GetAdByStatusAndUserID))
	mux.Post("/ad/confirm", authMiddleware.ThenFunc(app.adConfirmationHandler.ConfirmAd))
	mux.Post("/ad/cancel", authMiddleware.ThenFunc(app.adConfirmationHandler.CancelAd))
	mux.Post("/ad/done", authMiddleware.ThenFunc(app.adConfirmationHandler.DoneAd))
	mux.Get("/images/ad/:filename", http.HandlerFunc(app.adHandler.ServeAdImage))
	mux.Post("/ad/filtered/:user_id", authMiddleware.ThenFunc(app.adHandler.GetFilteredAdWithLikes))
	mux.Get("/ad/ad_id/:ad_id/user/:user_id", standardMiddleware.ThenFunc(app.adHandler.GetAdByAdIDAndUserID))

	// Ad Reviews
	mux.Post("/ad_review", authMiddleware.ThenFunc(app.adReviewHandler.CreateAdReview))
	mux.Get("/ad_review/:ad_id", standardMiddleware.ThenFunc(app.adReviewHandler.GetReviewsByAdID))
	mux.Put("/ad_review/:id", authMiddleware.ThenFunc(app.adReviewHandler.UpdateAdReview))
	mux.Del("/ad_review/:id", authMiddleware.ThenFunc(app.adReviewHandler.DeleteAdReview))

	// Ad Response
	mux.Post("/ad_responses", authMiddleware.ThenFunc(app.adResponseHandler.CreateAdResponse))

	// Ad Favorites
	mux.Post("/ad_favorites", authMiddleware.ThenFunc(app.adFavoriteHandler.AddAdToFavorites))
	mux.Del("/ad_favorites/user/:user_id/ad/:ad_id", authMiddleware.ThenFunc(app.adFavoriteHandler.RemoveAdFromFavorites))
	mux.Get("/ad_favorites/check/user/:user_id/ad/:ad_id", standardMiddleware.ThenFunc(app.adFavoriteHandler.IsAdFavorite))
	mux.Get("/ad_favorites/:user_id", standardMiddleware.ThenFunc(app.adFavoriteHandler.GetAdFavoritesByUser))

	// Work Ad
	mux.Post("/work_ad", authMiddleware.ThenFunc(app.workAdHandler.CreateWorkAd))
	mux.Get("/work_ad/get", standardMiddleware.ThenFunc(app.workAdHandler.GetWorksAd))
	mux.Get("/work_ad/:id", standardMiddleware.ThenFunc(app.workAdHandler.GetWorkAdByID))
	mux.Put("/work_ad/:id", authMiddleware.ThenFunc(app.workAdHandler.UpdateWorkAd))
	mux.Del("/work_ad/:id", authMiddleware.ThenFunc(app.workAdHandler.DeleteWorkAd))
	mux.Get("/work_ad/user/:user_id", authMiddleware.ThenFunc(app.workAdHandler.GetWorksAdByUserID))
	mux.Post("/work_ad/filtered", standardMiddleware.ThenFunc(app.workAdHandler.GetFilteredWorksAdPost))
	mux.Post("/work_ad/status", authMiddleware.ThenFunc(app.workAdHandler.GetWorksAdByStatusAndUserID))
	mux.Post("/work_ad/confirm", authMiddleware.ThenFunc(app.workAdConfirmationHandler.ConfirmWorkAd))
	mux.Post("/work_ad/cancel", authMiddleware.ThenFunc(app.workAdConfirmationHandler.CancelWorkAd))
	mux.Post("/work_ad/done", authMiddleware.ThenFunc(app.workAdConfirmationHandler.DoneWorkAd))
	mux.Get("/images/work_ad/:filename", http.HandlerFunc(app.workAdHandler.ServeWorkAdImage))
	mux.Post("/work_ad/filtered/:user_id", authMiddleware.ThenFunc(app.workAdHandler.GetFilteredWorksAdWithLikes))
	mux.Get("/work_ad/work_ad_id/:work_ad_id/user/:user_id", standardMiddleware.ThenFunc(app.workAdHandler.GetWorkAdByWorkIDAndUserID))

	// Work Ad Reviews
	mux.Post("/work_ad_review", authMiddleware.ThenFunc(app.workAdReviewHandler.CreateWorkAdReview))
	mux.Get("/work_ad_review/:work_ad_id", standardMiddleware.ThenFunc(app.workAdReviewHandler.GetWorkAdReviewsByWorkID))
	mux.Put("/work_ad_review/:id", authMiddleware.ThenFunc(app.workAdReviewHandler.UpdateWorkAdReview))
	mux.Del("/work_ad_review/:id", authMiddleware.ThenFunc(app.workAdReviewHandler.DeleteWorkAdReview))

	// Work Response
	mux.Post("/work_ad_responses", authMiddleware.ThenFunc(app.workAdResponseHandler.CreateWorkAdResponse))

	// Work Favorites
	mux.Post("/work_ad_favorites", authMiddleware.ThenFunc(app.workAdFavoriteHandler.AddWorkAdToFavorites))
	mux.Del("/work_ad_favorites/user/:user_id/work/:work_ad_id", authMiddleware.ThenFunc(app.workAdFavoriteHandler.RemoveWorkAdFromFavorites))
	mux.Get("/work_ad_favorites/check/user/:user_id/work/:work_ad_id", standardMiddleware.ThenFunc(app.workAdFavoriteHandler.IsWorkAdFavorite))
	mux.Get("/work_ad_favorites/:user_id", standardMiddleware.ThenFunc(app.workAdFavoriteHandler.GetWorkAdFavoritesByUser))

	// Rent Ad
	mux.Post("/rent_ad", authMiddleware.ThenFunc(app.rentAdHandler.CreateRentAd))
	mux.Get("/rent_ad/get", standardMiddleware.ThenFunc(app.rentAdHandler.GetRentsAd))
	mux.Get("/rent_ad/:id", standardMiddleware.ThenFunc(app.rentAdHandler.GetRentAdByID))
	mux.Put("/rent_ad/:id", authMiddleware.ThenFunc(app.rentAdHandler.UpdateRentAd))
	mux.Del("/rent_ad/:id", authMiddleware.ThenFunc(app.rentAdHandler.DeleteRentAd))
	mux.Get("/rent_ad/user/:user_id", authMiddleware.ThenFunc(app.rentAdHandler.GetRentsAdByUserID))
	mux.Post("/rent_ad/filtered", standardMiddleware.ThenFunc(app.rentAdHandler.GetFilteredRentsAdPost))
	mux.Post("/rent_ad/status", authMiddleware.ThenFunc(app.rentAdHandler.GetRentsAdByStatusAndUserID))
	mux.Post("/rent_ad/confirm", authMiddleware.ThenFunc(app.rentAdConfirmationHandler.ConfirmRentAd))
	mux.Post("/rent_ad/cancel", authMiddleware.ThenFunc(app.rentAdConfirmationHandler.CancelRentAd))
	mux.Post("/rent_ad/done", authMiddleware.ThenFunc(app.rentAdConfirmationHandler.DoneRentAd))
	mux.Get("/images/rents/:filename", http.HandlerFunc(app.rentAdHandler.ServeRentsAdImage))
	mux.Post("/rent_ad/filtered/:user_id", authMiddleware.ThenFunc(app.rentAdHandler.GetFilteredRentsAdWithLikes))
	mux.Get("/rent_ad/rent_ad_id/:rent_ad_id/user/:user_id", standardMiddleware.ThenFunc(app.rentAdHandler.GetRentAdByRentIDAndUserID))

	// Rent Ad Reviews
	mux.Post("/rent_ad_review", authMiddleware.ThenFunc(app.rentAdReviewHandler.CreateRentAdReview))
	mux.Get("/rent_ad_review/:rent_ad_id", standardMiddleware.ThenFunc(app.rentAdReviewHandler.GetRentAdReviewsByRentID))
	mux.Put("/rent_ad_review/:id", authMiddleware.ThenFunc(app.rentAdReviewHandler.UpdateRentAdReview))
	mux.Del("/rent_ad_review/:id", authMiddleware.ThenFunc(app.rentAdReviewHandler.DeleteRentAdReview))

	// Reent Response
	mux.Post("/rent_ad_responses", authMiddleware.ThenFunc(app.rentAdResponseHandler.CreateRentAdResponse))

	// Rent Ad Favorites
	mux.Post("/rent_ad_favorites", authMiddleware.ThenFunc(app.rentAdFavoriteHandler.AddRentAdToFavorites))
	mux.Del("/rent_ad_favorites/user/:user_id/rent_ad/:rent_ad_id", authMiddleware.ThenFunc(app.rentAdFavoriteHandler.RemoveRentAdFromFavorites))
	mux.Get("/rent_ad_favorites/check/user/:user_id/rent/:rent_ad_id", standardMiddleware.ThenFunc(app.rentAdFavoriteHandler.IsRentAdFavorite))
	mux.Get("/rent_ad_favorites/:user_id", standardMiddleware.ThenFunc(app.rentAdFavoriteHandler.GetRentAdFavoritesByUser))

	return standardMiddleware.Then(mux)
}
