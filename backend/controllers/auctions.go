package controllers

import (
	"backendAuction/services"
	"backendAuction/utils/cache"
	"database/sql"
	"net/http"
	"strconv"
)

type AuctionsController struct {
	DB *sql.DB
}

func (c *AuctionsController) GetAuctions(w http.ResponseWriter, req *http.Request) {
	limit := 20
	offset := 0

	if l := req.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	if o := req.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	service := services.NewAuctionsService(c.DB)
	data, status, err := service.GetAuctions(limit, offset)
	if err != nil {
		w.WriteHeader(status)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(status)
	w.Write(data)
}

// InvalidateCache clears all paginated auction cache entries.
func (c *AuctionsController) InvalidateCache() {
	cache.Cache.Flush()
}
