package sites

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// Record is the JSON shape embedded in Baystate's page <script> tag.
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

// extractStatus pulls the status text out of Baystate's inline HTML markup
// that appears inside date strings for cancelled/postponed auctions.
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

// cleanDate strips any inline HTML status markup and reformats the date to YYYY-MM-DD.
func cleanDate(date string) string {
	parts := strings.Split(date, " <s")
	if len(parts) > 0 {
		date = parts[0]
	}
	date = strings.ReplaceAll(date, "at", "@")
	parsedDate, err := time.Parse("Jan 2, 2006 @ 3:04PM", date)
	if err == nil {
		return parsedDate.Format("2006-01-02")
	}
	return date
}

// ScrapBaystate fetches baystateauction.com via Cloudflare Browser Rendering,
// extracts the embedded JSON auction data from the page <script> tag, and
// returns a slice of Auction structs.
func ScrapBaystate(ctx context.Context) ([]Auction, error) {
	const logo = "/baystate.webp"
	const link = "https://www.baystateauction.com/auctions"
	const targetURL = "https://www.baystateauction.com/auctions/state/ma"

	html, err := CFetch(ctx, targetURL)
	if err != nil {
		return nil, fmt.Errorf("baystate: cfetch: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("baystate: parse html: %w", err)
	}

	scriptText := doc.Find("#main > div.row.main > script").Text()
	if scriptText == "" {
		return nil, fmt.Errorf("baystate: script tag not found — selector may have changed")
	}

	// The script tag contains:  var auctions = [ {...}, {...} ];
	// We split at the first "[" to isolate the JSON array.
	parts := strings.SplitN(scriptText, "[", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("baystate: could not find JSON array in script tag")
	}
	jsText := "[" + strings.TrimSpace(parts[1])
	jsText = strings.ReplaceAll(jsText, "\n", "")
	jsText = strings.ReplaceAll(jsText, "\t", "")
	jsText = regexp.MustCompile(`\s{2,}`).ReplaceAllString(jsText, " ")
	jsText = strings.ReplaceAll(jsText, "},]", "}]") // strip trailing comma

	var data []Record
	if err := json.Unmarshal([]byte(jsText), &data); err != nil {
		// Log and return error instead of log.Fatalf, which would kill the process.
		return nil, fmt.Errorf("baystate: parse json: %w", err)
	}

	auctions := make([]Auction, 0, len(data))
	for _, record := range data {
		if strings.Contains(record.Date, "IS CANCELLED") || strings.Contains(record.Date, "Postponed") {
			record.Status = extractStatus(record.Date)
		} else {
			record.Status = "On Schedule"
		}
		record.Date = cleanDate(record.Date)

		auctions = append(auctions, Auction{
			SiteName: "baystate",
			Url:      link,
			Logo:     logo,
			Street:   record.Address,
			City:     record.City,
			Date:     record.Date,
			Time:     time.Now().Format("15:04"),
			Status:   record.Status,
			Deposit:  record.Deposit,
		})
	}

	return auctions, nil
}
