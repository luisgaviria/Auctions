package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCacheMiddleware(t *testing.T) {
	t.Run("GET adds cache headers", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := w.Header().Get("Cache-Control"); got != "public, max-age=300" {
				t.Errorf("Cache-Control = %q, want %q", got, "public, max-age=300")
			}
			if got := w.Header().Get("Vary"); got != "Accept-Encoding" {
				t.Errorf("Vary = %q, want %q", got, "Accept-Encoding")
			}
			w.WriteHeader(http.StatusOK)
		})

		response := httptest.NewRecorder()
		CacheMiddleware(next).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/auctions", nil))
		if response.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", response.Code, http.StatusOK)
		}
	})

	t.Run("non-GET does not add cache headers", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := w.Header().Get("Cache-Control"); got != "" {
				t.Errorf("Cache-Control = %q, want empty", got)
			}
			if got := w.Header().Get("Vary"); got != "" {
				t.Errorf("Vary = %q, want empty", got)
			}
		})

		response := httptest.NewRecorder()
		CacheMiddleware(next).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/auctions", nil))
	})

	t.Run("matching ETag returns not modified", func(t *testing.T) {
		nextCalled := false
		next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			nextCalled = true
		})
		request := httptest.NewRequest(http.MethodGet, "/auctions", nil)
		request.Header.Set("If-None-Match", `"v1"`)
		response := httptest.NewRecorder()
		response.Header().Set("ETag", `"v1"`)

		CacheMiddleware(next).ServeHTTP(response, request)
		if response.Code != http.StatusNotModified {
			t.Errorf("status = %d, want %d", response.Code, http.StatusNotModified)
		}
		if nextCalled {
			t.Error("next handler was called for a matching ETag")
		}
	})
}
