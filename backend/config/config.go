package config

import (
	"log"
	"os"
)

// GetDBURL returns the database URL from environment variables or logs a fatal error if not set.
func GetDBURL() string {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL is not set in the environment")
	}
	return dbURL
}

// GetFrontendURL returns the frontend URL from environment variables or a default value.
func GetFrontendURL() string {
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:4321"
	}
	return frontendURL
}

// GetPort returns the port from environment variables or a default value.
func GetPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	return port
}

// GetJWTSecret returns the JWT secret from environment variables or logs a fatal error if not set.
func GetJWTSecret() string {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET is not set in the environment")
	}
	return secret
} 