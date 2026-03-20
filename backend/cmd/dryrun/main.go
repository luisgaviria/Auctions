// cmd/dryrun/main.go
// Full population run — triggers all 12 scrapers via ScrapAllSites, then
// prints a grouped count of every auction in the database.
//
// Usage (from the backend/ directory):
//
//	go run ./cmd/dryrun
package main

import (
	"backendAuction/utils"
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
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
	log.Println("connected to Supabase")

	// 15 minutes: generous ceiling for 12 parallel scrapers + cleanup phase.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// ── Full scrape + upsert + cleanup ───────────────────────────────────────
	log.Println("=== starting full population run (12 scrapers) ===")
	start := time.Now()
	utils.ScrapAllSites(ctx, db)
	log.Printf("=== scrape complete in %s ===", time.Since(start).Round(time.Second))

	// ── Final DB verification ─────────────────────────────────────────────────
	log.Println("=== final database counts ===")
	rows, err := db.QueryContext(ctx, `
		SELECT   site_name,
		         status,
		         COUNT(*) AS n
		FROM     auctions
		GROUP BY site_name, status
		ORDER BY site_name, status;
	`)
	if err != nil {
		log.Fatalf("count query: %v", err)
	}
	defer rows.Close()

	type row struct {
		site, status string
		n            int
	}
	var results []row
	totals := map[string]int{}
	grand := 0

	for rows.Next() {
		var r row
		if err := rows.Scan(&r.site, &r.status, &r.n); err != nil {
			log.Printf("scan: %v", err)
			continue
		}
		results = append(results, r)
		totals[r.site] += r.n
		grand += r.n
	}

	fmt.Printf("\n%-18s %-26s %s\n", "site_name", "status", "count")
	fmt.Println("------------------------------------------------------------")
	prevSite := ""
	for _, r := range results {
		if prevSite != "" && r.site != prevSite {
			fmt.Printf("%-18s %-26s %d\n", "", "── site total ──", totals[prevSite])
			fmt.Println()
		}
		fmt.Printf("%-18s %-26s %d\n", r.site, r.status, r.n)
		prevSite = r.site
	}
	if prevSite != "" {
		fmt.Printf("%-18s %-26s %d\n", "", "── site total ──", totals[prevSite])
	}
	fmt.Println("------------------------------------------------------------")
	fmt.Printf("%-18s %-26s %d\n\n", "GRAND TOTAL", "", grand)

	if grand == 0 {
		log.Println("WARNING: 0 rows — check scraper logs above for errors")
	} else {
		log.Printf("OK: %d total auction rows in Supabase across %d sites", grand, len(totals))
	}
}
