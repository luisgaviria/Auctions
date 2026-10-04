package utils

import (
	"backendAuction/utils/sites"
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

// upsertAuctionSQL inserts a new auction or refreshes mutable fields on conflict.
//
// last_seen is stamped NOW() on every hit so the post-scrape sweep can identify
// rows that were genuinely removed from the source site (not seen for 6 h).
// date uses COALESCE so a scraper that temporarily returns a broken/null date
// does not overwrite a good date already stored in the DB.
var upsertAuctionSQL = `
	INSERT INTO auctions
		(address, city, state, time, logo, site_name, status, link, date, deposit, lat, lng,
		 zillow_url, street_view_url, registry_url, last_seen, city_slug, county_slug, address_slug,
		 legal_description, registry_deep_link, assessor_pid, registry_book, registry_page)
	VALUES
		($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW(), $16, $17, $18,
		 $19, $20, $21, $22, $23)
	ON CONFLICT ON CONSTRAINT uq_auctions_address_site DO UPDATE
		SET
			city              = EXCLUDED.city,
			status            = EXCLUDED.status,
			date              = COALESCE(EXCLUDED.date, auctions.date),
			deposit           = CASE WHEN auctions.deposit IS DISTINCT FROM EXCLUDED.deposit
			                         THEN EXCLUDED.deposit ELSE auctions.deposit END,
			time              = EXCLUDED.time,
			link              = EXCLUDED.link,
			zillow_url        = EXCLUDED.zillow_url,
			street_view_url   = EXCLUDED.street_view_url,
			registry_url      = EXCLUDED.registry_url,
			city_slug         = EXCLUDED.city_slug,
			county_slug       = EXCLUDED.county_slug,
			address_slug      = EXCLUDED.address_slug,
			legal_description  = COALESCE(EXCLUDED.legal_description,  auctions.legal_description),
			registry_deep_link = COALESCE(EXCLUDED.registry_deep_link, auctions.registry_deep_link),
			assessor_pid       = COALESCE(EXCLUDED.assessor_pid,        auctions.assessor_pid),
			registry_book      = COALESCE(EXCLUDED.registry_book,       auctions.registry_book),
			registry_page      = COALESCE(EXCLUDED.registry_page,       auctions.registry_page),
			last_seen         = EXCLUDED.last_seen,
			updated_at        = NOW()
	RETURNING id;`

// sweepStaleSQL hard-deletes rows that have not been seen by any scraper in the
// last 6 hours.  The 6-hour window means a single transient scraper failure
// leaves data intact; two consecutive missed runs signal genuine removal.
var sweepStaleSQL = `
	DELETE FROM auctions
	WHERE last_seen < NOW() - INTERVAL '6 hours';`

// upsertAuction writes a single already-normalised auction to the DB.
// Callers must run NormalizeAuction before calling this function.
// nullIfEmpty converts an empty string to a nil interface so lib/pq writes SQL
// NULL instead of an empty string.  This lets COALESCE in the upsert preserve
// a previously-stored non-empty value when the current scrape didn't capture one.
func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullIfZero(n int) interface{} {
	if n == 0 {
		return nil
	}
	return n
}

// atoiSafe converts a string to int, returning 0 on any parse error.
func atoiSafe(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

func upsertAuction(ctx context.Context, db *sql.DB, a sites.Auction) {
	// Skip auctions whose date has already passed (strictly before today in ET).
	// Same-day auctions whose time has elapsed are kept: we still upsert them
	// so that last_seen is refreshed on every scrape run. Without this, the
	// 6-hour sweep would delete a 9 AM auction by 3 PM simply because the
	// afternoon scraper skipped updating last_seen. The midnight ET SQL purge
	// (purgePastAuctions) is the correct place to remove yesterday's rows.
	if isPastDate(a.Date) {
		log.Printf("[upsert] skipping past date addr=%q site=%s date=%s", a.Street, a.SiteName, a.Date)
		return
	}
	// Log elapsed same-day auctions for visibility but do NOT skip the upsert.
	if isElapsedToday(a.Date, a.Time) {
		log.Printf("[upsert] elapsed today (still saving) addr=%q site=%s date=%s time=%s", a.Street, a.SiteName, a.Date, a.Time)
	}

	// Pass nil for empty date strings so the DB receives SQL NULL rather than
	// an invalid date literal (e.g. APG listings have no scheduled date).
	var dateParam interface{}
	if a.Date != "" {
		dateParam = a.Date
	}

	// Use a named prepared statement to avoid lib/pq's unnamed "" prepared
	// statement being shared with the 2-param UPDATE in migrateNonISODates.
	// When concurrent goroutines return different SQL to the same connection's
	// unnamed slot, pq can bind the wrong parameter count. A named statement
	// is isolated to this specific SQL and avoids the conflict entirely.
	stmt, prepErr := db.PrepareContext(ctx, upsertAuctionSQL)
	if prepErr != nil {
		log.Printf("[upsert] prepare error for %q (%s): %v", a.Street, a.SiteName, prepErr)
		return
	}
	defer stmt.Close()

	var id int
	err := stmt.QueryRowContext(ctx,
		a.Street,                        // $1  address
		a.City,                          // $2  city
		"Massachusetts",                 // $3  state
		a.Time,                          // $4  time
		a.Logo,                          // $5  logo
		a.SiteName,                      // $6  site_name
		a.Status,                        // $7  status
		a.Url,                           // $8  link
		dateParam,                       // $9  date
		nullIfZero(atoiSafe(a.Deposit)), // $10 deposit (integer)
		"0",                             // $11 lat
		"0",                             // $12 lng
		a.ZillowURL,                     // $13 zillow_url
		a.StreetViewURL,                 // $14 street_view_url
		a.RegistryURL,                   // $15 registry_url
		a.CitySlug,                      // $16 city_slug
		a.CountySlug,                    // $17 county_slug
		a.AddressSlug,                   // $18 address_slug
		nullIfEmpty(a.LegalDescription), // $19 legal_description
		nil,                             // $20 registry_deep_link — frontend builds dynamically; COALESCE preserves old rows
		nil,                             // $21 assessor_pid (set by second-pass)
		nullIfZero(a.RegistryBook),      // $22 registry_book
		nullIfZero(a.RegistryPage),      // $23 registry_page
	).Scan(&id)
	if err != nil {
		log.Printf("[upsert] error for %q (%s): %v", a.Street, a.SiteName, err)
		return
	}
	log.Printf("[upsert] saved auction id=%d  site=%s  addr=%q", id, a.SiteName, a.Street)
}

func sweepStaleAuctions(ctx context.Context, db *sql.DB) {
	res, err := db.ExecContext(ctx, sweepStaleSQL)
	if err != nil {
		log.Printf("[sweep] error: %v", err)
		return
	}
	n, _ := res.RowsAffected()
	log.Printf("[sweep] deleted %d auctions not seen in the last 6 hours", n)
}

// migrateNonISODates finds any auction whose date column contains a non-ISO
// string (e.g. "Apr 2, 2026", "04/02/2026") and rewrites it as ISO-8601
// ("2026-04-02") so the SQL purge, ORDER BY, and frontend all work correctly.
//
// If the date column is a native PostgreSQL DATE type, date::text always
// produces ISO-8601 output, so this function becomes a no-op — it scans zero
// rows and logs "standardized 0/0 non-ISO dates".  The function is still
// valuable as a defensive measure against future schema drift or manual edits.
func migrateNonISODates(ctx context.Context, db *sql.DB) {
	// Cast to text so the regex works regardless of whether the column is
	// DATE or VARCHAR.  Only fetch rows whose text representation is NOT
	// already in "YYYY-MM-DD" format.
	const sel = `
		SELECT id, date::text FROM auctions
		WHERE date IS NOT NULL
		  AND date::text !~ '^\d{4}-\d{2}-\d{2}$'`
	rows, err := db.QueryContext(ctx, sel)
	if err != nil {
		log.Printf("[migrate] query error: %v", err)
		return
	}
	defer rows.Close()

	type row struct {
		id   int
		date string
	}
	var toFix []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.date); err == nil {
			toFix = append(toFix, r)
		}
	}
	rows.Close()

	fixed := 0
	for _, r := range toFix {
		iso := normalizeDateToISO(r.date)
		if iso == r.date || iso == "" {
			log.Printf("[migrate] cannot normalize id=%d date=%q — leaving unchanged", r.id, r.date)
			continue
		}
		if _, err := db.ExecContext(ctx,
			`UPDATE auctions SET date = $1 WHERE id = $2`, iso, r.id); err != nil {
			log.Printf("[migrate] update id=%d: %v", r.id, err)
		} else {
			fixed++
		}
	}
	log.Printf("[migrate] standardized %d/%d non-ISO dates", fixed, len(toFix))
}

// purgePastAuctions hard-deletes every row whose date is earlier than today
// in Massachusetts (America/New_York).
//
// The CASE WHEN makes it format-blind:
//   - ISO dates (YYYY-MM-DD)  → cast to date and compare normally.
//   - Any other format        → treated as CURRENT_DATE - 1 day, which is
//     always < today, so the row is always deleted.
//
// This means legacy rows with "Apr 2, 2026", "04/02/2026", or any other
// non-ISO string are caught and removed without needing to know their format.
// Works identically whether the date column is DATE or VARCHAR.
func purgePastAuctions(ctx context.Context, db *sql.DB) {
	const q = `
		DELETE FROM auctions
		WHERE date IS NOT NULL
		  AND (
		    CASE
		      WHEN date::text ~ '^\d{4}-\d{2}-\d{2}$'
		        THEN date::date
		      ELSE CURRENT_DATE - INTERVAL '1 day'
		    END
		  ) < (CURRENT_TIMESTAMP AT TIME ZONE 'America/New_York')::date;`
	res, err := db.ExecContext(ctx, q)
	if err != nil {
		log.Printf("[purge] error: %v", err)
		return
	}
	n, _ := res.RowsAffected()
	log.Printf("[purge] deleted %d past auctions (any-format, ET-aware)", n)
}

// backfillMissingURLs finds existing auction rows that have NULL zillow_url or
// registry_url (rows inserted before the URL columns were added, or before the
// normalization pipeline populated them) and generates + saves the URLs.
// This runs at the start of every scrape so no manual migration is needed.
func backfillMissingURLs(ctx context.Context, db *sql.DB) {
	const sel = `
		SELECT id, address, city FROM auctions
		WHERE zillow_url IS NULL OR registry_url IS NULL`
	rows, err := db.QueryContext(ctx, sel)
	if err != nil {
		log.Printf("[backfill] query error: %v", err)
		return
	}
	defer rows.Close()

	type row struct {
		id      int
		address string
		city    string
	}
	var toFix []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.address, &r.city); err == nil {
			toFix = append(toFix, r)
		}
	}
	rows.Close()

	fixed := 0
	for _, r := range toFix {
		zillow, streetView, registry := buildAuctionURLs(r.address, r.city)
		_, err := db.ExecContext(ctx,
			`UPDATE auctions SET zillow_url=$1, street_view_url=$2, registry_url=$3 WHERE id=$4`,
			zillow, streetView, registry, r.id)
		if err != nil {
			log.Printf("[backfill] update id=%d: %v", r.id, err)
		} else {
			fixed++
		}
	}
	log.Printf("[backfill] populated URLs for %d/%d auctions", fixed, len(toFix))
}

// backfillSlugs fixes existing rows that have no city_slug / county_slug yet.
// For rows where city is still "Massachusetts" it also re-parses the real city
// from the address using CleanAuctionData before generating slugs.
// Runs at scrape startup alongside backfillMissingURLs.
func backfillSlugs(ctx context.Context, db *sql.DB) {
	const sel = `
		SELECT id, address, city FROM auctions
		WHERE city_slug IS NULL OR county_slug IS NULL OR address_slug IS NULL`
	rows, err := db.QueryContext(ctx, sel)
	if err != nil {
		log.Printf("[backfill-slugs] query error: %v", err)
		return
	}
	defer rows.Close()

	type row struct {
		id      int
		address string
		city    string
	}
	var toFix []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.address, &r.city); err == nil {
			toFix = append(toFix, r)
		}
	}
	rows.Close()

	fixed := 0
	for _, r := range toFix {
		// Re-derive city from address when the scraper left a generic placeholder.
		_, city := CleanAuctionData(r.address, r.city)

		citySlug := GenerateSlug(city)
		county := GetCounty(city)
		countySlug := ""
		if county != "" {
			countySlug = GenerateSlug(county + " County")
		}

		addressSlug := GenerateSlug(r.address + " " + city)

		_, err := db.ExecContext(ctx,
			`UPDATE auctions SET city=$1, city_slug=$2, county_slug=$3, address_slug=$4 WHERE id=$5`,
			city, citySlug, countySlug, addressSlug, r.id)
		if err != nil {
			log.Printf("[backfill-slugs] update id=%d: %v", r.id, err)
		} else {
			fixed++
		}
	}
	log.Printf("[backfill-slugs] populated slugs for %d/%d auctions", fixed, len(toFix))
}

// backfillRegistryCoordinates parses legal_description for every row that has
// a NULL registry_book and writes the book/page integers.  The query is keyed
// solely on registry_book IS NULL so it surfaces all 232+ legacy rows even
// when legal_description was never populated (those are skipped in Go).
// The previous query required legal_description IS NOT NULL, which silently
// returned 0 rows whenever the legal text was missing, hiding the gap.
func backfillRegistryCoordinates(ctx context.Context, db *sql.DB) {
	const sel = `
		SELECT id, site_name, legal_description
		FROM   auctions
		WHERE  registry_book IS NULL`

	rows, err := db.QueryContext(ctx, sel)
	if err != nil {
		log.Printf("[backfill-coords] query error: %v", err)
		return
	}
	defer rows.Close()

	type candidate struct {
		id       int
		siteName string
		legal    sql.NullString
	}
	var candidates []candidate
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.id, &c.siteName, &c.legal); err == nil {
			candidates = append(candidates, c)
		}
	}
	rows.Close()

	log.Printf("[backfill-coords] found %d rows with NULL registry_book", len(candidates))

	fixed, skipped := 0, 0
	for _, c := range candidates {
		if !c.legal.Valid || c.legal.String == "" {
			skipped++
			continue
		}
		book, page := ExtractBookPage(c.legal.String)
		if book == "" || page == "" {
			skipped++
			continue
		}
		_, err := db.ExecContext(ctx,
			`UPDATE auctions SET registry_book=$1, registry_page=$2 WHERE id=$3`,
			nullIfZero(atoiSafe(book)), nullIfZero(atoiSafe(page)), c.id)
		if err != nil {
			log.Printf("[backfill-coords] update error id=%d: %v", c.id, err)
		} else {
			log.Printf("[backfill] found match in id=%d (%s): Book %s, Page %s", c.id, c.siteName, book, page)
			fixed++
		}
	}
	log.Printf("[backfill-coords] complete: updated %d/%d rows (%d skipped — no legal text or no match)",
		fixed, len(candidates), skipped)
}

// backfillRegistryDeepLinks resolves the specific deed document URL for every
// row that has registry book/page coordinates but no deep link yet.
// It uses CFetch (Cloudflare Browser Rendering) to render the LAREDO
// hash-fragment search URL so the SPA routing and AJAX settle before we parse.
// Processes at most 25 rows per invocation to cap CF quota usage; subsequent
// runs will handle remaining rows.  Runs as a background goroutine.
func BackfillRegistryDeepLinks(db *sql.DB) {
	const maxPerRun = 25
	const sel = `
		SELECT id, registry_url, registry_book, registry_page
		FROM   auctions
		WHERE  registry_book  IS NOT NULL
		  AND  registry_page  IS NOT NULL
		  AND  registry_deep_link IS NULL
		  AND  registry_url   IS NOT NULL
		  AND  registry_url NOT LIKE '%barnstabledeeds%'
		LIMIT $1`

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	rows, err := db.QueryContext(ctx, sel, maxPerRun)
	if err != nil {
		log.Printf("[backfill-deeplink] query error: %v", err)
		return
	}

	type candidate struct {
		id          int
		registryURL string
		book        int
		page        int
	}
	var candidates []candidate
	for rows.Next() {
		var c candidate
		var book, page sql.NullInt64
		if err := rows.Scan(&c.id, &c.registryURL, &book, &page); err == nil {
			c.book = int(book.Int64)
			c.page = int(page.Int64)
			candidates = append(candidates, c)
		}
	}
	rows.Close()

	if len(candidates) == 0 {
		log.Printf("[backfill-deeplink] all rows already have deep links — nothing to do")
		return
	}
	log.Printf("[backfill-deeplink] resolving %d deed URL(s) via CF Browser Rendering", len(candidates))

	found, skipped := 0, 0
	for _, c := range candidates {
		base := strings.TrimRight(c.registryURL, "/")
		hashURL := fmt.Sprintf(
			"%s/#page=searchEntry&searchtype=Standard&recordtype=MR&book=%d&bookpage=%d",
			base, c.book, c.page,
		)
		log.Printf("[backfill-deeplink] id=%d fetching %s", c.id, hashURL)

		html, err := sites.CFetch(ctx, hashURL)
		if err != nil {
			log.Printf("[backfill-deeplink] id=%d cfetch error: %v", c.id, err)
			skipped++
			continue
		}

		docURL := ExtractDocumentURL(html, base)
		if docURL == "" {
			log.Printf("[backfill-deeplink] id=%d no document link found in rendered HTML (%d bytes)", c.id, len(html))
			skipped++
			continue
		}

		_, err = db.ExecContext(ctx,
			`UPDATE auctions SET registry_deep_link=$1 WHERE id=$2`, docURL, c.id)
		if err != nil {
			log.Printf("[backfill-deeplink] id=%d db update error: %v", c.id, err)
		} else {
			log.Printf("[backfill-deeplink] id=%d deep link stored: %s", c.id, docURL)
			found++
		}
	}
	log.Printf("[backfill-deeplink] complete: found=%d skipped=%d", found, skipped)
}
