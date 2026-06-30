package services

import (
	"backendAuction/models"
	"backendAuction/utils"
	"backendAuction/utils/cache"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type AuctionsService struct {
	DB *sql.DB
}

func NewAuctionsService(db *sql.DB) *AuctionsService {
	return &AuctionsService{DB: db}
}

type GetAuctionsResponse struct {
	Message  string               `json:"message"`
	Auctions []models.AuctionJSON `json:"auctions"`
}

// auctionCols is the explicit column list used by both queries.  Listing
// columns instead of SELECT * means adding new DB columns (e.g. last_seen)
// never breaks the scan regardless of migration order across environments.
const auctionCols = `id, address, city, state, time, logo, status, link,
	date, deposit, lat, lng, "createdAt", site_name, updated_at, last_seen,
	zillow_url, street_view_url, registry_url, assessor_pid, registry_deep_link,
	registry_book, registry_page`

// selectFilteredAuctions excludes terminal/past statuses, drops past-dated rows,
// and sorts upcoming auctions chronologically.
// $1=limit  $2=offset  $3=search pattern (use '%' to match all)
var selectFilteredAuctions = `
	SELECT ` + auctionCols + ` FROM auctions
	WHERE LOWER(status) NOT IN (
		'cancelled', 'sold', 'removed', 'canceled',
		'sold back to mortgagee', 'back to mortgagee',
		'past', '3rd party purchase', 'postponed'
	)
	AND (date >= (CURRENT_TIMESTAMP AT TIME ZONE 'America/New_York')::date OR date IS NULL)
	AND (address ILIKE $3 OR city ILIKE $3)
	ORDER BY date ASC NULLS LAST, time ASC NULLS LAST
	LIMIT $1 OFFSET $2`

// selectAuctionsInBounds returns geocoded auctions within a lat/lng bounding box.
// lat/lng are stored as text; we cast to float8 for the range check.
var selectAuctionsInBounds = `
	SELECT ` + auctionCols + ` FROM auctions
	WHERE LOWER(status) NOT IN (
		'cancelled', 'sold', 'removed', 'canceled',
		'sold back to mortgagee', 'back to mortgagee',
		'past', '3rd party purchase', 'postponed'
	)
	AND (date >= (CURRENT_TIMESTAMP AT TIME ZONE 'America/New_York')::date OR date IS NULL)
	AND lat != 0 AND lng != 0
	AND lat BETWEEN $1 AND $2
	AND lng BETWEEN $3 AND $4
	ORDER BY date ASC NULLS LAST, time ASC NULLS LAST
	LIMIT 200`

func (s *AuctionsService) GetAuctions(limit, offset int, search string) ([]byte, int, error) {
	easternTZ, _ := time.LoadLocation("America/New_York")
	todayStr := time.Now().In(easternTZ).Format("2006-01-02")
	cacheKey := fmt.Sprintf("auctions_%s_%d_%d_%s", todayStr, limit, offset, search)
	if cached, found := cache.Cache.Get(cacheKey); found {
		if data, ok := cached.([]byte); ok {
			return data, http.StatusOK, nil
		}
	}

	searchParam := "%"
	if s := strings.TrimSpace(search); s != "" {
		searchParam = "%" + s + "%"
	}

	auctions := make([]models.AuctionJSON, 0)
	rows, err := s.DB.Query(selectFilteredAuctions, limit, offset, searchParam)
	if err != nil {
		log.Printf("Database error: %v\n", err)
		return nil, http.StatusInternalServerError, err
	}
	defer rows.Close()
	for rows.Next() {
		auction := models.AuctionModel{}
		if err := rows.Scan(
			&auction.Id,
			&auction.Address,
			&auction.City,
			&auction.State,
			&auction.Time,
			&auction.Logo,
			&auction.Status,
			&auction.Link,
			&auction.Date,
			&auction.Deposit,
			&auction.Lat,
			&auction.Lng,
			&auction.Createdat,
			&auction.SiteName,
			&auction.UpdatedAt,
			&auction.LastSeen,
			&auction.ZillowURL,
			&auction.StreetViewURL,
			&auction.RegistryURL,
			&auction.AssessorPID,
			&auction.RegistryDeepLink,
			&auction.RegistryBook,
			&auction.RegistryPage,
		); err != nil {
			log.Printf("Error scanning auction: %v\n", err)
			return nil, http.StatusInternalServerError, err
		}
		auctions = append(auctions, auction.ToJSON())
	}
	if err := rows.Err(); err != nil {
		log.Printf("Error iterating rows: %v\n", err)
		return nil, http.StatusInternalServerError, err
	}

	response := GetAuctionsResponse{
		Message:  "Successfully fetched auctions",
		Auctions: auctions,
	}
	data, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshaling response: %v\n", err)
		return nil, http.StatusInternalServerError, err
	}
	cache.Cache.Set(cacheKey, data, 5*time.Minute)
	return data, http.StatusOK, nil
}

// CityMarketSummary is the aggregate badge data returned alongside the auction
// list for a city landing page.
type CityMarketSummary struct {
	City           string `json:"city"`
	County         string `json:"county"`
	AuctionCount   int    `json:"auction_count"`
	AverageDeposit string `json:"average_deposit"` // "$12,500" or "" when unavailable
}

// CityMarketResponse is the full response shape for GET /auctions/:county_slug/:city_slug.
type CityMarketResponse struct {
	Summary  CityMarketSummary    `json:"summary"`
	Auctions []models.AuctionJSON `json:"auctions"`
}

// activeFilter is the shared status + date predicate reused across queries.
// NULL status is treated as active (scraper hasn't set one yet).
const activeFilter = `
	(status IS NULL OR LOWER(status) NOT IN (
		'cancelled','sold','removed','canceled',
		'sold back to mortgagee','back to mortgagee',
		'past','3rd party purchase','postponed'
	))
	AND (date >= (CURRENT_TIMESTAMP AT TIME ZONE 'America/New_York')::date OR date IS NULL)`

// GetAuctionsBySlug returns all active auctions for a specific city+county slug
// pair together with a summary object (count, average deposit, display names).
// Returns 404 when the slug combination has no active auctions.
func (s *AuctionsService) GetAuctionsBySlug(countySlug, citySlug string) ([]byte, int, error) {
	// Normalise slugs to lowercase so URL casing never causes a miss.
	countySlug = strings.ToLower(strings.TrimSpace(countySlug))
	citySlug = strings.ToLower(strings.TrimSpace(citySlug))

	// ── 1. Summary aggregate ────────────────────────────────────────────────
	const summaryQ = `
		SELECT
			MAX(city)                                                          AS city_name,
			COUNT(*)                                                           AS auction_count,
			ROUND(AVG(NULLIF(deposit, 0)))                                     AS avg_deposit
		FROM auctions
		WHERE LOWER(county_slug) = $1
		  AND LOWER(city_slug)   = $2
		  AND ` + activeFilter

	log.Printf("[slug] summary query county=%q city=%q", countySlug, citySlug)

	var cityName string
	var count int
	var avgDeposit sql.NullFloat64

	if err := s.DB.QueryRow(summaryQ, countySlug, citySlug).
		Scan(&cityName, &count, &avgDeposit); err != nil {
		log.Printf("[slug] summary scan error county=%q city=%q: %v", countySlug, citySlug, err)
		return nil, http.StatusInternalServerError, err
	}

	log.Printf("[slug] summary result county=%q city=%q count=%d avg_deposit=%v city_name=%q",
		countySlug, citySlug, count, avgDeposit, cityName)

	if count == 0 {
		// Run a diagnostic query to distinguish "slugs don't exist" from
		// "slugs exist but all auctions are filtered out by status/date".
		var total int
		_ = s.DB.QueryRow(
			`SELECT COUNT(*) FROM auctions WHERE LOWER(county_slug)=$1 AND LOWER(city_slug)=$2`,
			countySlug, citySlug,
		).Scan(&total)
		log.Printf("[slug] 404 — total rows (unfiltered) for %q/%q: %d", countySlug, citySlug, total)
		return nil, http.StatusNotFound, fmt.Errorf("no active auctions found for %s/%s", countySlug, citySlug)
	}

	// Format average deposit as "$12,500" when data is available.
	avgDepositStr := ""
	if avgDeposit.Valid && avgDeposit.Float64 > 0 {
		avgDepositStr = fmt.Sprintf("$%s", formatWithCommas(int(avgDeposit.Float64)))
	}

	// Derive display county name from slug: "worcester-county" → "Worcester County"
	countyDisplay := slugToDisplay(countySlug)

	summary := CityMarketSummary{
		City:           cityName,
		County:         countyDisplay,
		AuctionCount:   count,
		AverageDeposit: avgDepositStr,
	}

	// ── 2. Auction rows ─────────────────────────────────────────────────────
	const rowsQ = `
		SELECT ` + auctionCols + `
		FROM auctions
		WHERE LOWER(county_slug) = $1
		  AND LOWER(city_slug)   = $2
		  AND ` + activeFilter + `
		ORDER BY date ASC NULLS LAST, time ASC NULLS LAST`

	log.Printf("[slug] rows query county=%q city=%q", countySlug, citySlug)
	rows, err := s.DB.Query(rowsQ, countySlug, citySlug)
	if err != nil {
		log.Printf("[slug] rows query error: %v", err)
		return nil, http.StatusInternalServerError, err
	}
	defer rows.Close()

	auctions := make([]models.AuctionJSON, 0, count)
	for rows.Next() {
		var a models.AuctionModel
		if err := rows.Scan(
			&a.Id, &a.Address, &a.City, &a.State, &a.Time, &a.Logo,
			&a.Status, &a.Link, &a.Date, &a.Deposit, &a.Lat, &a.Lng,
			&a.Createdat, &a.SiteName, &a.UpdatedAt, &a.LastSeen,
			&a.ZillowURL, &a.StreetViewURL, &a.RegistryURL,
			&a.AssessorPID, &a.RegistryDeepLink,
			&a.RegistryBook, &a.RegistryPage,
		); err != nil {
			log.Printf("[slug] scan error: %v", err)
			return nil, http.StatusInternalServerError, err
		}
		auctions = append(auctions, a.ToJSON())
	}
	if err := rows.Err(); err != nil {
		return nil, http.StatusInternalServerError, err
	}

	data, err := json.Marshal(CityMarketResponse{Summary: summary, Auctions: auctions})
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return data, http.StatusOK, nil
}

// formatWithCommas turns 12500 into "12,500".
func formatWithCommas(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

// slugToDisplay converts "worcester-county" → "Worcester County".
func slugToDisplay(slug string) string {
	words := strings.Split(slug, "-")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// CitySlugRow is one entry in the top-cities response.
type CitySlugRow struct {
	CountySlug   string `json:"county_slug"`
	CitySlug     string `json:"city_slug"`
	City         string `json:"city"`
	AuctionCount int    `json:"auction_count"`
}

// TopCitySlugsResponse wraps the markets list.
type TopCitySlugsResponse struct {
	Markets []CitySlugRow `json:"markets"`
}

// GetTopCitySlugs returns the N most active city+county slug pairs that have
// at least one active auction.  Used by the frontend getStaticPaths call to
// pre-render the top city landing pages at build time.
func (s *AuctionsService) GetTopCitySlugs(limit int) ([]byte, int, error) {
	const q = `
		SELECT county_slug, city_slug, MAX(city) AS city, COUNT(*) AS auction_count
		FROM auctions
		WHERE county_slug IS NOT NULL
		  AND city_slug   IS NOT NULL
		  AND ` + activeFilter + `
		GROUP BY county_slug, city_slug
		ORDER BY auction_count DESC
		LIMIT $1`

	rows, err := s.DB.Query(q, limit)
	if err != nil {
		log.Printf("[slugs] query error: %v", err)
		return nil, http.StatusInternalServerError, err
	}
	defer rows.Close()

	markets := make([]CitySlugRow, 0, limit)
	for rows.Next() {
		var r CitySlugRow
		if err := rows.Scan(&r.CountySlug, &r.CitySlug, &r.City, &r.AuctionCount); err != nil {
			log.Printf("[slugs] scan error: %v", err)
			return nil, http.StatusInternalServerError, err
		}
		markets = append(markets, r)
	}
	if err := rows.Err(); err != nil {
		return nil, http.StatusInternalServerError, err
	}

	data, err := json.Marshal(TopCitySlugsResponse{Markets: markets})
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return data, http.StatusOK, nil
}

// GetAuctionReport fetches a single auction by address_slug and returns a full
// property report including a due-diligence checklist.
// When zillow_url / registry_url are empty in the DB (pre-backfill rows) the
// function regenerates them on the fly so the report is always complete.
func (s *AuctionsService) GetAuctionReport(addressSlug string) ([]byte, int, error) {
	addressSlug = strings.ToLower(strings.TrimSpace(addressSlug))

	// Fetch the auction row.  address_slug is not unique (two auctioneers can
	// list the same property) so we take the first active result.
	const q = `
		SELECT ` + auctionCols + `, COALESCE(address_slug, '') AS address_slug
		FROM auctions
		WHERE address_slug = $1
		  AND ` + activeFilter + `
		ORDER BY date ASC NULLS LAST
		LIMIT 1`

	log.Printf("[report] query address_slug=%q", addressSlug)

	var a models.AuctionModel
	var scannedSlug string
	err := s.DB.QueryRow(q, addressSlug).Scan(
		&a.Id, &a.Address, &a.City, &a.State, &a.Time, &a.Logo,
		&a.Status, &a.Link, &a.Date, &a.Deposit, &a.Lat, &a.Lng,
		&a.Createdat, &a.SiteName, &a.UpdatedAt, &a.LastSeen,
		&a.ZillowURL, &a.StreetViewURL, &a.RegistryURL,
		&a.AssessorPID, &a.RegistryDeepLink,
		&a.RegistryBook, &a.RegistryPage,
		&scannedSlug,
	)
	if err == sql.ErrNoRows {
		log.Printf("[report] 404 address_slug=%q", addressSlug)
		return nil, http.StatusNotFound, fmt.Errorf("no report found for %q", addressSlug)
	}
	if err != nil {
		log.Printf("[report] scan error: %v", err)
		return nil, http.StatusInternalServerError, err
	}

	auctionJSON := a.ToJSON()

	// If URL columns are empty (row predates the backfill), generate on the fly.
	if auctionJSON.ZillowURL == "" || auctionJSON.RegistryURL == "" {
		z, sv, reg := utils.BuildAuctionURLsPublic(auctionJSON.Address, auctionJSON.City)
		if auctionJSON.ZillowURL == "" {
			auctionJSON.ZillowURL = z
		}
		if auctionJSON.StreetViewURL == "" {
			auctionJSON.StreetViewURL = sv
		}
		if auctionJSON.RegistryURL == "" {
			auctionJSON.RegistryURL = reg
		}
	}

	checklist := buildChecklist(auctionJSON)
	shareURL := "https://auctionandcompany.com/report/" + addressSlug

	report := models.AuctionReport{
		Auction:     auctionJSON,
		Checklist:   checklist,
		ShareURL:    shareURL,
		AddressSlug: scannedSlug,
	}

	data, err := json.Marshal(report)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return data, http.StatusOK, nil
}

// buildChecklist returns a personalised due-diligence checklist for a property.
// Items are ordered by the typical workflow an investor follows before bidding.
func buildChecklist(a models.AuctionJSON) []models.DueDiligenceItem {
	items := []models.DueDiligenceItem{
		{Task: "Drive-by inspection of the property", Category: "Physical"},
		{Task: "Confirm occupancy status (owner-occupied vs. tenant vs. vacant)", Category: "Physical"},
		{Task: "Photograph exterior condition and visible damage", Category: "Physical"},
	}

	// Financial items — personalise with deposit amount when available.
	depositLine := "Prepare certified check for deposit"
	if a.Deposit != "" {
		depositLine = "Prepare certified check for deposit (" + a.Deposit + ")"
	}
	items = append(items,
		models.DueDiligenceItem{Task: depositLine, Category: "Financial"},
		models.DueDiligenceItem{Task: "Research assessed value and recent comparable sales", Category: "Financial"},
		models.DueDiligenceItem{Task: "Estimate rehab costs for a realistic ARV calculation", Category: "Financial"},
		models.DueDiligenceItem{Task: "Calculate maximum bid (ARV × 70% − repairs − deposit)", Category: "Financial"},
	)

	// Legal items — link to registry when available.
	registryLine := "Verify all existing liens at the Registry of Deeds"
	if a.RegistryURL != "" {
		registryLine += " (" + a.RegistryURL + ")"
	}
	items = append(items,
		models.DueDiligenceItem{Task: registryLine, Category: "Legal"},
		models.DueDiligenceItem{Task: "Research outstanding real estate taxes and municipal charges", Category: "Legal"},
		models.DueDiligenceItem{Task: "Review title history for IRS liens or judgment liens", Category: "Legal"},
		models.DueDiligenceItem{Task: "Obtain and read the full terms of sale from the auctioneer", Category: "Legal"},
	)

	items = append(items,
		models.DueDiligenceItem{Task: "Confirm auction date, time, and location with auctioneer", Category: "Administrative"},
		models.DueDiligenceItem{Task: "Register or pre-qualify with the auctioneer if required", Category: "Administrative"},
	)

	return items
}

// GetAuctionsInBounds returns auctions whose coordinates fall within the given
// bounding box (south ≤ lat ≤ north, west ≤ lng ≤ east).
// Results are not cached because the bbox space is unbounded.
func (s *AuctionsService) GetAuctionsInBounds(south, north, west, east float64) ([]byte, int, error) {
	auctions := make([]models.AuctionJSON, 0)
	rows, err := s.DB.Query(selectAuctionsInBounds, south, north, west, east)
	if err != nil {
		log.Printf("Database error (bbox): %v\n", err)
		return nil, http.StatusInternalServerError, err
	}
	defer rows.Close()
	for rows.Next() {
		auction := models.AuctionModel{}
		if err := rows.Scan(
			&auction.Id,
			&auction.Address,
			&auction.City,
			&auction.State,
			&auction.Time,
			&auction.Logo,
			&auction.Status,
			&auction.Link,
			&auction.Date,
			&auction.Deposit,
			&auction.Lat,
			&auction.Lng,
			&auction.Createdat,
			&auction.SiteName,
			&auction.UpdatedAt,
			&auction.LastSeen,
			&auction.ZillowURL,
			&auction.StreetViewURL,
			&auction.RegistryURL,
			&auction.AssessorPID,
			&auction.RegistryDeepLink,
			&auction.RegistryBook,
			&auction.RegistryPage,
		); err != nil {
			log.Printf("Error scanning auction (bbox): %v\n", err)
			return nil, http.StatusInternalServerError, err
		}
		auctions = append(auctions, auction.ToJSON())
	}
	if err := rows.Err(); err != nil {
		log.Printf("Error iterating rows (bbox): %v\n", err)
		return nil, http.StatusInternalServerError, err
	}

	response := GetAuctionsResponse{
		Message:  "Successfully fetched auctions",
		Auctions: auctions,
	}
	data, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshaling response (bbox): %v\n", err)
		return nil, http.StatusInternalServerError, err
	}
	return data, http.StatusOK, nil
}
