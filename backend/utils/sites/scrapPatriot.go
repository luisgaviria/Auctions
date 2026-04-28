package sites

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const patriotBase = "https://patriotauctioneers.com"
const patriotLogo = "/patriot.webp"

// ScrapPatriot fetches the Patriot Auctioneers listing page and all detail
// pages.  It is a thin wrapper around ScrapPatriotWithCache with no cache.
func ScrapPatriot(ctx context.Context) ([]Auction, error) {
	return ScrapPatriotWithCache(ctx, nil)
}

// ScrapPatriotWithCache is like ScrapPatriot but accepts a map of already-known
// deposits keyed by full listing URL.  When a URL's deposit is already present
// in the map, the browser-based detail crawl is skipped entirely — saving one
// Cloudflare Browser Rendering API call per cached property.
// Pass nil (or an empty map) to always fetch detail pages.
//
// Detail fetches are dispatched to a pool of 3 parallel workers so multiple
// Cloudflare calls proceed concurrently instead of sequentially.
func ScrapPatriotWithCache(ctx context.Context, knownDeposits map[string]string) ([]Auction, error) {
	html, err := CFetch(ctx, patriotBase+"/auctions-in-massachusetts/")
	if err != nil {
		return nil, fmt.Errorf("patriot: list page: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("patriot: parse list: %w", err)
	}

	// ── Pass 1: collect raw listing data (no detail calls yet) ──────────────
	type listItem struct {
		rawAddress    string
		formattedDate string
		formattedTime string
		fullHref      string
		listStatus    string
	}
	var items []listItem

	doc.Find("#calendar > div > a").Each(func(_ int, a *goquery.Selection) {
		rawAddress := strings.TrimSpace(a.Find("h1").Text())
		rawDateRaw := strings.TrimSpace(a.Find(".auction-date").Text())

		listStatus := "On Schedule"
		if strings.Contains(strings.ToUpper(rawDateRaw), "CONTINUED") {
			listStatus = "Continued"
		}
		rawDate := strings.TrimSpace(strings.Split(rawDateRaw, "Continued")[0])

		href, _ := a.Attr("href")
		var fullHref string
		if strings.TrimSpace(href) != "" {
			fullHref = patriotBase + href
		} else {
			fullHref = patriotBase + "/auctions-in-massachusetts/"
		}

		formattedDate, formattedTime := parseDateAndTimePatriot(rawDate)
		items = append(items, listItem{rawAddress, formattedDate, formattedTime, fullHref, listStatus})
	})

	// ── Pass 2: fetch detail pages via 3-worker pool ─────────────────────────
	type detailResult struct {
		deposit string
		status  string
	}
	details := make([]detailResult, len(items))

	type workUnit struct{ idx int }
	work := make(chan workUnit, len(items))
	for i := range items {
		work <- workUnit{i}
	}
	close(work)

	const numWorkers = 3
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for wu := range work {
				item := items[wu.idx]
				if cached, ok := knownDeposits[item.fullHref]; ok && cached != "" {
					log.Printf("[patriot] deposit cached — skipping detail fetch url=%q", item.fullHref)
					details[wu.idx] = detailResult{cached, item.listStatus}
				} else {
					deposit, status := patriotDetail(ctx, item.fullHref)
					details[wu.idx] = detailResult{deposit, status}
				}
			}
		}()
	}
	wg.Wait()

	// ── Pass 3: assemble final auction slice ─────────────────────────────────
	var auctions []Auction
	for i, item := range items {
		// Patriot formats addresses as "47 Foss Road - Gardner, MA".
		street := item.rawAddress
		city := ""
		if parts := strings.SplitN(item.rawAddress, " - ", 2); len(parts) == 2 {
			street = strings.TrimSpace(parts[0])
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
			Url:      item.fullHref,
			Date:     item.formattedDate,
			Time:     item.formattedTime,
			Deposit:  details[i].deposit,
			Status:   details[i].status,
		})
	}

	return auctions, nil
}

// patriotDetail fetches a single Patriot auction detail page and returns the
// deposit string and status text. On any failure, safe defaults are returned
// so the listing is still recorded with partial data.
func patriotDetail(ctx context.Context, url string) (deposit, status string) {
	// Use CFetchSlow: block until .auction-box appears in the DOM, giving
	// JS-rendered terms time to paint before we grab the HTML.
	html, err := CFetchSlow(ctx, url, ".auction-box", 0)
	if err != nil {
		return "", "On Schedule"
	}
	log.Printf("[patriot] detail html_bytes=%d url=%q", len(html), url)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", "On Schedule"
	}

	// cleanText normalises whitespace: replaces non-breaking spaces (\xA0)
	// and other Unicode spaces with a plain space, then trims.
	cleanText := func(s string) string {
		s = strings.ReplaceAll(s, "\u00a0", " ") // &nbsp;
		s = strings.ReplaceAll(s, "\u200b", " ") // zero-width space
		return strings.TrimSpace(s)
	}

	// Primary: "$10,000 deposit" — dollar amount immediately before "deposit".
	depositBeforeRe := regexp.MustCompile(`(?i)(\$[\d,]+(?:\.\d{2})?)\s+deposit`)
	// Secondary: "deposit of $10,000" — dollar amount immediately after.
	depositAfterRe := regexp.MustCompile(`(?i)deposit\s+of\s+(\$[\d,]+(?:\.\d{2})?)`)
	// Fallback: any dollar amount.
	dollarRe := regexp.MustCompile(`\$[\d,]+(?:\.\d{2})?`)

	matchDeposit := func(text string) string {
		text = cleanText(text)
		if m := depositBeforeRe.FindStringSubmatch(text); len(m) > 1 {
			return m[1]
		}
		if m := depositAfterRe.FindStringSubmatch(text); len(m) > 1 {
			return m[1]
		}
		return ""
	}

	// First pass: iterate every <p> inside .auction-box and log each one so we
	// can see exactly which paragraph contains the deposit terms.
	log.Printf("[patriot] scanning .auction-box paragraphs for url=%q", url)
	doc.Find(".auction-box p").Each(func(i int, p *goquery.Selection) {
		text := cleanText(p.Text())
		log.Printf("[patriot]   p[%d] text=%q", i, text)
		if deposit == "" {
			deposit = matchDeposit(text)
			if deposit != "" {
				log.Printf("[patriot]   → matched deposit=%q at p[%d]", deposit, i)
			}
		}
	})

	// Second pass: .auction-terms (original selector) as a single block.
	if deposit == "" {
		termsText := cleanText(doc.Find(".auction-terms").Text())
		log.Printf("[patriot] .auction-terms text=%q", termsText)
		deposit = matchDeposit(termsText)
	}

	// Third pass: broader candidate selectors.
	if deposit == "" {
		for _, sel := range []string{
			".auction-details", ".terms", ".col-md-4",
		} {
			text := cleanText(doc.Find(sel).Text())
			if d := matchDeposit(text); d != "" {
				deposit = d
				log.Printf("[patriot] matched via selector %q deposit=%q", sel, deposit)
				break
			}
			if m := dollarRe.FindString(text); m != "" {
				deposit = m
				log.Printf("[patriot] fallback dollar via selector %q deposit=%q", sel, deposit)
				break
			}
		}
	}

	// Fourth pass: scan every <p> in the document (not scoped to .auction-box).
	// Catches pages where the terms are outside the expected container.
	if deposit == "" {
		log.Printf("[patriot] scanning ALL document paragraphs url=%q", url)
		doc.Find("p").Each(func(i int, p *goquery.Selection) {
			text := cleanText(p.Text())
			if deposit == "" && strings.Contains(strings.ToLower(text), "deposit") {
				log.Printf("[patriot]   doc-p[%d] text=%q", i, text)
				deposit = matchDeposit(text)
				if deposit != "" {
					log.Printf("[patriot]   → matched deposit=%q at doc-p[%d]", deposit, i)
				}
			}
		})
	}

	// Last resort: full document text.
	if deposit == "" {
		fullText := cleanText(doc.Text())
		if d := matchDeposit(fullText); d != "" {
			deposit = d
		} else if m := dollarRe.FindString(fullText); m != "" {
			deposit = m
		}
	}

	log.Printf("[patriot] final deposit=%q url=%q", deposit, url)

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
