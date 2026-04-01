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

		address = address + ", " + city + ", " + state

		data = append(data, Auction{
			Logo:    logo,
			Date:    date,
			Time:    time,
			Street:  address,
			Status:  status,
			Deposit: deposit,
			Url:     url,
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
