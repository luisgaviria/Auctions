package utils

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
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

// BuildRegistryDeepLink returns a masslandrecords.com URL that pre-fills the
// Book/Page search for the given city.  Returns "" when the city is not mapped
// or when book/page are empty.
//
// masslandrecords.com runs Fidlar LAREDO; the hash-fragment query format is:
//   {districtURL}/#page=searchEntry&searchtype=Standard&recordtype=MR&book={B}&bookpage={P}
// where MR = Mortgage (the record type cited in foreclosure notices).
func BuildRegistryDeepLink(city, book, page string) string {
	if book == "" || page == "" {
		return ""
	}
	base := GetRegistryURL(city)
	if base == "" || base == "https://www.masslandrecords.com" {
		// Barnstable uses its own portal — direct book/page linking not available.
		if strings.EqualFold(GetCounty(city), "Barnstable") {
			return ""
		}
		if base == "" {
			return ""
		}
	}
	return fmt.Sprintf(
		"%s/#page=searchEntry&searchtype=Standard&recordtype=MR&book=%s&bookpage=%s",
		base, url.QueryEscape(book), url.QueryEscape(page),
	)
}

// ── Vision Government Solutions (VGSI) assessor portal ───────────────────────

// vgsiTowns maps MA city/town names (UPPERCASE) to their VGSI base URL.
// The URL slug is typically {townname}ma but Worcester is "worcestma", etc.
// Expand this list as additional towns are confirmed on gis.vgsi.com.
var vgsiTowns = map[string]string{
	"ATTLEBORO":    "https://gis.vgsi.com/attleboroma",
	"BELMONT":      "https://gis.vgsi.com/belmontma",
	"BRAINTREE":    "https://gis.vgsi.com/braintrema",
	"BRIDGEWATER":  "https://gis.vgsi.com/bridgewaterma",
	"CHELMSFORD":   "https://gis.vgsi.com/chelmsfordma",
	"CHICOPEE":     "https://gis.vgsi.com/chicopeema",
	"DARTMOUTH":    "https://gis.vgsi.com/dartmouthma",
	"DRACUT":       "https://gis.vgsi.com/dracutma",
	"EAST BRIDGEWATER": "https://gis.vgsi.com/eastbridgewaterma",
	"FITCHBURG":    "https://gis.vgsi.com/fitchburgma",
	"FRAMINGHAM":   "https://gis.vgsi.com/framinghamma",
	"GLOUCESTER":   "https://gis.vgsi.com/gloucesterma",
	"HAVERHILL":    "https://gis.vgsi.com/haverhillma",
	"HOLDEN":       "https://gis.vgsi.com/holdenma",
	"HOLYOKE":      "https://gis.vgsi.com/holyokema",
	"LEOMINSTER":   "https://gis.vgsi.com/leominster",
	"LOWELL":       "https://gis.vgsi.com/lowellma",
	"LYNN":         "https://gis.vgsi.com/lynnma",
	"MALDEN":       "https://gis.vgsi.com/maldenma",
	"MARLBOROUGH":  "https://gis.vgsi.com/marlboroughma",
	"MEDFORD":      "https://gis.vgsi.com/medfordma",
	"METHUEN":      "https://gis.vgsi.com/methuenma",
	"MILFORD":      "https://gis.vgsi.com/milfordma",
	"NEEDHAM":      "https://gis.vgsi.com/needhamma",
	"NEW BEDFORD":  "https://gis.vgsi.com/newbedfordma",
	"NORTHAMPTON":  "https://gis.vgsi.com/northamptonma",
	"PEABODY":      "https://gis.vgsi.com/peabodyma",
	"PITTSFIELD":   "https://gis.vgsi.com/pittsfieldma",
	"PLYMOUTH":     "https://gis.vgsi.com/plymouthma",
	"QUINCY":       "https://gis.vgsi.com/quincyma",
	"RANDOLPH":     "https://gis.vgsi.com/randolphma",
	"REVERE":       "https://gis.vgsi.com/reverema",
	"SALEM":        "https://gis.vgsi.com/salemma",
	"SHREWSBURY":   "https://gis.vgsi.com/shrewsburyma",
	"SOMERVILLE":   "https://gis.vgsi.com/somervillema",
	"TAUNTON":      "https://gis.vgsi.com/tauntonma",
	"WALTHAM":      "https://gis.vgsi.com/walthamma",
	"WATERTOWN":    "https://gis.vgsi.com/watertownma",
	"WEST SPRINGFIELD": "https://gis.vgsi.com/westspringfieldma",
	"WESTFIELD":    "https://gis.vgsi.com/westfieldma",
	"WOBURN":       "https://gis.vgsi.com/woburnma",
	"WORCESTER":    "https://gis.vgsi.com/worcestma",
}

// GetVGSIBaseURL returns the VGSI base URL for a MA city, or "" if the city
// is not in the map.  Lookup is case-insensitive.
func GetVGSIBaseURL(city string) string {
	return vgsiTowns[strings.ToUpper(strings.TrimSpace(city))]
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

// vgsiHTTPClient is used for all VGSI requests.  Timeout is generous because
// VGSI sites can be slow; a 15-second per-request ceiling prevents runaway goroutines.
var vgsiHTTPClient = &http.Client{
	Timeout: 15 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		// Allow up to 5 redirects (ASP.NET sometimes chains a couple).
		if len(via) >= 5 {
			return http.ErrUseLastResponse
		}
		return nil
	},
}

// FetchAssessorPID queries the VGSI assessor portal for the given street+city,
// submits a POST search, and returns the numeric PID from the resulting URL.
// Returns ("", nil) when the city has no VGSI portal or the property is not found.
// Returns ("", err) on network / parse errors.
func FetchAssessorPID(ctx context.Context, street, city string) (string, error) {
	baseURL := GetVGSIBaseURL(city)
	if baseURL == "" {
		return "", nil // city not in VGSI map — not an error
	}
	searchURL := baseURL + "/Search.aspx"

	// Step 1: GET the search page to retrieve ASP.NET hidden fields + control IDs.
	getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("vgsi get request: %w", err)
	}
	getReq.Header.Set("User-Agent", "Mozilla/5.0 (compatible; AuctionBot/1.0)")

	resp, err := vgsiHTTPClient.Do(getReq)
	if err != nil {
		return "", fmt.Errorf("vgsi GET %s: %w", searchURL, err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("vgsi parse form: %w", err)
	}

	// Collect all ASP.NET hidden fields.
	formFields := url.Values{}
	doc.Find("input[type=hidden]").Each(func(_ int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		val, _ := s.Attr("value")
		if name != "" {
			formFields.Set(name, val)
		}
	})

	// Find the address text input — look for "txtAddress" in the control ID.
	addrField := ""
	doc.Find("input[type=text]").Each(func(_ int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		id, _ := s.Attr("id")
		if strings.Contains(strings.ToLower(id), "txtaddress") ||
			strings.Contains(strings.ToLower(name), "txtaddress") {
			addrField = name
		}
	})
	if addrField == "" {
		return "", fmt.Errorf("vgsi: could not find address field on %s", searchURL)
	}

	// Find the search button name (needed for ASP.NET postback).
	btnField := ""
	doc.Find("input[type=submit], input[type=button]").Each(func(_ int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		val, _ := s.Attr("value")
		if strings.Contains(strings.ToLower(val), "search") && name != "" {
			btnField = name
		}
	})

	// Build POST body: hidden fields + address + submit button.
	formFields.Set(addrField, NormalizeStreet(street)) // e.g. "123 MAIN ST"
	if btnField != "" {
		formFields.Set(btnField, "Search")
	}
	formFields.Set("__EVENTTARGET", "")
	formFields.Set("__EVENTARGUMENT", "")

	// Step 2: POST the search form.
	postReq, err := http.NewRequestWithContext(ctx, http.MethodPost, searchURL,
		strings.NewReader(formFields.Encode()))
	if err != nil {
		return "", fmt.Errorf("vgsi post request: %w", err)
	}
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("User-Agent", "Mozilla/5.0 (compatible; AuctionBot/1.0)")
	postReq.Header.Set("Referer", searchURL)

	postResp, err := vgsiHTTPClient.Do(postReq)
	if err != nil {
		return "", fmt.Errorf("vgsi POST %s: %w", searchURL, err)
	}
	defer postResp.Body.Close()

	// The final URL after redirects often contains ?pid=XXXXX.
	if m := pidRe.FindStringSubmatch(postResp.Request.URL.String()); len(m) == 2 {
		return m[1], nil
	}

	// No immediate redirect to a parcel page — parse the result HTML for links.
	body, err := io.ReadAll(postResp.Body)
	if err != nil {
		return "", fmt.Errorf("vgsi read body: %w", err)
	}

	// Look for the first Parcel.aspx?pid=X link in the response.
	if m := pidRe.FindStringSubmatch(string(body)); len(m) == 2 {
		return m[1], nil
	}

	return "", nil // no match found — not an error
}

// ── Second-pass assessor enrichment ─────────────────────────────────────────

// RunAssessorSecondPass queries all rows where assessor_pid is NULL and the
// city has a known VGSI portal, then attempts to fetch and store the PID.
// This is designed to run once after the main scrape completes.
// Requests are rate-limited to one per second to be a good citizen.
func RunAssessorSecondPass(ctx context.Context, db *sql.DB) {
	const sel = `
		SELECT id, address, city FROM auctions
		WHERE assessor_pid IS NULL
		  AND city IS NOT NULL AND city != ''`

	rows, err := db.QueryContext(ctx, sel)
	if err != nil {
		log.Printf("[assessor] second-pass query error: %v", err)
		return
	}
	defer rows.Close()

	type row struct {
		id      int
		address string
		city    string
	}
	var candidates []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.address, &r.city); err == nil {
			// Only include cities that are in the VGSI map.
			if GetVGSIBaseURL(r.city) != "" {
				candidates = append(candidates, r)
			}
		}
	}
	rows.Close()

	log.Printf("[assessor] second-pass: %d properties to enrich", len(candidates))
	enriched := 0

	for _, c := range candidates {
		select {
		case <-ctx.Done():
			log.Printf("[assessor] second-pass cancelled after %d/%d", enriched, len(candidates))
			return
		default:
		}

		pid, err := FetchAssessorPID(ctx, c.address, c.city)
		if err != nil {
			log.Printf("[assessor] id=%d addr=%q city=%q error: %v", c.id, c.address, c.city, err)
		} else if pid != "" {
			_, upErr := db.ExecContext(ctx,
				`UPDATE auctions SET assessor_pid=$1 WHERE id=$2`,
				pid, c.id,
			)
			if upErr != nil {
				log.Printf("[assessor] update id=%d: %v", c.id, upErr)
			} else {
				log.Printf("[assessor] enriched id=%d pid=%s addr=%q", c.id, pid, c.address)
				enriched++
			}
		}

		// 1-second delay between VGSI requests to avoid rate limiting.
		time.Sleep(time.Second)
	}

	log.Printf("[assessor] second-pass complete: enriched %d/%d", enriched, len(candidates))
}
