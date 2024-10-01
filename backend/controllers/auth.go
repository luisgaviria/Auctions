package controllers

import (
	"database/sql"
	"net/http"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Message string `json:"message"`
}

var insertIntoUserTable = `
	INSERT INTO (email, password) VALUES ('$s','$s')
`

var selectFromUserTable = `
	SELECT * FROM users 
`

type Controller struct {
	DB *sql.DB
}

func (c *Controller) Login(w http.ResponseWriter, req *http.Request) {

}

func (c *Controller) SignUp(w http.ResponseWriter, req *http.Request) {

}
