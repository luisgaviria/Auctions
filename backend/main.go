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
	router := mux.NewRouter().StrictSlash(true)

	var err error
	if os.Getenv("ENV") != "PROD" {
		err = godotenv.Load()
	}
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL is not set in the environment")
	}
	log.Println("Connecting to database at:", dbURL)

	db := utils.InitDb(dbURL)

	utils.InitTables(db)

	// go func() {
	// 	utils.ScrapAllSites(db)
	// }()

	authController := controllers.AuthController{DB: db}
	auctionController := controllers.AuctionsController{DB: db}

	authSubrouter := router.PathPrefix("/auth").Methods("POST").Subrouter()

	authSubrouter.HandleFunc("/signup", authController.SignUp)
	authSubrouter.HandleFunc("/login", authController.Login)

	auctionsSubrouter := router.PathPrefix("/auctions").Methods("GET").Subrouter()
	// auctionsSubrouter.Use(middleware.JwtValidator)
	auctionsSubrouter.HandleFunc("/", auctionController.GetAuctions)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000" // Default to port 8000 if PORT is not set
	}

	srv := &http.Server{
		Handler: router,
		Addr:    "127.0.0.1:" + port,
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Printf("Server is running on port %s", port)
	log.Fatal(srv.ListenAndServe())
}
