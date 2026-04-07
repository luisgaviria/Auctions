package sites

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	commonURL     = "https://www.commonwealthauctions.com/ma-auctions"
	commonAJAXURL = "https://www.commonwealthauctions.com/Components/Auction/Assets/AJAX/ListAuctions.php?Category=1"
)

// commonAJAXResponse is the JSON shape returned by the Commonwealth DataTable AJAX endpoint.
type commonAJAXResponse struct {
	Data []struct {
		ID       string `json:"ID"`
		Status   string `json:"Status"`
		RealDate string `json:"RealDate"` // Unix timestamp string
		Location string `json:"Location"`
		State    string `json:"State"`
		Deposit  string `json:"Deposit"`
		Date     string `json:"Date"` // "Friday, April 24, 2026 at 1:00 PM"
		Links    string `json:"Links"` // HTML anchor tag
	} `json:"data"`
}

// extractCommonHref pulls the href from the Links HTML snippet (e.g. <a href="...">...</a>).
func extractCommonHref(links string) string {
	start := strings.Index(links, `href="`)
	if start == -1 {
		return commonURL
	}
	start += len(`href="`)
	end := strings.Index(links[start:], `"`)
	if end == -1 {
		return commonURL
	}
	return links[start : start+end]
}

// commonZipRe matches a trailing US zip code (5 digits, optionally +4).
var commonZipRe = regexp.MustCompile(`\s+\d{5}(?:-\d{4})?$`)

// cleanCommonLocation sanitizes a raw Location string from the Commonwealth
// AJAX response before street/city splitting:
//
//  1. Replaces pipe characters (|) with ", " — some rows use "|" as a field
//     separator inside the location string.
//  2. Strips A/K/A aliases — keeps only the primary address (the part before
//     "A/K/A" or "AKA"), discarding the secondary identifier.
//  3. Strips trailing zip codes (e.g. "Newton 02458" → "Newton") so the
//     last-token city extraction does not return a zip code as the city name.
func cleanCommonLocation(loc string) string {
	// 1. Pipes → ", "
	loc = strings.ReplaceAll(loc, "|", ", ")

	// 2. Strip A/K/A alias — keep primary address only.
	if idx := strings.Index(strings.ToUpper(loc), "A/K/A"); idx != -1 {
		loc = loc[:idx]
	} else if idx := strings.Index(strings.ToUpper(loc), " AKA "); idx != -1 {
		loc = loc[:idx]
	}

	// 3. Strip trailing zip code.
	loc = commonZipRe.ReplaceAllString(loc, "")

	return strings.TrimRight(strings.TrimSpace(loc), ", ")
}

// splitCommonLocation cleans then splits a raw Location string like
// "2084 Washington Street Newton" into a street and city.
// Commonwealth appends the city as the last whitespace-separated token,
// so we split on the final space after sanitization.
func splitCommonLocation(location string) (street, city string) {
	if location == "" {
		return "", ""
	}
	loc := cleanCommonLocation(location)
	if loc == "" {
		return "", ""
	}
	i := strings.LastIndex(loc, " ")
	if i <= 0 {
		return loc, ""
	}
	return strings.TrimSpace(loc[:i]), strings.TrimSpace(loc[i+1:])
}

// parseCommonDate parses "Friday, April 24, 2026 at 1:00 PM" into separate date/time strings.
// Returns ("Apr 24, 2026", "1:00 PM") on success, or ("", "") on failure.
func parseCommonDate(raw string) (date, timeStr string) {
  // Clean the string: remove leading/trailing whitespace, commas, and spaces.
  clean := strings.TrimRight(strings.TrimSpace(raw), ", ")

  // The site uses "at" as a separator: "Friday, April 24, 2026 at 1:00 PM"
  // Go's layout string must match the format exactly.
  layout := "Monday, January 2, 2006 at 3:04 PM"
  
  t, err := time.Parse(layout, clean)
  if err != nil {
    log.Printf("[commonwealth] date parse error for %q: %v", raw, err)
    return "", ""
  }
  
  // Return in your preferred display format
  return t.Format("Jan 2, 2006"), t.Format("3:04 PM")
}

// ScrapCommon fetches auction listings from the Commonwealth Auctions DataTable
// AJAX endpoint directly, bypassing the JS-rendered page entirely.
func ScrapCommon(ctx context.Context) ([]Auction, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, commonAJAXURL,
		strings.NewReader("draw=1&start=0&length=200"))
	if err != nil {
		return nil, fmt.Errorf("commonwealth: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", commonURL)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("commonwealth: http post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("commonwealth: ajax returned %d: %.200s", resp.StatusCode, string(body))
	}

	var ajaxResp commonAJAXResponse
	if err := json.NewDecoder(resp.Body).Decode(&ajaxResp); err != nil {
		return nil, fmt.Errorf("commonwealth: decode json: %w", err)
	}

	auctions := make([]Auction, 0, len(ajaxResp.Data))
	for _, row := range ajaxResp.Data {
		if row.Location == "" {
			continue
		}
		date, timeStr := parseCommonDate(row.Date)
		street, city := splitCommonLocation(row.Location)
		auctions = append(auctions, Auction{
			SiteName: "commonwealth",
			Logo:     "/commonwealth.webp",
			Date:     date,
			Time:     timeStr,
			Street:   street,
			City:     city,
			Status:   row.Status,
			Deposit:  row.Deposit,
			Url:      extractCommonHref(row.Links),
		})
	}

	log.Printf("[commonwealth] ajax returned %d rows", len(auctions))
	return auctions, nil
}
