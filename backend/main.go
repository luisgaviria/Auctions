package main

import (
	"backendAuction/config"
	"backendAuction/utils"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables first
	if os.Getenv("ENV") != "PROD" {
		if err := godotenv.Load(); err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	dbURL := config.GetDBURL()
	// Log only the host so credentials are never written to Railway/stdout logs.
	if parsed, err := url.Parse(dbURL); err == nil {
		log.Printf("[db] connecting to host=%s", parsed.Host)
	} else {
		log.Println("[db] connecting to database (url unparseable)")
	}

	db := utils.InitDb(dbURL)
	utils.InitTables(db)

	// Purge past auctions before each scrape run so stale records don't linger.
	if res, err := db.Exec("DELETE FROM auctions WHERE date IS NOT NULL AND date < (CURRENT_TIMESTAMP AT TIME ZONE 'America/New_York')::date"); err != nil {
		log.Printf("[purge] failed to delete past auctions: %v", err)
	} else {
		n, _ := res.RowsAffected()
		log.Printf("[purge] deleted %d past auctions", n)
	}

	// utils.ScrapAllSites(db)

	router := newRouter(db, config.GetAllowedOrigins())

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
