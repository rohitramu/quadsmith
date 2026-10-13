package static

import (
	"context"
	"encoding/json"
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
	homeMeta := ResolvePageMeta(ctx, "/")
	if homeMeta.Title != DefaultTitle {
		t.Errorf("Expected default title for home, got %q", homeMeta.Title)
	}

	// Category route
	catMeta := ResolvePageMeta(ctx, "/components/hardware")
	if catMeta.Title != "Hardware Components — Quadsmith" {
		t.Errorf("Expected formatted category title, got %q", catMeta.Title)
	}

	// Collection route
	colMeta := ResolvePageMeta(ctx, "/components/hardware/flight-controllers")
	if colMeta.Title != "Flight Controllers Catalog — Quadsmith" {
		t.Errorf("Expected formatted collection title, got %q", colMeta.Title)
	}
}

func TestResolvePageMetaFromAPI_Build(t *testing.T) {
	ctx := context.Background()

	// Mock server-api Connect endpoint
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/quadsmith.api.BuildService/GetBuild" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"name":                  "Apex 5 Freestyle",
				"description":           "High-speed 6S bando ripper.",
				"primary_display_image": "https://quadsmith.net/blobs/apex.jpg",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	meta := ResolvePageMetaFromAPI(ctx, mockServer.URL, "/builds/apex-5-freestyle")

	if meta.Title != "Apex 5 Freestyle — FPV Drone Build — Quadsmith" {
		t.Errorf("Expected build title, got %q", meta.Title)
	}
	if meta.Description != "High-speed 6S bando ripper." {
		t.Errorf("Expected build description, got %q", meta.Description)
	}
	if meta.Image != "https://quadsmith.net/blobs/apex.jpg" {
		t.Errorf("Expected build image, got %q", meta.Image)
	}
	if meta.Type != "article" {
		t.Errorf("Expected type article, got %q", meta.Type)
	}
}

func TestResolvePageMetaFromAPI_User(t *testing.T) {
	ctx := context.Background()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/quadsmith.api.UserService/GetPublicProfile" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"id":         "fpv_pilot",
				"bio":        "Custom FPV drone builder and racer.",
				"avatar_url": "https://quadsmith.net/blobs/avatar.png",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	meta := ResolvePageMetaFromAPI(ctx, mockServer.URL, "/users/fpv_pilot")

	if meta.Title != "@fpv_pilot — User Profile — Quadsmith" {
		t.Errorf("Expected user title, got %q", meta.Title)
	}
	if meta.Description != "Custom FPV drone builder and racer." {
		t.Errorf("Expected user description, got %q", meta.Description)
	}
	if meta.Image != "https://quadsmith.net/blobs/avatar.png" {
		t.Errorf("Expected user avatar image, got %q", meta.Image)
	}
	if meta.Type != "profile" {
		t.Errorf("Expected type profile, got %q", meta.Type)
	}
}

func TestServeSPA_ServesInjectedIndex_Crawler(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "static_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	indexHTML := `<!doctype html><html><head><title>Old</title></head><body>App</body></html>`
	if err := os.WriteFile(filepath.Join(tempDir, "index.html"), []byte(indexHTML), 0644); err != nil {
		t.Fatalf("Failed to write index.html: %v", err)
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/quadsmith.api.BuildService/GetBuild" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{
				"name":                  "SpeedyBee Master 5",
				"description":           "Smooth cinematic freestyle build.",
				"primary_display_image": "https://quadsmith.net/blobs/speedybee.jpg",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	handler := ServeSPA(tempDir, mockServer.URL)

	req := httptest.NewRequest(http.MethodGet, "/builds/speedybee-master-5", nil)
	req.Header.Set("User-Agent", "Twitterbot/1.0")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200 for crawler, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Expected text/html content type, got %q", contentType)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "SpeedyBee Master 5 — FPV Drone Build — Quadsmith") {
		t.Errorf("Expected dynamic injected title in crawler response, got %s", body)
	}
	if !strings.Contains(body, `<meta property="og:image" content="https://quadsmith.net/blobs/speedybee.jpg" />`) {
		t.Errorf("Expected dynamic injected og:image in crawler response, got %s", body)
	}
}

func TestServeSPA_ServesRegularUser(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "static_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	indexHTML := `<!doctype html><html><head><title>Old</title></head><body>App</body></html>`
	if err := os.WriteFile(filepath.Join(tempDir, "index.html"), []byte(indexHTML), 0644); err != nil {
		t.Fatalf("Failed to write index.html: %v", err)
	}

	handler := ServeSPA(tempDir, "http://127.0.0.1:8080")

	req := httptest.NewRequest(http.MethodGet, "/builds/new", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected status 200 for regular user SPA request, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "App") {
		t.Errorf("Expected SPA HTML in response, got %s", body)
	}
}
