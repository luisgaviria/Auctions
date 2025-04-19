package controllers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SignUpResponse struct {
	Message string `json:"message"`
}

type LoginResponse struct {
	Message  string `json:"message"`
	JwtToken string `json:"jwt_token"`
}

var insertIntoUserTable = `INSERT INTO users (email, password) VALUES ($1, $2)`
var selectFromUserTable = `SELECT email, password FROM users WHERE email=$1`

type AuthController struct {
	DB *sql.DB
}

func (c *AuthController) SignUp(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	creds := &Credentials{}
	if err := json.NewDecoder(req.Body).Decode(creds); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error processing password", http.StatusInternalServerError)
		return
	}

	// Insert user into database
	if _, err = c.DB.Exec(insertIntoUserTable, creds.Email, string(hashedPassword)); err != nil {
		http.Error(w, "Error creating user", http.StatusInternalServerError)
		log.Println("Database error:", err)
		return
	}

	json.NewEncoder(w).Encode(SignUpResponse{Message: "Successfully registered user"})
}

func (c *AuthController) Login(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	creds := &Credentials{}
	if err := json.NewDecoder(req.Body).Decode(creds); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get stored credentials
	storedCreds := &Credentials{}
	err := c.DB.QueryRow(selectFromUserTable, creds.Email).Scan(&storedCreds.Email, &storedCreds.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Println("Database error:", err)
		return
	}

	// Compare passwords
	if err := bcrypt.CompareHashAndPassword([]byte(storedCreds.Password), []byte(creds.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	token, err := createToken(creds.Email)
	if err != nil {
		http.Error(w, "Error creating token", http.StatusInternalServerError)
		return
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		MaxAge:   3600 * 24, // 24 hours
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})

	json.NewEncoder(w).Encode(LoginResponse{
		Message:  "Successfully logged in",
		JwtToken: token,
	})
}

func (c *AuthController) Logout(w http.ResponseWriter, r *http.Request) {
	// Clear the token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false, // Set to true in production
		SameSite: http.SameSiteStrictMode,
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Successfully logged out",
	})
}

func createToken(email string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": email,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
		"iat": time.Now().Unix(),
	})

	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}
