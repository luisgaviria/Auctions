package sites

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// sriPostponedRe captures MM/DD/YYYY from statuses like
// "POSTPONED TO 05/20/2026@12:00PM".
var sriPostponedRe = regexp.MustCompile(`(?i)POSTPONED\s+TO\s+(\d{2}/\d{2}/\d{4})`)

// sriRegistryRe matches deed-registry citations embedded in SRI address cells,
// e.g. ", B13812/P506" or "B1133/P82".
var sriRegistryRe = regexp.MustCompile(`(?i),?\s*B\d+/P\d+\S*`)

func ScrapSri() []Auction {
	url := "http://www.auctionsri.com/scripts/auctions.asp?category=R"
	resp, err := http.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil
	}

	logo := "/ri.webp"
	var data []Auction

	doc.Find("body > center > center > font > b > table:nth-child(5) > tbody > tr").Each(func(i int, s *goquery.Selection) {
		tds := s.Find("td")
		date := strings.TrimSpace(tds.Eq(0).Text())
		time := strings.TrimSpace(tds.Eq(2).Text())
		address := strings.TrimSpace(tds.Eq(4).Text())
		city := strings.TrimSpace(tds.Eq(5).Text())
		state := strings.TrimSpace(tds.Eq(6).Text())
		// Index 7 = status ("On Schedule", "Cancelled", "Postponed to …", etc.)
		// Index 8 = deposit amount — confirmed from live page inspection
		status := strings.TrimSpace(tds.Eq(7).Text())
		deposit := strings.TrimSpace(tds.Eq(8).Text())

		// If the auction was postponed, the rescheduled date is embedded in the
		// status string ("POSTPONED TO 05/20/2026@12:00PM").  Overwrite the
		// scraped date so the row sorts to the correct future position.
		if m := sriPostponedRe.FindStringSubmatch(status); len(m) == 2 {
			date = m[1] // MM/DD/YYYY
		}

		if strings.Contains(address, "FEATURED:") {
			address = strings.ReplaceAll(address, "FEATURED:", "")
			address = strings.ReplaceAll(address, "Real Estate", "")
			address = strings.TrimSpace(address)
		}

		// Capture deed-registry citation before stripping it from the address.
		// "B13812/P506" becomes the LegalDescription for the deep-link builder.
		var legalDesc string
		if m := sriRegistryRe.FindString(address); m != "" {
			legalDesc = strings.TrimSpace(strings.TrimLeft(m, ", "))
		}
		// Remove deed-registry citations from the street address string.
		address = strings.TrimRight(strings.TrimSpace(sriRegistryRe.ReplaceAllString(address, "")), ",")

		// If the city name appears inside the address cell (some rows embed the
		// full "Street, City, State" in one column), truncate at the first comma
		// after the city so the state abbreviation is dropped.
		if city != "" {
			if idx := strings.Index(address, city); idx != -1 {
				rest := address[idx+len(city):]
				if ci := strings.Index(rest, ","); ci != -1 {
					address = strings.TrimSpace(address[:idx+len(city)+ci])
				}
			}
		}

		_ = state // state is excluded from the street field

		data = append(data, Auction{
			Logo:             logo,
			Date:             date,
			Time:             time,
			Street:           address,
			City:             city,
			Status:           status,
			Deposit:          deposit,
			Url:              url,
			LegalDescription: legalDesc,
		})
	})

	filteredData := []Auction{}
	for _, record := range data {
		if record.Date != "Date" {
			filteredData = append(filteredData, record)
		}
	}

	return filteredData
}
