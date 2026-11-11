package utils

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestBuildQuery(t *testing.T) {
	tests := []struct {
		name string
		row  auctionRow
		want string
	}{
		{
			name: "all address parts",
			row: auctionRow{
				Address: "  12 Main St  ",
				City:    sql.NullString{String: " Boston ", Valid: true},
				State:   sql.NullString{String: " MA ", Valid: true},
			},
			want: "12 Main St, Boston, MA, USA",
		},
		{
			name: "address only",
			row:  auctionRow{Address: "12 Main St"},
			want: "12 Main St, USA",
		},
		{
			name: "skip invalid optional fields",
			row: auctionRow{
				Address: "12 Main St",
				City:    sql.NullString{String: "Boston"},
				State:   sql.NullString{String: "MA"},
			},
			want: "12 Main St, USA",
		},
		{name: "empty address", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildQuery(tt.row); got != tt.want {
				t.Errorf("buildQuery() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCallMapTiler(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantLat    float64
		wantLng    float64
		wantError  string
	}{
		{
			name:       "coordinates use latitude longitude order",
			statusCode: http.StatusOK,
			body:       `{"features":[{"center":[-71.0589,42.3601],"context":[{"id":"region.123","text":"Massachusetts","short_code":"US-MA"}]}]}`,
			wantLat:    42.3601,
			wantLng:    -71.0589,
		},
		{
			name:       "skips unrelated first result and selects Massachusetts",
			statusCode: http.StatusOK,
			body:       `{"features":[{"center":[-74.006,40.7128],"context":[{"id":"region.456","text":"New York","short_code":"US-NY"}]},{"center":[-71.0589,42.3601],"context":[{"id":"region.123","text":"Massachusetts","short_code":"US-MA"}]}]}`,
			wantLat:    42.3601,
			wantLng:    -71.0589,
		},
		{
			name:       "API error",
			statusCode: http.StatusBadGateway,
			body:       "upstream unavailable",
			wantError:  "maptiler API 502: upstream unavailable",
		},
		{
			name:       "invalid JSON",
			statusCode: http.StatusOK,
			body:       "not json",
			wantError:  "decode maptiler response:",
		},
		{
			name:       "no results",
			statusCode: http.StatusOK,
			body:       `{"features":[]}`,
			wantError:  `no results for "Boston, MA"`,
		},
		{
			name:       "rejects result outside Massachusetts",
			statusCode: http.StatusOK,
			body:       `{"features":[{"center":[-74.006,40.7128],"context":[{"id":"region.456","text":"New York","short_code":"US-NY"}]}]}`,
			wantError:  `no result for "Boston, MA" matched Massachusetts`,
		},
		{
			name:       "rejects Massachusetts label with out-of-state coordinates",
			statusCode: http.StatusOK,
			body:       `{"features":[{"center":[-74.006,40.7128],"context":[{"id":"region.123","text":"Massachusetts","short_code":"US-MA"}]}]}`,
			wantError:  `no result for "Boston, MA" matched Massachusetts`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodGet {
					t.Errorf("request method = %q, want GET", req.Method)
				}
				if got := req.Header.Get("Origin"); got != "https://www.auctionandcompany.com" {
					t.Errorf("Origin header = %q, want configured site origin", got)
				}
				if got := req.URL.Query().Get("key"); got != "test key" {
					t.Errorf("API key query parameter = %q, want %q", got, "test key")
				}
				return &http.Response{
					StatusCode: tt.statusCode,
					Body:       io.NopCloser(strings.NewReader(tt.body)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			})}

			lat, lng, err := callMapTiler(context.Background(), client, "test key", "Boston, MA")
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("callMapTiler() error = %v, want substring %q", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("callMapTiler() error = %v", err)
			}
			if lat != tt.wantLat || lng != tt.wantLng {
				t.Errorf("callMapTiler() = (%v, %v), want (%v, %v)", lat, lng, tt.wantLat, tt.wantLng)
			}
		})
	}
}

func TestRetryDelay(t *testing.T) {
	tests := []struct {
		attempts int
		want     time.Duration
	}{
		{attempts: 0, want: time.Minute},
		{attempts: 1, want: time.Minute},
		{attempts: 2, want: 2 * time.Minute},
		{attempts: 4, want: 8 * time.Minute},
		{attempts: 20, want: 24 * time.Hour},
	}
	for _, tt := range tests {
		if got := retryDelay(tt.attempts); got != tt.want {
			t.Errorf("retryDelay(%d) = %s, want %s", tt.attempts, got, tt.want)
		}
	}
}

func TestInRunRetryDelay(t *testing.T) {
	tests := []struct {
		retry int
		want  time.Duration
	}{
		{retry: 1, want: time.Second},
		{retry: 2, want: 2 * time.Second},
		{retry: 6, want: 30 * time.Second},
	}
	for _, tt := range tests {
		if got := inRunRetryDelay(tt.retry); got != tt.want {
			t.Errorf("inRunRetryDelay(%d) = %s, want %s", tt.retry, got, tt.want)
		}
	}
}

func TestRecordGeocodeFailurePersistsRetryDetails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectExec(`(?s)UPDATE auctions\s+SET geocode_status = 'retry'.*geocode_error = \$1.*geocode_attempts = \$2.*geocode_retry_after = NOW\(\).*WHERE id = \$4`).
		WithArgs("no result matched Massachusetts", 5, int64((16 * time.Minute).Seconds()), 42).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = recordGeocodeFailure(context.Background(), db, auctionRow{ID: 42, Attempts: 2}, 3,
		errors.New("no result matched Massachusetts"))
	if err != nil {
		t.Fatalf("recordGeocodeFailure() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected persisted retry details: %v", err)
	}
}

func TestWriteCoordsMarksAuctionGeocoded(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectExec(`(?s)UPDATE auctions\s+SET lat = \$1, lng = \$2, geocode_status = 'geocoded'.*geocode_error = NULL.*geocode_attempts = geocode_attempts \+ \$3.*geocode_retry_after = NULL\s+WHERE id = \$4`).
		WithArgs("42.360100", "-71.058900", 3, 42).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := writeCoords(context.Background(), db, 42, 42.3601, -71.0589, 3); err != nil {
		t.Fatalf("writeCoords() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expected successful geocode state update: %v", err)
	}
}
