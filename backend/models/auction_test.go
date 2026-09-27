package models

import (
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestAuctionModelToJSON(t *testing.T) {
	model := AuctionModel{
		Id:           7,
		Address:      "12 Main St",
		City:         "Boston",
		State:        "MA",
		Date:         sql.NullTime{Time: time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC), Valid: true},
		Deposit:      sql.NullInt64{Int64: 1234567, Valid: true},
		Lat:          sql.NullFloat64{Float64: 42.3601, Valid: true},
		Lng:          sql.NullFloat64{Float64: -71.0589, Valid: true},
		ZillowURL:    sql.NullString{String: "https://example.com/zillow", Valid: true},
		RegistryBook: sql.NullInt64{Int64: 50303, Valid: true},
		RegistryPage: sql.NullInt64{Int64: 200, Valid: true},
	}

	got := model.ToJSON()
	if got.Id != 7 || got.Address != "12 Main St" || got.City != "Boston" {
		t.Errorf("identity fields were not preserved: %+v", got)
	}
	if got.Date != "Jan 2, 2026" {
		t.Errorf("Date = %q, want %q", got.Date, "Jan 2, 2026")
	}
	if got.Deposit != "$1,234,567" {
		t.Errorf("Deposit = %q, want %q", got.Deposit, "$1,234,567")
	}
	if got.Lat != "42.3601" || got.Lng != "-71.0589" {
		t.Errorf("coordinates = (%q, %q), want (42.3601, -71.0589)", got.Lat, got.Lng)
	}
	if got.ZillowURL != "https://example.com/zillow" || got.RegistryBook != 50303 || got.RegistryPage != 200 {
		t.Errorf("optional registry and URL fields were not preserved: %+v", got)
	}
}

func TestAuctionModelToJSONWithNullFields(t *testing.T) {
	got := (AuctionModel{}).ToJSON()
	if got.Date != "" || got.Deposit != "" {
		t.Errorf("null date and deposit should be empty, got date=%q deposit=%q", got.Date, got.Deposit)
	}
	if got.Lat != "0" || got.Lng != "0" {
		t.Errorf("null coordinates should default to zero, got (%q, %q)", got.Lat, got.Lng)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if strings.Contains(string(encoded), "registry_deep_link") || strings.Contains(string(encoded), "assessor_pid") {
		t.Errorf("empty optional fields should be omitted from JSON: %s", encoded)
	}
}
