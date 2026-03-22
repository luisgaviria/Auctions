package utils

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// mapTilerResponse is the minimal GeoJSON FeatureCollection shape returned
// by the MapTiler forward-geocoding API.
// features[0].center is [longitude, latitude] (GeoJSON order).
type mapTilerResponse struct {
	Features []struct {
		Center [2]float64 `json:"center"` // [lng, lat]
	} `json:"features"`
}

// auctionRow holds the columns needed for geocoding.
type auctionRow struct {
	ID      int
	Address string
	City    sql.NullString
	State   sql.NullString
}

const (
	// mapTilerGeocodeBase is the MapTiler forward-geocoding endpoint.
	// Query goes in the path: /geocoding/{query}.json?key={key}
	mapTilerGeocodeBase = "https://api.maptiler.com/geocoding"

	// requestsPerSecond caps how fast we hit the MapTiler API.
	// At 5 req/s a backfill of 200 rows takes ~40 s.
	requestsPerSecond = 5
)

// RunGeocodeWorker finds every auction row whose lat is still '0'
// (never geocoded), calls the MapTiler forward-geocoding API, and writes
// the coordinates back to the database.
//
// Guarantees:
//   - Each row is geocoded exactly once: we query WHERE lat = '0', and on
//     success write a real coordinate — future runs skip it automatically.
//   - If an address is later updated by the scraper, existing coordinates are
//     preserved; the upsert SQL never overwrites lat/lng.
//   - On API error or no-results the row stays at lat='0' and will be retried
//     on the next run.
func RunGeocodeWorker(ctx context.Context, db *sql.DB) {
	apiKey := os.Getenv("MAPTILER_API_KEY")
	if apiKey == "" {
		log.Fatal("[geocode] MAPTILER_API_KEY is not set")
	}

	rows, err := fetchUngeocoded(ctx, db)
	if err != nil {
		log.Fatalf("[geocode] failed to query ungeocoded rows: %v", err)
	}
	if len(rows) == 0 {
		log.Println("[geocode] all auctions already have coordinates — nothing to do")
		return
	}
	log.Printf("[geocode] found %d auction(s) to geocode", len(rows))

	// Ticker enforces the rate limit between API calls.
	ticker := time.NewTicker(time.Second / requestsPerSecond)
	defer ticker.Stop()

	client := &http.Client{Timeout: 10 * time.Second}

	success, skipped := 0, 0
	for _, row := range rows {
		// Respect context cancellation (e.g. GitHub Actions job timeout).
		select {
		case <-ctx.Done():
			log.Printf("[geocode] context cancelled — stopping (%d geocoded, %d skipped)", success, skipped)
			return
		case <-ticker.C:
		}

		query := buildQuery(row)
		if query == "" {
			log.Printf("[geocode] id=%d has no address — skipping", row.ID)
			skipped++
			continue
		}

		lat, lng, err := callMapTiler(ctx, client, apiKey, query)
		if err != nil {
			log.Printf("[geocode] id=%d query=%q ERROR: %v", row.ID, query, err)
			skipped++
			continue
		}

		if err := writeCoords(ctx, db, row.ID, lat, lng); err != nil {
			log.Printf("[geocode] id=%d failed to write coords: %v", row.ID, err)
			skipped++
			continue
		}

		log.Printf("[geocode] id=%-6d  lat=%-12f  lng=%-12f  %q", row.ID, lat, lng, query)
		success++
	}

	log.Printf("[geocode] complete — geocoded=%d  skipped=%d", success, skipped)
}

// fetchUngeocoded returns all auction rows where lat = '0'.
func fetchUngeocoded(ctx context.Context, db *sql.DB) ([]auctionRow, error) {
	const q = `
		SELECT id, address, city, state
		FROM   auctions
		WHERE  lat = '0'
		ORDER  BY id`

	rowsDB, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rowsDB.Close()

	var out []auctionRow
	for rowsDB.Next() {
		var r auctionRow
		if err := rowsDB.Scan(&r.ID, &r.Address, &r.City, &r.State); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rowsDB.Err()
}

// buildQuery assembles the geocoding search string from address components.
func buildQuery(r auctionRow) string {
	parts := []string{strings.TrimSpace(r.Address)}
	if r.City.Valid && r.City.String != "" {
		parts = append(parts, strings.TrimSpace(r.City.String))
	}
	if r.State.Valid && r.State.String != "" {
		parts = append(parts, strings.TrimSpace(r.State.String))
	}
	q := strings.Join(parts, ", ")
	if q == "" {
		return ""
	}
	// Append "USA" to bias results away from international matches.
	return q + ", USA"
}

// callMapTiler calls the MapTiler forward-geocoding API and returns the
// coordinates from features[0].center ([longitude, latitude]).
func callMapTiler(ctx context.Context, client *http.Client, apiKey, query string) (lat, lng float64, err error) {
	// MapTiler encodes the query in the URL path, not as a query parameter.
	reqURL := fmt.Sprintf("%s/%s.json?key=%s",
		mapTilerGeocodeBase,
		url.PathEscape(query),
		url.QueryEscape(apiKey),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("Origin", "https://www.auctionandcompany.com")

	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, err
	}
	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("maptiler API %d: %s", resp.StatusCode, string(body))
	}

	var result mapTilerResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, 0, fmt.Errorf("decode maptiler response: %w", err)
	}
	if len(result.Features) == 0 {
		return 0, 0, fmt.Errorf("no results for %q", query)
	}

	// GeoJSON center is [longitude, latitude].
	center := result.Features[0].Center
	return center[1], center[0], nil
}

// writeCoords updates the lat and lng columns for a single auction row.
func writeCoords(ctx context.Context, db *sql.DB, id int, lat, lng float64) error {
	const q = `UPDATE auctions SET lat = $1, lng = $2 WHERE id = $3`
	_, err := db.ExecContext(ctx, q, fmt.Sprintf("%f", lat), fmt.Sprintf("%f", lng), id)
	return err
}
