package main

import (
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

	// CORS middleware
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			frontendURL := os.Getenv("FRONTEND_URL")
			if frontendURL == "" {
				frontendURL = "http://localhost:4321"
			}

			w.Header().Set("Access-Control-Allow-Origin", frontendURL)
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL is not set in the environment")
	}
	log.Println("Connecting to database at:", dbURL)

	db := utils.InitDb(dbURL)
	utils.InitTables(db)

	// Initialize controllers
	authController := controllers.AuthController{DB: db}
	auctionController := controllers.AuctionsController{DB: db}
	favoritesController := controllers.FavoritesController{DB: db}

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

	// Health check route
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	srv := &http.Server{
		Handler:      router,
		Addr:         ":" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Printf("Server is running on port %s", port)
	log.Fatal(srv.ListenAndServe())
}
