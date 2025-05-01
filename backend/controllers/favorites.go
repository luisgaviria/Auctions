package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

type FavoritesController struct {
	DB *sql.DB
}

type FavoriteRequest struct {
	AuctionID int `json:"auction_id"`
}

type AuctionFavoriteResponse struct {
	Message  string         `json:"message"`
	Auctions []AuctionModel `json:"auctions"`
}

var getUserIDFromEmail = `SELECT id FROM users WHERE email = $1`

var addToFavorites = `
    INSERT INTO favorites (user_id, auction_id) 
    VALUES ($1, $2) 
    ON CONFLICT (user_id, auction_id) DO NOTHING
    RETURNING auction_id;`

var removeFromFavorites = `
    DELETE FROM favorites 
    WHERE user_id = $1 AND auction_id = $2
    RETURNING auction_id;`

var getFavorites = `
    SELECT a.* FROM auctions a
    INNER JOIN favorites f ON f.auction_id = a.id
    WHERE f.user_id = $1;`

func (c *FavoritesController) AddFavorite(w http.ResponseWriter, r *http.Request) {
	// Get email from context
	email := r.Context().Value("sub").(string)

	// Get user ID from email
	var userID int
	err := c.DB.QueryRow(getUserIDFromEmail, email).Scan(&userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var req FavoriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Verify auction exists
	var auctionExists bool
	err = c.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM auctions WHERE id = $1)", req.AuctionID).Scan(&auctionExists)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		fmt.Printf("Database error: %v\n", err)
		return
	}
	if !auctionExists {
		http.Error(w, "Auction not found", http.StatusNotFound)
		return
	}

	var addedAuctionID int
	err = c.DB.QueryRow(addToFavorites, userID, req.AuctionID).Scan(&addedAuctionID)
	if err != nil {
		http.Error(w, "Error adding favorite", http.StatusInternalServerError)
		fmt.Printf("Database error: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Added to favorites",
		"auction_id": addedAuctionID,
	})
}

func (c *FavoritesController) RemoveFavorite(w http.ResponseWriter, r *http.Request) {
	email := r.Context().Value("sub").(string)

	var userID int
	err := c.DB.QueryRow(getUserIDFromEmail, email).Scan(&userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var req FavoriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	var removedAuctionID int
	err = c.DB.QueryRow(removeFromFavorites, userID, req.AuctionID).Scan(&removedAuctionID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Favorite not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error removing favorite", http.StatusInternalServerError)
		fmt.Printf("Database error: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Removed from favorites",
		"auction_id": removedAuctionID,
	})
}

func (c *FavoritesController) GetFavorites(w http.ResponseWriter, r *http.Request) {
	email := r.Context().Value("sub").(string)

	var userID int
	err := c.DB.QueryRow(getUserIDFromEmail, email).Scan(&userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	rows, err := c.DB.Query(getFavorites, userID)
	if err != nil {
		http.Error(w, "Error retrieving favorites", http.StatusInternalServerError)
		fmt.Printf("Database error: %v\n", err)
		return
	}
	defer rows.Close()

	auctions := make([]AuctionModel, 0)
	for rows.Next() {
		var auction AuctionModel
		err := rows.Scan(&auction.Id, &auction.Address, &auction.City, &auction.State,
			&auction.Time, &auction.Logo, &auction.Status, &auction.Link, &auction.Date,
			&auction.Deposit, &auction.Lat, &auction.Lng, &auction.Createdat)
		if err != nil {
			http.Error(w, "Error scanning auction", http.StatusInternalServerError)
			fmt.Printf("Database error: %v\n", err)
			return
		}
		auctions = append(auctions, auction)
	}

	response := AuctionFavoriteResponse{
		Message:  "Successfully retrieved favorites",
		Auctions: auctions,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
