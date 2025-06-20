package services

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"backendAuction/models"
)

type AuthService struct {
	DB *sql.DB
}

func NewAuthService(db *sql.DB) *AuthService {
	return &AuthService{DB: db}
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

func (s *AuthService) SignUp(creds *models.Credentials) ([]byte, int, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if _, err = s.DB.Exec(insertIntoUserTable, creds.Email, string(hashedPassword)); err != nil {
		log.Println("Database error:", err)
		return nil, http.StatusInternalServerError, err
	}
	resp, _ := json.Marshal(SignUpResponse{Message: "Successfully registered user"})
	return resp, http.StatusOK, nil
}

func (s *AuthService) Login(creds *models.Credentials) ([]byte, int, error) {
	storedCreds := &models.Credentials{}
	err := s.DB.QueryRow(selectFromUserTable, creds.Email).Scan(&storedCreds.Email, &storedCreds.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, http.StatusUnauthorized, err
		}
		log.Println("Database error:", err)
		return nil, http.StatusInternalServerError, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(storedCreds.Password), []byte(creds.Password)); err != nil {
		return nil, http.StatusUnauthorized, err
	}
	token, err := createToken(creds.Email)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	resp, _ := json.Marshal(LoginResponse{
		Message:  "Successfully logged in",
		JwtToken: token,
	})
	return resp, http.StatusOK, nil
}

func (s *AuthService) Logout() ([]byte, int, error) {
	resp, _ := json.Marshal(map[string]string{"message": "Successfully logged out"})
	return resp, http.StatusOK, nil
}

func createToken(email string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": email,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
		"iat": time.Now().Unix(),
	})
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
} 