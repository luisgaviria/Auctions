package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Credentials struct {
	Email    string `json:"email", db:"email"`
	Password string `json:"password", db:"password"`
}

type SignUpResponse struct {
	Message string `json:"message"`
}

type LoginResponse struct {
	Message  string `json:"message"`
	JwtToken string `json:"jwt_token"`
}

var insertIntoUserTable = `
	INSERT INTO users (email, password) VALUES ($1,$2);
`

var selectFromUserTable = `
	SELECT id, password FROM users WHERE email=$1;
`

type Controller struct {
	DB *sql.DB
}

func (c *Controller) Login(w http.ResponseWriter, req *http.Request) {
	creds := &Credentials{}

	err := json.NewDecoder(req.Body).Decode(creds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	result := c.DB.QueryRow(selectFromUserTable, creds.Email)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println(err)
		return
	}
	storedCreds := &Credentials{}
	err = result.Scan(&storedCreds.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Println(err)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println(err)
		return
	}

	if err = bcrypt.CompareHashAndPassword([]byte(storedCreds.Password), []byte(creds.Password)); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
	}

	// generate token

	loginResponse := LoginResponse{Message: "Succesfully login user", JwtToken: "test"} // need to generate jwtToken
	data, err := json.Marshal(loginResponse)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (c *Controller) SignUp(w http.ResponseWriter, req *http.Request) {
	creds := &Credentials{}
	err := json.NewDecoder(req.Body).Decode(creds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(creds.Password), 8)

	if _, err = c.DB.Query(insertIntoUserTable, creds.Email, string(hashedPassword)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println(err)
		return
	}

	signUpResponse := SignUpResponse{Message: "Succesfully register user"}
	data, err := json.Marshal(signUpResponse)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// func createToken(id string) {

// }
