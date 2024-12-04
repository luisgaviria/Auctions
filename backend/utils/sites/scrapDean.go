package sites

import (
	"regexp"

	"github.com/gocolly/colly/v2"
)

func ScrapDean() []Auction {
	url := "https://deanassociatesinc.com/auctions/"
	c := colly.NewCollector()
	priceRegex := regexp.MustCompile(`\$[\d,]+`)

	auctions := make([]Auction, 0)

	c.OnHTML("html", func(e *colly.HTMLElement) {
		e.ForEach("#gatsby-focus-wrapper > main > section > div:nth-child(1) > div > table > tbody", func(i int, tbody *colly.HTMLElement) {
			tbody.ForEach("tr", func(_ int, tr *colly.HTMLElement) {
				auction := Auction{}
				tr.ForEach("td", func(i int, td *colly.HTMLElement) {
					// fmt.Println(i, ": ", td.Text)
					switch i {
					case 0:
						{
							auction.Date = td.Text
						}
					case 2:
						{
							auction.Street = td.Text
						}
					case 3:
						{
							auction.Deposit = priceRegex.FindString(td.Text)
						}
					}
				})
				auction.Logo = "/dean.webp"
				auction.Status = "Active"
				auction.City = "Masachussets"
				auctions = append(auctions, auction)
			})
		})
	})

	c.Visit(url)

	return auctions
}
