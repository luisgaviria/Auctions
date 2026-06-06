package models

import (
	"database/sql"
	"strconv"
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
	Deposit          sql.NullInt64  `json:"-"` // scraped integer deposit; format to string in ToJSON()
	Lat              sql.NullFloat64`json:"-"` // stored as double precision
	Lng              sql.NullFloat64`json:"-"` // stored as double precision
	Createdat        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updated_at"`
	LastSeen         sql.NullTime   `json:"-"` // internal sync column; not exposed in API responses
	ZillowURL        sql.NullString `json:"-"` // nullable; unwrapped in ToJSON()
	StreetViewURL    sql.NullString `json:"-"` // nullable; unwrapped in ToJSON()
	RegistryURL      sql.NullString `json:"-"` // nullable; unwrapped in ToJSON()
	AssessorPID      sql.NullString `json:"-"` // legacy column; retained for schema compatibility
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
	AssessorURL  string             `json:"assessor_url,omitempty"`
}

// formatDeposit takes a raw int and formats it to $X,XXX.
func formatDeposit(deposit int) string {
	if deposit == 0 {
		return ""
	}
	// simple comma formatting
	s := strconv.Itoa(deposit)
	if len(s) <= 3 {
		return "$" + s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return "$" + string(result)
}

// formatFloat takes a float and converts it to string.
func formatFloat(f float64) string {
	if f == 0 {
		return "0"
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// ToJSON converts a scanned AuctionModel into an API-safe AuctionJSON.
func (a AuctionModel) ToJSON() AuctionJSON {
	date := ""
	if a.Date.Valid {
		date = a.Date.Time.Format("Jan 2, 2006")
	}
	depositStr := ""
	if a.Deposit.Valid {
		depositStr = formatDeposit(int(a.Deposit.Int64))
	}
	
	latStr, lngStr := "0", "0"
	if a.Lat.Valid { latStr = formatFloat(a.Lat.Float64) }
	if a.Lng.Valid { lngStr = formatFloat(a.Lng.Float64) }

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
		Deposit:          depositStr,
		Lat:              latStr,
		Lng:              lngStr,
		ZillowURL:        a.ZillowURL.String,
		StreetViewURL:    a.StreetViewURL.String,
		RegistryURL:      a.RegistryURL.String,
		RegistryDeepLink: a.RegistryDeepLink.String,
		AssessorPID:      a.AssessorPID.String,
		RegistryBook:     int(a.RegistryBook.Int64),
		RegistryPage:     int(a.RegistryPage.Int64),
	}
} 