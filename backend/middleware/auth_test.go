package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuthMiddleware(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	validToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "user@example.com"})
	validString, err := validToken.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign test token: %v", err)
	}
	wrongSecretToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "user@example.com"})
	wrongSecretString, err := wrongSecretToken.SignedString([]byte("wrong-secret"))
	if err != nil {
		t.Fatalf("sign wrong-secret test token: %v", err)
	}

	tests := []struct {
		name           string
		authorization  string
		wantStatus     int
		wantNextCalled bool
	}{
		{name: "missing header", wantStatus: http.StatusUnauthorized},
		{name: "malformed header", authorization: "Bearer", wantStatus: http.StatusUnauthorized},
		{name: "invalid token", authorization: "Bearer not-a-token", wantStatus: http.StatusUnauthorized},
		{name: "bad signature", authorization: "Bearer " + wrongSecretString, wantStatus: http.StatusUnauthorized},
		{name: "valid token", authorization: "Bearer " + validString, wantStatus: http.StatusNoContent, wantNextCalled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				if got := r.Context().Value("sub"); got != "user@example.com" {
					t.Errorf("context sub = %v, want user@example.com", got)
				}
				w.WriteHeader(http.StatusNoContent)
			})

			req := httptest.NewRequest(http.MethodGet, "/private", nil)
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}
			response := httptest.NewRecorder()
			AuthMiddleware(next).ServeHTTP(response, req)

			if response.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if nextCalled != tt.wantNextCalled {
				t.Errorf("next called = %t, want %t", nextCalled, tt.wantNextCalled)
			}
		})
	}
}
