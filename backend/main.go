package main

import (
	"backendAuction/config"
	"backendAuction/controllers"
	"backendAuction/middleware"
	"backendAuction/utils"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables first
	if os.Getenv("ENV") != "PROD" {
		if err := godotenv.Load(); err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	router := mux.NewRouter().StrictSlash(true)

	router.Use(middleware.CacheMiddleware)
	// CORS middleware
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			frontendURL := config.GetFrontendURL()
			w.Header().Set("Access-Control-Allow-Origin", frontendURL)
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Cache-Control", "public, max-age=300")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	dbURL := config.GetDBURL()
	log.Println("Connecting to database at:", dbURL)

	db := utils.InitDb(dbURL)
	utils.InitTables(db)
	// utils.ScrapAllSites(db)

	// Initialize controllers
	authController := controllers.AuthController{DB: db}
	auctionController := controllers.AuctionsController{DB: db}
	favoritesController := controllers.FavoritesController{DB: db}
	scrapingController := controllers.ScrapingController{DB: db}

	// Auth routes
	authSubrouter := router.PathPrefix("/auth").Subrouter()
	authSubrouter.HandleFunc("/signup", authController.SignUp).Methods("POST", "OPTIONS")
	authSubrouter.HandleFunc("/login", authController.Login).Methods("POST", "OPTIONS")
	authSubrouter.HandleFunc("/logout", authController.Logout).Methods("POST", "OPTIONS")

	// Auctions routes
	auctionsSubrouter := router.PathPrefix("/auctions").Subrouter()
	auctionsSubrouter.HandleFunc("", auctionController.GetAuctions).Methods("GET", "OPTIONS")

	// Favorites routes
	favoritesSubrouter := router.PathPrefix("/favorites").Subrouter()
	favoritesSubrouter.HandleFunc("", middleware.AuthMiddleware(favoritesController.GetFavorites)).Methods("GET", "OPTIONS")
	favoritesSubrouter.HandleFunc("/add", middleware.AuthMiddleware(favoritesController.AddFavorite)).Methods("POST", "OPTIONS")
	favoritesSubrouter.HandleFunc("/remove", middleware.AuthMiddleware(favoritesController.RemoveFavorite)).Methods("POST", "OPTIONS")

	// Scraping routes
	scrapingSubrouter := router.PathPrefix("/scraping").Subrouter()
	scrapingSubrouter.HandleFunc("/start", scrapingController.StartScraping).Methods("POST", "OPTIONS")

	// Health check route
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Start server
	port := config.GetPort()

	srv := &http.Server{
		Handler:      router,
		Addr:         ":" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Printf("Server is running on port %s", port)
	log.Fatal(srv.ListenAndServe())
}
