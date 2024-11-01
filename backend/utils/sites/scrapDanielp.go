package sites

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

func ScrapDanielP() []Auction {
	logo := "/danielp.webp"
	url := "https://www.re-auctions.com/Auction-Schedule/PropertyAgentName/-1/sortBy/cf11"

	c := colly.NewCollector()

	auctions := make([]Auction, 0)

	c.OnHTML("html", func(e *colly.HTMLElement) {
		e.ForEach("#dnn_ctr376_ModuleContent > div", func(i int, divElement *colly.HTMLElement) {
			if i != 0 {
				// fmt.Println(i)
				imgUrl := divElement.DOM.Children().Find("img")
				// need to get along with other sutff
				attr, exists := imgUrl.Attr("src")
				var image string //
				if exists {
					image = "https://www.re-auctions.com" + attr
					_ = image
				}

				address := divElement.DOM.Children().Find("a").Text()

				var propertyType string
				var status string
				var deposit string
				divElement.DOM.Children().Find("li").Each(func(j int, li *goquery.Selection) {
					switch j {
					case 0:
						{
							propertyType = strings.TrimSpace(strings.Split(li.Text(), ":")[1])
							_ = propertyType
						}
					case 1:
						{
							status = strings.TrimSpace(strings.Split(li.Text(), ":")[1])
							_ = status
						}
					case 2:
						{
							deposit = strings.TrimSpace(strings.Split(li.Text(), ":")[1])
							_ = deposit
						}
					}
				})
				if len(strings.TrimSpace(divElement.DOM.Children().Find(".Postponed").Text())) > 0 {
					postponed := strings.TrimSpace(divElement.DOM.Children().Find(".Postponed").Text())
					_ = postponed

					date := strings.TrimSpace(strings.Split(divElement.DOM.Children().Find("b").Text(), "-")[0])
					time := strings.TrimSpace(strings.Split(divElement.DOM.Children().Find("b").Text(), "-")[1])
					auctions = append(auctions, Auction{
						Date:    date,
						Time:    time,
						Street:  address,
						City:    "",
						Status:  status,
						Logo:    logo,
						Url:     url,
						Deposit: deposit,
					})

				} else {
					var date string
					var time string
					bs := divElement.DOM.Find("b")
					bs.Each(func(i int, b *goquery.Selection) {
						if i == 1 {
							date = strings.TrimSpace(strings.Split(b.Text(), "-")[0])
							time = strings.TrimSpace(strings.Split(b.Text(), "-")[1])
						}
					})
					auctions = append(auctions, Auction{
						Date:    date,
						Time:    time,
						Street:  address,
						City:    "",
						Status:  status,
						Logo:    logo,
						Url:     url,
						Deposit: deposit,
					})
				}
			}

		})
	})

	c.Visit(url)
	return auctions
}
