package sites

import (
	"github.com/gocolly/colly/v2"
)

func ScrapDanielP() []Auction {
	// logo := "/danielp.webp"
	url := "https://www.re-auctions.com/Auction-Schedule/PropertyAgentName/-1/sortBy/cf11"

	c := colly.NewCollector()

	// auctions := make([]Auction, 0)

	c.OnHTML("html", func(e *colly.HTMLElement) {
		e.ForEach("#dnn_ctr376_ModuleContent > div", func(i int, divElement *colly.HTMLElement) {
			// fmt.Println(i)
			image := divElement.DOM.Children().Find("img")
			// need to get along with other sutff
			attr, exists := image.Attr("src")
			if exists {

			}
		})
	})

	c.Visit(url)
	return []Auction{}
}
