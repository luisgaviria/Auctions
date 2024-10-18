package sites

import (
	"github.com/gocolly/colly/v2"
)

type Auction struct {
	Date string `selector:"#comp-kykclvym > div > div > table > tbody"`
}

func ScrapHarvard() {
	url := "https://www.harvardauctioneers.com/"
	c := colly.NewCollector()

	auction := &Auction{}

	c.OnHTML("html", func(e *colly.HTMLElement) {
		e.Unmarshal(auction)
		e.ForEach("tr", func(i int, trElement *colly.HTMLElement) {
			trElement.ForEach("td", func(j int, tdElement *colly.HTMLElement) {
				switch j {
				case 0: // date
					{

					}
				case 1: // time
					{

					}
				case 2: // street
					{

					}
				case 3: // town?
					{

					}
				case 4: // deposit
					{

					}
				case 5: // comment?
					{

					}
				}
			})
		})
	})

	c.Visit(url)
}
