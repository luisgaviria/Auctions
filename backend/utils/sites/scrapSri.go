package sites

import (
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

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
		// Index 7 = status ("On Schedule", "Cancelled", etc.)
		// Index 8 = deposit amount — confirmed from live page inspection
		status := strings.TrimSpace(tds.Eq(7).Text())
		deposit := strings.TrimSpace(tds.Eq(8).Text())

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
