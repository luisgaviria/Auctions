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

	godotenv.Load()

	db := utils.InitDb(os.Getenv("DB_URL"))

	utils.InitTables(db)

	controller := controllers.Controller{DB: db}

	authSubrouter := router.PathPrefix("/auth").Methods("POST").Subrouter()

	authSubrouter.HandleFunc("/signup", controller.SignUp)
	authSubrouter.HandleFunc("/login", controller.Login)

	srv := &http.Server{
		Handler: router,
		Addr:    "127.0.0.1:8000",
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())

}
