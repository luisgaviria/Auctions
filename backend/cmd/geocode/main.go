// cmd/geocode/main.go
//
// Geocode worker — finds every auction with lat='0' and fills in
// real coordinates from the Radar API (5 req/s rate limit).
//
// Each auction is geocoded exactly once. If coordinates already exist
// (lat != '0') the row is skipped, even across repeated runs.
//
// Usage (from the backend/ directory):
//
//	MAPTILER_API_KEY=... go run ./cmd/geocode
//
// Or with the full .env file in development:
//
//	go run ./cmd/geocode
package main

import (
	"backendAuction/utils"
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load .env in non-prod environments (GitHub Actions sets ENV=PROD).
	if os.Getenv("ENV") != "PROD" {
		if err := godotenv.Load(); err != nil {
			log.Fatal("error loading .env file")
		}
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("db ping: %v", err)
	}
	log.Println("[geocode] connected to database")

	// 10-minute ceiling — enough to backfill hundreds of rows at 5 req/s.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	utils.RunGeocodeWorker(ctx, db)
}
