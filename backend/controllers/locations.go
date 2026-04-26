package controllers

import (
	"backendAuction/services"
	"database/sql"
	"net/http"

	"github.com/gorilla/mux"
)

type LocationsController struct {
	DB *sql.DB
}

// GetCounties handles GET /locations/counties.
// Returns all counties with active auctions and their counts.
func (c *LocationsController) GetCounties(w http.ResponseWriter, req *http.Request) {
	svc := services.NewLocationsService(c.DB)
	data, status, err := svc.GetCounties()
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(data)
}

// GetCitiesByCounty handles GET /locations/counties/{county_slug}/cities.
// Returns all cities in the county with active auctions and their counts.
func (c *LocationsController) GetCitiesByCounty(w http.ResponseWriter, req *http.Request) {
	countySlug := mux.Vars(req)["county_slug"]
	svc := services.NewLocationsService(c.DB)
	data, status, err := svc.GetCitiesByCounty(countySlug)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(data)
}
