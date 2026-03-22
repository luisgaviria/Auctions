package models

import (
	"database/sql"
	"time"
)

// AuctionModel represents an auction entity as stored/scanned from the DB.
// Date uses sql.NullTime so NULL database values scan cleanly without error.
type AuctionModel struct {
	Id        int          `json:"id"`
	Address   string       `json:"address"`
	City      string       `json:"city"`
	State     string       `json:"state"`
	Time      string       `json:"time"`
	Logo      string       `json:"logo"`
	SiteName  string       `json:"site_name"`
	Status    string       `json:"status"`
	Link      string       `json:"link"`
	Date      sql.NullTime `json:"-"` // scanned as nullable; use ToJSON() for API responses
	Deposit   string       `json:"deposit"`
	Lat       string       `json:"lat"`
	Lng       string       `json:"lng"`
	Createdat time.Time    `json:"createdAt"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// AuctionJSON is the API response shape. Date is a human-readable string ("Jan 2, 2006")
// or empty string when the auction has no scheduled date.
type AuctionJSON struct {
	Id       int    `json:"id"`
	Address  string `json:"address"`
	City     string `json:"city"`
	State    string `json:"state"`
	Time     string `json:"time"`
	Logo     string `json:"logo"`
	SiteName string `json:"site_name"`
	Status   string `json:"status"`
	Link     string `json:"link"`
	Date     string `json:"date"`
	Deposit  string `json:"deposit"`
	Lat      string `json:"lat"`
	Lng      string `json:"lng"`
}

// ToJSON converts a scanned AuctionModel into an API-safe AuctionJSON.
func (a AuctionModel) ToJSON() AuctionJSON {
	date := ""
	if a.Date.Valid {
		date = a.Date.Time.Format("Jan 2, 2006")
	}
	return AuctionJSON{
		Id:       a.Id,
		Address:  a.Address,
		City:     a.City,
		State:    a.State,
		Time:     a.Time,
		Logo:     a.Logo,
		SiteName: a.SiteName,
		Status:   a.Status,
		Link:     a.Link,
		Date:    date,
		Deposit: a.Deposit,
		Lat:     a.Lat,
		Lng:     a.Lng,
	}
} 