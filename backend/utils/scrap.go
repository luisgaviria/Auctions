package utils

import (
	"backendAuction/utils/sites"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

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

// qualityIsPoor returns true when the auction slice is empty, more than half
// of the records have a blank Street field, or more than half have a blank Date
// field — the latter catches scrapers that find addresses but fail to parse dates.
func qualityIsPoor(auctions []sites.Auction) bool {
	if len(auctions) == 0 {
		return true
	}
	emptyStreet, emptyDate := 0, 0
	for _, a := range auctions {
		if a.Street == "" {
			emptyStreet++
		}
		if a.Date == "" {
			emptyDate++
		}
	}
	n := len(auctions)
	return emptyStreet*2 > n || emptyDate*2 > n
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

	// Use a dedicated 3-minute context for the AI call so the main scraper's
	// overall deadline cannot cancel it prematurely.
	aiCtx, aiCancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer aiCancel()

	auctions, err := sites.RescueWithAI(aiCtx, html)
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
		client := &http.Client{Timeout: 120 * time.Second}
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
	migrateNonISODates(ctx, db)
	purgePastAuctions(ctx, db)
	backfillMissingURLs(ctx, db)
	backfillSlugs(ctx, db)

	targets := []scraperDef{
		{"baystate", sites.ScrapBaystate, "https://www.baystateauction.com/auctions/"},
		{"commonwealth", sites.ScrapCommon, "https://www.commonwealthauctions.com/ma-auctions"},
		{"patriot", sites.ScrapPatriot, "https://patriotauctioneers.com/auctions-in-massachusetts/"},
	}

	seen := make(map[string]bool)
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
			a = NormalizeAuction(a)
			key := a.Street + "|" + a.SiteName
			if seen[key] {
				log.Printf("[dedup] skipping duplicate addr=%q site=%s", a.Street, a.SiteName)
				continue
			}
			seen[key] = true
			upsertAuction(ctx, db, a)
		}
	}

	sweepStaleAuctions(ctx, db)
	log.Printf("[dryrun] complete")
}

// ScrapAllSites runs all 12 scrapers in parallel goroutines, streams results
// into the database as each scraper finishes, then marks any auction that was
// not seen in the last hour as status='Removed'.
//
// The context controls the overall timeout and propagates cancellation to all
// Cloudflare HTTP calls and database queries.
func ScrapAllSites(ctx context.Context, db *sql.DB) {
	// Pre-scrape maintenance: fast DB-only operations, no external HTTP.
	migrateNonISODates(ctx, db)
	purgePastAuctions(ctx, db)
	backfillMissingURLs(ctx, db)
	backfillSlugs(ctx, db)
	// NOTE: backfillRegistryCoordinates runs synchronously at the end of Pass 1.

	// Build a deposit cache for Patriot so detail page crawls are skipped for
	// properties whose deposit is already stored.  This saves one CF Browser
	// Rendering call per cached auction.
	patriotDepositCache := make(map[string]string)
	if dcRows, dcErr := db.QueryContext(ctx,
		`SELECT link, deposit FROM auctions WHERE site_name='patriot' AND deposit IS NOT NULL AND deposit != ''`); dcErr == nil {
		for dcRows.Next() {
			var link, deposit string
			if dcRows.Scan(&link, &deposit) == nil && link != "" {
				patriotDepositCache[link] = deposit
			}
		}
		dcRows.Close()
		log.Printf("[patriot] deposit cache loaded: %d entries", len(patriotDepositCache))
	}

	scrapers := []scraperDef{
		// Cloudflare-migrated scrapers (context-aware, no local Chrome)
		{"baystate", sites.ScrapBaystate, "https://www.baystateauction.com/auctions/"},
		{"commonwealth", sites.ScrapCommon, "https://www.commonwealthauctions.com/ma-auctions"},
		{"patriot", func(ctx context.Context) ([]sites.Auction, error) {
			return sites.ScrapPatriotWithCache(ctx, patriotDepositCache)
		}, "https://patriotauctioneers.com/auctions-in-massachusetts/"},

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

	// Drain results sequentially. This loop is single-threaded, so both
	// successfulSites and seen need no mutex — only this goroutine writes them.
	seen := make(map[string]bool)
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
			a = NormalizeAuction(a)
			key := a.Street + "|" + a.SiteName
			if seen[key] {
				log.Printf("[dedup] skipping duplicate addr=%q site=%s", a.Street, a.SiteName)
				continue
			}
			seen[key] = true
			upsertAuction(ctx, db, a)
		}
	}

	// Tag & Sweep — hard-delete any row not touched in the last 6 hours.
	// Runs once after ALL sites have upserted so a single failing scraper
	// does not prematurely delete its auctions.
	sweepStaleAuctions(ctx, db)

	log.Printf("[scraper] Pass 1 complete — all addresses and statuses committed to database")

	// Backfill: NormalizeAuction extracts book/page inline for every new row.
	// This pass also catches existing DB rows that pre-date book/page extraction.
	backfillRegistryCoordinates(ctx, db)

	// Deep-link backfill disabled: masslandrecords.com is purely PostBack-driven
	// (ASP.NET WebForms) — no GET/hash URL can pre-fill or submit a book/page
	// search. A Cloudflare Worker with Puppeteer is required to automate the form.
	// go BackfillRegistryDeepLinks(db)

	log.Printf("[scraper] run complete")
}
