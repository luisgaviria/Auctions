package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt"
)

type GetAuctionsResponse struct {
	Message string `json:"message"`
}

func GetAuctions(w http.ResponseWriter, r *http.Request) {
	props := r.Context().Value("props").(jwt.MapClaims)
	userEmail := fmt.Sprintf("%v", props["sub"])
	getAuctionsResponse := GetAuctionsResponse{Message: "logged as: " + userEmail}

	data, _ := json.Marshal(getAuctionsResponse)
	w.Write([]byte(data))
}
