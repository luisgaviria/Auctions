package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"backendAuction/services"
)

type FavoritesController struct {
	DB *sql.DB
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
	email := r.Context().Value("sub").(string)
	var req services.FavoriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}
	service := services.NewFavoritesService(c.DB)
	resp, status, err := service.AddFavorite(email, &req)
	if err != nil {
		w.WriteHeader(status)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(status)
	w.Write(resp)
}

func (c *FavoritesController) RemoveFavorite(w http.ResponseWriter, r *http.Request) {
	email := r.Context().Value("sub").(string)
	var req services.FavoriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}
	service := services.NewFavoritesService(c.DB)
	resp, status, err := service.RemoveFavorite(email, &req)
	if err != nil {
		w.WriteHeader(status)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(status)
	w.Write(resp)
}

func (c *FavoritesController) GetFavorites(w http.ResponseWriter, r *http.Request) {
	email := r.Context().Value("sub").(string)
	service := services.NewFavoritesService(c.DB)
	resp, status, err := service.GetFavorites(email)
	if err != nil {
		w.WriteHeader(status)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(status)
	w.Write(resp)
}
