package middleware

import (
	"net/http"
	"strings"
)

func CacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add cache headers for GET requests
		if r.Method == "GET" {
			w.Header().Set("Cache-Control", "public, max-age=300")
			w.Header().Set("Vary", "Accept-Encoding")

			// Handle browser cache validation
			if match := r.Header.Get("If-None-Match"); match != "" {
				etag := w.Header().Get("ETag")
				if strings.Trim(match, `"`) == strings.Trim(etag, `"`) {
					w.WriteHeader(http.StatusNotModified)
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}
