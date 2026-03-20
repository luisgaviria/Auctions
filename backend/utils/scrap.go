package utils

import (
	"backendAuction/utils/sites"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// upsertAuctionSQL inserts a new auction or refreshes mutable fields on conflict.
//
// updated_at is ALWAYS bumped on every scrape run — even when nothing changed —
// so the cleanup phase can reliably identify rows that were NOT seen in the last
// scrape (i.e., removed from the source site). The CASE expressions ensure that
// status/date/deposit only change when the source data actually differs.
var upsertAuctionSQL = `
	INSERT INTO auctions
		(address, city, state, time, logo, site_name, status, link, date, deposit, lat, lng)
	VALUES
		($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	ON CONFLICT (address, site_name) DO UPDATE
		SET
			status     = CASE WHEN auctions.status  IS DISTINCT FROM EXCLUDED.status
			                  THEN EXCLUDED.status  ELSE auctions.status  END,
			date       = CASE WHEN auctions.date    IS DISTINCT FROM EXCLUDED.date
			                  THEN EXCLUDED.date    ELSE auctions.date    END,
			deposit    = CASE WHEN auctions.deposit IS DISTINCT FROM EXCLUDED.deposit
			                  THEN EXCLUDED.deposit ELSE auctions.deposit END,
			time       = EXCLUDED.time,
			link       = EXCLUDED.link,
			updated_at = NOW()
	RETURNING id;`

// markStaleSQL marks auctions that were not seen in the last hour as Removed.
// Run once per site_name after all upserts for that site complete.
var markStaleSQL = `
	UPDATE auctions
	SET    status = 'Removed'
	WHERE  site_name  = $1
	  AND  updated_at < NOW() - INTERVAL '1 hour'
	  AND  status    != 'Removed';`

// normalizeStatus maps cancellation/sale/past variants to the canonical "Removed"
// value so the DB stays consistent regardless of how individual sites phrase it.
// Bare "Postponed" is also removed; "Postponed to [date]" is kept as-is.
func normalizeStatus(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	if strings.Contains(lower, "cancel") ||
		strings.Contains(lower, "sold") ||
		strings.Contains(lower, "mortgagee") ||
		strings.Contains(lower, "past") ||
		strings.Contains(lower, "3rd party") ||
		strings.Contains(lower, "purchased") ||
		lower == "removed" ||
		lower == "postponed" {
		return "Removed"
	}
	return raw
}

// normalizeTime nulls out times that are clearly placeholder/default values
// scrapers emit when no real time is known (midnight zeros, 12:00 AM, etc.).
func normalizeTime(raw string) string {
	switch strings.TrimSpace(raw) {
	case "00:00:00", "00:00", "12:00 AM", "12:00 am", "12:00AM", "12:00am":
		return ""
	}
	return raw
}

func upsertAuction(ctx context.Context, db *sql.DB, a sites.Auction) {
	a.Status = normalizeStatus(a.Status)
	a.Time = normalizeTime(a.Time)
	// Pass nil for empty date strings so the DB receives SQL NULL rather than
	// an invalid date literal (e.g. APG listings have no scheduled date).
	var dateParam interface{}
	if a.Date != "" {
		dateParam = a.Date
	}

	var id int
	err := db.QueryRowContext(ctx, upsertAuctionSQL,
		a.Street,
		a.City,
		"Massachusetts",
		a.Time,
		a.Logo,
		a.SiteName,
		a.Status,
		a.Url,
		dateParam,
		a.Deposit,
		"0",
		"0",
	).Scan(&id)
	if err != nil {
		log.Printf("[upsert] error for %q (%s): %v", a.Street, a.SiteName, err)
		return
	}
	log.Printf("[upsert] saved auction id=%d  site=%s  addr=%q", id, a.SiteName, a.Street)
}

func markStaleAuctions(ctx context.Context, db *sql.DB, siteName string) {
	res, err := db.ExecContext(ctx, markStaleSQL, siteName)
	if err != nil {
		log.Printf("[cleanup] error for site=%s: %v", siteName, err)
		return
	}
	if n, _ := res.RowsAffected(); n > 0 {
		log.Printf("[cleanup] marked %d stale auctions as Removed for site=%s", n, siteName)
	}
}

// scraperDef pairs a canonical site name with a context-aware scraper function.
// fallbackURL is non-empty for CF-based scrapers; it is used to re-fetch HTML
// for the AI self-healing fallback when the scraper returns 0 results.
type scraperDef struct {
	name        string
	fn          func(ctx context.Context) ([]sites.Auction, error)
	fallbackURL string
}

// wrapLegacy adapts the old func() []Auction signature to the new interface.
// The legacy colly/goquery scrapers do not accept a context but are otherwise
// unmodified — they will be migrated individually in a future phase.
func wrapLegacy(fn func() []sites.Auction) func(ctx context.Context) ([]sites.Auction, error) {
	return func(_ context.Context) ([]sites.Auction, error) {
		return fn(), nil
	}
}

// qualityIsPoor returns true when the auction slice is empty or more than half
// of the records have a blank Street (address) field — a sign that the scraper
// found the container HTML but failed to parse the data rows correctly.
func qualityIsPoor(auctions []sites.Auction) bool {
	if len(auctions) == 0 {
		return true
	}
	empty := 0
	for _, a := range auctions {
		if a.Street == "" {
			empty++
		}
	}
	return empty*2 > len(auctions) // >50% empty
}

// aiRescue fetches HTML from fallbackURL and passes it to Gemini for auction
// extraction. It first tries a plain http.Get (fast, no CF quota consumed);
// if that returns suspiciously little HTML (< 5 KB — likely a bare shell that
// needs JS rendering), it retries via CFetch. Returns nil on any error.
func aiRescue(ctx context.Context, siteName, fallbackURL string) []sites.Auction {
	log.Printf("[AI_HEAL] attempting rescue for site=%s url=%s", siteName, fallbackURL)

	html := aiRescueFetchHTML(ctx, siteName, fallbackURL)
	if html == "" {
		return nil
	}

	auctions, err := sites.RescueWithAI(ctx, html)
	if err != nil {
		log.Printf("[AI_HEAL] gemini failed for site=%s: %v", siteName, err)
		return nil
	}
	for i := range auctions {
		if auctions[i].SiteName == "" {
			auctions[i].SiteName = siteName
		}
	}
	log.Printf("[AI_HEAL] recovered %d auctions for site=%s", len(auctions), siteName)
	return auctions
}

// aiRescueFetchHTML fetches the URL with a plain http.Get first. If the
// response is < 5 KB (JS-rendered shell), it falls back to CFetch so that
// fully-rendered HTML is available for Gemini. Returns "" on all errors.
func aiRescueFetchHTML(ctx context.Context, siteName, url string) string {
	log.Printf("[AI_HEAL] plain http.Get url=%s", url)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err == nil {
		client := &http.Client{Timeout: 20 * time.Second}
		resp, err := client.Do(req)
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if len(body) >= 5_000 {
				log.Printf("[AI_HEAL] plain http OK site=%s bytes=%d", siteName, len(body))
				return string(body)
			}
			log.Printf("[AI_HEAL] plain http too small (%d bytes) — falling back to CFetch", len(body))
		} else {
			log.Printf("[AI_HEAL] plain http error site=%s: %v — falling back to CFetch", siteName, err)
		}
	}

	html, err := sites.CFetch(ctx, url)
	if err != nil {
		log.Printf("[AI_HEAL] cfetch also failed for site=%s: %v", siteName, err)
		return ""
	}
	return html
}

// DryRunCFScrapers runs only the three Cloudflare-migrated scrapers (Baystate,
// Commonwealth, Patriot) sequentially, upserts results, then runs the cleanup
// phase for those three sites. Used for verifying the CF migration without
// triggering the full legacy scraper suite.
func DryRunCFScrapers(ctx context.Context, db *sql.DB) {
	targets := []scraperDef{
		{"baystate", sites.ScrapBaystate, "https://www.baystateauction.com/auctions/"},
		{"commonwealth", sites.ScrapCommon, "https://www.commonwealthauctions.com/ma-auctions"},
		{"patriot", sites.ScrapPatriot, "https://patriotauctioneers.com/auctions-in-massachusetts/"},
	}

	var successfulSites []string
	for _, def := range targets {
		log.Printf("[dryrun] starting %s", def.name)
		auctions, err := def.fn(ctx)
		if err != nil {
			log.Printf("[dryrun] %s FAILED: %v", def.name, err)
			continue
		}
		log.Printf("[dryrun] %s returned %d auctions", def.name, len(auctions))
		for i := range auctions {
			if auctions[i].SiteName == "" {
				auctions[i].SiteName = def.name
			}
		}
		successfulSites = append(successfulSites, def.name)
		for _, a := range auctions {
			upsertAuction(ctx, db, a)
		}
	}

	log.Printf("[dryrun] cleanup phase for sites: %v", successfulSites)
	for _, siteName := range successfulSites {
		markStaleAuctions(ctx, db, siteName)
	}
	log.Printf("[dryrun] complete")
}

// ScrapAllSites runs all 12 scrapers in parallel goroutines, streams results
// into the database as each scraper finishes, then marks any auction that was
// not seen in the last hour as status='Removed'.
//
// The context controls the overall timeout and propagates cancellation to all
// Cloudflare HTTP calls and database queries.
func ScrapAllSites(ctx context.Context, db *sql.DB) {
	scrapers := []scraperDef{
		// Cloudflare-migrated scrapers (context-aware, no local Chrome)
		{"baystate", sites.ScrapBaystate, "https://www.baystateauction.com/auctions/"},
		{"commonwealth", sites.ScrapCommon, "https://www.commonwealthauctions.com/ma-auctions"},
		{"patriot", sites.ScrapPatriot, "https://patriotauctioneers.com/auctions-in-massachusetts/"},

		// Legacy colly/goquery scrapers (wrapped for uniform interface)
		{"amg", wrapLegacy(sites.ScrapAMG), ""},
		{"apg", wrapLegacy(sites.ScrapApg), ""},
		{"danielp", wrapLegacy(sites.ScrapDanielP), ""},
		{"dean", wrapLegacy(sites.ScrapDean), ""},
		{"harvard", wrapLegacy(sites.ScrapHarvard), ""},
		{"jake", wrapLegacy(sites.ScrapJake), ""},
		{"sri", wrapLegacy(sites.ScrapSri), ""},
		{"sullivan", wrapLegacy(sites.ScrapSullivan), ""},
		{"tache", wrapLegacy(sites.ScrapTache), ""},
	}

	type result struct {
		def      scraperDef
		auctions []sites.Auction
		err      error
	}

	// Buffer == len(scrapers) so goroutines never block on send
	ch := make(chan result, len(scrapers))
	var wg sync.WaitGroup

	for _, s := range scrapers {
		wg.Add(1)
		go func(def scraperDef) {
			defer wg.Done()
			// Recover from panics in individual scrapers so one bad scraper
			// cannot crash the orchestrator or orphan the WaitGroup counter.
			defer func() {
				if r := recover(); r != nil {
					ch <- result{def: def, err: fmt.Errorf("panic: %v", r)}
				}
			}()

			auctions, err := def.fn(ctx)

			// Stamp site_name on any auction the scraper did not set itself.
			// (Legacy scrapers never set SiteName; migrated ones set it explicitly.)
			for i := range auctions {
				if auctions[i].SiteName == "" {
					auctions[i].SiteName = def.name
				}
			}

			ch <- result{def: def, auctions: auctions, err: err}
		}(s)
	}

	// Close the channel once every goroutine has sent its result
	go func() {
		wg.Wait()
		close(ch)
	}()

	// Drain results sequentially. This loop is single-threaded, so successfulSites
	// needs no mutex — only this goroutine writes to it.
	var successfulSites []string
	for res := range ch {
		if res.err != nil {
			log.Printf("[scraper] %s FAILED: %v", res.def.name, res.err)
			// AI rescue: re-fetch the landing page and let Gemini extract auctions.
			if res.def.fallbackURL != "" {
				rescued := aiRescue(ctx, res.def.name, res.def.fallbackURL)
				if len(rescued) > 0 {
					res.auctions = rescued
					res.err = nil
				}
			}
			if res.err != nil {
				continue
			}
		}
		log.Printf("[scraper] %s returned %d auctions", res.def.name, len(res.auctions))
		// AI rescue: scraper succeeded but data quality is poor — try Gemini fallback.
		// Triggers when: (a) zero results, or (b) >50% of addresses are empty strings.
		if res.def.fallbackURL != "" && qualityIsPoor(res.auctions) {
			rescued := aiRescue(ctx, res.def.name, res.def.fallbackURL)
			if len(rescued) > 0 {
				res.auctions = rescued
			}
		}
		successfulSites = append(successfulSites, res.def.name)
		for _, a := range res.auctions {
			upsertAuction(ctx, db, a)
		}
	}

	// Cleanup phase — runs after ALL upserts are committed
	log.Printf("[cleanup] starting stale-auction cleanup for %d sites", len(successfulSites))
	for _, siteName := range successfulSites {
		markStaleAuctions(ctx, db, siteName)
	}
	log.Printf("[scraper] run complete")
}
