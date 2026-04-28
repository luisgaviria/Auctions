package models

import (
	"database/sql"
	"time"
)

// AuctionModel represents an auction entity as stored/scanned from the DB.
// Date uses sql.NullTime so NULL database values scan cleanly without error.
type AuctionModel struct {
	Id               int            `json:"id"`
	Address          string         `json:"address"`
	City             string         `json:"city"`
	State            string         `json:"state"`
	Time             string         `json:"time"`
	Logo             string         `json:"logo"`
	SiteName         string         `json:"site_name"`
	Status           string         `json:"status"`
	Link             string         `json:"link"`
	Date             sql.NullTime   `json:"-"` // scanned as nullable; use ToJSON() for API responses
	Deposit          string         `json:"deposit"`
	Lat              string         `json:"lat"`
	Lng              string         `json:"lng"`
	Createdat        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updated_at"`
	LastSeen         sql.NullTime   `json:"-"` // internal sync column; not exposed in API responses
	ZillowURL        sql.NullString `json:"-"` // nullable; unwrapped in ToJSON()
	StreetViewURL    sql.NullString `json:"-"` // nullable; unwrapped in ToJSON()
	RegistryURL      sql.NullString `json:"-"` // nullable; unwrapped in ToJSON()
	AssessorPID      sql.NullString `json:"-"` // VGSI parcel ID; set by second-pass scraper
	RegistryDeepLink sql.NullString `json:"-"` // masslandrecords.com book/page deep link
	RegistryBook     sql.NullInt64  `json:"-"` // numeric book number for dynamic URL building
	RegistryPage     sql.NullInt64  `json:"-"` // numeric page number for dynamic URL building
}

// AuctionJSON is the API response shape. Date is a human-readable string ("Jan 2, 2006")
// or empty string when the auction has no scheduled date.
type AuctionJSON struct {
	Id               int    `json:"id"`
	Address          string `json:"address"`
	City             string `json:"city"`
	State            string `json:"state"`
	Time             string `json:"time"`
	Logo             string `json:"logo"`
	SiteName         string `json:"site_name"`
	Status           string `json:"status"`
	Link             string `json:"link"`
	Date             string `json:"date"`
	Deposit          string `json:"deposit"`
	Lat              string `json:"lat"`
	Lng              string `json:"lng"`
	ZillowURL        string `json:"zillow_url"`
	StreetViewURL    string `json:"street_view_url"`
	RegistryURL      string `json:"registry_url"`
	RegistryDeepLink string `json:"registry_deep_link,omitempty"`
	AssessorPID      string `json:"assessor_pid,omitempty"`
	RegistryBook     int    `json:"registry_book,omitempty"`
	RegistryPage     int    `json:"registry_page,omitempty"`
}

// DueDiligenceItem is one task in the shareable property report checklist.
type DueDiligenceItem struct {
	Task     string `json:"task"`
	Category string `json:"category"`
}

// AuctionReport is the full response for GET /report/{address_slug}.
type AuctionReport struct {
	Auction      AuctionJSON        `json:"auction"`
	Checklist    []DueDiligenceItem `json:"checklist"`
	ShareURL     string             `json:"share_url"`
	AddressSlug  string             `json:"address_slug"`
	AssessorURL  string             `json:"assessor_url,omitempty"` // full VGSI parcel URL
}

// ToJSON converts a scanned AuctionModel into an API-safe AuctionJSON.
func (a AuctionModel) ToJSON() AuctionJSON {
	date := ""
	if a.Date.Valid {
		date = a.Date.Time.Format("Jan 2, 2006")
	}
	return AuctionJSON{
		Id:               a.Id,
		Address:          a.Address,
		City:             a.City,
		State:            a.State,
		Time:             a.Time,
		Logo:             a.Logo,
		SiteName:         a.SiteName,
		Status:           a.Status,
		Link:             a.Link,
		Date:             date,
		Deposit:          a.Deposit,
		Lat:              a.Lat,
		Lng:              a.Lng,
		ZillowURL:        a.ZillowURL.String,
		StreetViewURL:    a.StreetViewURL.String,
		RegistryURL:      a.RegistryURL.String,
		RegistryDeepLink: a.RegistryDeepLink.String,
		AssessorPID:      a.AssessorPID.String,
		RegistryBook:     int(a.RegistryBook.Int64),
		RegistryPage:     int(a.RegistryPage.Int64),
	}
} 