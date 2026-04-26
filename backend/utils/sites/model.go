package sites

import "fmt"

type Auction struct {
	Date        string
	Time        string
	Street      string
	City        string
	Deposit     string
	Status      string
	Logo        string
	Url         string
	SiteName    string // identifies the source scraper, e.g. "baystate"
	ZillowURL        string
	StreetViewURL    string
	RegistryURL      string
	CitySlug         string
	CountySlug       string
	AddressSlug      string
	LegalDescription string // raw Book/Page citation from the notice, e.g. "B13812/P506"
	RegistryDeepLink string // computed masslandrecords.com deep link (book+page pre-filled)
}

func (auction *Auction) Print() {
	fmt.Printf(`
	Date: %s
	Time: %s
	Street: %s
	City: %s
	Deposit: %s
	Status: %s
	`, auction.Date, auction.Time, auction.Street, auction.City, auction.Deposit, auction.Status)
}
