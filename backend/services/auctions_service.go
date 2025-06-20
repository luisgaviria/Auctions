package services

import (
	"backendAuction/models"
	"backendAuction/utils/cache"
	"database/sql"
	"encoding/json"
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
	Message  string         `json:"message"`
	Auctions []models.AuctionModel `json:"auctions"`
}

var selectFromAuctionsTable = `SELECT * FROM auctions;`

func (s *AuctionsService) GetAuctions() ([]byte, int, error) {
	if cached, found := cache.Cache.Get(cache.AuctionsKey); found {
		if data, ok := cached.([]byte); ok {
			return data, http.StatusOK, nil
		}
	}
	auctions := make([]models.AuctionModel, 0)
	rows, err := s.DB.Query(selectFromAuctionsTable)
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
		); err != nil {
			log.Printf("Error scanning auction: %v\n", err)
			return nil, http.StatusInternalServerError, err
		}
		auctions = append(auctions, auction)
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
	cache.Cache.Set(cache.AuctionsKey, data, 5*time.Minute)
	return data, http.StatusOK, nil
} 