package models

// Credentials represents user login/signup credentials.
type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
} 