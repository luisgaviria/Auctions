package sites

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

// dateRe and timeRe extract date/time from <b> text that may also contain
// a status prefix (e.g. "Cancelled\n\n3/23/2026" or "3/25/2026 - 10:00 AM").
var (
	dpDateRe = regexp.MustCompile(`(\d{1,2}/\d{1,2}/\d{4})`)
	dpTimeRe = regexp.MustCompile(`(\d{1,2}:\d{2}\s*[APap][Mm])`)
)

func ScrapDanielP() []Auction {
	logo := "/danielp.webp"
	url := "https://www.re-auctions.com/Auction-Schedule/PropertyAgentName/-1/sortBy/cf11"

	c := colly.NewCollector()

	auctions := make([]Auction, 0)

	c.OnHTML("html", func(e *colly.HTMLElement) {
		e.ForEach("#dnn_ctr376_ModuleContent > div", func(i int, divElement *colly.HTMLElement) {
			if i != 0 {
				address := divElement.DOM.Children().Find("a").Text()

				var propertyType string
				var status string
				var deposit string
				divElement.DOM.Children().Find("li").Each(func(j int, li *goquery.Selection) {
					parts := strings.SplitN(li.Text(), ":", 2)
					if len(parts) < 2 {
						return
					}
					val := strings.TrimSpace(parts[1])
					switch j {
					case 0:
						propertyType = val
						_ = propertyType
					case 1:
						status = val
					case 2:
						deposit = val
					}
				})

				// Extract date and time from the relevant <b> tag.
				// The text may contain a status prefix and/or newlines before the date.
				var date, time string
				divElement.DOM.Find("b").Each(func(idx int, b *goquery.Selection) {
					if idx == 1 {
						raw := b.Text()
						if m := dpDateRe.FindString(raw); m != "" {
							date = m
						}
						if m := dpTimeRe.FindString(raw); m != "" {
							time = strings.TrimSpace(m)
						}
					}
				})

				// Postponed path: override date/time if the .Postponed selector has data.
				if postponed := strings.TrimSpace(divElement.DOM.Children().Find(".Postponed").Text()); postponed != "" {
					raw := divElement.DOM.Children().Find("b").Text()
					if m := dpDateRe.FindString(raw); m != "" {
						date = m
					}
					if m := dpTimeRe.FindString(raw); m != "" {
						time = strings.TrimSpace(m)
					}
				}

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
		})
	})

	c.Visit(url)
	return auctions
}
