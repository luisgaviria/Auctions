package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"backendAuction/services"
	"backendAuction/models"
)

var insertIntoUserTable = `INSERT INTO users (email, password) VALUES ($1, $2)`
var selectFromUserTable = `SELECT email, password FROM users WHERE email=$1`

type AuthController struct {
	DB *sql.DB
}

func (c *AuthController) SignUp(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	creds := &models.Credentials{}
	if err := json.NewDecoder(req.Body).Decode(creds); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	service := services.NewAuthService(c.DB)
	resp, status, err := service.SignUp(creds)
	if err != nil {
		w.WriteHeader(status)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(status)
	w.Write(resp)
}

func (c *AuthController) Login(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	creds := &models.Credentials{}
	if err := json.NewDecoder(req.Body).Decode(creds); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	service := services.NewAuthService(c.DB)
	resp, status, err := service.Login(creds)
	if err != nil {
		w.WriteHeader(status)
		w.Write([]byte(err.Error()))
		return
	}
	// Set cookie if login successful
	if status == http.StatusOK {
		var loginResp services.LoginResponse
		if err := json.Unmarshal(resp, &loginResp); err == nil {
			http.SetCookie(w, &http.Cookie{
				Name:     "token",
				Value:    loginResp.JwtToken,
				MaxAge:   3600 * 24,
				HttpOnly: true,
				Secure:   false,
				SameSite: http.SameSiteStrictMode,
				Path:     "/",
			})
		}
	}
	w.WriteHeader(status)
	w.Write(resp)
}

func (c *AuthController) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})
	service := services.NewAuthService(c.DB)
	resp, status, _ := service.Logout()
	w.WriteHeader(status)
	w.Write(resp)
}

func createToken(email string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": email,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
		"iat": time.Now().Unix(),
	})

	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}
