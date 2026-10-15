package auth

import (
	"net/http"
	"testing"

	"connectrpc.com/connect"
)

func TestExtractBaseURL_EnvOverrides(t *testing.T) {
	// APP_URL
	t.Setenv("APP_URL", "https://quadsmith.dev/")
	if url := extractBaseURL(nil); url != "https://quadsmith.dev" {
		t.Errorf("expected https://quadsmith.dev, got %s", url)
	}

	// WEB_URL
	t.Setenv("APP_URL", "")
	t.Setenv("WEB_URL", "http://localhost:5173/")
	if url := extractBaseURL(nil); url != "http://localhost:5173" {
		t.Errorf("expected http://localhost:5173, got %s", url)
	}

	// BASE_URL
	t.Setenv("WEB_URL", "")
	t.Setenv("BASE_URL", "https://staging.quadsmith.io")
	if url := extractBaseURL(nil); url != "https://staging.quadsmith.io" {
		t.Errorf("expected https://staging.quadsmith.io, got %s", url)
	}
}

func TestExtractBaseURL_Headers(t *testing.T) {
	t.Setenv("APP_URL", "")
	t.Setenv("WEB_URL", "")
	t.Setenv("BASE_URL", "")

	// Origin header
	h := http.Header{}
	h.Set("Origin", "http://localhost:5173/")
	if url := extractBaseURL(h); url != "http://localhost:5173" {
		t.Errorf("expected http://localhost:5173, got %s", url)
	}

	// Referer header
	h = http.Header{}
	h.Set("Referer", "https://quadsmith.dev/login?ref=header")
	if url := extractBaseURL(h); url != "https://quadsmith.dev" {
		t.Errorf("expected https://quadsmith.dev, got %s", url)
	}

	// X-Forwarded-Host with X-Forwarded-Proto
	h = http.Header{}
	h.Set("X-Forwarded-Host", "app.quadsmith.com, proxy.local")
	h.Set("X-Forwarded-Proto", "https")
	if url := extractBaseURL(h); url != "https://app.quadsmith.com" {
		t.Errorf("expected https://app.quadsmith.com, got %s", url)
	}

	// Host header fallback
	h = http.Header{}
	h.Set("Host", "custom.domain:9000")
	if url := extractBaseURL(h); url != "http://custom.domain:9000" {
		t.Errorf("expected http://custom.domain:9000, got %s", url)
	}
}

func TestExtractBaseURL_DefaultFallback(t *testing.T) {
	t.Setenv("APP_URL", "")
	t.Setenv("WEB_URL", "")
	t.Setenv("BASE_URL", "")
	t.Setenv("PORT", "")

	if url := extractBaseURL(nil); url != "http://localhost:8080" {
		t.Errorf("expected http://localhost:8080, got %s", url)
	}

	t.Setenv("PORT", "3000")
	if url := extractBaseURL(nil); url != "http://localhost:3000" {
		t.Errorf("expected http://localhost:3000, got %s", url)
	}

	t.Setenv("PORT", "80")
	if url := extractBaseURL(nil); url != "http://localhost" {
		t.Errorf("expected http://localhost, got %s", url)
	}

	t.Setenv("PORT", "443")
	if url := extractBaseURL(nil); url != "https://localhost" {
		t.Errorf("expected https://localhost, got %s", url)
	}
}

func TestGetBaseURL_ConnectRequest(t *testing.T) {
	t.Setenv("APP_URL", "")
	t.Setenv("WEB_URL", "")
	t.Setenv("BASE_URL", "")
	t.Setenv("PORT", "8080")

	req := connect.NewRequest(&struct{}{})
	req.Header().Set("Origin", "http://localhost:5173")

	if url := getBaseURL(req); url != "http://localhost:5173" {
		t.Errorf("expected http://localhost:5173, got %s", url)
	}

	if url := getBaseURL[*struct{}](nil); url != "http://localhost:8080" {
		t.Errorf("expected http://localhost:8080, got %s", url)
	}
}
