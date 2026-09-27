package main

import "testing"

func TestWithConnectTimeout(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"postgres://db.example/auctions", "postgres://db.example/auctions?connect_timeout=15"},
		{"postgres://db.example/auctions?sslmode=require", "postgres://db.example/auctions?sslmode=require&connect_timeout=15"},
		{"postgres://db.example/auctions?connect_timeout=5", "postgres://db.example/auctions?connect_timeout=5"},
	}
	for _, tt := range tests {
		if got := withConnectTimeout(tt.input); got != tt.want {
			t.Errorf("withConnectTimeout(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
