package sites

import (
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
)

// dateRe and timeRe extract date/time from <b> text that may also contain
// a status prefix (e.g. "Cancelled\n\n3/23/2026" or "3/25/2026 - 10:00 AM").
var (
	dpDateRe = regexp.MustCompile(`(\d{1,2}/\d{1,2}/\d{4})`)
	dpTimeRe = regexp.MustCompile(`(\d{1,2}:\d{2}\s*[APap][Mm])`)
)

// normalizeDPDate converts M/D/YYYY or MM/DD/YYYY to "2006-01-02" (ISO).
// Returns the input unchanged if it cannot be parsed.
func normalizeDPDate(s string) string {
	for _, layout := range []string{"01/02/2006", "1/2/2006", "1/02/2006", "01/2/2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return s
}

func ScrapDanielP() []Auction {
	logo := "/danielp.webp"
	url := "https://www.re-auctions.com/Auction-Schedule/PropertyAgentName/-1/sortBy/cf11"

	c := colly.NewCollector()

	auctions := make([]Auction, 0)

	c.OnHTML("html", func(e *colly.HTMLElement) {
		e.ForEach("#dnn_ctr376_ModuleContent > div", func(i int, divElement *colly.HTMLElement) {
			if i != 0 {
				// Clone the anchor and strip child elements (e.g. <b> containing
				// the date) before reading text.  Without this, .Text() returns
				// the full concatenated string "90 SUFFOLK ROAD, NEWTON , MAApr 7, 2026".
				aClone := divElement.DOM.Children().Find("a").Clone()
				aClone.Find("b, span, div").Remove()
				address := strings.TrimSpace(aClone.Text())

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
							date = normalizeDPDate(m)
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
						date = normalizeDPDate(m)
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
