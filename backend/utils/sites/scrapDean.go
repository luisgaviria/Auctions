package sites

import (
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

// deanWeekdayRe strips a leading day-of-week prefix that Dean Associates
// prepends to date strings, e.g. "WEDNESDAY, AUGUST 13, 2025, 3:00 PM".
var deanWeekdayRe = regexp.MustCompile(`(?i)^(?:MONDAY|TUESDAY|WEDNESDAY|THURSDAY|FRIDAY|SATURDAY|SUNDAY),?\s*`)

// deanDateTimeRe extracts the date and time components after any weekday has
// been removed.  Handles both formats the site has used historically:
//
//	"AUGUST 13, 2025, 3:00 PM"  (comma-separated)
//	"Mar 23, 20263:00 PM"       (year runs straight into time, no comma/space)
var deanDateTimeRe = regexp.MustCompile(`(?i)([A-Za-z]+ \d+,?\s*\d{4}),?\s*(\d+:\d+\s*[AP]M)`)

// canParseDeanDate returns true if s can be interpreted as a date by the
// layouts Dean Associates produces.  Title-cases the input first so
// "AUGUST 13, 2025" is accepted by Go's time.Parse.
func canParseDeanDate(s string) bool {
	// strings.Title is deprecated but correct for ASCII month names. //nolint:staticcheck
	normalized := strings.Title(strings.ToLower(strings.TrimSpace(s)))
	for _, layout := range []string{"January 2, 2006", "Jan 2, 2006", "2006-01-02"} {
		if _, err := time.Parse(layout, normalized); err == nil {
			return true
		}
	}
	return false
}

func ScrapDean() []Auction {
	url := "https://deanassociatesinc.com/auctions/"
	c := colly.NewCollector()
	priceRegex := regexp.MustCompile(`\$[\d,]+`)

	auctions := make([]Auction, 0)

	c.OnHTML("html", func(e *colly.HTMLElement) {
		e.ForEach("#gatsby-focus-wrapper > main > section > div:nth-child(1) > div > table > tbody", func(i int, tbody *colly.HTMLElement) {
			tbody.ForEach("tr", func(_ int, tr *colly.HTMLElement) {
				auction := Auction{}
				skipRow := false

				tr.ForEach("td", func(i int, td *colly.HTMLElement) {
					if skipRow {
						return
					}
					switch i {
					case 0:
						raw := strings.TrimRight(strings.TrimSpace(td.Text), ", ")

						// Step 1: uppercase for uniform matching.
						upper := strings.ToUpper(raw)

						// Step 2: strip leading weekday name ("WEDNESDAY, " → "").
						stripped := strings.TrimSpace(deanWeekdayRe.ReplaceAllString(upper, ""))
						stripped = strings.TrimRight(stripped, ", ")

						// Step 3: extract date + time via regex.
						m := deanDateTimeRe.FindStringSubmatch(stripped)
						if len(m) != 3 {
							log.Printf("[dean] date regex no match for %q (raw: %q) — skipping row", stripped, raw)
							skipRow = true
							return
						}

						dateStr := strings.TrimRight(strings.TrimSpace(m[1]), ", ")

						// Step 4: validate the date is parseable before accepting it.
						if !canParseDeanDate(dateStr) {
							log.Printf("[dean] unparseable date %q (raw: %q) — skipping row", dateStr, raw)
							skipRow = true
							return
						}

						auction.Date = dateStr
						auction.Time = strings.TrimSpace(m[2])

					case 2:
						auction.Street = td.Text
					case 3:
						auction.Deposit = priceRegex.FindString(td.Text)
					}
				})

				if !skipRow {
					auction.Logo = "/dean.webp"
					auction.Status = "Active"
					auction.City = "Massachusetts"
					auctions = append(auctions, auction)
				}
			})
		})
	})

	c.Visit(url)
	return auctions
}
