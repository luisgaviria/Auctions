package sites

import "testing"

func TestPatriotResolveURL(t *testing.T) {
	tests := []struct {
		name string
		href string
		want string
	}{
		{
			name: "absolute URL",
			href: "https://patriotauctioneers.com/auction/21936/listing/",
			want: "https://patriotauctioneers.com/auction/21936/listing/",
		},
		{
			name: "root-relative URL",
			href: "/auction/21936/listing/",
			want: "https://patriotauctioneers.com/auction/21936/listing/",
		},
		{
			name: "path-relative URL",
			href: "auction/21936/listing/",
			want: "https://patriotauctioneers.com/auction/21936/listing/",
		},
		{
			name: "protocol-relative URL",
			href: "//patriotauctioneers.com/auction/21936/listing/",
			want: "https://patriotauctioneers.com/auction/21936/listing/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := patriotResolveURL(tt.href)
			if err != nil {
				t.Fatalf("patriotResolveURL(%q) error = %v", tt.href, err)
			}
			if got != tt.want {
				t.Errorf("patriotResolveURL(%q) = %q, want %q", tt.href, got, tt.want)
			}
		})
	}
}

func TestPatriotResolveURLRejectsMalformedHref(t *testing.T) {
	if _, err := patriotResolveURL("http://[::1"); err == nil {
		t.Fatal("patriotResolveURL accepted malformed href")
	}
}
