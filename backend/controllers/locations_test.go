package controllers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
)

func TestLocationsControllerGetCountiesError(t *testing.T) {
	db, mock := newControllerMockDB(t)
	mock.ExpectQuery("SELECT county_slug").WillReturnError(errors.New("database unavailable"))
	response := httptest.NewRecorder()
	(&LocationsController{DB: db}).GetCounties(response, httptest.NewRequest(http.MethodGet, "/locations/counties", nil))
	if response.Code != http.StatusInternalServerError {
		t.Errorf("GetCounties() status = %d", response.Code)
	}
}

func TestLocationsControllerGetCitiesByCounty(t *testing.T) {
	db, mock := newControllerMockDB(t)
	mock.ExpectQuery("SELECT city_slug, MAX\\(city\\) AS city, COUNT\\(\\*\\) AS auction_count").
		WithArgs("suffolk-county").
		WillReturnRows(sqlmock.NewRows([]string{"city_slug", "city", "auction_count"}).AddRow("boston", "BOSTON", 3))
	request := httptest.NewRequest(http.MethodGet, "/locations/counties/suffolk-county/cities", nil)
	request = mux.SetURLVars(request, map[string]string{"county_slug": " SUFFOLK-COUNTY "})
	response := httptest.NewRecorder()
	(&LocationsController{DB: db}).GetCitiesByCounty(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/json" {
		t.Errorf("GetCitiesByCounty() response = %d, headers %v", response.Code, response.Header())
	}
}
