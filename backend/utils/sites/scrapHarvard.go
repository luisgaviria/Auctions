package sites

import (
	"strings"

	"github.com/gocolly/colly/v2"
)

func ScrapHarvard() []Auction {
	url := "https://www.harvardauctioneers.com/"
	c := colly.NewCollector()

	auctions := make([]Auction, 0)

	c.OnHTML("html", func(e *colly.HTMLElement) {
		e.ForEach("#comp-kykclvym > div > div > table > tbody", func(_ int, tbody *colly.HTMLElement) {
			tbody.ForEach("tr", func(_ int, trElement *colly.HTMLElement) {
				auction := Auction{}
				trElement.ForEach("td", func(i int, tdElement *colly.HTMLElement) {
					switch i {
					case 0: // date
						{
							auction.Date = strings.Trim(tdElement.Text, "\t")
							auction.Date = strings.Trim(auction.Date, "\n")
							auction.Date = strings.ReplaceAll(auction.Date, "/21", "/2021")
						}
					case 1: // time
						{
							auction.Time = strings.Trim(tdElement.Text, "\t")
							auction.Time = strings.Trim(auction.Time, "\n")
						}
					case 2: // street
						{
							auction.Street = strings.ReplaceAll(tdElement.Text, "\n", "")
							auction.Street = strings.ReplaceAll(auction.Street, "\t", "")
							auction.Street = strings.ReplaceAll(auction.Street, "  +", " ")
							auction.Street = strings.Split(auction.Street, ",")[0]
						}
					case 3: // city?
						{
							auction.City = strings.Trim(tdElement.Text, "\t")
							auction.City = strings.Trim(auction.City, "\n")
						}
					case 4: // deposit
						{
							auction.Deposit = strings.Trim(tdElement.Text, " ")
						}
					case 5: // comment?
						{
							if len(tdElement.Text) > 0 {
								auction.Status = "Sold"
							} else {
								auction.Status = "Available"
							}
						}
					}
				})
				auction.Logo = `/baystate.webp`
				auction.Url = `https://www.harvardauctioneers.com/`
				auction.Print()
				auctions = append(auctions, auction)
			})
		})
	})

	c.Visit(url)

	return auctions
}
