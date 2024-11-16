package sites

import (
	"strings"

	"github.com/gocolly/colly/v2"
)

func ScrapApg() []Auction { // write additional support for date formats
	logo := "/apg.webp"
	url := "https://apg-online.com/auction-schedule/"

	c := colly.NewCollector()

	auctions := make([]Auction, 0)

	c.OnHTML("html", func(e *colly.HTMLElement) {
		e.ForEach("#content > div.columns.three.properties > div", func(i int, divElement *colly.HTMLElement) {
			ddTexts := divElement.ChildTexts("dd")
			if len(ddTexts) > 1 {
				auctions = append(auctions, Auction{
					Status:  ddTexts[0],
					Date:    ddTexts[1],
					Street:  strings.ReplaceAll(ddTexts[2], ".,", ","),
					Deposit: ddTexts[len(ddTexts)-1],
					Url:     url,
					Logo:    logo,
				})
			}
		})
	})

	c.Visit(url)

	return auctions
}
