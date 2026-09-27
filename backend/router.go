package main

import (
	"backendAuction/controllers"
	"backendAuction/middleware"
	"database/sql"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func newRouter(db *sql.DB, allowedOrigins []string) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	router.Use(middleware.CacheMiddleware)

	allowedSet := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowedSet[origin] = struct{}{}
	}
	log.Printf("[cors] allowed origins: %v", allowedOrigins)

	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if _, ok := allowedSet[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Cache-Control", "public, max-age=300")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	authController := controllers.AuthController{DB: db}
	auctionController := controllers.AuctionsController{DB: db}
	favoritesController := controllers.FavoritesController{DB: db}
	scrapingController := controllers.ScrapingController{DB: db}
	locationsController := controllers.LocationsController{DB: db}

	authSubrouter := router.PathPrefix("/auth").Subrouter()
	authSubrouter.HandleFunc("/signup", authController.SignUp).Methods("POST", "OPTIONS")
	authSubrouter.HandleFunc("/login", authController.Login).Methods("POST", "OPTIONS")
	authSubrouter.HandleFunc("/logout", authController.Logout).Methods("POST", "OPTIONS")

	auctionsSubrouter := router.PathPrefix("/auctions").Subrouter()
	auctionsSubrouter.HandleFunc("", auctionController.GetAuctions).Methods("GET", "OPTIONS")
	auctionsSubrouter.HandleFunc("/slugs", auctionController.GetTopSlugs).Methods("GET", "OPTIONS")
	auctionsSubrouter.HandleFunc("/{county_slug}/{city_slug}", auctionController.GetAuctionsBySlug).Methods("GET", "OPTIONS")

	router.HandleFunc("/report/{address_slug}", auctionController.GetReport).Methods("GET", "OPTIONS")

	locationsSubrouter := router.PathPrefix("/locations").Subrouter()
	locationsSubrouter.HandleFunc("/counties", locationsController.GetCounties).Methods("GET", "OPTIONS")
	locationsSubrouter.HandleFunc("/counties/{county_slug}/cities", locationsController.GetCitiesByCounty).Methods("GET", "OPTIONS")

	favoritesSubrouter := router.PathPrefix("/favorites").Subrouter()
	favoritesSubrouter.HandleFunc("", middleware.AuthMiddleware(favoritesController.GetFavorites)).Methods("GET", "OPTIONS")
	favoritesSubrouter.HandleFunc("/add", middleware.AuthMiddleware(favoritesController.AddFavorite)).Methods("POST", "OPTIONS")
	favoritesSubrouter.HandleFunc("/remove", middleware.AuthMiddleware(favoritesController.RemoveFavorite)).Methods("POST", "OPTIONS")

	scrapingSubrouter := router.PathPrefix("/scraping").Subrouter()
	scrapingSubrouter.HandleFunc("/start", scrapingController.StartScraping).Methods("POST", "OPTIONS")

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	return router
}
