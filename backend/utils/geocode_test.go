package utils

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"strings"
	"testing"
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
			body:       `{"features":[{"center":[-71.0589,42.3601]}]}`,
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
