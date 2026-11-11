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
		Center  [2]float64        `json:"center"` // [lng, lat]
		Context []mapTilerContext `json:"context"`
	} `json:"features"`
}

type mapTilerContext struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	ShortCode string `json:"short_code"`
}

// auctionRow holds the columns needed for geocoding.
type auctionRow struct {
	ID       int
	Address  string
	City     sql.NullString
	State    sql.NullString
	Attempts int
}

const (
	// mapTilerGeocodeBase is the MapTiler forward-geocoding endpoint.
	// Query goes in the path: /geocoding/{query}.json?key={key}
	mapTilerGeocodeBase = "https://api.maptiler.com/geocoding"

	// requestsPerSecond caps how fast we hit the MapTiler API.
	// At 5 req/s a backfill of 200 rows takes ~40 s.
	requestsPerSecond = 5

	maxGeocodeAttemptsPerRun = 3
	maxGeocodeRetryDelay     = 24 * time.Hour
)

// Massachusetts' approximate extent is a second guard against accepting an
// unrelated result even if the API's region context is incorrect.
const (
	massachusettsMinLng = -73.508
	massachusettsMaxLng = -69.858
	massachusettsMinLat = 41.187
	massachusettsMaxLat = 42.887
)

// RunGeocodeWorker retries due geocoding rows, records failures, and saves only
// results that MapTiler identifies as belonging to Massachusetts.
//
// Guarantees:
//   - Geocoding failures are persisted with their cause, attempt count, and
//     next retry time.
//   - If an address is later updated by the scraper, existing coordinates are
//     preserved; the upsert SQL never overwrites lat/lng.
func RunGeocodeWorker(ctx context.Context, db *sql.DB) error {
	apiKey := os.Getenv("MAPTILER_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("MAPTILER_API_KEY is not set")
	}

	rows, err := fetchUngeocoded(ctx, db)
	if err != nil {
		return fmt.Errorf("query ungeocoded rows: %w", err)
	}
	if len(rows) == 0 {
		log.Println("[geocode] all auctions already have coordinates — nothing to do")
		return nil
	}
	log.Printf("[geocode] found %d auction(s) to geocode", len(rows))

	// Each API request, including retries, consumes a rate-limit token.
	ticker := time.NewTicker(time.Second / requestsPerSecond)
	defer ticker.Stop()

	client := &http.Client{Timeout: 10 * time.Second}

	success, failed := 0, 0
	for _, row := range rows {
		query := buildQuery(row)
		if query == "" {
			err := fmt.Errorf("address is empty")
			if recordErr := recordGeocodeFailure(ctx, db, row, 0, err); recordErr != nil {
				return fmt.Errorf("record missing-address status for auction %d: %w", row.ID, recordErr)
			}
			log.Printf("[geocode] id=%d status=retry error=%v", row.ID, err)
			failed++
			continue
		}

		lat, lng, attempts, err := geocodeWithRetry(ctx, ticker, client, apiKey, query)
		if err != nil {
			if recordErr := recordGeocodeFailure(ctx, db, row, attempts, err); recordErr != nil {
				return fmt.Errorf("record geocoding failure for auction %d: %w (original error: %v)", row.ID, recordErr, err)
			}
			log.Printf("[geocode] id=%d status=retry attempts=%d retry_after=%s query=%q error=%v",
				row.ID, row.Attempts+attempts, retryDelay(row.Attempts+attempts), query, err)
			failed++
			continue
		}

		if err := writeCoords(ctx, db, row.ID, lat, lng, attempts); err != nil {
			return fmt.Errorf("save coordinates for auction %d: %w", row.ID, err)
		}

		log.Printf("[geocode] id=%-6d status=geocoded attempts=%d lat=%-12f lng=%-12f %q",
			row.ID, row.Attempts+attempts, lat, lng, query)
		success++
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("geocoding cancelled after %d geocoded and %d failed: %w", success, failed, err)
	}
	log.Printf("[geocode] complete — geocoded=%d  retry scheduled=%d", success, failed)
	return nil
}

func geocodeWithRetry(ctx context.Context, ticker *time.Ticker, client *http.Client, apiKey, query string) (float64, float64, int, error) {
	var lastErr error
	for attempt := 1; attempt <= maxGeocodeAttemptsPerRun; attempt++ {
		if attempt > 1 {
			if err := waitContext(ctx, inRunRetryDelay(attempt-1)); err != nil {
				return 0, 0, attempt - 1, err
			}
		}
		select {
		case <-ctx.Done():
			return 0, 0, attempt - 1, ctx.Err()
		case <-ticker.C:
		}

		lat, lng, err := callMapTiler(ctx, client, apiKey, query)
		if err == nil {
			return lat, lng, attempt, nil
		}
		lastErr = err
	}
	return 0, 0, maxGeocodeAttemptsPerRun, lastErr
}

func inRunRetryDelay(retry int) time.Duration {
	if retry < 1 {
		retry = 1
	}
	delay := time.Second
	for i := 1; i < retry && delay < 30*time.Second; i++ {
		delay *= 2
	}
	if delay > 30*time.Second {
		return 30 * time.Second
	}
	return delay
}

func waitContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func retryDelay(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	delay := time.Minute
	for i := 1; i < attempts && delay < maxGeocodeRetryDelay; i++ {
		delay *= 2
	}
	if delay > maxGeocodeRetryDelay {
		return maxGeocodeRetryDelay
	}
	return delay
}

// fetchUngeocoded returns rows that are pending or whose persisted retry time
// has elapsed.
func fetchUngeocoded(ctx context.Context, db *sql.DB) ([]auctionRow, error) {
	const q = `
		SELECT id, address, city, state, geocode_attempts
		FROM   auctions
		WHERE  lat = '0'
		  AND (geocode_status = 'pending'
		       OR (geocode_status = 'retry'
		           AND (geocode_retry_after IS NULL OR geocode_retry_after <= NOW())))
		ORDER  BY id`

	rowsDB, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rowsDB.Close()

	var out []auctionRow
	for rowsDB.Next() {
		var r auctionRow
		if err := rowsDB.Scan(&r.ID, &r.Address, &r.City, &r.State, &r.Attempts); err != nil {
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

// callMapTiler returns the first result that is explicitly identified as
// Massachusetts and has coordinates within the state's approximate extent.
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

	for _, feature := range result.Features {
		// GeoJSON center is [longitude, latitude].
		candidateLng, candidateLat := feature.Center[0], feature.Center[1]
		if !isMassachusettsContext(feature.Context) || !isMassachusettsCoordinates(candidateLat, candidateLng) {
			continue
		}
		return candidateLat, candidateLng, nil
	}
	return 0, 0, fmt.Errorf("no result for %q matched Massachusetts", query)
}

func isMassachusettsContext(contexts []mapTilerContext) bool {
	for _, item := range contexts {
		if strings.EqualFold(item.ShortCode, "US-MA") ||
			strings.EqualFold(item.Text, "Massachusetts") ||
			(strings.HasPrefix(item.ID, "region.") && strings.EqualFold(item.Text, "Massachusetts")) {
			return true
		}
	}
	return false
}

func isMassachusettsCoordinates(lat, lng float64) bool {
	return lat >= massachusettsMinLat && lat <= massachusettsMaxLat &&
		lng >= massachusettsMinLng && lng <= massachusettsMaxLng
}

func recordGeocodeFailure(ctx context.Context, db *sql.DB, row auctionRow, attempts int, cause error) error {
	totalAttempts := row.Attempts + attempts
	delay := retryDelay(totalAttempts)
	const q = `
		UPDATE auctions
		SET geocode_status = 'retry',
		    geocode_error = $1,
		    geocode_attempts = $2,
		    geocode_retry_after = NOW() + ($3 * INTERVAL '1 second')
		WHERE id = $4`
	_, err := db.ExecContext(ctx, q, cause.Error(), totalAttempts, int64(delay.Seconds()), row.ID)
	return err
}

// writeCoords updates coordinates and clears any persisted failure state.
func writeCoords(ctx context.Context, db *sql.DB, id int, lat, lng float64, attempts int) error {
	const q = `
		UPDATE auctions
		SET lat = $1, lng = $2, geocode_status = 'geocoded',
		    geocode_error = NULL, geocode_attempts = geocode_attempts + $3,
		    geocode_retry_after = NULL
		WHERE id = $4`
	_, err := db.ExecContext(ctx, q, fmt.Sprintf("%f", lat), fmt.Sprintf("%f", lng), attempts, id)
	return err
}
