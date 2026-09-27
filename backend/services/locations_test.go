package services

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestLocationsServiceGetCounties(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT county_slug, COUNT(*) AS auction_count")).
		WillReturnRows(sqlmock.NewRows([]string{"county_slug", "auction_count"}).
			AddRow("worcester-county", 12).
			AddRow("suffolk-county", 5))

	data, status, err := NewLocationsService(db).GetCounties()
	if err != nil || status != http.StatusOK {
		t.Fatalf("GetCounties() = status %d, err %v", status, err)
	}
	var response CountiesResponse
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Counties) != 2 || response.Counties[0].County != "Worcester County" || response.Counties[0].AuctionCount != 12 {
		t.Errorf("unexpected counties response: %+v", response)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLocationsServiceGetCitiesByCounty(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT city_slug, MAX(city) AS city, COUNT(*) AS auction_count")).
		WithArgs("worcester-county").
		WillReturnRows(sqlmock.NewRows([]string{"city_slug", "city", "auction_count"}).
			AddRow("north-bridge", "NORTH BRIDGE", 4))

	data, status, err := NewLocationsService(db).GetCitiesByCounty(" Worcester-County ")
	if err != nil || status != http.StatusOK {
		t.Fatalf("GetCitiesByCounty() = status %d, err %v", status, err)
	}
	var response CountyCitiesResponse
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}
	if response.County != "Worcester County" || response.CountySlug != "worcester-county" || len(response.Cities) != 1 {
		t.Errorf("unexpected county/cities response: %+v", response)
	}
	if got := response.Cities[0]; got.City != "North Bridge" || got.AuctionCount != 4 {
		t.Errorf("unexpected city row: %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLocationsServiceGetCitiesByCountyEmptyAndQueryError(t *testing.T) {
	t.Run("empty result", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery("SELECT city_slug").WithArgs("empty-county").
			WillReturnRows(sqlmock.NewRows([]string{"city_slug", "city", "auction_count"}))
		data, status, err := NewLocationsService(db).GetCitiesByCounty("empty-county")
		if status != http.StatusNotFound || err == nil || data != nil {
			t.Errorf("GetCitiesByCounty() = (%q, %d, %v), want nil/404/error", data, status, err)
		}
	})

	t.Run("query error", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery("SELECT city_slug").WithArgs("failed-county").WillReturnError(errors.New("query failed"))
		data, status, err := NewLocationsService(db).GetCitiesByCounty("failed-county")
		if status != http.StatusInternalServerError || err == nil || data != nil {
			t.Errorf("GetCitiesByCounty() = (%q, %d, %v), want nil/500/error", data, status, err)
		}
	})
}

func TestLocationsServiceGetCitiesForRelated(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery("SELECT city_slug, MAX\\(city\\) AS city, COUNT\\(\\*\\) AS auction_count").
		WithArgs("norfolk-county", "boston", 3).
		WillReturnRows(sqlmock.NewRows([]string{"city_slug", "city", "auction_count"}).
			AddRow("quincy", "QUINCY", 8))

	cities, err := NewLocationsService(db).GetCitiesForRelated(" Norfolk-County ", " BOSTON ", 3)
	if err != nil {
		t.Fatalf("GetCitiesForRelated() error = %v", err)
	}
	if len(cities) != 1 || cities[0].City != "Quincy" || cities[0].CountySlug != "norfolk-county" {
		t.Errorf("unexpected related cities: %+v", cities)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
