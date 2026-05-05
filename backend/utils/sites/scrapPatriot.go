package sites

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const patriotBase = "https://patriotauctioneers.com"
const patriotLogo = "/patriot.webp"

// patriotForceDetailFetch bypasses the deposit cache so every detail page is
// visited fresh.  Set to true for a one-time run to populate legal_description
// for all Patriot rows; flip back to false once the column is fully populated.
const patriotForceDetailFetch = true

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
	listURL := patriotBase + "/auctions-in-massachusetts/"
	var html string
	if body, ok := patriotDetailPlainHTTP(ctx, listURL); ok {
		log.Printf("[patriot] list page plain-HTTP hit bytes=%d", len(body))
		html = body
	} else {
		log.Printf("[patriot] list page plain-HTTP failed — falling back to CFetch")
		var err error
		html, err = CFetch(ctx, listURL)
		if err != nil {
			return nil, fmt.Errorf("patriot: list page: %w", err)
		}
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

	// ── Pass 2: fetch detail pages via 10-worker pool ────────────────────────
	type detailResult struct {
		deposit string
		status  string
		legal   string // raw text containing Book/Page citation
	}
	details := make([]detailResult, len(items))

	type urlResult struct {
		deposit string
		legal   string
	}
	var dedupMu sync.Mutex
	urlCache := make(map[string]*urlResult) // URL → completed result (nil = in-flight)

	type workUnit struct{ idx int }
	work := make(chan workUnit, len(items))
	for i := range items {
		work <- workUnit{i}
	}
	close(work)

	const numWorkers = 10
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for wu := range work {
				item := items[wu.idx]
				// Normalise the URL: strip a trailing slash so that DB links
				// stored with or without it both match.
				normURL := strings.TrimRight(item.fullHref, "/")

				// 1. Supabase deposit cache — skipped when patriotForceDetailFetch
				//    is true so legal_description is populated on every property.
				if !patriotForceDetailFetch {
					if cached, ok := knownDeposits[normURL]; ok && cached != "" {
						log.Printf("[patriot] deposit cached — skipping detail fetch url=%q", normURL)
						details[wu.idx] = detailResult{cached, item.listStatus, ""}
						continue
					}
					if cached, ok := knownDeposits[normURL+"/"]; ok && cached != "" {
						log.Printf("[patriot] deposit cached (slash) — skipping detail fetch url=%q", normURL)
						details[wu.idx] = detailResult{cached, item.listStatus, ""}
						continue
					}
				}

				// 2. Deduplicate concurrent fetches for the same URL.
				dedupMu.Lock()
				if res, seen := urlCache[normURL]; seen && res != nil {
					dedupMu.Unlock()
					details[wu.idx] = detailResult{res.deposit, item.listStatus, res.legal}
					continue
				}
				urlCache[normURL] = nil
				dedupMu.Unlock()

				deposit, legal := patriotDetail(ctx, item.fullHref)
				// Status always comes from the list page — never from the detail
				// page CSS selector, which can be stale or missing.
				details[wu.idx] = detailResult{deposit, item.listStatus, legal}

				dedupMu.Lock()
				urlCache[normURL] = &urlResult{deposit, legal}
				dedupMu.Unlock()
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
			SiteName:         "patriot",
			Logo:             patriotLogo,
			Street:           street,
			City:             city,
			Url:              item.fullHref,
			Date:             item.formattedDate,
			Time:             item.formattedTime,
			Deposit:          details[i].deposit,
			Status:           details[i].status,
			LegalDescription: details[i].legal,
		})
	}

	return auctions, nil
}

// patriotDetailHTTPClient is a shared client for the plain-HTTP fast path.
var patriotHTTPClient = &http.Client{Timeout: 10 * time.Second}

// patriotDetail fetches a single Patriot auction detail page and returns the
// deposit string and status text. On any failure, safe defaults are returned
// so the listing is still recorded with partial data.
//
// Fast path: plain HTTP GET with a desktop User-Agent. Patriot serves its
// auction terms in the initial HTML, so no JS execution is needed most of the
// time. Only falls back to CFetchFast (domcontentloaded + block_ads +
// block_images) when the plain GET misses the deposit.
func patriotDetail(ctx context.Context, url string) (deposit, legal string) {
	if html, ok := patriotDetailPlainHTTP(ctx, url); ok {
		log.Printf("[patriot] plain-HTTP hit html_bytes=%d url=%q", len(html), url)
		if dep, leg := patriotParseDetail(html, url); dep != "" {
			return dep, leg
		}
		log.Printf("[patriot] plain-HTTP deposit empty — falling back to CFetchFast url=%q", url)
	}

	html, err := CFetchFast(ctx, url)
	if err != nil {
		return "", ""
	}
	log.Printf("[patriot] CFetchFast html_bytes=%d url=%q", len(html), url)
	return patriotParseDetail(html, url)
}

// patriotDetailPlainHTTP performs a plain HTTP GET with a desktop User-Agent.
// Returns (html, true) on a 200 OK, ("", false) on any error or non-200.
func patriotDetailPlainHTTP(ctx context.Context, url string) (string, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := patriotHTTPClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if err != nil {
			log.Printf("[patriot] plain-HTTP error url=%q err=%v", url, err)
		} else {
			log.Printf("[patriot] plain-HTTP status=%d url=%q — falling back", resp.StatusCode, url)
			resp.Body.Close()
		}
		return "", false
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false
	}
	return string(body), true
}

// patriotLegalRe detects paragraphs containing a Book/Page registry citation.
var patriotLegalRe = regexp.MustCompile(`(?i)(?:book|bk)\s*\.?\s*\d+`)

// patriotParseDetail extracts deposit and legal description from a Patriot detail page HTML.
func patriotParseDetail(html, url string) (deposit, legal string) {
	log.Printf("[patriot] detail html_bytes=%d url=%q", len(html), url)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", ""
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
		if legal == "" && patriotLegalRe.MatchString(text) {
			legal = text
			log.Printf("[patriot]   → matched legal=%q at p[%d]", legal, i)
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

	log.Printf("[patriot] final deposit=%q legal=%q url=%q", deposit, legal, url)
	return deposit, legal
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
