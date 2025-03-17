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

	err := godotenv.Load()
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

	go func() {
		utils.ScrapAllSites(db)
	}()

	authController := controllers.AuthController{DB: db}
	auctionController := controllers.AuctionsController{DB: db}

	authSubrouter := router.PathPrefix("/auth").Methods("POST").Subrouter()

	authSubrouter.HandleFunc("/signup", authController.SignUp)
	authSubrouter.HandleFunc("/login", authController.Login)

	auctionsSubrouter := router.PathPrefix("/auctions").Methods("GET").Subrouter()
	// auctionsSubrouter.Use(middleware.JwtValidator)
	auctionsSubrouter.HandleFunc("/", auctionController.GetAuctions)

	srv := &http.Server{
		Handler: router,
		Addr:    "127.0.0.1:8000",
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())

}
