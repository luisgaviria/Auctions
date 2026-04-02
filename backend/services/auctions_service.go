package services

import (
	"backendAuction/models"
	"backendAuction/utils/cache"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
	date, deposit, lat, lng, "createdAt", site_name, updated_at, last_seen`

// selectFilteredAuctions excludes terminal/past statuses, drops past-dated rows,
// and sorts upcoming auctions chronologically.
var selectFilteredAuctions = `
	SELECT ` + auctionCols + ` FROM auctions
	WHERE LOWER(status) NOT IN (
		'cancelled', 'sold', 'removed', 'canceled',
		'sold back to mortgagee', 'back to mortgagee',
		'past', '3rd party purchase', 'postponed'
	)
	AND (date >= (CURRENT_DATE AT TIME ZONE 'UTC' AT TIME ZONE 'America/New_York')::date OR date IS NULL)
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
	AND (date >= (CURRENT_DATE AT TIME ZONE 'UTC' AT TIME ZONE 'America/New_York')::date OR date IS NULL)
	AND lat != '0' AND lng != '0'
	AND lat::float8 BETWEEN $1 AND $2
	AND lng::float8 BETWEEN $3 AND $4
	ORDER BY date ASC NULLS LAST, time ASC NULLS LAST
	LIMIT 200`

func (s *AuctionsService) GetAuctions(limit, offset int) ([]byte, int, error) {
	easternTZ, _ := time.LoadLocation("America/New_York")
	todayStr := time.Now().In(easternTZ).Format("2006-01-02")
	cacheKey := fmt.Sprintf("auctions_%s_%d_%d", todayStr, limit, offset)
	if cached, found := cache.Cache.Get(cacheKey); found {
		if data, ok := cached.([]byte); ok {
			return data, http.StatusOK, nil
		}
	}

	auctions := make([]models.AuctionJSON, 0)
	rows, err := s.DB.Query(selectFilteredAuctions, limit, offset)
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
