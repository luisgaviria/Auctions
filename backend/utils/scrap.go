package utils

import (
	"backendAuction/utils/sites"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	_ "time/tzdata" // embed IANA timezone DB so LoadLocation works on any host
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
		 legal_description, registry_deep_link, assessor_pid)
	VALUES
		($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW(), $16, $17, $18,
		 $19, $20, $21)
	ON CONFLICT ON CONSTRAINT uq_auctions_address_site DO UPDATE
		SET
			status            = CASE WHEN auctions.status  IS DISTINCT FROM EXCLUDED.status
			                         THEN EXCLUDED.status  ELSE auctions.status  END,
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
			last_seen         = EXCLUDED.last_seen,
			updated_at        = NOW()
	RETURNING id;`

// sweepStaleSQL hard-deletes rows that have not been seen by any scraper in the
// last 6 hours.  The 6-hour window means a single transient scraper failure
// leaves data intact; two consecutive missed runs signal genuine removal.
var sweepStaleSQL = `
	DELETE FROM auctions
	WHERE last_seen < NOW() - INTERVAL '6 hours';`

// easternLoc is America/New_York, used for all "is this date in the past?"
// comparisons so the server's own timezone (often UTC on Railway) does not
// cause early-morning scrape runs to see yesterday's date.
// Falls back to UTC and logs a warning if the tz database is unavailable.
var easternLoc = func() *time.Location {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Printf("[tz] America/New_York unavailable, falling back to UTC: %v", err)
		return time.UTC
	}
	return loc
}()

// cleanAddrRe matches boilerplate fragments that scrapers embed in raw address
// strings: deed-registry citations (B123/P456), free-text clauses starting
// with Mortgage/Registry/Book/View property, and parenthetical qualifiers
// like "(CHESTNUT HILL)" that some scrapers append to city names.
var cleanAddrRe = regexp.MustCompile(
	`(?i)` +
		`\([^)]*\)` + // parenthetical qualifiers, e.g. (CHESTNUT HILL)
		`|,?\s*(?:B\d+/P\d+\S*` + // deed-registry book/page refs
		`|(?:Mortgage|Registry|Book)\b[^,\n]*` + // clause keywords
		`|View\s*property[^\n]*` + // "View property…" trailers
		`|Pre-?\s*Register[^\n]*` + // "Pre-Register" / "Pre Register"
		`|AS\s+SCHEDULED[^\n]*)`, // "AS SCHEDULED" status text
)

// gluedDateRe matches a month name glued directly onto a state abbreviation
// with no space — e.g. "MAApr" or "MAJan".  DanielP's <a> element text
// concatenates the auction date onto the address without a separator.
// The replacement keeps only the state abbreviation (group $1).
var gluedDateRe = regexp.MustCompile(
	`(?i)(MA|MASSACHUSETTS)(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\w*[\s,\d]*`,
)

// CleanStreetAddress removes boilerplate from a raw scraper address and trims
// trailing punctuation.  Applied to every auction's Street field before the
// DB upsert so no scraper-specific caller needs to handle it.
func CleanStreetAddress(s string) string {
	// 1. Strip glued date text after state abbreviation (DanielP scraper).
	//    "90 SUFFOLK ROAD, NEWTON , MAApr 7, 2026" → "90 SUFFOLK ROAD, NEWTON , MA"
	s = gluedDateRe.ReplaceAllString(s, "$1")

	// 2. Strip other boilerplate (parentheticals, deed refs, junk phrases).
	s = cleanAddrRe.ReplaceAllString(s, "")

	// 3. Collapse space-before-comma: "NEWTON , MA" → "NEWTON, MA"
	s = regexp.MustCompile(`\s+,`).ReplaceAllString(s, ",")

	return strings.TrimRight(strings.TrimSpace(s), ",;- ")
}

// urlJunkRe strips any remaining boilerplate that CleanStreetAddress may not
// catch when the address is used only for URL generation (not stored in the DB).
// Handles edge cases like trailing state abbreviations in parentheses.
var urlJunkRe = regexp.MustCompile(`(?i)\([^)]*\)|\bMA\s*,?\s*MA\b`)

// GenerateSlug converts a display string into a URL-safe slug.
//
//	"Jamaica Plain"   → "jamaica-plain"
//	"Worcester County" → "worcester-county"
//	"  North Adams  " → "north-adams"
func GenerateSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevHyphen := false
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevHyphen = false
		case r == ' ' || r == '-' || r == '_':
			if !prevHyphen {
				b.WriteRune('-')
				prevHyphen = true
			}
		}
		// all other characters (apostrophes, commas, etc.) are dropped
	}
	return strings.Trim(b.String(), "-")
}

// maTokenRe matches a comma-segment that is exactly the MA state abbreviation,
// optionally followed by a zip code.  It deliberately does NOT match
// "MASSACHUSETTS" so that SRI's "…, MA, MASSACHUSETTS" format skips the
// redundant long-form token and correctly lands on the preceding "MA".
var maTokenRe = regexp.MustCompile(`(?i)^MA(\s+\d{5}(-\d{4})?)?$`)

// cityLegalStripRe removes legal-description boilerplate that leaks into the
// city field when scrapers embed Book/Page text on the same line as the address.
// e.g. "MA, PAGE 98" → "MA", "MILLBURY, Book 45231" → "MILLBURY"
var cityLegalStripRe = regexp.MustCompile(`(?i)[,\s]+(Recorded\b|See\s+Mortgage|Book\s+\d|Bk\.?\s*\d|Page\s+\d|Pg\.?\s*\d|B\d{4,}\b).*$`)

// addrTitleCase returns s in title case, e.g. "HAVERHILL" → "Haverhill".
// Used only for city/street display — stored fields are already uppercased by
// NormalizeStreet so this is called before that step.
func addrTitleCase(s string) string {
	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// CleanAuctionData resolves the correct street and city from a raw scraper
// address when the city field has been set to a generic "Massachusetts"
// placeholder.
//
// All three address formats used by MA auctioneers share the same structure —
// the real city always sits one comma-segment to the left of the "MA" token —
// so a single parser covers every case.  A per-auctioneer switch would add
// maintenance overhead with no benefit.
//
//	Dean:    "1-7 5TH AVENUE, HAVERHILL, MA, PAGE 336"   → "Haverhill"
//	DanielP: "1600 WEST STREET, STOUGHTON, MA"           → "Stoughton"
//	SRI:     "174 HAMILTON STREET, SOUTHBRIDGE, MA, MASSACHUSETTS" → "Southbridge"
//
// Returns the street (everything before the city segment) and city in title
// case.  Falls back to rawAddr/rawCity unchanged if no MA token is found.
func CleanAuctionData(rawAddr, rawCity string) (street, city string) {
	trimCity := strings.TrimSpace(rawCity)

	// Strip legal-description boilerplate that may have leaked into the city field.
	trimCity = strings.TrimSpace(cityLegalStripRe.ReplaceAllString(trimCity, ""))

	// If rawCity is already a specific town, honour it and just strip the
	// city+state tail from the address so the street field stays clean.
	isFallback := strings.EqualFold(trimCity, "massachusetts") ||
		strings.EqualFold(trimCity, "ma") ||
		trimCity == ""

	parts := strings.Split(rawAddr, ",")

	// Find the index of the "MA" state token scanning right-to-left.
	maIdx := -1
	for i := len(parts) - 1; i >= 1; i-- {
		if maTokenRe.MatchString(strings.TrimSpace(parts[i])) {
			maIdx = i
			break
		}
	}

	if maIdx <= 0 {
		// No MA token found — return inputs as-is.
		return addrTitleCase(strings.TrimSpace(rawAddr)), addrTitleCase(trimCity)
	}

	parsedCity := addrTitleCase(strings.TrimSpace(parts[maIdx-1]))
	parsedStreet := addrTitleCase(strings.TrimSpace(strings.Join(parts[:maIdx-1], ",")))

	if isFallback {
		return parsedStreet, parsedCity
	}
	// City was already known — just strip the city+state tail from the street.
	return parsedStreet, addrTitleCase(trimCity)
}

// streetRangeRe matches a leading street-number range like "61-63" or "2-4"
// and captures only the first number.  Zillow search cannot resolve ranges —
// using the lower number produces a valid result for the property.
var streetRangeRe = regexp.MustCompile(`^(\d+)-\d+\b`)

// NormalizeAddressForURL returns a URL-safe version of the address string:
// parentheticals stripped, known junk phrases removed, commas dropped, and
// consecutive whitespace collapsed to single spaces.  The result is suitable
// for path-encoding (replace spaces with "+" or "-" depending on the target).
//
// Examples:
//
//	"90 SUFFOLK ROAD, NEWTON (CHESTNUT HILL), MA" → "90 SUFFOLK ROAD NEWTON MA"
//	"59 COCHATO PARK, RANDOLPHPre-Register, MA"  → "59 COCHATO PARK RANDOLPH MA"
func NormalizeAddressForURL(address string) string {
	s := cleanAddrRe.ReplaceAllString(address, "")
	s = urlJunkRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, ",", "")
	// Collapse any run of whitespace (gaps left by removed tokens) to one space.
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}

// streetAbbrevs expands common US street-suffix abbreviations to their full
// form.  Each regex only fires when the abbreviation sits at the end of the
// string or immediately before a comma, so "ST JAMES ST" becomes
// "ST JAMES STREET" (the leading "ST" is not a suffix and is left alone).
// Patterns are applied after uppercasing, so matching is always uppercase.
var streetAbbrevs = []struct {
	re   *regexp.Regexp
	full string
}{
	{regexp.MustCompile(`\bST(,|$)`), "STREET$1"},
	{regexp.MustCompile(`\bRD(,|$)`), "ROAD$1"},
	{regexp.MustCompile(`\bAVE(,|$)`), "AVENUE$1"},
	{regexp.MustCompile(`\bBLVD(,|$)`), "BOULEVARD$1"},
	{regexp.MustCompile(`\bDR(,|$)`), "DRIVE$1"},
	{regexp.MustCompile(`\bLN(,|$)`), "LANE$1"},
	{regexp.MustCompile(`\bCT(,|$)`), "COURT$1"},
	{regexp.MustCompile(`\bPL(,|$)`), "PLACE$1"},
	{regexp.MustCompile(`\bTERR?(,|$)`), "TERRACE$1"},
	{regexp.MustCompile(`\bHWY(,|$)`), "HIGHWAY$1"},
	{regexp.MustCompile(`\bCIR(,|$)`), "CIRCLE$1"},
	{regexp.MustCompile(`\bPKWY(,|$)`), "PARKWAY$1"},
}

// NormalizeStreet uppercases s and expands suffix abbreviations so that
// "47 Foss Rd" and "47 FOSS ROAD" resolve to the same conflict key.
func NormalizeStreet(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	for _, abbr := range streetAbbrevs {
		s = abbr.re.ReplaceAllString(s, abbr.full)
	}
	return s
}

// dateLayouts is the ordered list of formats that any scraper or Gemini rescue
// call can produce.  Layouts are tried in order; the first match wins.
// "2006-01-02" is listed first so already-normalized dates are free.
var dateLayouts = []string{
	"2006-01-02",      // ISO-8601 (DanielP, Dean after normalization)
	"Jan 2, 2006",     // Gemini output, Commonwealth ("Apr 2, 2026")
	"January 2, 2006", // long month (Dean raw, some SRI)
	"01/02/2006",      // zero-padded MM/DD/YYYY (SRI)
	"1/2/2006",        // single-digit M/D/YYYY (SRI, DanielP raw)
	"1/02/2006",       // mixed single-month + zero-day
	"01/2/2006",       // mixed zero-month + single-day
}

// normalizeDateToISO converts any date string a scraper or Gemini may produce
// into ISO-8601 "2006-01-02".  Returns the original string unchanged if no
// layout matches so the upsert still runs (PostgreSQL will reject invalid
// DATE literals and the error will surface in the [upsert] log line).
func normalizeDateToISO(s string) string {
	if s == "" {
		return ""
	}
	// Title-case normalises "AUGUST 13, 2025" → "August 13, 2025" and
	// "APR 2, 2026" → "Apr 2, 2026" so time.Parse layouts match.
	normalized := strings.Title(strings.ToLower(strings.TrimSpace(s))) //nolint:staticcheck
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, normalized); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return s
}

// buildAuctionURLs populates the three generated URL fields for an auction.
//
// Zillow search URL format:
//
//	https://www.zillow.com/homes/STREET-CITY-MA/
//
// Spaces are replaced with dashes, commas are stripped, and the city is only
// appended when non-empty so an unknown city never produces a double-dash.
// Example: "91 GENEVA AVE", "DORCHESTER" → "91-GENEVA-AVE-DORCHESTER-MA"
//
// Google Maps "Search" URL (no API key required, opens the search result):
//
//	https://www.google.com/maps/search/?api=1&query=STREET,+CITY,+MA
//
// Registry URL is looked up from the MA Registry of Deeds map by city.
func buildAuctionURLs(street, city string) (zillowURL, streetViewURL, registryURL string) {
	// Build the full address string and sanitize it for use in URLs.
	// NormalizeAddressForURL strips parentheticals, junk text, and commas so
	// values like "90 SUFFOLK ROAD, NEWTON (CHESTNUT HILL), MA" become clean.
	rawFull := street
	if city != "" {
		rawFull += " " + city
	}
	rawFull += " MA"
	clean := NormalizeAddressForURL(rawFull) // "90 SUFFOLK ROAD NEWTON MA"

	// Zillow path format: spaces → "+", no commas, "_rb/" suffix for address search.
	// Example: https://www.zillow.com/homes/90+SUFFOLK+ROAD+NEWTON+MA_rb/
	// Street-number ranges (e.g. "61-63") are reduced to the first number so
	// Zillow can resolve the search — "61-63+DEXTER+ST" finds nothing.
	zillowClean := streetRangeRe.ReplaceAllString(clean, "$1")
	zillowURL = "https://www.zillow.com/homes/" +
		strings.ReplaceAll(zillowClean, " ", "+") + "_rb/"

	// Google Maps uses standard percent-encoding for the query parameter.
	mapsQuery := street
	if city != "" {
		mapsQuery += ", " + city
	}
	mapsQuery += ", MA"
	streetViewURL = "https://www.google.com/maps/search/?api=1&query=" +
		url.QueryEscape(mapsQuery)

	registryURL = GetRegistryURL(city)
	return
}

// BuildAuctionURLsPublic is the exported wrapper around buildAuctionURLs.
func BuildAuctionURLsPublic(street, city string) (zillowURL, streetViewURL, registryURL string) {
	return buildAuctionURLs(street, city)
}

// NormalizeAuction applies every address-cleaning and normalization step in
// one place so the result is identical whether called from ScrapAllSites,
// DryRunCFScrapers, or any future caller.  The returned copy is safe to pass
// directly to upsertAuction.
func NormalizeAuction(a sites.Auction) sites.Auction {
	a.Street = CleanStreetAddress(strings.TrimSpace(a.Street))

	// Resolve city from the address when the scraper set a generic placeholder.
	// CleanAuctionData is called before NormalizeStreet so title-cased output
	// is then uppercased uniformly by NormalizeStreet below.
	a.Street, a.City = CleanAuctionData(a.Street, a.City)

	a.Street = NormalizeStreet(a.Street)
	a.City = strings.TrimSpace(a.City)
	a.Date = normalizeDateToISO(a.Date)
	a.Status = normalizeStatus(a.Status)
	a.Time = normalizeTime(a.Time)
	a.ZillowURL, a.StreetViewURL, a.RegistryURL = buildAuctionURLs(a.Street, a.City)

	// Build registry deep link if the scraper provided a legal description.
	if a.LegalDescription != "" {
		book, page := ExtractBookPage(a.LegalDescription)
		a.RegistryDeepLink = BuildRegistryDeepLink(a.City, book, page)
	}

	a.CitySlug = GenerateSlug(a.City)
	county := GetCounty(a.City)
	if county != "" {
		a.CountySlug = GenerateSlug(county + " County")
	}
	// Address slug combines street + city for human-readable shareable URLs:
	// "123 MAIN STREET" + "Worcester" → "123-main-street-worcester"
	a.AddressSlug = GenerateSlug(a.Street + " " + a.City)
	return a
}

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

// todayET returns midnight of the current day in America/New_York.
func todayET() time.Time {
	now := time.Now().In(easternLoc)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, easternLoc)
}

// isPastDate returns true when dateStr represents a date strictly before
// today in America/New_York.  time.Parse returns a UTC time so we rebuild
// the calendar date in ET before comparing.
// Unrecognised formats return false so the record is still saved.
func isPastDate(dateStr string) bool {
	if dateStr == "" {
		return false
	}
	normalized := strings.Title(strings.ToLower(strings.TrimSpace(dateStr))) //nolint:staticcheck
	today := todayET()
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, normalized); err == nil {
			// Rebuild in ET: time.Parse returns UTC regardless of input.
			day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, easternLoc)
			return day.Before(today)
		}
	}
	return false
}

// timeLayouts covers every time format the scrapers and Gemini produce.
var timeLayouts = []string{
	"3:04 PM", "3:04 pm", "3:04PM", "3:04pm",
	"3:04 AM", "3:04 am", "3:04AM", "3:04am",
	"15:04",
}

// isElapsedToday returns true when the auction date is today in ET AND the
// listed start time has already passed in ET.  This skips same-day auctions
// that have already occurred (e.g. a 10 AM auction seen at 8 PM).
// Returns false when either string is empty, the date is not today, or the
// time string cannot be parsed — the auction is kept in those ambiguous cases.
func isElapsedToday(dateStr, timeStr string) bool {
	if dateStr == "" || timeStr == "" {
		return false
	}
	normalized := strings.Title(strings.ToLower(strings.TrimSpace(dateStr))) //nolint:staticcheck
	today := todayET()
	nowET := time.Now().In(easternLoc)

	var auctionDay time.Time
	found := false
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, normalized); err == nil {
			auctionDay = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, easternLoc)
			found = true
			break
		}
	}
	if !found || !auctionDay.Equal(today) {
		return false
	}

	clean := strings.TrimSpace(timeStr)
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, clean); err == nil {
			auctionDT := time.Date(nowET.Year(), nowET.Month(), nowET.Day(),
				t.Hour(), t.Minute(), 0, 0, easternLoc)
			return auctionDT.Before(nowET)
		}
	}
	return false
}

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
		a.Street,        // $1  address
		a.City,          // $2  city
		"Massachusetts", // $3  state
		a.Time,          // $4  time
		a.Logo,          // $5  logo
		a.SiteName,      // $6  site_name
		a.Status,        // $7  status
		a.Url,           // $8  link
		dateParam,       // $9  date
		a.Deposit,       // $10 deposit
		"0",             // $11 lat
		"0",             // $12 lng
		a.ZillowURL,           // $13 zillow_url
		a.StreetViewURL,       // $14 street_view_url
		a.RegistryURL,         // $15 registry_url
		a.CitySlug,            // $16 city_slug
		a.CountySlug,          // $17 county_slug
		a.AddressSlug,         // $18 address_slug
		nullIfEmpty(a.LegalDescription),  // $19 legal_description
		nullIfEmpty(a.RegistryDeepLink),  // $20 registry_deep_link
		nil,                              // $21 assessor_pid (set by second-pass)
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
//   • ISO dates (YYYY-MM-DD)  → cast to date and compare normally.
//   • Any other format        → treated as CURRENT_DATE - 1 day, which is
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

// backfillRegistryDeepLinks computes registry_deep_link for any row that has a
// legal_description but no registry_deep_link yet.  Runs at scrape startup so
// rows scraped before this feature was deployed get their deep links populated.
func backfillRegistryDeepLinks(ctx context.Context, db *sql.DB) {
	const sel = `
		SELECT id, legal_description, city FROM auctions
		WHERE legal_description IS NOT NULL
		  AND legal_description != ''
		  AND (registry_deep_link IS NULL OR registry_deep_link = '')`

	rows, err := db.QueryContext(ctx, sel)
	if err != nil {
		log.Printf("[backfill-deeplinks] query error: %v", err)
		return
	}
	defer rows.Close()

	type row struct {
		id    int
		legal string
		city  string
	}
	var toFix []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.legal, &r.city); err == nil {
			toFix = append(toFix, r)
		}
	}
	rows.Close()

	fixed := 0
	for _, r := range toFix {
		book, page := ExtractBookPage(r.legal)
		link := BuildRegistryDeepLink(r.city, book, page)
		if link == "" {
			continue // book/page not found in this legal description
		}
		_, err := db.ExecContext(ctx,
			`UPDATE auctions SET registry_deep_link=$1 WHERE id=$2`,
			link, r.id)
		if err != nil {
			log.Printf("[backfill-deeplinks] update id=%d: %v", r.id, err)
		} else {
			fixed++
		}
	}
	log.Printf("[backfill-deeplinks] populated deep links for %d/%d auctions", fixed, len(toFix))
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
	// NOTE: backfillRegistryDeepLinks and RunAssessorSecondPass run in the
	// enrichment phase AFTER all 12 scrapers have committed their data.

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

	// ── Enrichment phase ────────────────────────────────────────────────────────
	// Uses a FRESH context derived from Background(), NOT from the caller's ctx.
	// This means the enrichment is never killed by the scrape deadline — even if
	// the dryrun or HTTP handler's 60-minute timer fires, enrichment continues
	// independently.  Row-by-row commits mean any partial run is preserved.
	enrichCtx, enrichCancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer enrichCancel()

	// Level 2 (mortgage): compute registry deep links from scraped legal_description.
	// Fast — pure DB read + string math, no external HTTP.
	backfillRegistryDeepLinks(enrichCtx, db)

	// Level 1 (deed) + assessor PID: hit VGSI portals for rows still missing data.
	RunAssessorSecondPass(enrichCtx, db)

	log.Printf("[scraper] run complete")
}
