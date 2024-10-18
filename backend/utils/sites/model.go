package sites

import "fmt"

type Auction struct {
	Date    string
	Time    string
	Street  string
	City    string
	Deposit string
	Status  string
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
