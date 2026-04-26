package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type LocationsService struct {
	DB *sql.DB
}

func NewLocationsService(db *sql.DB) *LocationsService {
	return &LocationsService{DB: db}
}

type CountyRow struct {
	CountySlug   string `json:"county_slug"`
	County       string `json:"county"`
	AuctionCount int    `json:"auction_count"`
}

type CountiesResponse struct {
	Counties []CountyRow `json:"counties"`
}

type CityRow struct {
	CitySlug     string `json:"city_slug"`
	City         string `json:"city"`
	CountySlug   string `json:"county_slug"`
	AuctionCount int    `json:"auction_count"`
}

type CountyCitiesResponse struct {
	County     string    `json:"county"`
	CountySlug string    `json:"county_slug"`
	Cities     []CityRow `json:"cities"`
}

// GetCounties returns all counties that have at least one active auction,
// ordered by auction count descending.
func (s *LocationsService) GetCounties() ([]byte, int, error) {
	const q = `
		SELECT county_slug, COUNT(*) AS auction_count
		FROM auctions
		WHERE county_slug IS NOT NULL
		  AND county_slug != ''
		  AND ` + activeFilter + `
		GROUP BY county_slug
		ORDER BY auction_count DESC, county_slug ASC`

	rows, err := s.DB.Query(q)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	defer rows.Close()

	counties := make([]CountyRow, 0, 16)
	for rows.Next() {
		var slug string
		var count int
		if err := rows.Scan(&slug, &count); err != nil {
			return nil, http.StatusInternalServerError, err
		}
		counties = append(counties, CountyRow{
			CountySlug:   slug,
			County:       slugToDisplay(slug),
			AuctionCount: count,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, http.StatusInternalServerError, err
	}

	data, err := json.Marshal(CountiesResponse{Counties: counties})
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return data, http.StatusOK, nil
}

// GetCitiesByCounty returns all cities in a county that have at least one
// active auction, ordered by auction count descending.
func (s *LocationsService) GetCitiesByCounty(countySlug string) ([]byte, int, error) {
	countySlug = strings.ToLower(strings.TrimSpace(countySlug))

	const q = `
		SELECT city_slug, MAX(city) AS city, COUNT(*) AS auction_count
		FROM auctions
		WHERE LOWER(county_slug) = $1
		  AND city_slug IS NOT NULL
		  AND city_slug != ''
		  AND ` + activeFilter + `
		GROUP BY city_slug
		ORDER BY auction_count DESC, city_slug ASC`

	rows, err := s.DB.Query(q, countySlug)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	defer rows.Close()

	cities := make([]CityRow, 0, 32)
	for rows.Next() {
		var citySlug, city string
		var count int
		if err := rows.Scan(&citySlug, &city, &count); err != nil {
			return nil, http.StatusInternalServerError, err
		}
		cities = append(cities, CityRow{
			CitySlug:     citySlug,
			City:         titleCaseStr(city),
			CountySlug:   countySlug,
			AuctionCount: count,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, http.StatusInternalServerError, err
	}

	if len(cities) == 0 {
		return nil, http.StatusNotFound, fmt.Errorf("no active auctions in county %q", countySlug)
	}

	data, err := json.Marshal(CountyCitiesResponse{
		County:     slugToDisplay(countySlug),
		CountySlug: countySlug,
		Cities:     cities,
	})
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return data, http.StatusOK, nil
}

// titleCaseStr title-cases a DB uppercase city string ("NEW BEDFORD" → "New Bedford").
func titleCaseStr(s string) string {
	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// GetCitiesForRelated returns up to `limit` active cities in the same county,
// excluding the given city slug. Used by the city page's Related Markets section.
func (s *LocationsService) GetCitiesForRelated(countySlug, excludeCitySlug string, limit int) ([]CityRow, error) {
	countySlug = strings.ToLower(strings.TrimSpace(countySlug))
	excludeCitySlug = strings.ToLower(strings.TrimSpace(excludeCitySlug))

	const q = `
		SELECT city_slug, MAX(city) AS city, COUNT(*) AS auction_count
		FROM auctions
		WHERE LOWER(county_slug) = $1
		  AND LOWER(city_slug)  != $2
		  AND city_slug IS NOT NULL
		  AND city_slug != ''
		  AND ` + activeFilter + `
		GROUP BY city_slug
		ORDER BY auction_count DESC, city_slug ASC
		LIMIT $3`

	rows, err := s.DB.Query(q, countySlug, excludeCitySlug, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cities := make([]CityRow, 0, limit)
	for rows.Next() {
		var citySlug, city string
		var count int
		if err := rows.Scan(&citySlug, &city, &count); err != nil {
			return nil, err
		}
		cities = append(cities, CityRow{
			CitySlug:     citySlug,
			City:         titleCaseStr(city),
			CountySlug:   countySlug,
			AuctionCount: count,
		})
	}
	return cities, rows.Err()
}
