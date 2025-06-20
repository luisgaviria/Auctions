package controllers

import (
	"backendAuction/services"
	"backendAuction/utils/cache"
	"database/sql"
	"net/http"
)

var selectFromAuctionsTable = `SELECT * FROM auctions;`

type AuctionsController struct {
	DB *sql.DB
}

func (c *AuctionsController) GetAuctions(w http.ResponseWriter, req *http.Request) {
	service := services.NewAuctionsService(c.DB)
	data, status, err := service.GetAuctions()
	if err != nil {
		w.WriteHeader(status)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(status)
	w.Write(data)
}

// Add a method to invalidate cache when auctions are updated
func (c *AuctionsController) InvalidateCache() {
	cache.Cache.Delete(cache.AuctionsKey)
}
