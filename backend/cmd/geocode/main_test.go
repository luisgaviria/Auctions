package main

import "testing"

func TestGetDatabaseURL(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "")
		if got, err := getDatabaseURL(); got != "" || err == nil || err.Error() != "DATABASE_URL not set" {
			t.Errorf("getDatabaseURL() = (%q, %v), want empty URL and missing-variable error", got, err)
		}
	})

	t.Run("configured", func(t *testing.T) {
		t.Setenv("DATABASE_URL", "postgres://db.example/auctions")
		if got, err := getDatabaseURL(); err != nil || got != "postgres://db.example/auctions" {
			t.Errorf("getDatabaseURL() = (%q, %v)", got, err)
		}
	})
}
