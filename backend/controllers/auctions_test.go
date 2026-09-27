package controllers

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"backendAuction/utils/cache"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
)

func TestAuctionsControllerGetAuctionsUsesDefaultsOnInvalidQuery(t *testing.T) {
	db, mock := newControllerMockDB(t)
	mock.ExpectQuery(`(?s)SELECT .*FROM auctions`).WithArgs(20, 0, "%").
		WillReturnError(errors.New("database unavailable"))
	request := httptest.NewRequest(http.MethodGet, "/auctions?limit=0&offset=-1&search=%20&north=invalid&south=1&east=2&west=0", nil)
	response := httptest.NewRecorder()
	(&AuctionsController{DB: db}).GetAuctions(response, request)
	if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), "database unavailable") {
		t.Errorf("GetAuctions() response = %d %q", response.Code, response.Body.String())
	}
}

func TestAuctionsControllerGetTopSlugsCapsLimit(t *testing.T) {
	db, mock := newControllerMockDB(t)
	mock.ExpectQuery(`(?s)SELECT county_slug, city_slug, MAX\(city\) AS city, COUNT\(\*\) AS auction_count.*LIMIT \$1`).
		WithArgs(50).
		WillReturnRows(sqlmock.NewRows([]string{"county_slug", "city_slug", "city", "auction_count"}))
	request := httptest.NewRequest(http.MethodGet, "/auctions/slugs?limit=501", nil)
	response := httptest.NewRecorder()
	(&AuctionsController{DB: db}).GetTopSlugs(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/json" || response.Body.String() != `{"markets":[]}` {
		t.Errorf("GetTopSlugs() response = %d, headers %v, body %q", response.Code, response.Header(), response.Body.String())
	}
}

func TestAuctionsControllerGetAuctionsBySlugAndReportErrors(t *testing.T) {
	t.Run("slug service error", func(t *testing.T) {
		db, mock := newControllerMockDB(t)
		mock.ExpectQuery(`(?s)SELECT\s+MAX\(city\).*FROM auctions`).
			WithArgs("norfolk-county", "quincy").WillReturnError(errors.New("database unavailable"))
		request := httptest.NewRequest(http.MethodGet, "/auctions/norfolk-county/quincy", nil)
		request = mux.SetURLVars(request, map[string]string{"county_slug": "NORFOLK-COUNTY", "city_slug": "Quincy"})
		response := httptest.NewRecorder()
		(&AuctionsController{DB: db}).GetAuctionsBySlug(response, request)
		if response.Code != http.StatusInternalServerError {
			t.Errorf("GetAuctionsBySlug() status = %d", response.Code)
		}
	})

	t.Run("report not found", func(t *testing.T) {
		db, mock := newControllerMockDB(t)
		mock.ExpectQuery(`(?s)SELECT .*address_slug.*FROM auctions`).
			WithArgs("missing-report").WillReturnError(sql.ErrNoRows)
		request := httptest.NewRequest(http.MethodGet, "/report/missing-report", nil)
		request = mux.SetURLVars(request, map[string]string{"address_slug": "Missing-Report"})
		response := httptest.NewRecorder()
		(&AuctionsController{DB: db}).GetReport(response, request)
		if response.Code != http.StatusNotFound {
			t.Errorf("GetReport() status = %d", response.Code)
		}
	})
}

func TestAuctionsControllerInvalidateCache(t *testing.T) {
	cache.Cache.Set("auctions_test_entry", []byte("cached"), 0)
	cache.Cache.Set("other_test_entry", []byte("keep"), 0)
	t.Cleanup(func() {
		cache.Cache.Delete("auctions_test_entry")
		cache.Cache.Delete("other_test_entry")
	})
	(&AuctionsController{}).InvalidateCache()
	if _, found := cache.Cache.Get("auctions_test_entry"); found {
		t.Error("auction cache entry was not invalidated")
	}
	if _, found := cache.Cache.Get("other_test_entry"); !found {
		t.Error("unrelated cache entry was invalidated")
	}
}
