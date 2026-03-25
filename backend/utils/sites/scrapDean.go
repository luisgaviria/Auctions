package sites

import (
	"regexp"
	"strings"

	"github.com/gocolly/colly/v2"
)

func ScrapDean() []Auction {
	url := "https://deanassociatesinc.com/auctions/"
	c := colly.NewCollector()
	priceRegex := regexp.MustCompile(`\$[\d,]+`)
	// Match date+time regardless of whether the site uses 3-letter or full month
	// names, and whether a weekday prefix (e.g. "WEDNESDAY, ") is present.
	// Handles both old "Mar 23, 20263:00 PM" and new "WEDNESDAY, AUGUST 13, 2025,3:00 PM".
	dateTimeRe := regexp.MustCompile(`(?i)(?:\w+,\s*)?([A-Za-z]+ \d+,?\s*\d{4}),?\s*(\d+:\d+\s*[AP]M)`)

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
							raw := strings.TrimRight(strings.TrimSpace(td.Text), ",")
							if m := dateTimeRe.FindStringSubmatch(raw); len(m) == 3 {
								auction.Date = strings.TrimRight(strings.TrimSpace(m[1]), ",")
								auction.Time = strings.TrimSpace(m[2])
							} else {
								auction.Date = raw
							}
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
