package services

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestFavoritesServiceAddFavorite(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(sqlmock.Sqlmock)
		wantStatus int
		wantError  bool
	}{
		{
			name: "success",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(getUserIDFromEmail)).WithArgs("person@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(23))
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM auctions WHERE id = \$1\)`).WithArgs(81).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
				mock.ExpectQuery(regexp.QuoteMeta(addToFavorites)).WithArgs(23, 81).
					WillReturnRows(sqlmock.NewRows([]string{"auction_id"}).AddRow(81))
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "user not found",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(getUserIDFromEmail)).WithArgs("missing@example.com").WillReturnError(sql.ErrNoRows)
			},
			wantStatus: http.StatusNotFound,
			wantError:  true,
		},
		{
			name: "auction does not exist",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(getUserIDFromEmail)).WithArgs("person@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(23))
				mock.ExpectQuery("SELECT EXISTS").WithArgs(81).
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
			},
			wantStatus: http.StatusNotFound,
			wantError:  true,
		},
		{
			name: "auction existence query fails",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(getUserIDFromEmail)).WithArgs("person@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(23))
				mock.ExpectQuery("SELECT EXISTS").WithArgs(81).WillReturnError(errors.New("database unavailable"))
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockDB(t)
			email := "person@example.com"
			if tt.name == "user not found" {
				email = "missing@example.com"
			}
			tt.setup(mock)
			data, status, err := NewFavoritesService(db).AddFavorite(email, &FavoriteRequest{AuctionID: 81})
			if status != tt.wantStatus || (err != nil) != tt.wantError {
				t.Fatalf("AddFavorite() = status %d, err %v, want status %d/error %t", status, err, tt.wantStatus, tt.wantError)
			}
			if err == nil {
				var response map[string]interface{}
				if decodeErr := json.Unmarshal(data, &response); decodeErr != nil {
					t.Fatal(decodeErr)
				}
				if response["message"] != "Added to favorites" || response["auction_id"] != float64(81) {
					t.Errorf("unexpected response: %#v", response)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestFavoritesServiceRemoveFavorite(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(sqlmock.Sqlmock)
		wantStatus int
		wantError  bool
	}{
		{
			name: "success",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(getUserIDFromEmail)).WithArgs("person@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(23))
				mock.ExpectQuery(regexp.QuoteMeta(removeFromFavorites)).WithArgs(23, 81).
					WillReturnRows(sqlmock.NewRows([]string{"auction_id"}).AddRow(81))
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "user not found",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(getUserIDFromEmail)).WithArgs("person@example.com").WillReturnError(sql.ErrNoRows)
			},
			wantStatus: http.StatusNotFound,
			wantError:  true,
		},
		{
			name: "favorite not found",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(getUserIDFromEmail)).WithArgs("person@example.com").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(23))
				mock.ExpectQuery(regexp.QuoteMeta(removeFromFavorites)).WithArgs(23, 81).WillReturnError(sql.ErrNoRows)
			},
			wantStatus: http.StatusNotFound,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := newMockDB(t)
			tt.setup(mock)
			data, status, err := NewFavoritesService(db).RemoveFavorite("person@example.com", &FavoriteRequest{AuctionID: 81})
			if status != tt.wantStatus || (err != nil) != tt.wantError {
				t.Fatalf("RemoveFavorite() = status %d, err %v", status, err)
			}
			if err == nil {
				var response map[string]interface{}
				if decodeErr := json.Unmarshal(data, &response); decodeErr != nil {
					t.Fatal(decodeErr)
				}
				if response["message"] != "Removed from favorites" {
					t.Errorf("unexpected response: %#v", response)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestFavoritesServiceGetFavorites(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(getUserIDFromEmail)).WithArgs("person@example.com").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(23))
		mock.ExpectQuery(regexp.QuoteMeta(getFavorites)).WithArgs(23).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "address", "city", "state", "time", "logo", "status", "link", "date", "deposit", "lat", "lng", "created_at", "site_name", "updated_at",
			}).AddRow(81, "12 Main St", "Boston", "MA", "10:00 AM", "", "Active", "https://example.test", nil, nil, 0.0, 0.0, time.Now(), "test", time.Now()))

		data, status, err := NewFavoritesService(db).GetFavorites("person@example.com")
		if err != nil || status != http.StatusOK {
			t.Fatalf("GetFavorites() = status %d, err %v", status, err)
		}
		var response FavoritesResponse
		if err := json.Unmarshal(data, &response); err != nil {
			t.Fatal(err)
		}
		if len(response.Auctions) != 1 || response.Auctions[0].Id != 81 || response.Auctions[0].Address != "12 Main St" {
			t.Errorf("unexpected favorites response: %+v", response)
		}
	})

	t.Run("unknown user", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(getUserIDFromEmail)).WithArgs("missing@example.com").WillReturnError(sql.ErrNoRows)
		data, status, err := NewFavoritesService(db).GetFavorites("missing@example.com")
		if status != http.StatusNotFound || err == nil || data != nil {
			t.Errorf("GetFavorites() = (%q, %d, %v), want nil/404/error", data, status, err)
		}
	})
}
