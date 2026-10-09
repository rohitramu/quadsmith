package static

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInjectMetaTags(t *testing.T) {
	htmlTemplate := `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <title>Default Title</title>
  </head>
  <body>
    <div id="root"></div>
  </body>
</html>`

	meta := PageMeta{
		Title:       "Test Build — Quadsmith",
		Description: "A high-performance freestyle drone build.",
		Image:       "https://quadsmith.net/img/build.jpg",
		Type:        "article",
		Canonical:   "https://quadsmith.net/builds/test-build",
	}

	result := InjectMetaTags(htmlTemplate, meta)

	if !strings.Contains(result, "<title>Test Build — Quadsmith</title>") {
		t.Errorf("Expected injected title, got:\n%s", result)
	}
	if !strings.Contains(result, `<meta property="og:title" content="Test Build — Quadsmith" />`) {
		t.Errorf("Expected og:title tag, got:\n%s", result)
	}
	if !strings.Contains(result, `<meta property="og:image" content="https://quadsmith.net/img/build.jpg" />`) {
		t.Errorf("Expected og:image tag, got:\n%s", result)
	}
	if !strings.Contains(result, `<meta property="og:url" content="https://quadsmith.net/builds/test-build" />`) {
		t.Errorf("Expected og:url tag, got:\n%s", result)
	}
	if !strings.Contains(result, `<meta name="twitter:card" content="summary_large_image" />`) {
		t.Errorf("Expected twitter:card tag, got:\n%s", result)
	}
}

func TestResolvePageMeta_Fallbacks(t *testing.T) {
	ctx := context.Background()

	// Home page
	homeMeta := ResolvePageMeta(ctx, nil, "/")
	if homeMeta.Title != DefaultTitle {
		t.Errorf("Expected default title for home, got %q", homeMeta.Title)
	}

	// Category route
	catMeta := ResolvePageMeta(ctx, nil, "/components/hardware")
	if catMeta.Title != "Hardware Components — Quadsmith" {
		t.Errorf("Expected formatted category title, got %q", catMeta.Title)
	}

	// Collection route
	colMeta := ResolvePageMeta(ctx, nil, "/components/hardware/flight-controllers")
	if colMeta.Title != "Flight Controllers Catalog — Quadsmith" {
		t.Errorf("Expected formatted collection title, got %q", colMeta.Title)
	}
}

func TestServeSPA_ServesInjectedIndex(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "static_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	indexHTML := `<!doctype html><html><head><title>Old</title></head><body>App</body></html>`
	if err := os.WriteFile(filepath.Join(tempDir, "index.html"), []byte(indexHTML), 0644); err != nil {
		t.Fatalf("Failed to write index.html: %v", err)
	}

	handler := ServeSPA(tempDir, nil)

	req := httptest.NewRequest(http.MethodGet, "/components/propulsion/motors", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Expected text/html content type, got %q", contentType)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Motors Catalog — Quadsmith") {
		t.Errorf("Expected injected title in response, got %s", body)
	}
}
