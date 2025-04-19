package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type FavoritesController struct {
	DB *sql.DB
}

type FavoriteRequest struct {
	AuctionID string `json:"auction_id"`
}

var addToFavorites = `
    INSERT INTO favorites (user_id, auction_id) 
    VALUES ($1, $2) 
    ON CONFLICT (user_id, auction_id) DO NOTHING;`

var removeFromFavorites = `
    DELETE FROM favorites 
    WHERE user_id = $1 AND auction_id = $2;`

var getFavorites = `
    SELECT a.* FROM auctions a
    INNER JOIN favorites f ON f.auction_id = a.id
    WHERE f.user_id = $1;`

func (c *FavoritesController) AddFavorite(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	var req FavoriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := c.DB.Exec(addToFavorites, userID, req.AuctionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Added to favorites",
	})
}

func (c *FavoritesController) RemoveFavorite(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	var req FavoriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := c.DB.Exec(removeFromFavorites, userID, req.AuctionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Removed from favorites",
	})
}

func (c *FavoritesController) GetFavorites(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)

	rows, err := c.DB.Query(getFavorites, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var auctions []AuctionModel
	for rows.Next() {
		var auction AuctionModel
		if err := rows.Scan(&auction.Id, &auction.Address, &auction.City, &auction.State,
			&auction.Time, &auction.Logo, &auction.Status, &auction.Link, &auction.Date,
			&auction.Deposit, &auction.Lat, &auction.Lng, &auction.Createdat); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		auctions = append(auctions, auction)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "Favorites retrieved successfully",
		"auctions": auctions,
	})
}
