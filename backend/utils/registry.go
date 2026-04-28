package utils

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ── Book / Page extraction ────────────────────────────────────────────────────

// bookPageRe covers the common formats found in MA foreclosure notices:
//
//   SRI compact:     "B13812/P506"  or  ", B13812/P506"
//   Prose format:    "Book 13812, Page 506"  or  "Book 13812 Page 506"
//   Abbreviated:     "Bk 13812 Pg 506"  or  "Bk/Pg: 13812/506"
//   Numeric slash:   "13812/506"  (when preceded by "Book" context — not matched alone)
//
// Group 1 = book number, Group 2 = page number.
var bookPageRe = regexp.MustCompile(
	`(?i)` +
		`(?:B(?:ook)?\.?\s*|Bk\.?\s*)` + // "B", "Book", "Bk" (with optional dot/space)
		`(\d+)` + // group 1: book number
		`(?:\s*[/,]\s*|\s+)` + // separator: "/" or "," or whitespace
		`(?:P(?:age|g)?\.?\s*)?` + // optional "P", "Page", "Pg" prefix on page
		`(\d+)`, // group 2: page number
)

// ExtractBookPage parses a legal description string and returns the
// Book and Page numbers as strings, or empty strings if not found.
func ExtractBookPage(legal string) (book, page string) {
	m := bookPageRe.FindStringSubmatch(legal)
	if len(m) < 3 {
		return "", ""
	}
	return m[1], m[2]
}

// ── Registry deep-link builder ────────────────────────────────────────────────

// BuildRegistryDeepLink returns a masslandrecords.com URL that opens the
// Book/Page search directly for the given city's registry district.
// Returns "" when the city is not mapped or when book is empty.
//
// Verified URL format (Worcester district, Book 61467 Page 24):
//   https://www.masslandrecords.com/Worcester/Default.aspx?p=BookSearch&b=61467&pg=24
//
// City may include a state suffix ("Southbridge, Massachusetts") — normalised
// automatically before the registry-district lookup.
// page is optional: when empty, a book-only search URL is returned.
func BuildRegistryDeepLink(city, book, page string) string {
	if book == "" {
		return ""
	}
	base := GetRegistryURL(city)
	// GetRegistryURL returns the generic portal when city is unknown — skip those.
	if base == "" || base == "https://www.masslandrecords.com" {
		return ""
	}
	// Barnstable uses its own portal — query-string format not available.
	if strings.EqualFold(GetCounty(city), "Barnstable") {
		return ""
	}
	if page == "" {
		return fmt.Sprintf("%s/Default.aspx?p=BookSearch&b=%s", base, book)
	}
	return fmt.Sprintf("%s/Default.aspx?p=BookSearch&b=%s&pg=%s", base, book, page)
}

// ── Vision Government Solutions (VGSI) assessor portal ───────────────────────

// vgsiTowns maps MA city/town names (UPPERCASE) to their VGSI base URL.
// Pattern is gis.vgsi.com/{town}ma — Worcester is the anomaly ("worcestma").
// Verify new entries at https://gis.vgsi.com/{town}ma/Search.aspx before adding.
var vgsiTowns = map[string]string{
	// ── Bristol County ───────────────────────────────────────────────────────
	"ATTLEBORO":         "https://gis.vgsi.com/attleboroma",
	"DARTMOUTH":         "https://gis.vgsi.com/dartmouthma",
	"DIGHTON":           "https://gis.vgsi.com/dightonma",
	"FALL RIVER":        "https://gis.vgsi.com/fallriverma",
	"MANSFIELD":         "https://gis.vgsi.com/mansfieldma",
	"NEW BEDFORD":       "https://gis.vgsi.com/newbedfordma",
	"NORTH ATTLEBOROUGH": "https://gis.vgsi.com/northattleboroughma",
	"NORTON":            "https://gis.vgsi.com/nortonma",
	"RAYNHAM":           "https://gis.vgsi.com/raynhamma",
	"SEEKONK":           "https://gis.vgsi.com/seekonkma",
	"TAUNTON":           "https://gis.vgsi.com/tauntonma",
	"WESTPORT":          "https://gis.vgsi.com/westportma",

	// ── Essex County ─────────────────────────────────────────────────────────
	"ANDOVER":           "https://gis.vgsi.com/andoverma",
	"BEVERLY":           "https://gis.vgsi.com/beverlyma",
	"GLOUCESTER":        "https://gis.vgsi.com/gloucesterma",
	"HAVERHILL":         "https://gis.vgsi.com/haverhillma",
	"LYNN":              "https://gis.vgsi.com/lynnma",
	"METHUEN":           "https://gis.vgsi.com/methuenma",
	"PEABODY":           "https://gis.vgsi.com/peabodyma",
	"SALEM":             "https://gis.vgsi.com/salemma",

	// ── Hampden County ───────────────────────────────────────────────────────
	"AGAWAM":            "https://gis.vgsi.com/agawamma",
	"CHICOPEE":          "https://gis.vgsi.com/chicopeema",
	"HOLYOKE":           "https://gis.vgsi.com/holyokema",
	"LUDLOW":            "https://gis.vgsi.com/ludlowma",
	"PALMER":            "https://gis.vgsi.com/palmerma",
	"SPRINGFIELD":       "https://gis.vgsi.com/springfieldma",
	"WEST SPRINGFIELD":  "https://gis.vgsi.com/westspringfieldma",
	"WESTFIELD":         "https://gis.vgsi.com/westfieldma",

	// ── Middlesex County ─────────────────────────────────────────────────────
	"BELMONT":           "https://gis.vgsi.com/belmontma",
	"CHELMSFORD":        "https://gis.vgsi.com/chelmsfordma",
	"DRACUT":            "https://gis.vgsi.com/dracutma",
	"FRAMINGHAM":        "https://gis.vgsi.com/framinghamma",
	"LOWELL":            "https://gis.vgsi.com/lowellma",
	"MALDEN":            "https://gis.vgsi.com/maldenma",
	"MARLBOROUGH":       "https://gis.vgsi.com/marlboroughma",
	"MEDFORD":           "https://gis.vgsi.com/medfordma",
	"SOMERVILLE":        "https://gis.vgsi.com/somervillema",
	"WALTHAM":           "https://gis.vgsi.com/walthamma",
	"WATERTOWN":         "https://gis.vgsi.com/watertownma",
	"WOBURN":            "https://gis.vgsi.com/woburnma",

	// ── Norfolk County ───────────────────────────────────────────────────────
	"BRAINTREE":         "https://gis.vgsi.com/braintrema",
	"NEEDHAM":           "https://gis.vgsi.com/needhamma",
	"QUINCY":            "https://gis.vgsi.com/quincyma",
	"RANDOLPH":          "https://gis.vgsi.com/randolphma",

	// ── Plymouth County ──────────────────────────────────────────────────────
	"BRIDGEWATER":       "https://gis.vgsi.com/bridgewaterma",
	"BROCKTON":          "https://gis.vgsi.com/brocktonma",
	"EAST BRIDGEWATER":  "https://gis.vgsi.com/eastbridgewaterma",
	"MARSHFIELD":        "https://gis.vgsi.com/marshfieldma",
	"PLYMOUTH":          "https://gis.vgsi.com/plymouthma",
	"ROCKLAND":          "https://gis.vgsi.com/rocklandma",
	"WHITMAN":           "https://gis.vgsi.com/whitmanma",

	// ── Suffolk County ───────────────────────────────────────────────────────
	"REVERE":            "https://gis.vgsi.com/reverema",

	// ── Worcester County (South district) ────────────────────────────────────
	"AUBURN":            "https://gis.vgsi.com/auburnma",
	"BLACKSTONE":        "https://gis.vgsi.com/blackstonema",
	"CHARLTON":          "https://gis.vgsi.com/charltonma",
	"DOUGLAS":           "https://gis.vgsi.com/douglasma",
	"DUDLEY":            "https://gis.vgsi.com/dudleyma",
	"GRAFTON":           "https://gis.vgsi.com/graftonma",
	"MILFORD":           "https://gis.vgsi.com/milfordma",
	"MILLBURY":          "https://gis.vgsi.com/millburyma",
	"NORTHBRIDGE":       "https://gis.vgsi.com/northbridgema",
	"OXFORD":            "https://gis.vgsi.com/oxfordma",
	"SHREWSBURY":        "https://gis.vgsi.com/shrewsburyma",
	"SOUTHBRIDGE":       "https://gis.vgsi.com/southbridgema",
	"STURBRIDGE":        "https://gis.vgsi.com/sturbridgema",
	"SUTTON":            "https://gis.vgsi.com/suttonma",
	"UXBRIDGE":          "https://gis.vgsi.com/uxbridgema",
	"WEBSTER":           "https://gis.vgsi.com/websterma",
	"WORCESTER":         "https://gis.vgsi.com/worcestma",

	// ── Worcester County (North district) ────────────────────────────────────
	"FITCHBURG":         "https://gis.vgsi.com/fitchburgma",
	"HOLDEN":            "https://gis.vgsi.com/holdenma",
	"LEOMINSTER":        "https://gis.vgsi.com/leominster",
	"LEICESTER":         "https://gis.vgsi.com/leicesterma",
	"NORTHAMPTON":       "https://gis.vgsi.com/northamptonma",
	"PAXTON":            "https://gis.vgsi.com/paxtonma",
	"RUTLAND":           "https://gis.vgsi.com/rutlandma",
	"SPENCER":           "https://gis.vgsi.com/spencerma",
	"STERLING":          "https://gis.vgsi.com/sterlingma",
	"WARE":              "https://gis.vgsi.com/warema",
}

// GetVGSIBaseURL returns the VGSI base URL for a MA city, or "" if the city
// is not in the map.  Lookup is case-insensitive; state suffixes like
// ", Massachusetts" are stripped automatically.
func GetVGSIBaseURL(city string) string {
	return vgsiTowns[normalizeCity(city)]
}

// BuildAssessorURL constructs the full VGSI parcel detail URL from a PID.
// Returns "" when the city is not in the VGSI map or pid is empty.
func BuildAssessorURL(city, pid string) string {
	base := GetVGSIBaseURL(city)
	if base == "" || pid == "" {
		return ""
	}
	return base + "/Parcel.aspx?pid=" + url.QueryEscape(pid)
}

// ── VGSI parcel ID scraper ────────────────────────────────────────────────────

// pidRe extracts the numeric PID from a VGSI Parcel.aspx URL.
var pidRe = regexp.MustCompile(`(?i)[?&]pid=(\d+)`)

// desktopUA is a modern Chrome user-agent string.  Government portals (VGSI)
// often block obvious bot strings like "AuctionBot/1.0".
const desktopUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

// vgsiDisclaimerCookies lists every cookie name VGSI portals use to gate the
// disclaimer page.  Pre-seeding all of them bypasses the splash screen so the
// search form is returned directly on the first request.
var vgsiDisclaimerCookies = []*http.Cookie{
	{Name: "VisionDisclaimer", Value: "accepted", Path: "/"},
	{Name: "vicisDisclaimer",  Value: "true",     Path: "/"},
	{Name: "disclaimer",       Value: "true",     Path: "/"},
	{Name: "Disclaimer",       Value: "accepted", Path: "/"},
}

// newVGSIClient creates a per-call HTTP client with its own cookie jar.
// Disclaimer cookies are pre-seeded for the given base URL so the portal
// serves the search form directly instead of a splash/disclaimer page.
// A modern desktop User-Agent is set on every request via the header helpers
// below; this client intentionally does NOT set a global Transport UA.
func newVGSIClient(baseURL string) *http.Client {
	jar, _ := cookiejar.New(nil)
	if u, err := url.Parse(baseURL); err == nil {
		jar.SetCookies(u, vgsiDisclaimerCookies)
	}
	return &http.Client{
		Timeout: 15 * time.Second,
		Jar:     jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
}

// vgsiHeaders adds standard browser headers to a request so it is
// indistinguishable from a normal Chrome GET/POST.
func vgsiHeaders(req *http.Request, referer string) {
	req.Header.Set("User-Agent", desktopUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
}

// vgsiRangeRe matches a leading street-number range (e.g. "193-195").
// Group 1 captures only the first number for the simplified retry.
var vgsiRangeRe = regexp.MustCompile(`^(\d+)-\d+\b`)

// vgsiUnitRe matches common unit suffixes: "APT 2", "UNIT B", "#3", etc.
var vgsiUnitRe = regexp.MustCompile(`(?i)[,\s]+(?:apt|unit|ste|suite|#|fl|floor)\s*[\w-]+\s*$`)

// simplifyVGSIAddr returns a shorter search-friendly version of a normalized
// street string by stripping hyphenated ranges and unit designators.
// Returns the input unchanged when no simplification applies.
func simplifyVGSIAddr(s string) string {
	s = vgsiRangeRe.ReplaceAllString(s, "$1")
	s = vgsiUnitRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// vgsiSearchOnce performs a single GET→POST cycle against a VGSI Search.aspx
// page using addrText as the address query.  Returns the PID string on success,
// "" when the property is not found, or an error on network/parse failure.
// A fresh GET is required each time because ASP.NET __VIEWSTATE is request-scoped.
func vgsiSearchOnce(ctx context.Context, client *http.Client, searchURL, referer, addrText string) (string, error) {
	getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("vgsi get: %w", err)
	}
	vgsiHeaders(getReq, referer)

	resp, err := client.Do(getReq)
	if err != nil {
		return "", fmt.Errorf("vgsi GET %s: %w", searchURL, err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("vgsi parse: %w", err)
	}

	// Collect ASP.NET hidden fields (ViewState, EventValidation, …).
	formFields := url.Values{}
	doc.Find("input[type=hidden]").Each(func(_ int, s *goquery.Selection) {
		if name, _ := s.Attr("name"); name != "" {
			val, _ := s.Attr("value")
			formFields.Set(name, val)
		}
	})

	// Locate the address text box — VGSI IDs it with "txtAddress".
	addrField := ""
	doc.Find("input[type=text]").Each(func(_ int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		id, _ := s.Attr("id")
		lo := strings.ToLower
		if strings.Contains(lo(id), "txtaddress") || strings.Contains(lo(name), "txtaddress") {
			addrField = name
		}
	})
	if addrField == "" {
		// Disclaimer or splash page was returned — cookies didn't bypass it.
		log.Printf("[assessor] no address field on %s — disclaimer page may still be active", searchURL)
		return "", nil
	}

	btnField := ""
	doc.Find("input[type=submit], input[type=button]").Each(func(_ int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		val, _ := s.Attr("value")
		if strings.Contains(strings.ToLower(val), "search") && name != "" {
			btnField = name
		}
	})

	formFields.Set(addrField, addrText)
	if btnField != "" {
		formFields.Set(btnField, "Search")
	}
	formFields.Set("__EVENTTARGET", "")
	formFields.Set("__EVENTARGUMENT", "")

	postReq, err := http.NewRequestWithContext(ctx, http.MethodPost, searchURL,
		strings.NewReader(formFields.Encode()))
	if err != nil {
		return "", fmt.Errorf("vgsi post: %w", err)
	}
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	vgsiHeaders(postReq, searchURL)

	postResp, err := client.Do(postReq)
	if err != nil {
		return "", fmt.Errorf("vgsi POST %s: %w", searchURL, err)
	}
	defer postResp.Body.Close()

	if m := pidRe.FindStringSubmatch(postResp.Request.URL.String()); len(m) == 2 {
		return m[1], nil
	}
	body, err := io.ReadAll(postResp.Body)
	if err != nil {
		return "", fmt.Errorf("vgsi read body: %w", err)
	}
	if m := pidRe.FindStringSubmatch(string(body)); len(m) == 2 {
		return m[1], nil
	}
	return "", nil
}

// FetchAssessorPID queries the VGSI assessor portal for the given street+city
// and returns the numeric parcel PID.
// Returns ("", nil) for non-VGSI cities or when the property is not found.
// Two search attempts are made: the full normalized address first, then a
// simplified version that strips hyphenated ranges (193-195 → 193) and unit
// designators (APT, UNIT, #, etc.) so ambiguous addresses still resolve.
func FetchAssessorPID(ctx context.Context, street, city string) (string, error) {
	baseURL := GetVGSIBaseURL(city)
	if baseURL == "" {
		return "", nil
	}
	searchURL := baseURL + "/Search.aspx"
	client := newVGSIClient(baseURL)

	// Step 0: warm session — visit home page to collect any additional cookies
	// the portal sets on first load (session ID, anti-CSRF tokens, etc.).
	if homeReq, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/", nil); err == nil {
		vgsiHeaders(homeReq, "")
		if homeResp, herr := client.Do(homeReq); herr == nil {
			homeResp.Body.Close()
		}
	}

	normalized := NormalizeStreet(street)

	// Attempt 1: full normalized address.
	pid, err := vgsiSearchOnce(ctx, client, searchURL, baseURL+"/", normalized)
	if err != nil {
		return "", err
	}
	if pid != "" {
		log.Printf("[assessor] found pid=%s addr=%q city=%q", pid, normalized, city)
		return pid, nil
	}

	// Attempt 2: simplified address (strip range / unit).
	if simplified := simplifyVGSIAddr(normalized); simplified != normalized {
		log.Printf("[assessor] retrying simplified addr=%q (original=%q)", simplified, normalized)
		pid, err = vgsiSearchOnce(ctx, client, searchURL, baseURL+"/", simplified)
		if err != nil {
			return "", err
		}
		if pid != "" {
			log.Printf("[assessor] found pid=%s simplified addr=%q city=%q", pid, simplified, city)
			return pid, nil
		}
	}

	return "", nil
}

// ── Chain-of-title: deed link from VGSI parcel page ─────────────────────────

// fetchDeedLinkFromParcel GETs the VGSI Parcel page for the given pid and
// returns a registry deep link built from the most recent sale's Book/Page.
// Returns "" when the page cannot be fetched, has no sales table, or the
// first row has no Book/Page — all non-fatal.
func fetchDeedLinkFromParcel(ctx context.Context, baseURL, pid, city string) string {
	if pid == "" || baseURL == "" {
		return ""
	}
	parcelURL := baseURL + "/Parcel.aspx?pid=" + url.QueryEscape(pid)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parcelURL, nil)
	if err != nil {
		return ""
	}
	vgsiHeaders(req, baseURL+"/Search.aspx")

	client := newVGSIClient(baseURL)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[assessor] parcel fetch pid=%s: %v", pid, err)
		return ""
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return ""
	}

	// Walk every <table> looking for one whose header row contains both
	// "book" and "page" columns — that is the Sales / Ownership History table.
	var deedLink string
	doc.Find("table").EachWithBreak(func(_ int, tbl *goquery.Selection) bool {
		colIdx := map[string]int{}
		tbl.Find("tr").First().Find("th, td").Each(func(i int, cell *goquery.Selection) {
			h := strings.TrimSpace(strings.ToLower(cell.Text()))
			if h == "book" || h == "page" {
				colIdx[h] = i
			}
		})
		bookCol, hasBook := colIdx["book"]
		pageCol, hasPage := colIdx["page"]
		if !hasBook || !hasPage {
			return true // not the sales table — keep searching
		}

		// The first data row (index 1) is the most recent sale.
		allRows := tbl.Find("tr")
		if allRows.Length() < 2 {
			return false
		}
		cells := allRows.Eq(1).Find("td")
		book := strings.TrimSpace(cells.Eq(bookCol).Text())
		page := strings.TrimSpace(cells.Eq(pageCol).Text())
		if book != "" && page != "" {
			deedLink = BuildRegistryDeepLink(city, book, page)
		}
		return false // stop after first matching table
	})

	if deedLink != "" {
		log.Printf("[assessor] deed link found pid=%s city=%q: %s", pid, city, deedLink)
	}
	return deedLink
}

// ── Second-pass assessor enrichment ─────────────────────────────────────────

// RunAssessorSecondPass queries rows where assessor_pid OR registry_deep_link is
// still NULL (so rows where both are already populated are skipped entirely —
// no external government portal is hit for them).
// For each candidate it fetches the VGSI PID, then scrapes the Sales History
// table for the deed Book/Page to build a Level-1 registry deep link.
// Work is distributed across a pool of 3 parallel workers; every successful
// result is committed immediately so a timeout cannot discard partial progress.
func RunAssessorSecondPass(ctx context.Context, db *sql.DB) {
	const numWorkers = 3

	// Count rows that are already fully enriched (both fields present).
	var alreadyEnriched int
	db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM auctions
		WHERE assessor_pid    IS NOT NULL AND assessor_pid    != ''
		  AND registry_deep_link IS NOT NULL AND registry_deep_link != ''`).Scan(&alreadyEnriched)

	// Only fetch rows that are still missing at least one enrichment field.
	const sel = `
		SELECT id, address, city FROM auctions
		WHERE (assessor_pid IS NULL OR registry_deep_link IS NULL)
		  AND city IS NOT NULL AND city != ''`

	rows, err := db.QueryContext(ctx, sel)
	if err != nil {
		log.Printf("[Enrichment] query error: %v", err)
		return
	}
	defer rows.Close()

	type candidate struct {
		id      int
		address string
		city    string
	}
	var candidates []candidate
	for rows.Next() {
		var r candidate
		if err := rows.Scan(&r.id, &r.address, &r.city); err == nil {
			if GetVGSIBaseURL(r.city) != "" {
				candidates = append(candidates, r)
			}
		}
	}
	rows.Close()

	log.Printf("[Enrichment] Found %d deals needing research. Skipping %d already enriched. Starting %d parallel workers.",
		len(candidates), alreadyEnriched, numWorkers)

	if len(candidates) == 0 {
		return
	}

	work := make(chan candidate, len(candidates))
	for _, c := range candidates {
		work <- c
	}
	close(work)

	var (
		mu       sync.Mutex
		enriched int
	)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for c := range work {
				select {
				case <-ctx.Done():
					return
				default:
				}

				pid, err := FetchAssessorPID(ctx, c.address, c.city)
				if err != nil {
					log.Printf("[Enrichment] id=%d addr=%q city=%q pid error: %v", c.id, c.address, c.city, err)
					time.Sleep(time.Second)
					continue
				}
				if pid == "" {
					time.Sleep(time.Second)
					continue
				}

				// Level 1: deed Book/Page from the Sales History table.
				deedLink := fetchDeedLinkFromParcel(ctx, GetVGSIBaseURL(c.city), pid, c.city)

				// Commit immediately — partial progress is never lost.
				var upErr error
				if deedLink != "" {
					_, upErr = db.ExecContext(ctx,
						`UPDATE auctions SET assessor_pid=$1, registry_deep_link=$2 WHERE id=$3`,
						pid, deedLink, c.id)
				} else {
					_, upErr = db.ExecContext(ctx,
						`UPDATE auctions SET assessor_pid=$1 WHERE id=$2`,
						pid, c.id)
				}
				if upErr != nil {
					log.Printf("[Enrichment] db update id=%d: %v", c.id, upErr)
				} else {
					log.Printf("[Enrichment] id=%d pid=%s deed=%q addr=%q", c.id, pid, deedLink, c.address)
					mu.Lock()
					enriched++
					mu.Unlock()
				}

				time.Sleep(time.Second)
			}
		}()
	}
	wg.Wait()

	log.Printf("[Enrichment] complete: enriched %d/%d", enriched, len(candidates))
}
