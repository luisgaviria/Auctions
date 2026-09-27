package services

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"backendAuction/models"
	"backendAuction/utils/cache"
	"github.com/DATA-DOG/go-sqlmock"
)

func auctionFixtureColumns() []string {
	columns := strings.Split(auctionCols, ",")
	for i := range columns {
		columns[i] = strings.TrimSpace(columns[i])
	}
	return columns
}

func auctionFixtureValues() []driver.Value {
	now := time.Date(2026, time.January, 2, 10, 0, 0, 0, time.UTC)
	return []driver.Value{
		int64(81), "12 Main St", "Boston", "MA", "10:00 AM", "", "Active", "https://auction.example/81",
		nil, int64(5000), float64(42.3601), float64(-71.0589), now, "test", now, now,
		"", "", "", "", "", int64(0), int64(0),
	}
}

func oneAuctionRows() *sqlmock.Rows {
	return sqlmock.NewRows(auctionFixtureColumns()).AddRow(auctionFixtureValues()...)
}

func TestAuctionsServiceGetAuctionsAndCache(t *testing.T) {
	db, mock := newMockDB(t)
	search := "unique-auction-cache-test"
	eastern, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	cacheKey := "auctions_" + time.Now().In(eastern).Format("2006-01-02") + "_7_4_" + search
	cache.Cache.Delete(cacheKey)
	t.Cleanup(func() { cache.Cache.Delete(cacheKey) })

	mock.ExpectQuery(`(?s)SELECT .*FROM auctions`).WithArgs(7, 4, "%"+search+"%").
		WillReturnRows(oneAuctionRows())
	service := NewAuctionsService(db)
	data, status, err := service.GetAuctions(7, 4, search)
	if err != nil || status != http.StatusOK {
		t.Fatalf("GetAuctions() = status %d, err %v", status, err)
	}
	var response GetAuctionsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Auctions) != 1 || response.Auctions[0].Id != 81 || response.Auctions[0].Deposit != "$5,000" {
		t.Errorf("unexpected auctions response: %+v", response)
	}

	cached, status, err := service.GetAuctions(7, 4, search)
	if err != nil || status != http.StatusOK || string(cached) != string(data) {
		t.Errorf("cached GetAuctions() differs: status %d, err %v", status, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAuctionsServiceGetAuctionsBySlug(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(`(?s)SELECT\s+MAX\(city\).*FROM auctions`).
		WithArgs("worcester-county", "southbridge").
		WillReturnRows(sqlmock.NewRows([]string{"city_name", "auction_count", "avg_deposit"}).AddRow("SOUTHBRIDGE", 1, 12500.0))
	mock.ExpectQuery(`(?s)SELECT .*FROM auctions`).
		WithArgs("worcester-county", "southbridge").
		WillReturnRows(oneAuctionRows())

	data, status, err := NewAuctionsService(db).GetAuctionsBySlug(" Worcester-County ", " SOUTHBRIDGE ")
	if err != nil || status != http.StatusOK {
		t.Fatalf("GetAuctionsBySlug() = status %d, err %v", status, err)
	}
	var response CityMarketResponse
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}
	if response.Summary.City != "SOUTHBRIDGE" || response.Summary.County != "Worcester County" || response.Summary.AverageDeposit != "$12,500" {
		t.Errorf("unexpected market summary: %+v", response.Summary)
	}
	if len(response.Auctions) != 1 || response.Auctions[0].Id != 81 {
		t.Errorf("unexpected market auctions: %+v", response.Auctions)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAuctionsServiceGetTopCitySlugs(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(`(?s)SELECT county_slug, city_slug, MAX\(city\) AS city, COUNT\(\*\) AS auction_count.*LIMIT \$1`).
		WithArgs(10).
		WillReturnRows(sqlmock.NewRows([]string{"county_slug", "city_slug", "city", "auction_count"}).
			AddRow("norfolk-county", "quincy", "QUINCY", 8))

	data, status, err := NewAuctionsService(db).GetTopCitySlugs(10)
	if err != nil || status != http.StatusOK {
		t.Fatalf("GetTopCitySlugs() = status %d, err %v", status, err)
	}
	var response TopCitySlugsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Markets) != 1 || response.Markets[0].CitySlug != "quincy" || response.Markets[0].AuctionCount != 8 {
		t.Errorf("unexpected top-slugs response: %+v", response)
	}
}

func TestAuctionsServiceGetAuctionReport(t *testing.T) {
	db, mock := newMockDB(t)
	columns := append(auctionFixtureColumns(), "address_slug")
	values := append(auctionFixtureValues(), "12-main-st-boston")
	mock.ExpectQuery(`(?s)SELECT .*address_slug.*FROM auctions`).
		WithArgs("12-main-st-boston").
		WillReturnRows(sqlmock.NewRows(columns).AddRow(values...))

	data, status, err := NewAuctionsService(db).GetAuctionReport(" 12-MAIN-ST-BOSTON ")
	if err != nil || status != http.StatusOK {
		t.Fatalf("GetAuctionReport() = status %d, err %v", status, err)
	}
	var report models.AuctionReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.AddressSlug != "12-main-st-boston" || report.ShareURL != "https://auctionandcompany.com/report/12-main-st-boston" {
		t.Errorf("unexpected report identity: %+v", report)
	}
	if report.Auction.ZillowURL == "" || report.Auction.RegistryURL == "" || len(report.Checklist) != 13 {
		t.Errorf("report URLs/checklist were not populated: auction=%+v checklist=%d", report.Auction, len(report.Checklist))
	}
}

func TestAuctionsServiceGetAuctionReportNotFound(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(`(?s)SELECT .*address_slug.*FROM auctions`).
		WithArgs("missing-address").WillReturnError(sql.ErrNoRows)
	data, status, err := NewAuctionsService(db).GetAuctionReport("Missing-Address")
	if status != http.StatusNotFound || err == nil || data != nil {
		t.Errorf("GetAuctionReport() = (%q, %d, %v), want nil/404/error", data, status, err)
	}
}

func TestAuctionsServiceGetAuctionsInBounds(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(`(?s)SELECT .*FROM auctions`).
		WithArgs(42.0, 43.0, -72.0, -71.0).
		WillReturnRows(oneAuctionRows())

	data, status, err := NewAuctionsService(db).GetAuctionsInBounds(42, 43, -72, -71)
	if err != nil || status != http.StatusOK {
		t.Fatalf("GetAuctionsInBounds() = status %d, err %v", status, err)
	}
	var response GetAuctionsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Auctions) != 1 || response.Auctions[0].Id != 81 {
		t.Errorf("unexpected bounds response: %+v", response)
	}
}

func TestAuctionsServiceSlugNotFound(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(`(?s)SELECT\s+MAX\(city\).*FROM auctions`).
		WithArgs("empty-county", "empty-city").
		WillReturnRows(sqlmock.NewRows([]string{"city_name", "auction_count", "avg_deposit"}).AddRow(nil, 0, nil))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM auctions`).
		WithArgs("empty-county", "empty-city").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	data, status, err := NewAuctionsService(db).GetAuctionsBySlug("empty-county", "empty-city")
	if status != http.StatusNotFound || err == nil || data != nil {
		t.Errorf("GetAuctionsBySlug() = (%q, %d, %v), want nil/404/error", data, status, err)
	}
}
