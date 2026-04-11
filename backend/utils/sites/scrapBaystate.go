package sites

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

// resolveStatus returns the canonical status for a Baystate record.
// It checks two sources in order:
//  1. The JSON "status" field (Baystate sometimes populates this directly).
//  2. The "date" field, which may contain embedded HTML like
//     <span class='...'>CANCELLED</span> appended after the date string.
//
// Both checks are case-insensitive, so CANCELLED / Cancelled / cancelled all
// resolve to "Cancelled".
func resolveStatus(jsonStatus, dateField string) string {
	for _, s := range []string{jsonStatus, dateField} {
		lower := strings.ToLower(s)
		if strings.Contains(lower, "cancel") {
			return "Cancelled"
		}
		if strings.Contains(lower, "postpone") {
			return "Postponed"
		}
	}
	return "On Schedule"
}

// parseBaystateDateTime extracts date (YYYY-MM-DD) and time (e.g. "10:00 AM")
// from a raw Baystate date string that may contain trailing HTML status markup.
// Example input: "Thursday, April 09, 2026 @ 10:00 am <span ...>CANCELLED</span>"
func parseBaystateDateTime(raw string) (date, timeStr string) {
	// Drop everything from the first HTML tag onward.
	if idx := strings.Index(raw, "<"); idx != -1 {
		raw = raw[:idx]
	}
	raw = strings.TrimSpace(raw)

	// Normalise lowercase am/pm → AM/PM (Go's time layout is case-sensitive).
	switch {
	case strings.HasSuffix(raw, " am"):
		raw = raw[:len(raw)-3] + " AM"
	case strings.HasSuffix(raw, " pm"):
		raw = raw[:len(raw)-3] + " PM"
	}

	// Primary format: "Thursday, April 09, 2026 @ 10:00 AM"
	if t, err := time.Parse("Monday, January 02, 2006 @ 3:04 PM", raw); err == nil {
		return t.Format("2006-01-02"), t.Format("3:04 PM")
	}

	// Fallback: date only, no time component.
	if idx := strings.Index(raw, " @ "); idx != -1 {
		if t, err := time.Parse("Monday, January 02, 2006", raw[:idx]); err == nil {
			return t.Format("2006-01-02"), ""
		}
	}

	return raw, ""
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
		status := resolveStatus(record.Status, record.Date)

		log.Printf("[baystate] parsing addr=%q raw_status=%q raw_date=%q → resolved=%q",
			record.Address, record.Status, record.Date, status)

		// Skip cancelled listings — they're over and clutter the feed.
		if status == "Cancelled" {
			log.Printf("[baystate] skipping cancelled: %q", record.Address)
			continue
		}

		parsedDate, parsedTime := parseBaystateDateTime(record.Date)

		auctions = append(auctions, Auction{
			SiteName: "baystate",
			Url:      link,
			Logo:     logo,
			Street:   record.Address,
			City:     record.City,
			Date:     parsedDate,
			Time:     parsedTime,
			Status:   status,
			Deposit:  record.Deposit,
		})
	}

	return auctions, nil
}
