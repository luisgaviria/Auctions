package utils

import "strings"

// normalizeCity strips a state suffix ("Southbridge, Massachusetts" →
// "SOUTHBRIDGE") and uppercases the result so all city lookups are
// case-insensitive and state-suffix-insensitive.
func normalizeCity(city string) string {
	s := strings.TrimSpace(city)
	if i := strings.Index(s, ","); i != -1 {
		s = strings.TrimSpace(s[:i])
	}
	return strings.ToUpper(s)
}

// registryURLs maps each MA Registry of Deeds to its direct search portal.
// URLs sourced from massrods.com (Massachusetts Registers of Deeds Association).
var registryURLs = map[string]string{
	// Barnstable County
	"BARNSTABLE": "https://www.barnstabledeeds.org",

	// Berkshire County — three separate districts
	"BERKSHIRE MIDDLE": "https://www.masslandrecords.com/BerkshireMiddle",
	"BERKSHIRE NORTH":  "https://www.masslandrecords.com/BerkshireNorth",
	"BERKSHIRE SOUTH":  "https://www.masslandrecords.com/BerkshireSouth",

	// Bristol County — three districts
	"BRISTOL FALL RIVER": "https://www.masslandrecords.com/BristolFallRiver",
	"BRISTOL NORTH":      "https://www.masslandrecords.com/BristolNorth",
	"BRISTOL SOUTH":      "https://www.masslandrecords.com/BristolSouth",

	// Dukes County (Martha's Vineyard)
	"DUKES": "https://www.masslandrecords.com/Dukes",

	// Essex County — two districts
	"ESSEX NORTH": "https://www.masslandrecords.com/EssexNorth",
	"ESSEX SOUTH": "https://www.masslandrecords.com/EssexSouth",

	// Franklin County
	"FRANKLIN": "https://www.masslandrecords.com/Franklin",

	// Hampden County
	"HAMPDEN": "https://www.masslandrecords.com/Hampden",

	// Hampshire County
	"HAMPSHIRE": "https://www.masslandrecords.com/Hampshire",

	// Middlesex County — two districts
	"MIDDLESEX NORTH": "https://www.masslandrecords.com/MiddlesexNorth",
	"MIDDLESEX SOUTH": "https://www.masslandrecords.com/MiddlesexSouth",

	// Nantucket County
	"NANTUCKET": "https://www.masslandrecords.com/Nantucket",

	// Norfolk County
	"NORFOLK": "https://www.masslandrecords.com/Norfolk",

	// Plymouth County
	"PLYMOUTH": "https://www.masslandrecords.com/Plymouth",

	// Suffolk County — covers Boston, Chelsea, Revere, Winthrop
	"SUFFOLK": "https://www.masslandrecords.com/Suffolk",

	// Worcester County — two districts
	"WORCESTER":      "https://www.masslandrecords.com/Worcester",
	"WORCESTER NORTH": "https://www.masslandrecords.com/WorcesterNorth",
}

// cityToRegistry maps every MA city/town to its Registry of Deeds key.
// Cities are stored in UPPERCASE so lookup is case-insensitive.
var cityToRegistry = map[string]string{
	// ── Barnstable County ────────────────────────────────────────────────────
	"BARNSTABLE":  "BARNSTABLE",
	"BOURNE":      "BARNSTABLE",
	"BREWSTER":    "BARNSTABLE",
	"CHATHAM":     "BARNSTABLE",
	"DENNIS":      "BARNSTABLE",
	"EASTHAM":     "BARNSTABLE",
	"FALMOUTH":    "BARNSTABLE",
	"HARWICH":     "BARNSTABLE",
	"MASHPEE":     "BARNSTABLE",
	"ORLEANS":     "BARNSTABLE",
	"PROVINCETOWN": "BARNSTABLE",
	"SANDWICH":    "BARNSTABLE",
	"TRURO":       "BARNSTABLE",
	"WELLFLEET":   "BARNSTABLE",
	"YARMOUTH":    "BARNSTABLE",

	// ── Berkshire North ──────────────────────────────────────────────────────
	"ADAMS":        "BERKSHIRE NORTH",
	"CHESHIRE":     "BERKSHIRE NORTH",
	"CLARKSBURG":   "BERKSHIRE NORTH",
	"FLORIDA":      "BERKSHIRE NORTH",
	"HANCOCK":      "BERKSHIRE NORTH",
	"LANESBOROUGH": "BERKSHIRE NORTH",
	"NEW ASHFORD":  "BERKSHIRE NORTH",
	"NORTH ADAMS":  "BERKSHIRE NORTH",
	"SAVOY":        "BERKSHIRE NORTH",
	"WILLIAMSTOWN": "BERKSHIRE NORTH",
	"WINDSOR":      "BERKSHIRE NORTH",

	// ── Berkshire Middle ─────────────────────────────────────────────────────
	"BECKET":        "BERKSHIRE MIDDLE",
	"DALTON":        "BERKSHIRE MIDDLE",
	"HINSDALE":      "BERKSHIRE MIDDLE",
	"LEE":           "BERKSHIRE MIDDLE",
	"LENOX":         "BERKSHIRE MIDDLE",
	"OTIS":          "BERKSHIRE MIDDLE",
	"PERU":          "BERKSHIRE MIDDLE",
	"PITTSFIELD":    "BERKSHIRE MIDDLE",
	"RICHMOND":      "BERKSHIRE MIDDLE",
	"STOCKBRIDGE":   "BERKSHIRE MIDDLE",
	"WASHINGTON":    "BERKSHIRE MIDDLE",
	"WEST STOCKBRIDGE": "BERKSHIRE MIDDLE",

	// ── Berkshire South ──────────────────────────────────────────────────────
	"ALFORD":       "BERKSHIRE SOUTH",
	"EGREMONT":     "BERKSHIRE SOUTH",
	"GREAT BARRINGTON": "BERKSHIRE SOUTH",
	"MONTEREY":     "BERKSHIRE SOUTH",
	"MT WASHINGTON": "BERKSHIRE SOUTH",
	"NEW MARLBOROUGH": "BERKSHIRE SOUTH",
	"SANDISFIELD":  "BERKSHIRE SOUTH",
	"SHEFFIELD":    "BERKSHIRE SOUTH",
	"TYRINGHAM":    "BERKSHIRE SOUTH",

	// ── Bristol Fall River ───────────────────────────────────────────────────
	"FALL RIVER": "BRISTOL FALL RIVER",
	"FREETOWN":   "BRISTOL FALL RIVER",
	"SOMERSET":   "BRISTOL FALL RIVER",
	"SWANSEA":    "BRISTOL FALL RIVER",
	"WESTPORT":   "BRISTOL FALL RIVER",

	// ── Bristol North ────────────────────────────────────────────────────────
	"ATTLEBORO":      "BRISTOL NORTH",
	"BERKLEY":        "BRISTOL NORTH",
	"DIGHTON":        "BRISTOL NORTH",
	"EASTON":         "BRISTOL NORTH",
	"MANSFIELD":      "BRISTOL NORTH",
	"NORTH ATTLEBORO": "BRISTOL NORTH",
	"NORTON":         "BRISTOL NORTH",
	"RAYNHAM":        "BRISTOL NORTH",
	"REHOBOTH":       "BRISTOL NORTH",
	"SEEKONK":        "BRISTOL NORTH",
	"TAUNTON":        "BRISTOL NORTH",

	// ── Bristol South ────────────────────────────────────────────────────────
	"ACUSHNET":       "BRISTOL SOUTH",
	"DARTMOUTH":      "BRISTOL SOUTH",
	"FAIRHAVEN":      "BRISTOL SOUTH",
	"NEW BEDFORD":    "BRISTOL SOUTH",

	// ── Dukes County ─────────────────────────────────────────────────────────
	"AQUINNAH":    "DUKES",
	"CHILMARK":    "DUKES",
	"EDGARTOWN":   "DUKES",
	"GOSNOLD":     "DUKES",
	"OAK BLUFFS":  "DUKES",
	"TISBURY":     "DUKES",
	"WEST TISBURY": "DUKES",

	// ── Essex North ──────────────────────────────────────────────────────────
	"AMESBURY":    "ESSEX NORTH",
	"BOXFORD":     "ESSEX NORTH",
	"GEORGETOWN":  "ESSEX NORTH",
	"GROVELAND":   "ESSEX NORTH",
	"HAVERHILL":   "ESSEX NORTH",
	"LAWRENCE":    "ESSEX NORTH",
	"MERRIMAC":    "ESSEX NORTH",
	"METHUEN":     "ESSEX NORTH",
	"NEWBURY":     "ESSEX NORTH",
	"NEWBURYPORT": "ESSEX NORTH",
	"NORTH ANDOVER": "ESSEX NORTH",
	"ROWLEY":      "ESSEX NORTH",
	"SALISBURY":   "ESSEX NORTH",
	"WEST NEWBURY": "ESSEX NORTH",

	// ── Essex South ──────────────────────────────────────────────────────────
	"ANDOVER":     "ESSEX SOUTH",
	"BEVERLY":     "ESSEX SOUTH",
	"DANVERS":     "ESSEX SOUTH",
	"ESSEX":       "ESSEX SOUTH",
	"GLOUCESTER":  "ESSEX SOUTH",
	"HAMILTON":    "ESSEX SOUTH",
	"IPSWICH":     "ESSEX SOUTH",
	"LYNN":        "ESSEX SOUTH",
	"LYNNFIELD":   "ESSEX SOUTH",
	"MANCHESTER":  "ESSEX SOUTH",
	"MARBLEHEAD":  "ESSEX SOUTH",
	"MIDDLETON":   "ESSEX SOUTH",
	"NAHANT":      "ESSEX SOUTH",
	"PEABODY":     "ESSEX SOUTH",
	"ROCKPORT":    "ESSEX SOUTH",
	"SALEM":       "ESSEX SOUTH",
	"SAUGUS":      "ESSEX SOUTH",
	"SWAMPSCOTT":  "ESSEX SOUTH",
	"TOPSFIELD":   "ESSEX SOUTH",
	"WENHAM":      "ESSEX SOUTH",

	// ── Franklin County ──────────────────────────────────────────────────────
	"ASHFIELD":     "FRANKLIN",
	"BERNARDSTON":  "FRANKLIN",
	"BUCKLAND":     "FRANKLIN",
	"CHARLEMONT":   "FRANKLIN",
	"COLRAIN":      "FRANKLIN",
	"CONWAY":       "FRANKLIN",
	"DEERFIELD":    "FRANKLIN",
	"ERVING":       "FRANKLIN",
	"GILL":         "FRANKLIN",
	"GREENFIELD":   "FRANKLIN",
	"HAWLEY":       "FRANKLIN",
	"HEATH":        "FRANKLIN",
	"LEVERETT":     "FRANKLIN",
	"LEYDEN":       "FRANKLIN",
	"MONROE":       "FRANKLIN",
	"MONTAGUE":     "FRANKLIN",
	"NEW SALEM":    "FRANKLIN",
	"NORTHFIELD":   "FRANKLIN",
	"ORANGE":       "FRANKLIN",
	"ROWE":         "FRANKLIN",
	"SHELBURNE":    "FRANKLIN",
	"SHUTESBURY":   "FRANKLIN",
	"SUNDERLAND":   "FRANKLIN",
	"WARWICK":      "FRANKLIN",
	"WENDELL":      "FRANKLIN",
	"WHATELY":      "FRANKLIN",

	// ── Hampden County ───────────────────────────────────────────────────────
	"AGAWAM":        "HAMPDEN",
	"BLANDFORD":     "HAMPDEN",
	"BRIMFIELD":     "HAMPDEN",
	"CHESTER":       "HAMPDEN",
	"CHICOPEE":      "HAMPDEN",
	"EAST LONGMEADOW": "HAMPDEN",
	"GRANVILLE":     "HAMPDEN",
	"HAMPDEN":       "HAMPDEN",
	"HOLLAND":       "HAMPDEN",
	"HOLYOKE":       "HAMPDEN",
	"LONGMEADOW":    "HAMPDEN",
	"LUDLOW":        "HAMPDEN",
	"MONSON":        "HAMPDEN",
	"MONTGOMERY":    "HAMPDEN",
	"PALMER":        "HAMPDEN",
	"RUSSELL":       "HAMPDEN",
	"SOUTHWICK":     "HAMPDEN",
	"SPRINGFIELD":   "HAMPDEN",
	"TOLLAND":       "HAMPDEN",
	"WALES":         "HAMPDEN",
	"WEST SPRINGFIELD": "HAMPDEN",
	"WESTFIELD":     "HAMPDEN",
	"WILBRAHAM":     "HAMPDEN",

	// ── Hampshire County ─────────────────────────────────────────────────────
	"AMHERST":       "HAMPSHIRE",
	"BELCHERTOWN":   "HAMPSHIRE",
	"CHESTERFIELD":  "HAMPSHIRE",
	"CUMMINGTON":    "HAMPSHIRE",
	"EASTHAMPTON":   "HAMPSHIRE",
	"GOSHEN":        "HAMPSHIRE",
	"GRANBY":        "HAMPSHIRE",
	"HADLEY":        "HAMPSHIRE",
	"HATFIELD":      "HAMPSHIRE",
	"HUNTINGTON":    "HAMPSHIRE",
	"MIDDLEFIELD":   "HAMPSHIRE",
	"NORTHAMPTON":   "HAMPSHIRE",
	"PELHAM":        "HAMPSHIRE",
	"PLAINFIELD":    "HAMPSHIRE",
	"SOUTH HADLEY":  "HAMPSHIRE",
	"SOUTHAMPTON":   "HAMPSHIRE",
	"WARE":          "HAMPSHIRE",
	"WESTHAMPTON":   "HAMPSHIRE",
	"WILLIAMSBURG":  "HAMPSHIRE",
	"WORTHINGTON":   "HAMPSHIRE",

	// ── Middlesex North ──────────────────────────────────────────────────────
	"BILLERICA":     "MIDDLESEX NORTH",
	"CHELMSFORD":    "MIDDLESEX NORTH",
	"DRACUT":        "MIDDLESEX NORTH",
	"DUNSTABLE":     "MIDDLESEX NORTH",
	"LOWELL":        "MIDDLESEX NORTH",
	"TEWKSBURY":     "MIDDLESEX NORTH",
	"TYNGSBOROUGH":  "MIDDLESEX NORTH",
	"WESTFORD":      "MIDDLESEX NORTH",

	// ── Middlesex South ──────────────────────────────────────────────────────
	"ACTON":         "MIDDLESEX SOUTH",
	"ARLINGTON":     "MIDDLESEX SOUTH",
	"ASHBY":         "MIDDLESEX SOUTH",
	"ASHLAND":       "MIDDLESEX SOUTH",
	"AYER":          "MIDDLESEX SOUTH",
	"BEDFORD":       "MIDDLESEX SOUTH",
	"BELMONT":       "MIDDLESEX SOUTH",
	"BOXBOROUGH":    "MIDDLESEX SOUTH",
	"BURLINGTON":    "MIDDLESEX SOUTH",
	"CAMBRIDGE":     "MIDDLESEX SOUTH",
	"CARLISLE":      "MIDDLESEX SOUTH",
	"CONCORD":       "MIDDLESEX SOUTH",
	"EVERETT":       "MIDDLESEX SOUTH",
	"FRAMINGHAM":    "MIDDLESEX SOUTH",
	"GROTON":        "MIDDLESEX SOUTH",
	"HOLLISTON":     "MIDDLESEX SOUTH",
	"HOPKINTON":     "MIDDLESEX SOUTH",
	"HUDSON":        "MIDDLESEX SOUTH",
	"LEXINGTON":     "MIDDLESEX SOUTH",
	"LINCOLN":       "MIDDLESEX SOUTH",
	"LITTLETON":     "MIDDLESEX SOUTH",
	"MALDEN":        "MIDDLESEX SOUTH",
	"MARLBOROUGH":   "MIDDLESEX SOUTH",
	"MAYNARD":       "MIDDLESEX SOUTH",
	"MEDFORD":       "MIDDLESEX SOUTH",
	"MELROSE":       "MIDDLESEX SOUTH",
	"NATICK":        "MIDDLESEX SOUTH",
	"NEWTON":        "MIDDLESEX SOUTH",
	"NORTH READING": "MIDDLESEX SOUTH",
	"PEPPERELL":     "MIDDLESEX SOUTH",
	"READING":       "MIDDLESEX SOUTH",
	"SHERBORN":      "MIDDLESEX SOUTH",
	"SHIRLEY":       "MIDDLESEX SOUTH",
	"SOMERVILLE":    "MIDDLESEX SOUTH",
	"STONEHAM":      "MIDDLESEX SOUTH",
	"STOW":          "MIDDLESEX SOUTH",
	"SUDBURY":       "MIDDLESEX SOUTH",
	"TOWNSEND":      "MIDDLESEX SOUTH",
	"WAKEFIELD":     "MIDDLESEX SOUTH",
	"WATERTOWN":     "MIDDLESEX SOUTH",
	"WAYLAND":       "MIDDLESEX SOUTH",
	"WESTON":        "MIDDLESEX SOUTH",
	"WILMINGTON":    "MIDDLESEX SOUTH",
	"WINCHESTER":    "MIDDLESEX SOUTH",
	"WOBURN":        "MIDDLESEX SOUTH",

	// ── Nantucket County ─────────────────────────────────────────────────────
	"NANTUCKET": "NANTUCKET",

	// ── Norfolk County ───────────────────────────────────────────────────────
	"AVON":          "NORFOLK",
	"BELLINGHAM":    "NORFOLK",
	"BRAINTREE":     "NORFOLK",
	"BROOKLINE":     "NORFOLK",
	"CANTON":        "NORFOLK",
	"COHASSET":      "NORFOLK",
	"DEDHAM":        "NORFOLK",
	"DOVER":         "NORFOLK",
	"FOXBOROUGH":    "NORFOLK",
	"FRANKLIN":      "NORFOLK",
	"HOLBROOK":      "NORFOLK",
	"MEDFIELD":      "NORFOLK",
	"MEDWAY":        "NORFOLK",
	"MILLIS":        "NORFOLK",
	"MILTON":        "NORFOLK",
	"NEEDHAM":       "NORFOLK",
	"NORFOLK":       "NORFOLK",
	"NORWOOD":       "NORFOLK",
	"PLAINVILLE":    "NORFOLK",
	"QUINCY":        "NORFOLK",
	"RANDOLPH":      "NORFOLK",
	"SHARON":        "NORFOLK",
	"STOUGHTON":     "NORFOLK",
	"WALPOLE":       "NORFOLK",
	"WELLESLEY":     "NORFOLK",
	"WESTWOOD":      "NORFOLK",
	"WEYMOUTH":      "NORFOLK",
	"WRENTHAM":      "NORFOLK",

	// ── Plymouth County ──────────────────────────────────────────────────────
	"ABINGTON":      "PLYMOUTH",
	"BRIDGEWATER":   "PLYMOUTH",
	"BROCKTON":      "PLYMOUTH",
	"CARVER":        "PLYMOUTH",
	"DUXBURY":       "PLYMOUTH",
	"EAST BRIDGEWATER": "PLYMOUTH",
	"HALIFAX":       "PLYMOUTH",
	"HANOVER":       "PLYMOUTH",
	"HANSON":        "PLYMOUTH",
	"HINGHAM":       "PLYMOUTH",
	"HULL":          "PLYMOUTH",
	"KINGSTON":      "PLYMOUTH",
	"LAKEVILLE":     "PLYMOUTH",
	"MARION":        "PLYMOUTH",
	"MARSHFIELD":    "PLYMOUTH",
	"MATTAPOISETT":  "PLYMOUTH",
	"MIDDLEBOROUGH": "PLYMOUTH",
	"NORWELL":       "PLYMOUTH",
	"PEMBROKE":      "PLYMOUTH",
	"PLYMOUTH":      "PLYMOUTH",
	"PLYMPTON":      "PLYMOUTH",
	"ROCHESTER":     "PLYMOUTH",
	"ROCKLAND":      "PLYMOUTH",
	"SCITUATE":      "PLYMOUTH",
	"WAREHAM":       "PLYMOUTH",
	"WEST BRIDGEWATER": "PLYMOUTH",
	"WHITMAN":       "PLYMOUTH",

	// ── Suffolk County ───────────────────────────────────────────────────────
	"BOSTON":  "SUFFOLK",
	"CHELSEA": "SUFFOLK",
	"REVERE":  "SUFFOLK",
	"WINTHROP": "SUFFOLK",

	// ── Worcester (South/Main district) ──────────────────────────────────────
	"AUBURN":        "WORCESTER",
	"BARRE":         "WORCESTER",
	"BERLIN":        "WORCESTER",
	"BLACKSTONE":    "WORCESTER",
	"BOLTON":        "WORCESTER",
	"BOYLSTON":      "WORCESTER",
	"BROOKFIELD":    "WORCESTER",
	"CHARLTON":      "WORCESTER",
	"CLINTON":       "WORCESTER",
	"DOUGLAS":       "WORCESTER",
	"DUDLEY":        "WORCESTER",
	"EAST BROOKFIELD": "WORCESTER",
	"GRAFTON":       "WORCESTER",
	"HARDWICK":      "WORCESTER",
	"HARVARD":       "WORCESTER",
	"HOLDEN":        "WORCESTER",
	"HOPEDALE":      "WORCESTER",
	"HUBBARDSTON":   "WORCESTER",
	"LANCASTER":     "WORCESTER",
	"LEICESTER":     "WORCESTER",
	"LUNENBURG":     "WORCESTER",
	"MENDON":        "WORCESTER",
	"MILFORD":       "WORCESTER",
	"MILLBURY":      "WORCESTER",
	"MILLVILLE":     "WORCESTER",
	"NEW BRAINTREE": "WORCESTER",
	"NORTHBOROUGH":  "WORCESTER",
	"NORTHBRIDGE":   "WORCESTER",
	"NORTH BROOKFIELD": "WORCESTER",
	"OAKHAM":        "WORCESTER",
	"OXFORD":        "WORCESTER",
	"PAXTON":        "WORCESTER",
	"PRINCETON":     "WORCESTER",
	"RUTLAND":       "WORCESTER",
	"SHREWSBURY":    "WORCESTER",
	"SOUTHBOROUGH":  "WORCESTER",
	"SOUTHBRIDGE":   "WORCESTER",
	"SPENCER":       "WORCESTER",
	"STERLING":      "WORCESTER",
	"STURBRIDGE":    "WORCESTER",
	"SUTTON":        "WORCESTER",
	"UPTON":         "WORCESTER",
	"UXBRIDGE":      "WORCESTER",
	"WARREN":        "WORCESTER",
	"WEBSTER":       "WORCESTER",
	"WEST BOYLSTON": "WORCESTER",
	"WEST BROOKFIELD": "WORCESTER",
	"WESTBOROUGH":   "WORCESTER",
	"WORCESTER":     "WORCESTER",
	"WINCHENDON":    "WORCESTER NORTH",

	// ── Worcester North ──────────────────────────────────────────────────────
	"ASHBURNHAM":   "WORCESTER NORTH",
	"ATHOL":        "WORCESTER NORTH",
	"FITCHBURG":    "WORCESTER NORTH",
	"GARDNER":      "WORCESTER NORTH",
	"LEOMINSTER":   "WORCESTER NORTH",
	"PETERSHAM":    "WORCESTER NORTH",
	"PHILLIPSTON":  "WORCESTER NORTH",
	"ROYALSTON":    "WORCESTER NORTH",
	"TEMPLETON":    "WORCESTER NORTH",
	"WESTMINSTER":  "WORCESTER NORTH",
}

// GetRegistryURL returns the direct Registry of Deeds search portal URL for
// the given Massachusetts city or town.  The lookup is case-insensitive.
// If the city is not found in the map, the general masslandrecords.com portal
// is returned as a fallback.
// GetCounty returns the plain county name for a Massachusetts city, e.g.
// "Worcester" or "Middlesex".  Registry district suffixes ("NORTH", "SOUTH",
// "FALL RIVER") are stripped so all towns in the same county share one name.
// Returns "" when the city is not in the map.
func GetCounty(city string) string {
	key := normalizeCity(city)
	registry, ok := cityToRegistry[key]
	if !ok {
		return ""
	}
	// Registry keys: "WORCESTER NORTH", "MIDDLESEX SOUTH", "BRISTOL FALL RIVER"
	// → first word is always the county name.
	first := strings.Fields(registry)[0]
	return strings.ToUpper(first[:1]) + strings.ToLower(first[1:])
}

func GetRegistryURL(city string) string {
	key := normalizeCity(city)
	if registry, ok := cityToRegistry[key]; ok {
		if url, ok := registryURLs[registry]; ok {
			return url
		}
	}
	return "https://www.masslandrecords.com"
}
