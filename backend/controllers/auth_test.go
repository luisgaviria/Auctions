package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuthControllerRejectsInvalidJSON(t *testing.T) {
	controller := &AuthController{}
	for _, handler := range []struct {
		name string
		fn   http.HandlerFunc
	}{
		{name: "signup", fn: controller.SignUp},
		{name: "login", fn: controller.Login},
	} {
		t.Run(handler.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/auth/"+handler.name, strings.NewReader("{"))
			handler.fn(response, request)
			if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "Invalid request body") {
				t.Errorf("response = %d %q", response.Code, response.Body.String())
			}
		})
	}
}

func TestAuthControllerLogoutClearsCookie(t *testing.T) {
	response := httptest.NewRecorder()
	(&AuthController{}).Logout(response, httptest.NewRequest(http.MethodPost, "/auth/logout", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("Logout() status = %d", response.Code)
	}
	cookie := response.Result().Cookies()
	if len(cookie) != 1 || cookie[0].Name != "token" || cookie[0].MaxAge >= 0 || !cookie[0].HttpOnly {
		t.Errorf("logout cookie = %+v", cookie)
	}
	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["message"] != "Successfully logged out" {
		t.Errorf("logout body = %q, err %v", response.Body.String(), err)
	}
}

func TestControllerCreateToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "controller-test-secret")
	tokenString, err := createToken("person@example.com")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte("controller-test-secret"), nil
	})
	if err != nil || !parsed.Valid {
		t.Fatalf("created token invalid: %v", err)
	}
	claims := parsed.Claims.(jwt.MapClaims)
	if claims["sub"] != "person@example.com" {
		t.Errorf("token subject = %v", claims["sub"])
	}
	if exp, ok := claims["exp"].(float64); !ok || time.Unix(int64(exp), 0).Before(time.Now()) {
		t.Errorf("token expiration missing or expired: %v", claims["exp"])
	}
}
