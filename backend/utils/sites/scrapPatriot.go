package sites

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const patriotBase = "https://patriotauctioneers.com"
const patriotLogo = "/patriot.webp"

// ScrapPatriot fetches the Patriot Auctioneers listing page via Cloudflare
// Browser Rendering, then fetches each individual auction detail page to
// extract the deposit amount and status. The N+1 detail fetches are HTTP
// requests to Cloudflare — no local browser tabs are opened.
func ScrapPatriot(ctx context.Context) ([]Auction, error) {
	html, err := CFetch(ctx, patriotBase+"/auctions-in-massachusetts/")
	if err != nil {
		return nil, fmt.Errorf("patriot: list page: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("patriot: parse list: %w", err)
	}

	var auctions []Auction
	doc.Find("#calendar > div > a").Each(func(_ int, a *goquery.Selection) {
		rawAddress := strings.TrimSpace(a.Find("h1").Text())
		rawDate := strings.TrimSpace(a.Find(".auction-date").Text())
		rawDate = strings.TrimSpace(strings.Split(rawDate, "Continued")[0])

		href, _ := a.Attr("href")
		// Guard: if the listing anchor has no href (e.g. scrape degraded),
		// fall back to the Massachusetts auctions index rather than storing
		// the bare base URL which offers no useful detail for the user.
		var fullHref string
		if strings.TrimSpace(href) != "" {
			fullHref = patriotBase + href
		} else {
			fullHref = patriotBase + "/auctions-in-massachusetts/"
		}

		formattedDate, formattedTime := parseDateAndTimePatriot(rawDate)
		deposit, status := patriotDetail(ctx, fullHref)

		// Patriot formats addresses as "47 Foss Road - Gardner, MA".
		// Split on " - " to separate the street from the city/state token.
		street := rawAddress
		city := ""
		if parts := strings.SplitN(rawAddress, " - ", 2); len(parts) == 2 {
			street = strings.TrimSpace(parts[0])
			// City is the word before the state abbreviation: "Gardner, MA" → "Gardner"
			cityState := strings.TrimSpace(parts[1])
			if comma := strings.Index(cityState, ","); comma != -1 {
				city = strings.TrimSpace(cityState[:comma])
			} else {
				city = cityState
			}
		}

		auctions = append(auctions, Auction{
			SiteName: "patriot",
			Logo:     patriotLogo,
			Street:   street,
			City:     city,
			Url:      fullHref,
			Date:     formattedDate,
			Time:     formattedTime,
			Deposit:  deposit,
			Status:   status,
		})
	})

	return auctions, nil
}

// patriotDetail fetches a single Patriot auction detail page and returns the
// deposit string and status text. On any failure, safe defaults are returned
// so the listing is still recorded with partial data.
func patriotDetail(ctx context.Context, url string) (deposit, status string) {
	html, err := CFetch(ctx, url)
	if err != nil {
		return "", "On Schedule"
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", "On Schedule"
	}

	// Extract the first $X,XXX dollar amount from the detail page.
	// Try selectors from most-specific to least-specific so a site redesign
	// degrades gracefully rather than returning an empty deposit.
	dollarRe := regexp.MustCompile(`\$[0-9,]+`)
	candidateSelectors := []string{
		".auction-terms",
		".auction-details",
		".terms",
		"#calendar > div:nth-child(2) > div > div.col-md-4 > div:nth-child(3) > p",
		"#calendar > div:nth-child(2) > div > div.col-md-4",
		".col-md-4",
	}
	for _, sel := range candidateSelectors {
		text := doc.Find(sel).Text()
		if m := dollarRe.FindString(text); m != "" {
			deposit = m
			break
		}
	}
	// Last resort: scan the entire document for any dollar amount.
	if deposit == "" {
		if m := dollarRe.FindString(doc.Text()); m != "" {
			deposit = m
		}
	}

	status = strings.TrimSpace(doc.Find(
		"#calendar > div:nth-child(2) > div > div.col-md-4 > div:nth-child(1) > p > span.text-red > strong",
	).Text())
	if status == "" {
		status = "On Schedule"
	}

	return deposit, status
}

// parseDateAndTimePatriot parses "Monday Mar 10 @ 11:00 am" into
// ("2006-01-02", "11:00 am"). Returns empty strings on parse failure.
func parseDateAndTimePatriot(dateTimeStr string) (string, string) {
	re := regexp.MustCompile(`(\w+ \w+ \d+) @ (\d+:\d+ [ap]m)`)
	match := re.FindStringSubmatch(dateTimeStr)
	if len(match) != 3 {
		return "", ""
	}

	dateStr := strings.TrimSpace(match[1])
	timeStr := strings.TrimSpace(match[2])

	parsedDate, err := time.Parse("Monday Jan 2", dateStr)
	if err != nil {
		return "", timeStr
	}

	currentYear := time.Now().Year()
	parsedDate = parsedDate.AddDate(currentYear-parsedDate.Year(), 0, 0)
	return parsedDate.Format("2006-01-02"), timeStr
}
