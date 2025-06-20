package models

import "time"

// AuctionModel represents an auction entity.
type AuctionModel struct {
	Id        int       `json:"id"`
	Address   string    `json:"address"`
	City      string    `json:"city"`
	State     string    `json:"state"`
	Time      string    `json:"time"`
	Logo      string    `json:"logo"`
	Status    string    `json:"status"`
	Link      string    `json:"link"`
	Date      time.Time `json:"date"`
	Deposit   string    `json:"deposit"`
	Lat       string    `json:"lat"`
	Lng       string    `json:"lng"`
	Createdat time.Time `json:"createdAt"`
} 