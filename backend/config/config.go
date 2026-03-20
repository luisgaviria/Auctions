package config

import (
	"log"
	"os"
	"strings"
)

// GetDBURL returns the database URL from environment variables or logs a fatal error if not set.
func GetDBURL() string {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set in the environment")
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

// GetAllowedOrigins returns the list of allowed CORS origins from ALLOWED_ORIGINS
// (comma-separated). Falls back to FRONTEND_URL for backwards compatibility.
// Logs a warning if neither variable is set.
func GetAllowedOrigins() []string {
	raw := os.Getenv("ALLOWED_ORIGINS")
	if raw == "" {
		fallback := os.Getenv("FRONTEND_URL")
		if fallback == "" {
			log.Println("[cors] WARNING: ALLOWED_ORIGINS and FRONTEND_URL are both unset — defaulting to http://localhost:4321")
			fallback = "http://localhost:4321"
		}
		return []string{fallback}
	}
	var origins []string
	for _, o := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(o); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
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