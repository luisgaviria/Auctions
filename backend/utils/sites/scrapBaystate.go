package sites

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
)

type Record struct {
	Date    string `json:"date"`
	Address string `json:"address"`
	City    string `json:"city"`
	State   string `json:"state"`
	Deposit string `json:"deposit"`
	Status  string `json:"status,omitempty"`
	Link    string `json:"link,omitempty"`
	Logo    string `json:"logo,omitempty"`
}

func extractStatus(date string) string {
	parts := strings.Split(date, "'>")
	if len(parts) > 1 {
		statusParts := strings.Split(parts[1], "</")
		if len(statusParts) > 0 {
			return statusParts[0]
		}
	}
	return ""
}

func cleanDate(date string) string {
	parts := strings.Split(date, " <s")
	if len(parts) > 0 {
		date = parts[0]
	}
	date = strings.ReplaceAll(date, "at", "@")
	parsedDate, err := time.Parse("Jan 2, 2006 @ 3:04PM", date) // Adjust format based on the actual date
	if err == nil {
		return parsedDate.Format("2006-01-02")
	}
	return date // Return the original string if parsing fails
}

func ScrapBaystate() []Auction {
	const logo = "/baystate.webp"
	const link = "https://www.baystateauction.com/auctions"
	page := rod.New().MustConnect().MustPage("https://www.baystateauction.com/auctions/state/ma")
	page.MustWaitStable()
	scriptWithScripts := page.MustElement("#main > div.row.main > script")

	jsText := scriptWithScripts.MustText()
	jsText = "[" + strings.TrimSpace(strings.SplitN(jsText, "[", 2)[1])
	jsText = strings.ReplaceAll(jsText, "\n", "")
	jsText = strings.ReplaceAll(jsText, "\t", "")
	jsText = regexp.MustCompile(`\s{2,}`).ReplaceAllString(jsText, " ") // Replace multiple spaces with one
	jsText = strings.Replace(jsText, "},]", "}]", -1)                   // Replace the trailing comma

	// Parse the cleaned string into JSON
	var data []Record
	err := json.Unmarshal([]byte(jsText), &data)

	if err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}

	var filteredData []Auction
	for _, record := range data {
		// Extract status from the date string
		if strings.Contains(record.Date, "IS CANCELLED") || strings.Contains(record.Date, "Postponed") {
			record.Status = extractStatus(record.Date)
		} else {
			record.Status = "On Schedule"
		}

		// Clean the date and reformat it
		record.Date = cleanDate(record.Date)

		// Add additional fields
		record.Link = link
		record.Logo = logo

		// Add to the filtered list

		auction := Auction{
			Url:     record.Link,
			Logo:    record.Logo,
			Street:  record.Address,
			City:    record.City,
			Date:    record.Date,
			Time:    time.Now().String(),
			Status:  record.Status,
			Deposit: record.Deposit,
		}
		filteredData = append(filteredData, auction)
	}

	fmt.Print(filteredData)
	return filteredData
} // add pagination here
