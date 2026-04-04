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

// deanDateLayouts mirrors the full layout list in utils/scrap.go so that any
// date format Dean Associates (or a future site) uses is handled consistently.
// Keep this in sync with the dateLayouts var in the parent package.
var deanDateLayouts = []string{
	"2006-01-02",      // ISO-8601
	"Jan 2, 2006",     // short month
	"January 2, 2006", // long month (most common Dean format)
	"01/02/2006",      // zero-padded MM/DD/YYYY
	"1/2/2006",        // single-digit M/D/YYYY
	"1/02/2006",       // mixed
	"01/2/2006",       // mixed
}

// normalizeDeanDate parses a Dean Associates date string (which may be all-caps
// and/or have trailing commas, e.g. "AUGUST 13, 2025,") and returns an
// ISO-8601 "2006-01-02" string.  Returns "" if the input cannot be parsed so
// the caller treats it as a signal to skip the row.
func normalizeDeanDate(s string) string {
	if s == "" {
		return ""
	}
	// Title-case so "AUGUST 13, 2025" → "August 13, 2025" for time.Parse.
	// strings.Title is deprecated but correct for ASCII month names. //nolint:staticcheck
	normalized := strings.Title(strings.ToLower(strings.TrimSpace(s)))
	for _, layout := range deanDateLayouts {
		if t, err := time.Parse(layout, normalized); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return ""
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

						// Step 4: normalise to ISO-8601; skip the row if parsing fails.
						isoDate := normalizeDeanDate(dateStr)
						if isoDate == "" {
							log.Printf("[dean] unparseable date %q (raw: %q) — skipping row", dateStr, raw)
							skipRow = true
							return
						}

						auction.Date = isoDate
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
