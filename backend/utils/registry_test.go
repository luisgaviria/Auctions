package utils

import "testing"

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
		{city: "  boston ", want: "https://www.masslandrecords.com/Suffolk"},
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
		city string
		want string
	}{
		{city: "Winchendon", want: "Worcester"},
		{city: "Fall River", want: "Bristol"},
		{city: "unknown", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.city, func(t *testing.T) {
			if got := GetCounty(tt.city); got != tt.want {
				t.Errorf("GetCounty(%q) = %q, want %q", tt.city, got, tt.want)
			}
		})
	}
}
