package linkpreview

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

	pb "quadsmith/api/gen/quadsmith"
)

func TestExtractMetadata_FullOpenGraph(t *testing.T) {
	pageURL, err := url.Parse("https://www.getfpv.com/tbs-source-one-v5-5-frame-kit.html")
	if err != nil {
		t.Fatalf("Failed to parse page URL: %v", err)
	}

	htmlContent := `
<!DOCTYPE html>
<html>
<head>
    <title>TBS Source One V5 5" Frame Kit - GetFPV</title>
    <meta name="description" content="The TBS SOURCE ONE is a wide-X configuration frame kit.">
    <meta property="og:title" content="TBS Source One V5 5&quot; Frame Kit">
    <meta property="og:description" content="Affordable freestyle drone frame by Team BlackSheep.">
    <meta property="og:image" content="/media/catalog/product/s/o/source-one-v5.jpg">
    <meta property="og:site_name" content="GetFPV">
    <link rel="icon" href="/favicon.png" type="image/png">
</head>
<body>
    <h1>TBS Source One</h1>
</body>
</html>
`

	meta := ExtractMetadata(pageURL, htmlContent)

	if meta.Title != `TBS Source One V5 5" Frame Kit` {
		t.Errorf("Expected title 'TBS Source One V5 5\" Frame Kit', got %q", meta.Title)
	}
	if meta.Description != "Affordable freestyle drone frame by Team BlackSheep." {
		t.Errorf("Expected og:description, got %q", meta.Description)
	}
	expectedImage := "https://www.getfpv.com/media/catalog/product/s/o/source-one-v5.jpg"
	if meta.Image != expectedImage {
		t.Errorf("Expected resolved image URL %q, got %q", expectedImage, meta.Image)
	}
	if meta.SiteName != "GetFPV" {
		t.Errorf("Expected site_name 'GetFPV', got %q", meta.SiteName)
	}
	expectedFavicon := "https://www.getfpv.com/favicon.png"
	if meta.Favicon != expectedFavicon {
		t.Errorf("Expected favicon %q, got %q", expectedFavicon, meta.Favicon)
	}
}

func TestExtractMetadata_FallbackTwitterAndHTMLTags(t *testing.T) {
	pageURL, _ := url.Parse("https://rotorriot.com/products/frame")
	htmlContent := `
<html>
<head>
    <title>Rotor Riot CL2 Frame</title>
    <meta name="twitter:title" content="Twitter CL2 Frame">
    <meta name="twitter:description" content="Freestyle frame by Rotor Riot.">
    <meta name="twitter:image" content="https://rotorriot.com/image.jpg">
</head>
</html>
`
	meta := ExtractMetadata(pageURL, htmlContent)
	if meta.Title != "Twitter CL2 Frame" {
		t.Errorf("Expected twitter:title fallback, got %q", meta.Title)
	}
	if meta.Description != "Freestyle frame by Rotor Riot." {
		t.Errorf("Expected twitter:description, got %q", meta.Description)
	}
	if meta.Image != "https://rotorriot.com/image.jpg" {
		t.Errorf("Expected twitter:image, got %q", meta.Image)
	}
	if meta.SiteName != "rotorriot.com" {
		t.Errorf("Expected hostname site name 'rotorriot.com', got %q", meta.SiteName)
	}
	if meta.Favicon != "https://rotorriot.com/favicon.ico" {
		t.Errorf("Expected default favicon.ico, got %q", meta.Favicon)
	}
}

func TestValidateHostSecurity_SSRFBlocked(t *testing.T) {
	ctx := context.Background()

	blockedHosts := []string{
		"localhost",
		"127.0.0.1",
		"::1",
		"service.internal",
		"device.local",
	}

	for _, host := range blockedHosts {
		err := validateHostSecurity(ctx, host)
		if err == nil {
			t.Errorf("Expected host %q to be blocked by validateHostSecurity, but it succeeded", host)
		}
	}
}

func TestFetchLinkPreview_InvalidScheme(t *testing.T) {
	ctx := context.Background()
	_, err := FetchLinkPreview(ctx, "file:///etc/passwd")
	if err == nil {
		t.Errorf("Expected error for file:// scheme, got nil")
	}
	if !strings.Contains(err.Error(), "invalid URL scheme") {
		t.Errorf("Expected 'invalid URL scheme' error, got %v", err)
	}
}

func TestFetchLinkPreview_Cache(t *testing.T) {
	ctx := context.Background()
	dummyURL := "https://example-test-cached-domain.com/item"

	expected := &pb.GetLinkPreviewResponse{
		Url:         dummyURL,
		Title:       "Cached Test Item",
		Description: "Cached description",
		Image:       "https://example-test-cached-domain.com/img.jpg",
		SiteName:    "example-test-cached-domain.com",
		Favicon:     "https://example-test-cached-domain.com/favicon.ico",
	}

	previewCache.Store(dummyURL, cacheEntry{
		response:  expected,
		expiresAt: time.Now().Add(1 * time.Hour),
	})

	got, err := FetchLinkPreview(ctx, dummyURL)
	if err != nil {
		t.Fatalf("Unexpected error fetching from cache: %v", err)
	}
	if got.Title != expected.Title {
		t.Errorf("Expected cached title %q, got %q", expected.Title, got.Title)
	}
}
