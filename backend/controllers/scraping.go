package controllers

import (
	"backendAuction/utils"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type ScrapingController struct {
	DB *sql.DB
}

type ScrapingResponse struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}

func (c *ScrapingController) StartScraping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// The HTTP request context is cancelled as soon as the response is sent,
	// so we detach from it and use a fresh context with an explicit timeout.
	// 10 minutes is generous for 12 parallel scrapers.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		utils.ScrapAllSites(ctx, c.DB)
	}()

	json.NewEncoder(w).Encode(ScrapingResponse{
		Message: "Scraping started successfully",
	})
}
