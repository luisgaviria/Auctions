package config

import "testing"

func TestGetFrontendURL(t *testing.T) {
	t.Setenv("FRONTEND_URL", "")
	if got := GetFrontendURL(); got != "http://localhost:4321" {
		t.Errorf("GetFrontendURL() = %q, want default URL", got)
	}

	t.Setenv("FRONTEND_URL", "https://frontend.example")
	if got := GetFrontendURL(); got != "https://frontend.example" {
		t.Errorf("GetFrontendURL() = %q, want configured URL", got)
	}
}

func TestGetAllowedOrigins(t *testing.T) {
	t.Run("configured list", func(t *testing.T) {
		t.Setenv("ALLOWED_ORIGINS", " https://one.example, ,https://two.example ")
		got := GetAllowedOrigins()
		if len(got) != 2 || got[0] != "https://one.example" || got[1] != "https://two.example" {
			t.Errorf("GetAllowedOrigins() = %#v", got)
		}
	})

	t.Run("frontend fallback", func(t *testing.T) {
		t.Setenv("ALLOWED_ORIGINS", "")
		t.Setenv("FRONTEND_URL", "https://frontend.example")
		got := GetAllowedOrigins()
		if len(got) != 1 || got[0] != "https://frontend.example" {
			t.Errorf("GetAllowedOrigins() = %#v", got)
		}
	})

	t.Run("default fallback", func(t *testing.T) {
		t.Setenv("ALLOWED_ORIGINS", "")
		t.Setenv("FRONTEND_URL", "")
		got := GetAllowedOrigins()
		if len(got) != 1 || got[0] != "http://localhost:4321" {
			t.Errorf("GetAllowedOrigins() = %#v", got)
		}
	})
}

func TestGetPort(t *testing.T) {
	t.Setenv("PORT", "")
	if got := GetPort(); got != "8000" {
		t.Errorf("GetPort() = %q, want default port", got)
	}

	t.Setenv("PORT", "9000")
	if got := GetPort(); got != "9000" {
		t.Errorf("GetPort() = %q, want configured port", got)
	}
}
