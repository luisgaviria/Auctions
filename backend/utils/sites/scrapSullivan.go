package sites

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func ScrapSullivan() []Auction {
	siteUrl := "https://sullivan-auctioneers.com/massachusetts/"
	auctions := []Auction{}

	time.Sleep(1 * time.Second)

	res, err := http.Get(siteUrl)
	if err != nil {
		return nil
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil
	}

	doc.Find("#table-view tbody tr").Each(func(i int, s *goquery.Selection) {
		auction := Auction{}

		// Date and Time (combined in one <td>, need to split)
		dateAndTime := s.Find("td:nth-child(1) a").Text()
		auction.Date, auction.Time = parseDateAndTimeSullivan(dateAndTime)

		// Status
		auction.Status = strings.TrimSpace(s.Find("td:nth-child(2) span").Text())

		// Street
		auction.Street = strings.TrimSpace(s.Find("td:nth-child(3)").Text())

		// City, ST  (need to clean)
		cityState := strings.TrimSpace(s.Find("td:nth-child(4)").Text())
		auction.City = cleanCity(cityState) // Remove ", MA"

		// URL
		if url, exists := s.Find("td:nth-child(1) a").Attr("href"); exists {
			auction.Url = siteUrl + strings.TrimSpace(url)
		}

		auction.Deposit = "$5,000" // deposit set to default 5,000$

		auction.Logo = "/sullivan.webp"

		auctions = append(auctions, auction)
	})

	return auctions
}

func parseDateAndTimeSullivan(dateTimeStr string) (string, string) {
	// Regular expression to match "Wed. Feb. 12, 2025 at 12 pm"  or with different month, day and time
	re := regexp.MustCompile(`(\w{3}\.?\s+\w{3}\.?\s+\d{1,2},\s+\d{4})\s+at\s+(\d{1,2}\s*[ap]m)`)
	match := re.FindStringSubmatch(dateTimeStr)

	//For the postponed cases
	re2 := regexp.MustCompile(`(\w{3}\.?\s+\w{3}\.?\s+\d{1,2})\s+at\s+(\d{1,2}\s*[ap]m)`)
	match2 := re2.FindStringSubmatch(dateTimeStr)

	if len(match) == 3 {
		dateStr := strings.TrimSpace(match[1])
		timeStr := strings.TrimSpace(match[2])
		parsedDate, err := time.Parse("Mon. Jan. 2, 2006", dateStr)
		if err != nil {
			return "", timeStr
		}
		formattedDate := parsedDate.Format("2006-01-02")
		return formattedDate, timeStr
	} else if len(match2) == 3 {
		dateStr := strings.TrimSpace(match2[1])
		timeStr := strings.TrimSpace(match2[2])
		parsedDate, err := time.Parse("Mon. Jan. 2", dateStr)
		if err != nil {
			return "", timeStr
		}
		formattedDate := parsedDate.Format("2006-01-02")
		return formattedDate, timeStr
	}
	return "", "" // Return empty if no match
}

// cleanCity removes the ", MA" suffix (or any ", [State]")
func cleanCity(cityState string) string {
	parts := strings.Split(cityState, ",")
	if len(parts) > 0 {
		return strings.TrimSpace(parts[0])
	}
	return cityState // Return original if no comma
}
