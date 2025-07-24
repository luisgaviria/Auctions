package controllers

import (
	"backendAuction/utils"
	"database/sql"
	"encoding/json"
	"net/http"
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

	go func() {
		utils.ScrapAllSites(c.DB)
	}()

	json.NewEncoder(w).Encode(ScrapingResponse{
		Message: "Scraping started successfully",
	})
}
