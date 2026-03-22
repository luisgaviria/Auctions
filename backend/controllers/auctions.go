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
	q := req.URL.Query()
	service := services.NewAuctionsService(c.DB)

	// Bounding-box search: all four params must be present and valid.
	northStr := q.Get("north")
	southStr := q.Get("south")
	eastStr := q.Get("east")
	westStr := q.Get("west")

	if northStr != "" && southStr != "" && eastStr != "" && westStr != "" {
		north, errN := strconv.ParseFloat(northStr, 64)
		south, errS := strconv.ParseFloat(southStr, 64)
		east, errE := strconv.ParseFloat(eastStr, 64)
		west, errW := strconv.ParseFloat(westStr, 64)

		if errN == nil && errS == nil && errE == nil && errW == nil {
			data, status, err := service.GetAuctionsInBounds(south, north, west, east)
			if err != nil {
				w.WriteHeader(status)
				w.Write([]byte(err.Error()))
				return
			}
			w.WriteHeader(status)
			w.Write(data)
			return
		}
	}

	// Standard paginated query.
	limit := 20
	offset := 0

	if l := q.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	if o := q.Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

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
