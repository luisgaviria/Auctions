package utils

import (
	"regexp"
	"strings"
)

// ── Book / Page extraction ────────────────────────────────────────────────────

// bookPageRe matches every Book/Page format found in MA foreclosure legal
// descriptions.  It is intentionally broad to catch the variations seen in
// production logs across all 10 scrapers:
//
//	Bk 50303, Pg 200              (Sullivan / Dean)
//	Book 67605, Page 300          (Dean prose)
//	in Bk 9251, Pg 341            (mid-sentence)
//	Middlesex (South) Cty. in Bk 48017, Pg 54
//	B13812/P506                   (SRI compact, no spaces)
//	B./P. 13812/506               (dotted single-letter variant)
//
// Alternation order (book|bk|b) ensures "Book" and "Bk" are tried before the
// single-letter "B" fallback so longer prefixes consume correctly.
// Group 1 = book number, Group 2 = page number.
var bookPageRe = regexp.MustCompile(
	`(?i)` +
		`(?:book|bk|b)\.?\s*` + // prefix: "Book", "Bk", "B", "B." with optional space
		`(\d+)` + // group 1: book number
		`[\s,/]+` + // separator: whitespace, comma, or slash
		`(?:(?:page|pg|p)\.?\s*)?` + // optional page prefix: "Page", "Pg", "P", "P."
		`(\d+)`, // group 2: page number
)

// bookPageBareRe is a fallback for bare numeric citations like "12345/678"
// that appear without any letter prefix.  Requires a 4-5 digit book number to
// avoid false positives on short unrelated numbers.
var bookPageBareRe = regexp.MustCompile(`\b(\d{4,5})/(\d{1,4})\b`)

// ExtractBookPage parses a legal description string and returns the
// Book and Page numbers as strings, or empty strings if not found.
// It first tries the full-prefix regex, then falls back to the bare NNNNN/NNN
// pattern used by some scrapers that strip the "B/P" letters.
func ExtractBookPage(legal string) (book, page string) {
	if m := bookPageRe.FindStringSubmatch(legal); len(m) >= 3 {
		return m[1], m[2]
	}
	// Fallback: bare numeric citation (e.g. "50303/200")
	if m := bookPageBareRe.FindStringSubmatch(legal); len(m) >= 3 {
		return m[1], m[2]
	}
	return "", ""
}

// ── Registry deep-link extraction ─────────────────────────────────────────────

// laredoDocRe matches a Tyler Technologies LAREDO document anchor in a
// fully-rendered search-results page.  The href may be relative
// (Document.aspx?id=…) or absolute (https://…/District/Document.aspx?id=…).
// Group 1 captures the raw href value.
var laredoDocRe = regexp.MustCompile(`(?i)href="([^"]*Document\.aspx\?[^"]*id=\d+[^"]*)"`)

// ExtractDocumentURL scans a fully-rendered LAREDO search-results HTML page
// for the first document link and returns an absolute URL.
// registryBase is the district root (e.g. "https://www.masslandrecords.com/MiddlesexSouth").
// Returns "" when no document link is found.
func ExtractDocumentURL(html, registryBase string) string {
	m := laredoDocRe.FindStringSubmatch(html)
	if len(m) < 2 {
		return ""
	}
	href := m[1]
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	// Relative URL — make it absolute using the registry base.
	base := strings.TrimRight(registryBase, "/")
	if !strings.HasPrefix(href, "/") {
		href = "/" + href
	}
	return base + href
}
