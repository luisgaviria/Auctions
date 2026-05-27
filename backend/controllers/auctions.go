package controllers

import (
	"backendAuction/services"
	"backendAuction/utils/cache"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
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

	search := q.Get("search")
	data, status, err := service.GetAuctions(limit, offset, search)
	if err != nil {
		w.WriteHeader(status)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(status)
	w.Write(data)
}

// GetTopSlugs handles GET /auctions/slugs[?limit=N].
// Returns the top N (default 50) most active city+county slug pairs.
// Used by the Astro frontend's getStaticParams to pre-render city pages.
func (c *AuctionsController) GetTopSlugs(w http.ResponseWriter, req *http.Request) {
	limit := 50
	if l := req.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}
	service := services.NewAuctionsService(c.DB)
	data, status, err := service.GetTopCitySlugs(limit)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(data)
}

// GetAuctionsBySlug handles GET /auctions/{county_slug}/{city_slug}.
// Returns active auctions for the given city together with a summary badge
// (count + average deposit).  404 when the slug pair has no active auctions.
func (c *AuctionsController) GetAuctionsBySlug(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	countySlug := vars["county_slug"]
	citySlug := vars["city_slug"]

	service := services.NewAuctionsService(c.DB)
	data, status, err := service.GetAuctionsBySlug(countySlug, citySlug)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(data)
}

// GetReport handles GET /report/{address_slug}.
// Returns the shareable property report (auction details + due-diligence checklist).
func (c *AuctionsController) GetReport(w http.ResponseWriter, req *http.Request) {
	addressSlug := mux.Vars(req)["address_slug"]
	service := services.NewAuctionsService(c.DB)
	data, status, err := service.GetAuctionReport(addressSlug)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(data)
}

// InvalidateCache clears all paginated auction cache entries.
func (c *AuctionsController) InvalidateCache() {
	cache.Cache.Flush()
}
