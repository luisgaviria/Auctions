package controllers

import (
	"backendAuction/utils/cache"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

var selectFromAuctionsTable = `SELECT * FROM auctions;`

type AuctionsController struct {
	DB *sql.DB
}

type GetAuctionsResponse struct {
	Message  string         `json:"message"`
	Auctions []AuctionModel `json:"auctions"`
}

type AuctionModel struct {
	Id        int       `json:"id"`
	Address   string    `json:"address"`
	City      string    `json:"city"`
	State     string    `json:"state"`
	Time      string    `json:"time"`
	Logo      string    `json:"logo"`
	Status    string    `json:"status"`
	Link      string    `json:"link"`
	Date      time.Time `json:"date"`
	Deposit   string    `json:"deposit"`
	Lat       string    `json:"lat"`
	Lng       string    `json:"lng"`
	Createdat time.Time `json:"createdAt"`
}

func (c *AuctionsController) GetAuctions(w http.ResponseWriter, req *http.Request) {
	// Try to get from cache first
	if cached, found := cache.Cache.Get(cache.AuctionsKey); found {
		if data, ok := cached.([]byte); ok {
			w.WriteHeader(http.StatusOK)
			w.Write(data)
			return
		}
	}

	// If not in cache, get from database
	auctions := make([]AuctionModel, 0)
	rows, err := c.DB.Query(selectFromAuctionsTable)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Database error: %v\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		auction := AuctionModel{}
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
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		auctions = append(auctions, auction)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating rows: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := GetAuctionsResponse{
		Message:  "Successfully fetched auctions",
		Auctions: auctions,
	}

	data, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshaling response: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Store in cache for 5 minutes
	cache.Cache.Set(cache.AuctionsKey, data, 5*time.Minute)

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// Add a method to invalidate cache when auctions are updated
func (c *AuctionsController) InvalidateCache() {
	cache.Cache.Delete(cache.AuctionsKey)
}
