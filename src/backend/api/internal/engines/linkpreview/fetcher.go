package linkpreview

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	pb "quadsmith/api/gen/quadsmith"
)

// cacheEntry wraps a cached response with an expiration time.
type cacheEntry struct {
	response  *pb.GetLinkPreviewResponse
	expiresAt time.Time
}

var (
	previewCache   sync.Map
	cacheTTL       = 24 * time.Hour
	maxPayloadSize = int64(512 * 1024) // 512 KB
	httpClient     = &http.Client{
		Timeout: 3 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("stopped after 5 redirects")
			}
			// Enforce SSRF checks on redirect target
			return validateHostSecurity(req.Context(), req.URL.Hostname())
		},
	}
)

// validateHostSecurity checks whether a hostname is safe against SSRF attacks.
func validateHostSecurity(ctx context.Context, hostname string) error {
	lower := strings.ToLower(strings.TrimSpace(hostname))
	if lower == "" || lower == "localhost" || strings.HasSuffix(lower, ".local") || strings.HasSuffix(lower, ".internal") {
		return fmt.Errorf("forbidden hostname: %s", hostname)
	}

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", hostname)
	if err != nil {
		return fmt.Errorf("failed to resolve host %s: %w", hostname, err)
	}

	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("access to internal/private IP address (%s) is forbidden", ip.String())
		}
	}
	return nil
}

// FetchLinkPreview retrieves, extracts, and caches Open Graph and HTML metadata for a URL.
func FetchLinkPreview(ctx context.Context, rawURL string) (*pb.GetLinkPreviewResponse, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return nil, errors.New("empty URL provided")
	}

	parsedURL, err := url.Parse(trimmed)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return nil, fmt.Errorf("invalid URL scheme: %s", parsedURL.Scheme)
	}
	if parsedURL.Hostname() == "" {
		return nil, errors.New("missing hostname in URL")
	}

	// Check in-memory cache
	if val, ok := previewCache.Load(trimmed); ok {
		entry := val.(cacheEntry)
		if time.Now().Before(entry.expiresAt) {
			return entry.response, nil
		}
		previewCache.Delete(trimmed)
	}

	// SSRF security check
	if err := validateHostSecurity(ctx, parsedURL.Hostname()); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, trimmed, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "QuadsmithBot/1.0 (+https://quadsmith.net)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP error %d received from %s", resp.StatusCode, parsedURL.Hostname())
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxPayloadSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	meta := ExtractMetadata(parsedURL, string(bodyBytes))

	preview := &pb.GetLinkPreviewResponse{
		Url:         trimmed,
		Title:       meta.Title,
		Description: meta.Description,
		Image:       meta.Image,
		SiteName:    meta.SiteName,
		Favicon:     meta.Favicon,
	}

	previewCache.Store(trimmed, cacheEntry{
		response:  preview,
		expiresAt: time.Now().Add(cacheTTL),
	})

	return preview, nil
}

// ExtractedMeta represents parsed HTML metadata.
type ExtractedMeta struct {
	Title       string
	Description string
	Image       string
	SiteName    string
	Favicon     string
}

var (
	titleTagRegex = regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
	metaTagRegex  = regexp.MustCompile(`(?i)<meta\s+[^>]*>`)
	linkTagRegex  = regexp.MustCompile(`(?i)<link\s+[^>]*>`)
	attrRegex     = regexp.MustCompile(`([a-zA-Z0-9_\-:]+)\s*=\s*["']([^"']*)["']`)
)

// ExtractMetadata scans an HTML string for Open Graph, Twitter, and standard metadata.
func ExtractMetadata(pageURL *url.URL, body string) ExtractedMeta {
	var meta ExtractedMeta

	// Fallback domain-based site name
	meta.SiteName = pageURL.Hostname()
	if strings.HasPrefix(meta.SiteName, "www.") {
		meta.SiteName = strings.TrimPrefix(meta.SiteName, "www.")
	}

	// 1. Scan title tag as initial fallback
	if match := titleTagRegex.FindStringSubmatch(body); len(match) > 1 {
		meta.Title = cleanText(match[1])
	}

	hasOGTitle := false

	// 2. Scan meta tags for Open Graph and Twitter
	metaTags := metaTagRegex.FindAllString(body, -1)
	for _, tag := range metaTags {
		attrs := parseAttributes(tag)
		prop := strings.ToLower(attrs["property"])
		name := strings.ToLower(attrs["name"])
		content := attrs["content"]
		if content == "" {
			continue
		}

		switch {
		case prop == "og:title":
			meta.Title = cleanText(content)
			hasOGTitle = true
		case name == "twitter:title":
			if !hasOGTitle {
				meta.Title = cleanText(content)
			}
		case prop == "og:description" || name == "description" || name == "twitter:description":
			if meta.Description == "" || prop == "og:description" {
				meta.Description = cleanText(content)
			}
		case prop == "og:image" || name == "twitter:image":
			if meta.Image == "" || prop == "og:image" {
				meta.Image = resolveURL(pageURL, strings.TrimSpace(content))
			}
		case prop == "og:site_name":
			meta.SiteName = cleanText(content)
		}
	}

	// 3. Scan link tags for favicon
	linkTags := linkTagRegex.FindAllString(body, -1)
	for _, tag := range linkTags {
		attrs := parseAttributes(tag)
		rel := strings.ToLower(attrs["rel"])
		href := attrs["href"]
		if href == "" {
			continue
		}

		if rel == "icon" || rel == "shortcut icon" || strings.Contains(rel, "icon") {
			meta.Favicon = resolveURL(pageURL, strings.TrimSpace(href))
			break
		}
	}

	// Default fallback favicon
	if meta.Favicon == "" {
		meta.Favicon = fmt.Sprintf("%s://%s/favicon.ico", pageURL.Scheme, pageURL.Host)
	}

	// Default fallback title
	if meta.Title == "" {
		meta.Title = meta.SiteName
	}

	return meta
}

func parseAttributes(tag string) map[string]string {
	attrs := make(map[string]string)
	matches := attrRegex.FindAllStringSubmatch(tag, -1)
	for _, m := range matches {
		if len(m) > 2 {
			attrs[strings.ToLower(m[1])] = m[2]
		}
	}
	return attrs
}

func cleanText(text string) string {
	unescaped := html.UnescapeString(text)
	normalized := strings.Join(strings.Fields(unescaped), " ")
	return strings.TrimSpace(normalized)
}

func resolveURL(base *url.URL, relative string) string {
	if relative == "" {
		return ""
	}
	parsed, err := url.Parse(relative)
	if err != nil {
		return relative
	}
	return base.ResolveReference(parsed).String()
}
