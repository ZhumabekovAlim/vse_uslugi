package main

import (
	"net/http"

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

	// Service
	mux.Post("/service", authMiddleware.ThenFunc(app.serviceHandler.CreateService))                             //РАБОТАЕТ
	mux.Get("/service/get", authMiddleware.ThenFunc(app.serviceHandler.GetServices))                            //РАБОТАЕТ
	mux.Get("/service/:id", authMiddleware.ThenFunc(app.serviceHandler.GetServiceByID))                         //РАБОТАЕТ
	mux.Put("/service/:id", authMiddleware.ThenFunc(app.serviceHandler.UpdateService))                          //РАБОТАЕТ
	mux.Del("/service/:id", authMiddleware.ThenFunc(app.serviceHandler.DeleteService))                          //РАБОТАЕТ
	mux.Get("/service/sort/:type/user/:user_id", authMiddleware.ThenFunc(app.serviceHandler.GetServicesSorted)) //user_id - id пользователя который авторизован
	mux.Get("/service/user/:user_id", authMiddleware.ThenFunc(app.serviceHandler.GetServiceByUserID))           //РАБОТАЕТ
	mux.Post("/service/filtered", authMiddleware.ThenFunc(app.serviceHandler.GetFilteredServicesPost))          //РАБОТАЕТ

	// Categories
	mux.Post("/category", authMiddleware.ThenFunc(app.categoryHandler.CreateCategory)) //РАБОТАЕТ
	mux.Get("/category", authMiddleware.ThenFunc(app.categoryHandler.GetAllCategories))
	mux.Get("/category/:id", authMiddleware.ThenFunc(app.categoryHandler.GetCategoryByID))
	mux.Put("/category/:id", authMiddleware.ThenFunc(app.categoryHandler.UpdateCategory))
	mux.Del("/category/:id", authMiddleware.ThenFunc(app.categoryHandler.DeleteCategory))

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

	return standardMiddleware.Then(mux)
}
