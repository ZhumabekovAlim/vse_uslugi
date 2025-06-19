package main

import (
	"log"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/bmizerany/pat"
	"github.com/justinas/alice"
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
	mux.Get("/user/:id", authMiddleware.ThenFunc(app.userHandler.GetUserByID))                 //РАБОТАЕТ
	mux.Put("/user/:id", authMiddleware.ThenFunc(app.userHandler.UpdateUser))                  //РАБОТАЕТ
	mux.Del("/user/:id", authMiddleware.ThenFunc(app.userHandler.DeleteUser))                  //ИСПРАВИТЬ
	mux.Post("/user/sign_up", standardMiddleware.ThenFunc(app.userHandler.SignUp))             //РАБОТАЕТ
	mux.Post("/user/sign_in", standardMiddleware.ThenFunc(app.userHandler.SignIn))             //РАБОТАЕТ
	mux.Post("/user/change_number", standardMiddleware.ThenFunc(app.userHandler.ChangeNumber)) //РАБОТАЕТ
	mux.Post("/user/change_email", standardMiddleware.ThenFunc(app.userHandler.ChangeEmail))   //РАБОТАЕТ
	mux.Put("/user/:id/city", authMiddleware.ThenFunc(app.userHandler.ChangeCityForUser))
	mux.Post("/user/:id/upgrade", authMiddleware.ThenFunc(app.userHandler.UpdateToWorker))
	mux.Post("/users/check_duplicate", standardMiddleware.ThenFunc(app.userHandler.CheckUserDuplicate))
	mux.Post("/user/request_reset", authMiddleware.ThenFunc(app.userHandler.RequestPasswordReset))
	mux.Post("/user/verify_reset_code", authMiddleware.ThenFunc(app.userHandler.VerifyResetCode))
	mux.Post("/user/reset_password", authMiddleware.ThenFunc(app.userHandler.ResetPassword))

	// Service
	mux.Post("/service", authMiddleware.ThenFunc(app.serviceHandler.CreateService))                             //РАБОТАЕТ
	mux.Get("/service/get", authMiddleware.ThenFunc(app.serviceHandler.GetServices))                            //РАБОТАЕТ
	mux.Get("/service/:id", authMiddleware.ThenFunc(app.serviceHandler.GetServiceByID))                         //РАБОТАЕТ
	mux.Put("/service/:id", authMiddleware.ThenFunc(app.serviceHandler.UpdateService))                          //РАБОТАЕТ
	mux.Del("/service/:id", authMiddleware.ThenFunc(app.serviceHandler.DeleteService))                          //РАБОТАЕТ
	mux.Get("/service/sort/:type/user/:user_id", authMiddleware.ThenFunc(app.serviceHandler.GetServicesSorted)) //user_id - id пользователя который авторизован
	mux.Get("/service/user/:user_id", authMiddleware.ThenFunc(app.serviceHandler.GetServiceByUserID))           //РАБОТАЕТ
	mux.Post("/service/filtered", authMiddleware.ThenFunc(app.serviceHandler.GetFilteredServicesPost))          //РАБОТАЕТ
	mux.Post("/service/filter", authMiddleware.ThenFunc(app.serviceHandler.GetServicesByStatusAndUserID))

	// Categories
	mux.Post("/category", authMiddleware.ThenFunc(app.categoryHandler.CreateCategory)) //РАБОТАЕТ
	mux.Get("/category", authMiddleware.ThenFunc(app.categoryHandler.GetAllCategories))
	mux.Get("/category/:id", authMiddleware.ThenFunc(app.categoryHandler.GetCategoryByID))
	mux.Put("/category/:id", authMiddleware.ThenFunc(app.categoryHandler.UpdateCategory))
	mux.Del("/category/:id", authMiddleware.ThenFunc(app.categoryHandler.DeleteCategory))
	mux.Get("/uploads/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Trying to serve:", r.URL.Path)
		http.StripPrefix("/uploads/", http.FileServer(http.Dir("cmd/uploads"))).ServeHTTP(w, r)
	}))

	// Reviews
	mux.Post("/review", authMiddleware.ThenFunc(app.reviewsHandler.CreateReview))                     //РАБОТАЕТ
	mux.Get("/review/:service_id", authMiddleware.ThenFunc(app.reviewsHandler.GetReviewsByServiceID)) //РАБОТАЕТ
	mux.Put("/review/:id", authMiddleware.ThenFunc(app.reviewsHandler.UpdateReview))                  //РАБОТАЕТ
	mux.Del("/review/:id", authMiddleware.ThenFunc(app.reviewsHandler.DeleteReview))                  //РАБОТАЕТ

	// Service Favorites
	mux.Post("/favorites", authMiddleware.ThenFunc(app.serviceFavorite.AddToFavorites)) //РАБОТАЕТ
	mux.Del("/favorites/user/:user_id/service/:service_is", authMiddleware.ThenFunc(app.serviceFavorite.RemoveFromFavorites))
	mux.Get("/favorites/check/user/:user_id/service/:service_id", authMiddleware.ThenFunc(app.serviceFavorite.IsFavorite)) //РАБОТАЕТ
	mux.Get("/favorites/:user_id", authMiddleware.ThenFunc(app.serviceFavorite.GetFavoritesByUser))                        //РАБОТАЕТ

	// Subcategories
	mux.Post("/subcategory", authMiddleware.ThenFunc(app.subcategoryHandler.CreateSubcategory))
	mux.Get("/subcategory", authMiddleware.ThenFunc(app.subcategoryHandler.GetAllSubcategories))
	mux.Get("/subcategory/cat/:category_id", authMiddleware.ThenFunc(app.subcategoryHandler.GetByCategory))
	mux.Get("/subcategory/:id", authMiddleware.ThenFunc(app.subcategoryHandler.GetSubcategoryByID))
	mux.Put("/subcategory/:id", authMiddleware.ThenFunc(app.subcategoryHandler.UpdateSubcategoryByID))
	mux.Del("/subcategory/:id", authMiddleware.ThenFunc(app.subcategoryHandler.DeleteSubcategoryByID))

	// City
	mux.Post("/city", authMiddleware.ThenFunc(app.cityHandler.CreateCity))
	mux.Get("/city", authMiddleware.ThenFunc(app.cityHandler.GetCities))
	mux.Get("/city/:id", authMiddleware.ThenFunc(app.cityHandler.GetCityByID))
	mux.Put("/city/:id", authMiddleware.ThenFunc(app.cityHandler.UpdateCity))
	mux.Del("/city/:id", authMiddleware.ThenFunc(app.cityHandler.DeleteCity))

	// Chat
	mux.Get("/ws", authMiddleware.ThenFunc(app.WebSocketHandler))

	mux.Post("/api/chats", authMiddleware.ThenFunc(app.chatHandler.CreateChat))
	mux.Get("/api/chats/:id", authMiddleware.ThenFunc(app.chatHandler.GetChatByID))
	mux.Get("/api/chats", authMiddleware.ThenFunc(app.chatHandler.GetAllChats))
	mux.Del("/api/chats/:id", authMiddleware.ThenFunc(app.chatHandler.DeleteChat))

	mux.Post("/api/messages", authMiddleware.ThenFunc(app.messageHandler.CreateMessage))
	mux.Get("/api/messages/:chatId", authMiddleware.ThenFunc(app.messageHandler.GetMessagesForChat))
	mux.Del("/api/messages/:messageId", authMiddleware.ThenFunc(app.messageHandler.DeleteMessage))

	mux.Get("/api/users/messages", authMiddleware.ThenFunc(app.messageHandler.GetMessagesByUserIDs))

	return standardMiddleware.Then(mux)
}
func fileServerWithContentType(root string) http.Handler {
	fs := http.FileServer(http.Dir(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(root, r.URL.Path[len("/uploads/"):]) // убираем префикс вручную
		ext := filepath.Ext(path)
		mimeType := mime.TypeByExtension(ext)
		if mimeType != "" {
			w.Header().Set("Content-Type", mimeType)
		} else {
			w.Header().Set("Content-Type", "application/octet-stream")
		}
		fs.ServeHTTP(w, r)
	})
}
