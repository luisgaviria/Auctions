package sites

import (
	"fmt"
	"strings"
	"time"

	"github.com/gocolly/colly"
)

func ScrapJake() []Auction {
	url := "https://www.jkauctioneers.com/list1.htm"
	var data []Auction
	logo := "/jake.webp"

	c := colly.NewCollector()

	c.OnHTML("body font p[align='left']", func(e *colly.HTMLElement) {
		fonts := e.DOM.Find("font")
		href, exists := e.DOM.Find("a").Attr("href")
		if !exists {
			return
		}
		href = "https://www.jkauctioneers.com/" + href
		address := strings.Split(fonts.Eq(0).Text(), "\n")[1]
		date := strings.TrimSpace(fonts.Eq(1).Text())
		status := strings.TrimSpace(fonts.Eq(2).Text())
		if status == "" {
			status = "On Schedule"
		}

		c2 := colly.NewCollector()
		c2.OnHTML("body", func(e2 *colly.HTMLElement) {
			termsText := strings.Split(e2.Text, "TERMS")[1]
			price := strings.Split(termsText, "Dollars")[0]
			deposit := strings.Split(strings.TrimSpace(strings.Split(price, "(")[1]), ")")[0]
			dateParsed, err := time.Parse("Monday, January 2, 2006", strings.TrimSpace(strings.Split(date, "AT")[0]))
			if err != nil {
				dateParsed, err = time.Parse("Monday January 2, 2006", strings.TrimSpace(strings.Split(date, "AT")[0]))
				if err != nil {
					fmt.Println(err)
					return
				}
			}

			data = append(data, Auction{
				Date:    dateParsed.Format("01/02/2006"),
				Time:    "", // Time is not parsed in the original code
				Street:  address,
				City:    "", // City is not parsed in the original code
				Deposit: deposit,
				Status:  status,
				Logo:    logo,
				Url:     href,
			})
		})
		c2.Visit(href)
	})

	err := c.Visit(url)
	if err != nil {
		return nil
	}

	if len(data) > 0 {
		data = data[1:]
	}

	return data
}
