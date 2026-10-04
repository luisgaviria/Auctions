package utils

import (
	"backendAuction/utils/sites"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"
)

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

// depositCleanRe strips all non-numeric characters from a deposit string.
var depositCleanRe = regexp.MustCompile(`[^0-9]`)

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
	a.Deposit = depositCleanRe.ReplaceAllString(a.Deposit, "")

	// Extract Book/Page coordinates from the legal description and store the
	// raw integers.  The frontend builds the full URL dynamically from the
	// district base URL (registry_url) + these coordinates.
	if a.LegalDescription != "" {
		book, page := ExtractBookPage(a.LegalDescription)
		a.RegistryBook = atoiSafe(book)
		a.RegistryPage = atoiSafe(page)
		if book != "" && page != "" {
			log.Printf("[book-page] extracted Book=%s Page=%s addr=%q", book, page, a.Street)
		}
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
