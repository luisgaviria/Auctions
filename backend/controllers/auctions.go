package controllers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

var selectFromAuctionsTable = `SELECT * FROM auctions;`

type AuctionsController struct {
	DB *sql.DB
}

type GetAuctionsResponse struct {
	Message  string         `json:"message"`
	Auctions []AuctionModel `json:"auctions"`
}

type AuctionModel struct {
	Id        string    `json:"id", db:"id"`
	Address   string    `json:"address", db:"address"`
	City      string    `json:"city", db:"city"`
	State     string    `json:"state", db:"state"`
	Time      string    `json:"time", db:"time"`
	Logo      string    `json:"logo", db:"logo"`
	Status    string    `json:"status", db:"status"`
	Link      string    `json:"link", db:"link"`
	Date      time.Time `json:"date", db:"date"`
	Deposit   string    `json:"deposit", db:"deposit"`
	Lat       string    `json:"lat", db:"lat"`
	Lng       string    `json:"lng", db:"lng"`
	Createdat time.Time `json:"createdAt", db:"createdat"`
}

func (c *AuctionsController) GetAuctions(w http.ResponseWriter, req *http.Request) {
	auctions := make([]AuctionModel, 0)
	rows, err := c.DB.Query(selectFromAuctionsTable)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		auction := AuctionModel{}
		if err := rows.Scan(&auction.Id, &auction.Address, &auction.City, &auction.State, &auction.Time, &auction.Logo, &auction.Status, &auction.Link, &auction.Date, &auction.Deposit, &auction.Lat, &auction.Lng, &auction.Createdat); err != nil {
			panic(err)
		}
		auctions = append(auctions, auction)
	}
	if err := rows.Err(); err != nil {
		panic(err)
	}
	response := GetAuctionsResponse{Message: "Succesfully fetched auctions", Auctions: auctions}

	data, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
