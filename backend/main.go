package main

import (
	"backendAuction/controllers"
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

	// CORS middleware with environment variables already loaded
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

	go func() {
		utils.ScrapAllSites(db)
	}()

	authController := controllers.AuthController{DB: db}
	auctionController := controllers.AuctionsController{DB: db}

	authSubrouter := router.PathPrefix("/auth").Subrouter()
	authSubrouter.HandleFunc("/signup", authController.SignUp).Methods("POST", "OPTIONS")
	authSubrouter.HandleFunc("/login", authController.Login).Methods("POST", "OPTIONS")

	auctionsSubrouter := router.PathPrefix("/auctions").Subrouter()
	auctionsSubrouter.HandleFunc("/", auctionController.GetAuctions).Methods("GET", "OPTIONS")

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

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
