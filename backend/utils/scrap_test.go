package utils

import (
	"testing"

	"backendAuction/utils/sites"
)

func TestCleanStreetAddress(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"  90 SUFFOLK ROAD, NEWTON (CHESTNUT HILL), MA  ", "90 SUFFOLK ROAD, NEWTON, MA"},
		{"12 MAIN ST, MA, B13812/P506", "12 MAIN ST, MA"},
		{"12 MAIN ST, View property for details", "12 MAIN ST"},
		{"90 MAIN ST, MAApr 7, 2026", "90 MAIN ST, MA"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := CleanStreetAddress(tt.input); got != tt.want {
				t.Errorf("CleanStreetAddress(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCleanAuctionData(t *testing.T) {
	tests := []struct {
		name       string
		address    string
		city       string
		wantStreet string
		wantCity   string
	}{
		{
			name:       "derive city from state-suffixed address",
			address:    "1600 WEST STREET, STOUGHTON, MA",
			city:       "Massachusetts",
			wantStreet: "1600 West Street",
			wantCity:   "Stoughton",
		},
		{
			name:       "preserve known city",
			address:    "12 MAIN ST, BOSTON, MA",
			city:       "Cambridge",
			wantStreet: "12 Main St",
			wantCity:   "Cambridge",
		},
		{
			name:       "title-case fallback without state token",
			address:    "12 MAIN STREET",
			city:       "NORTH ADAMS",
			wantStreet: "12 Main Street",
			wantCity:   "North Adams",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			street, city := CleanAuctionData(tt.address, tt.city)
			if street != tt.wantStreet || city != tt.wantCity {
				t.Errorf("CleanAuctionData() = (%q, %q), want (%q, %q)", street, city, tt.wantStreet, tt.wantCity)
			}
		})
	}
}

func TestNormalizeHelpers(t *testing.T) {
	t.Run("slug", func(t *testing.T) {
		if got := GenerateSlug("1600 WEST STREET Stoughton"); got != "1600-west-street-stoughton" {
			t.Errorf("GenerateSlug() = %q, want %q", got, "1600-west-street-stoughton")
		}
	})

	t.Run("address for URL", func(t *testing.T) {
		got := NormalizeAddressForURL("90 SUFFOLK ROAD, NEWTON (CHESTNUT HILL), MA")
		if got != "90 SUFFOLK ROAD NEWTON MA" {
			t.Errorf("NormalizeAddressForURL() = %q", got)
		}
	})

	t.Run("street suffix", func(t *testing.T) {
		if got := NormalizeStreet(" 47 Foss Rd "); got != "47 FOSS ROAD" {
			t.Errorf("NormalizeStreet() = %q, want %q", got, "47 FOSS ROAD")
		}
	})

	t.Run("date formats", func(t *testing.T) {
		tests := []struct {
			input string
			want  string
		}{
			{"2026-04-02", "2026-04-02"},
			{"APRIL 2, 2026", "2026-04-02"},
			{"4/2/2026", "2026-04-02"},
			{"not a date", "not a date"},
			{"", ""},
		}
		for _, tt := range tests {
			if got := normalizeDateToISO(tt.input); got != tt.want {
				t.Errorf("normalizeDateToISO(%q) = %q, want %q", tt.input, got, tt.want)
			}
		}
	})

	t.Run("status normalization", func(t *testing.T) {
		tests := []struct {
			input string
			want  string
		}{
			{"  CANCELLED ", "Removed"},
			{"Sold to third party", "Removed"},
			{"Postponed", "Removed"},
			{"Postponed to April 10, 2026", "Postponed to April 10, 2026"},
			{"Active", "Active"},
		}
		for _, tt := range tests {
			if got := normalizeStatus(tt.input); got != tt.want {
				t.Errorf("normalizeStatus(%q) = %q, want %q", tt.input, got, tt.want)
			}
		}
	})

	t.Run("placeholder times", func(t *testing.T) {
		for _, input := range []string{"00:00:00", "00:00", "12:00 AM", "12:00am"} {
			if got := normalizeTime(input); got != "" {
				t.Errorf("normalizeTime(%q) = %q, want empty", input, got)
			}
		}
		if got := normalizeTime("10:30 AM"); got != "10:30 AM" {
			t.Errorf("normalizeTime() changed a real time to %q", got)
		}
	})
}

func TestBuildAuctionURLsPublic(t *testing.T) {
	zillowURL, mapsURL, registryURL := BuildAuctionURLsPublic("12 Main St", "Boston")
	if zillowURL != "https://www.zillow.com/homes/12+Main+St+Boston+MA_rb/" {
		t.Errorf("Zillow URL = %q", zillowURL)
	}
	if mapsURL != "https://www.google.com/maps/search/?api=1&query=12+Main+St%2C+Boston%2C+MA" {
		t.Errorf("Maps URL = %q", mapsURL)
	}
	if registryURL != "https://www.masslandrecords.com/Suffolk" {
		t.Errorf("Registry URL = %q", registryURL)
	}
}

func TestNormalizeAuction(t *testing.T) {
	input := sites.Auction{
		Street:           "1600 WEST ST, STOUGHTON, MA, B13812/P506",
		City:             "Massachusetts",
		Date:             "APRIL 2, 2026",
		Time:             "12:00 AM",
		Deposit:          "$10,000",
		Status:           "Postponed to April 10, 2026",
		LegalDescription: "Bk 50303, Pg 200",
	}

	got := NormalizeAuction(input)
	if got.Street != "1600 WEST STREET" || got.City != "Stoughton" {
		t.Errorf("normalized address = (%q, %q), want (%q, %q)", got.Street, got.City, "1600 WEST STREET", "Stoughton")
	}
	if got.Date != "2026-04-02" || got.Time != "" || got.Deposit != "10000" {
		t.Errorf("normalized date/time/deposit = (%q, %q, %q)", got.Date, got.Time, got.Deposit)
	}
	if got.Status != input.Status {
		t.Errorf("scheduled postponement status = %q, want unchanged %q", got.Status, input.Status)
	}
	if got.RegistryBook != 50303 || got.RegistryPage != 200 {
		t.Errorf("registry citation = (%d, %d), want (50303, 200)", got.RegistryBook, got.RegistryPage)
	}
	if got.CitySlug != "stoughton" || got.CountySlug != "norfolk-county" || got.AddressSlug != "1600-west-street-stoughton" {
		t.Errorf("slugs = (%q, %q, %q), derived from street=%q city=%q", got.CitySlug, got.CountySlug, got.AddressSlug, got.Street, got.City)
	}
	if input.Street != "1600 WEST ST, STOUGHTON, MA, B13812/P506" {
		t.Errorf("NormalizeAuction mutated its input: %q", input.Street)
	}
}

func TestIsPastDate(t *testing.T) {
	today := todayET()
	yesterday := today.AddDate(0, 0, -1).Format("Jan 2, 2006")
	tomorrow := today.AddDate(0, 0, 1).Format("Jan 2, 2006")

	if !isPastDate(yesterday) {
		t.Errorf("isPastDate(%q) = false, want true", yesterday)
	}
	if isPastDate(today.Format("Jan 2, 2006")) {
		t.Error("today should not be considered a past date")
	}
	if isPastDate(tomorrow) {
		t.Errorf("isPastDate(%q) = true, want false", tomorrow)
	}
	if isPastDate("not a date") {
		t.Error("an unrecognized date should not be considered past")
	}
	if isPastDate("") {
		t.Error("an empty date should not be considered past")
	}

}
