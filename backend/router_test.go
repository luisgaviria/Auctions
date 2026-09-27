package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRouterHealthAndCORS(t *testing.T) {
	router := newRouter(nil, []string{"https://frontend.example"})

	t.Run("health route and allowed origin", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/health", nil)
		request.Header.Set("Origin", "https://frontend.example")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK || response.Body.String() != "OK" {
			t.Errorf("health response = %d %q", response.Code, response.Body.String())
		}
		if got := response.Header().Get("Access-Control-Allow-Origin"); got != "https://frontend.example" {
			t.Errorf("allowed origin header = %q", got)
		}
	})

	t.Run("preflight is answered before route handlers", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodOptions, "/favorites", nil)
		request.Header.Set("Origin", "https://frontend.example")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK || response.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Errorf("preflight response = %d, headers %v", response.Code, response.Header())
		}
	})

	t.Run("unlisted origin is not echoed", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/health", nil)
		request.Header.Set("Origin", "https://attacker.example")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("unlisted origin was allowed: %q", got)
		}
	})
}
