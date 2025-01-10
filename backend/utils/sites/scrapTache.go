package sites

import (
	"fmt"
	"log"
	"strings"

	"github.com/gocolly/colly/v2"
)

func ScrapTache() []Auction {
	url := "https://docs.google.com/spreadsheets/u/1/d/14nrcaKBhCA61FcnBwU6EbiDbRQtOP-gQVxJVvxg5_o0/pubhtml/sheet?headers=false&gid=0"
	c := colly.NewCollector(
		colly.MaxDepth(1),
	)

	var data []Auction

	logo := "https://auction-site-ma.herokuapp.com/auction_photos/tache.webp"

	c.OnHTML("body > div > div > div > table > tbody", func(e *colly.HTMLElement) {
		// Get all the tr elements inside the tbody
		trs := e.DOM.Children().Nodes

		// Remove the first four rows
		if len(trs) <= 4 {
			return
		}

		for i := 0; i < 4; i++ {
			e.DOM.Children().First().Remove()
		}

		trs = e.DOM.Children().Nodes

		e.ForEach("tr", func(index int, el *colly.HTMLElement) {

			tds := el.DOM.Find("td")
			if tds.Length() < 8 {
				return
			}

			date := strings.Replace(tds.Eq(0).Text(), "/21", "/2021", 1)
			time := tds.Eq(1).Text()
			address := tds.Eq(2).Text()
			city := tds.Eq(3).Text()
			state := tds.Eq(4).Text()
			zip := tds.Eq(5).Text()
			status := tds.Eq(6).Text()
			deposit := tds.Eq(7).Text()

			fmt.Println(zip)

			fullAddress := address + " " + city + ", " + state

			if !strings.Contains(status, "PP") {
				data = append(data, Auction{
					Logo:    logo,
					Date:    date,
					Time:    time,
					Street:  fullAddress,
					City:    city,
					Deposit: deposit,
					Status:  status,
					Url:     url,
				})
			}
		})
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	err := c.Visit(url)
	if err != nil {
		fmt.Println(err)
	}

	return data
}
