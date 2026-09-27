package controllers

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFavoritesControllerRejectsInvalidJSON(t *testing.T) {
	controller := &FavoritesController{}
	handlers := []struct {
		name string
		fn   http.HandlerFunc
	}{
		{name: "add", fn: controller.AddFavorite},
		{name: "remove", fn: controller.RemoveFavorite},
	}

	for _, handler := range handlers {
		t.Run(handler.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/favorites/"+handler.name, strings.NewReader("{"))
			request = request.WithContext(context.WithValue(request.Context(), "sub", "person@example.com"))
			response := httptest.NewRecorder()
			handler.fn(response, request)
			if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "Invalid request format") {
				t.Errorf("response = %d %q", response.Code, response.Body.String())
			}
		})
	}
}

func TestFavoritesControllerGetFavoritesPropagatesServiceError(t *testing.T) {
	db, mock := newControllerMockDB(t)
	mock.ExpectQuery("SELECT id FROM users WHERE email = \\$1").
		WithArgs("missing@example.com").WillReturnError(sql.ErrNoRows)
	request := httptest.NewRequest(http.MethodGet, "/favorites", nil)
	request = request.WithContext(context.WithValue(request.Context(), "sub", "missing@example.com"))
	response := httptest.NewRecorder()
	(&FavoritesController{DB: db}).GetFavorites(response, request)
	if response.Code != http.StatusNotFound {
		t.Errorf("GetFavorites() status = %d, want %d", response.Code, http.StatusNotFound)
	}
}
