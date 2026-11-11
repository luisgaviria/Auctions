package utils

import (
	"net/url"
	"strings"
	"testing"
)

func TestExtractBookPage(t *testing.T) {
	tests := []struct {
		name     string
		legal    string
		wantBook string
		wantPage string
	}{
		{name: "book and page words", legal: "Book 67605, Page 300", wantBook: "67605", wantPage: "300"},
		{name: "abbreviated citation", legal: "Bk 50303, Pg 200", wantBook: "50303", wantPage: "200"},
		{name: "citation within prose", legal: "Middlesex County in Bk 48017, Pg 54", wantBook: "48017", wantPage: "54"},
		{name: "compact citation", legal: "B13812/P506", wantBook: "13812", wantPage: "506"},
		{name: "dotted citation", legal: "B./P. 13812/506", wantBook: "13812", wantPage: "506"},
		{name: "bare numeric citation", legal: "record 50303/200", wantBook: "50303", wantPage: "200"},
		{name: "unrelated numbers", legal: "Lot 12, parcel 345", wantBook: "", wantPage: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBook, gotPage := ExtractBookPage(tt.legal)
			if gotBook != tt.wantBook || gotPage != tt.wantPage {
				t.Errorf("ExtractBookPage(%q) = (%q, %q), want (%q, %q)", tt.legal, gotBook, gotPage, tt.wantBook, tt.wantPage)
			}
		})
	}
}

func TestExtractDocumentURL(t *testing.T) {
	tests := []struct {
		name         string
		html         string
		registryBase string
		want         string
	}{
		{
			name:         "relative link",
			html:         `<a href="Document.aspx?id=12345">Open</a>`,
			registryBase: "https://www.masslandrecords.com/MiddlesexSouth/",
			want:         "https://www.masslandrecords.com/MiddlesexSouth/Document.aspx?id=12345",
		},
		{
			name:         "root-relative link",
			html:         `<a href="/MiddlesexSouth/Document.aspx?id=12345">Open</a>`,
			registryBase: "https://www.masslandrecords.com/MiddlesexSouth",
			want:         "https://www.masslandrecords.com/MiddlesexSouth/MiddlesexSouth/Document.aspx?id=12345",
		},
		{
			name:         "absolute link",
			html:         `<a href="https://records.example/Document.aspx?id=12345">Open</a>`,
			registryBase: "https://www.masslandrecords.com/MiddlesexSouth",
			want:         "https://records.example/Document.aspx?id=12345",
		},
		{
			name:         "no document link",
			html:         `<a href="Search.aspx">Search</a>`,
			registryBase: "https://www.masslandrecords.com/MiddlesexSouth",
			want:         "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractDocumentURL(tt.html, tt.registryBase); got != tt.want {
				t.Errorf("ExtractDocumentURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetRegistryURL(t *testing.T) {
	tests := []struct {
		city string
		want string
	}{
		{city: "Barnstable", want: "https://www.barnstabledeeds.org"},
		{city: "North Adams", want: "https://www.masslandrecords.com/BerkshireNorth"},
		{city: "Pittsfield", want: "https://www.masslandrecords.com/BerkshireMiddle"},
		{city: "Great Barrington", want: "https://www.masslandrecords.com/BerkshireSouth"},
		{city: "Fall River", want: "https://www.masslandrecords.com/BristolFallRiver"},
		{city: "Attleboro", want: "https://www.masslandrecords.com/BristolNorth"},
		{city: "New Bedford", want: "https://www.masslandrecords.com/BristolSouth"},
		{city: "Edgartown", want: "https://www.masslandrecords.com/Dukes"},
		{city: "Haverhill", want: "https://www.masslandrecords.com/EssexNorth"},
		{city: "Salem", want: "https://www.masslandrecords.com/EssexSouth"},
		{city: "Greenfield", want: "https://www.masslandrecords.com/Franklin"},
		{city: "Springfield", want: "https://www.masslandrecords.com/Hampden"},
		{city: "Northampton", want: "https://www.masslandrecords.com/Hampshire"},
		{city: "Lowell", want: "https://www.masslandrecords.com/MiddlesexNorth"},
		{city: "Cambridge", want: "https://www.masslandrecords.com/MiddlesexSouth"},
		{city: "Nantucket", want: "https://www.masslandrecords.com/Nantucket"},
		{city: "Quincy", want: "https://www.masslandrecords.com/Norfolk"},
		{city: "Plymouth", want: "https://www.masslandrecords.com/Plymouth"},
		{city: "  boston ", want: "https://www.masslandrecords.com/Suffolk"},
		{city: "Worcester", want: "https://www.masslandrecords.com/Worcester"},
		{city: "Fitchburg", want: "https://www.masslandrecords.com/WorcesterNorth"},
		{city: "Southbridge, Massachusetts", want: "https://www.masslandrecords.com/Worcester"},
		{city: "not a Massachusetts town", want: "https://www.masslandrecords.com"},
	}

	for _, tt := range tests {
		t.Run(tt.city, func(t *testing.T) {
			if got := GetRegistryURL(tt.city); got != tt.want {
				t.Errorf("GetRegistryURL(%q) = %q, want %q", tt.city, got, tt.want)
			}
		})
	}
}

func TestGetCounty(t *testing.T) {
	tests := []struct {
		name string
		city string
		want string
	}{
		{name: "Worcester main district", city: "Southbridge", want: "Worcester"},
		{name: "Worcester north district", city: "Winchendon", want: "Worcester"},
		{name: "Middlesex north district", city: "Lowell", want: "Middlesex"},
		{name: "Middlesex south district", city: "Cambridge", want: "Middlesex"},
		{name: "Bristol fall river district", city: "Fall River", want: "Bristol"},
		{name: "Berkshire north district", city: "North Adams", want: "Berkshire"},
		{name: "Barnstable county", city: "Barnstable", want: "Barnstable"},
		{name: "Dukes county", city: "Edgartown", want: "Dukes"},
		{name: "Essex county", city: "Salem", want: "Essex"},
		{name: "Franklin county", city: "Greenfield", want: "Franklin"},
		{name: "Hampden county", city: "Springfield", want: "Hampden"},
		{name: "Hampshire county", city: "Northampton", want: "Hampshire"},
		{name: "town name with county suffix", city: "Franklin", want: "Norfolk"},
		{name: "Nantucket county", city: "Nantucket", want: "Nantucket"},
		{name: "Plymouth county", city: "Plymouth", want: "Plymouth"},
		{name: "Suffolk county", city: "Boston", want: "Suffolk"},
		{name: "case and whitespace insensitive", city: "  sOuThBrIdGe  ", want: "Worcester"},
		{name: "state suffix ignored", city: "Southbridge, Massachusetts", want: "Worcester"},
		{name: "unknown city", city: "unknown", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetCounty(tt.city); got != tt.want {
				t.Errorf("GetCounty(%q) = %q, want %q", tt.city, got, tt.want)
			}
		})
	}
}

func TestCityToRegistryMappingsAreComplete(t *testing.T) {
	usedRegistries := make(map[string]bool, len(registryURLs))

	for city, registry := range cityToRegistry {
		if got := normalizeCity(city); got != city {
			t.Errorf("city key %q is not normalized (normalizeCity = %q)", city, got)
		}

		registryURL, ok := registryURLs[registry]
		if !ok {
			t.Errorf("city %q references registry %q without a URL", city, registry)
			continue
		}
		usedRegistries[registry] = true

		parsedURL, err := url.ParseRequestURI(registryURL)
		if err != nil || parsedURL.Scheme != "https" || parsedURL.Host == "" {
			t.Errorf("registry %q has invalid HTTPS URL %q", registry, registryURL)
		}
		if got := GetRegistryURL(city); got != registryURL {
			t.Errorf("GetRegistryURL(%q) = %q, want %q", city, got, registryURL)
		}
		if got := GetCounty(city); got == "" {
			t.Errorf("GetCounty(%q) returned an empty county", city)
		}
	}

	for registry := range registryURLs {
		if !usedRegistries[registry] {
			t.Errorf("registry %q has a URL but no city mappings", registry)
		}
	}

	for city := range cityToRegistry {
		if strings.TrimSpace(city) != city {
			t.Errorf("city key %q has leading or trailing whitespace", city)
		}
	}
}
