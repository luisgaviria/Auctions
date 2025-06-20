package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"backendAuction/models"
)

type FavoritesService struct {
	DB *sql.DB
}

func NewFavoritesService(db *sql.DB) *FavoritesService {
	return &FavoritesService{DB: db}
}

type FavoriteRequest struct {
	AuctionID int `json:"auction_id"`
}

type FavoritesResponse struct {
	Message  string                `json:"message"`
	Auctions []models.AuctionModel `json:"auctions"`
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

func (s *FavoritesService) AddFavorite(email string, req *FavoriteRequest) ([]byte, int, error) {
	var userID int
	err := s.DB.QueryRow(getUserIDFromEmail, email).Scan(&userID)
	if err != nil {
		return nil, http.StatusNotFound, err
	}
	var auctionExists bool
	err = s.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM auctions WHERE id = $1)", req.AuctionID).Scan(&auctionExists)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if !auctionExists {
		return nil, http.StatusNotFound, fmt.Errorf("auction not found")
	}
	var addedAuctionID int
	err = s.DB.QueryRow(addToFavorites, userID, req.AuctionID).Scan(&addedAuctionID)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	resp, _ := json.Marshal(map[string]interface{}{
		"message":    "Added to favorites",
		"auction_id": addedAuctionID,
	})
	return resp, http.StatusOK, nil
}

func (s *FavoritesService) RemoveFavorite(email string, req *FavoriteRequest) ([]byte, int, error) {
	var userID int
	err := s.DB.QueryRow(getUserIDFromEmail, email).Scan(&userID)
	if err != nil {
		return nil, http.StatusNotFound, err
	}
	var removedAuctionID int
	err = s.DB.QueryRow(removeFromFavorites, userID, req.AuctionID).Scan(&removedAuctionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, http.StatusNotFound, err
		}
		return nil, http.StatusInternalServerError, err
	}
	resp, _ := json.Marshal(map[string]interface{}{
		"message":    "Removed from favorites",
		"auction_id": removedAuctionID,
	})
	return resp, http.StatusOK, nil
}

func (s *FavoritesService) GetFavorites(email string) ([]byte, int, error) {
	var userID int
	err := s.DB.QueryRow(getUserIDFromEmail, email).Scan(&userID)
	if err != nil {
		return nil, http.StatusNotFound, err
	}
	rows, err := s.DB.Query(getFavorites, userID)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	defer rows.Close()
	auctions := make([]models.AuctionModel, 0)
	for rows.Next() {
		var auction models.AuctionModel
		err := rows.Scan(&auction.Id, &auction.Address, &auction.City, &auction.State,
			&auction.Time, &auction.Logo, &auction.Status, &auction.Link, &auction.Date,
			&auction.Deposit, &auction.Lat, &auction.Lng, &auction.Createdat)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
		auctions = append(auctions, auction)
	}
	response := FavoritesResponse{
		Message:  "Successfully retrieved favorites",
		Auctions: auctions,
	}
	resp, _ := json.Marshal(response)
	return resp, http.StatusOK, nil
} 