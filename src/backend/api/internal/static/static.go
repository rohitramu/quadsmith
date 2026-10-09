package static

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PageMeta defines the Open Graph, Twitter, and SEO metadata for a page.
type PageMeta struct {
	Title       string
	Description string
	Image       string
	Type        string
	Canonical   string
}

const (
	DefaultTitle       = "Quadsmith — FPV Drone Component Platform"
	DefaultDescription = "Design, optimize, and evaluate custom FPV drone builds. Calculate thrust-to-weight, simulate flight times, and verify hardware compatibility."
	DefaultOGImage     = "/og-default.png"
)

var (
	titleTagRegex = regexp.MustCompile(`(?i)<title[^>]*>.*?</title>`)
	headTagRegex  = regexp.MustCompile(`(?i)</head>`)
)

// Collection table mapping from collection slug to SQL table name.
var collectionTableMap = map[string]string{
	"motors":                       "motors",
	"frames":                       "frames",
	"batteries":                    "batteries",
	"electronic-speed-controllers": "electronic_speed_controllers",
	"electronic_speed_controllers": "electronic_speed_controllers",
	"flight-controllers":           "flight_controllers",
	"flight_controllers":           "flight_controllers",
	"cameras":                      "cameras",
	"video-transmitters":           "video_transmitters",
	"video_transmitters":           "video_transmitters",
	"receivers":                    "receivers",
	"antennas":                     "antennas",
	"propellers":                   "propellers",
	"gps-receivers":                "gps_receivers",
	"gps_receivers":                "gps_receivers",
}

// ServeSPA serves a Single Page Application, dynamically injecting Open Graph and Twitter
// card tags into index.html for crawlers and browser visits.
func ServeSPA(dir string, pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cleanPath := filepath.Clean(r.URL.Path)
		fullPath := filepath.Join(dir, cleanPath)
		info, err := os.Stat(fullPath)
		if err == nil && !info.IsDir() {
			// Static asset exists on disk (JS, CSS, images, etc.)
			http.FileServer(http.Dir(dir)).ServeHTTP(w, r)
			return
		}

		// Read base index.html
		indexPath := filepath.Join(dir, "index.html")
		indexBytes, err := os.ReadFile(indexPath)
		if err != nil {
			http.Error(w, "index.html not found", http.StatusNotFound)
			return
		}

		meta := ResolvePageMeta(r.Context(), pool, r.URL.Path)
		meta.Canonical = resolveAbsoluteURL(r, r.URL.Path)
		meta.Image = resolveAbsoluteURL(r, meta.Image)

		renderedHTML := InjectMetaTags(string(indexBytes), meta)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(renderedHTML))
	}
}

// ResolvePageMeta resolves metadata for the requested URL path from PostgreSQL.
func ResolvePageMeta(ctx context.Context, pool *pgxpool.Pool, path string) PageMeta {
	meta := PageMeta{
		Title:       DefaultTitle,
		Description: DefaultDescription,
		Image:       DefaultOGImage,
		Type:        "website",
	}

	trimmed := strings.Trim(strings.TrimSpace(path), "/")
	if trimmed == "" || trimmed == "builds" {
		return meta
	}

	parts := strings.Split(trimmed, "/")

	// Route: /builds/:buildId
	if len(parts) == 2 && parts[0] == "builds" && pool != nil {
		buildId := parts[1]
		var name, desc string
		var primaryDisplayImage *string

		err := pool.QueryRow(ctx, `
			SELECT name, description, primary_display_image 
			FROM builds 
			WHERE id = $1 OR uuid::text = $1 
			LIMIT 1`, buildId).Scan(&name, &desc, &primaryDisplayImage)
		if err == nil {
			meta.Title = fmt.Sprintf("%s — FPV Drone Build — Quadsmith", name)
			if strings.TrimSpace(desc) != "" {
				meta.Description = desc
			}
			if primaryDisplayImage != nil && strings.TrimSpace(*primaryDisplayImage) != "" {
				meta.Image = *primaryDisplayImage
			}
			meta.Type = "article"
			return meta
		}
	}

	// Route: /components/:categoryId/:collectionId/:productId
	if len(parts) >= 4 && parts[0] == "components" && pool != nil {
		collectionSlug := parts[2]
		productId := parts[3]
		tableName, exists := collectionTableMap[collectionSlug]
		if exists {
			var manufacturer, name, desc string
			var primaryDisplayImage *string

			query := fmt.Sprintf(`
				SELECT manufacturer, name, description, primary_display_image 
				FROM %s 
				WHERE id = $1 OR uuid::text = $1 
				LIMIT 1`, tableName)

			err := pool.QueryRow(ctx, query, productId).Scan(&manufacturer, &name, &desc, &primaryDisplayImage)
			if err == nil {
				meta.Title = fmt.Sprintf("%s %s — Quadsmith", manufacturer, name)
				if strings.TrimSpace(desc) != "" {
					meta.Description = desc
				}
				if primaryDisplayImage != nil && strings.TrimSpace(*primaryDisplayImage) != "" {
					meta.Image = *primaryDisplayImage
				}
				meta.Type = "article"
				return meta
			}
		}
	}

	// Route: /components/:categoryId/:collectionId
	if len(parts) == 3 && parts[0] == "components" {
		colTitle := formatSlug(parts[2])
		meta.Title = fmt.Sprintf("%s Catalog — Quadsmith", colTitle)
		meta.Description = fmt.Sprintf("Browse and compare %s specifications, flight weights, and hardware compatibility on Quadsmith.", colTitle)
		return meta
	}

	// Route: /components/:categoryId
	if len(parts) == 2 && parts[0] == "components" {
		catTitle := formatSlug(parts[1])
		meta.Title = fmt.Sprintf("%s Components — Quadsmith", catTitle)
		meta.Description = fmt.Sprintf("Explore %s drone components and hardware on Quadsmith.", catTitle)
		return meta
	}

	return meta
}

// InjectMetaTags replaces the <title> tag and inserts Open Graph and Twitter Card tags before </head>.
func InjectMetaTags(htmlTemplate string, meta PageMeta) string {
	escapedTitle := html.EscapeString(meta.Title)
	escapedDesc := html.EscapeString(meta.Description)
	escapedImage := html.EscapeString(meta.Image)
	escapedCanonical := html.EscapeString(meta.Canonical)
	escapedType := html.EscapeString(meta.Type)

	// Replace existing title
	res := titleTagRegex.ReplaceAllString(htmlTemplate, fmt.Sprintf("<title>%s</title>", escapedTitle))

	var tagsBuilder strings.Builder
	tagsBuilder.WriteString(fmt.Sprintf("\n    <meta name=\"description\" content=\"%s\" />", escapedDesc))
	tagsBuilder.WriteString("\n    <meta property=\"og:site_name\" content=\"Quadsmith\" />")
	tagsBuilder.WriteString(fmt.Sprintf("\n    <meta property=\"og:type\" content=\"%s\" />", escapedType))
	tagsBuilder.WriteString(fmt.Sprintf("\n    <meta property=\"og:url\" content=\"%s\" />", escapedCanonical))
	tagsBuilder.WriteString(fmt.Sprintf("\n    <meta property=\"og:title\" content=\"%s\" />", escapedTitle))
	tagsBuilder.WriteString(fmt.Sprintf("\n    <meta property=\"og:description\" content=\"%s\" />", escapedDesc))
	tagsBuilder.WriteString(fmt.Sprintf("\n    <meta property=\"og:image\" content=\"%s\" />", escapedImage))
	tagsBuilder.WriteString("\n    <meta name=\"twitter:card\" content=\"summary_large_image\" />")
	tagsBuilder.WriteString(fmt.Sprintf("\n    <meta name=\"twitter:title\" content=\"%s\" />", escapedTitle))
	tagsBuilder.WriteString(fmt.Sprintf("\n    <meta name=\"twitter:description\" content=\"%s\" />", escapedDesc))
	tagsBuilder.WriteString(fmt.Sprintf("\n    <meta name=\"twitter:image\" content=\"%s\" />\n  ", escapedImage))

	injected := tagsBuilder.String() + "</head>"
	res = headTagRegex.ReplaceAllString(res, injected)
	return res
}

func formatSlug(slug string) string {
	parts := strings.Split(slug, "-")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}

func resolveAbsoluteURL(r *http.Request, rawURL string) string {
	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		return rawURL
	}
	scheme := "https"
	if r.TLS == nil {
		protoHeader := r.Header.Get("X-Forwarded-Proto")
		if protoHeader != "" {
			scheme = protoHeader
		} else if r.Host != "" && (strings.HasPrefix(r.Host, "localhost") || strings.HasPrefix(r.Host, "127.0.0.1")) {
			scheme = "http"
		}
	}
	host := r.Host
	if host == "" {
		host = "quadsmith.net"
	}
	trimmed := strings.TrimPrefix(rawURL, "/")
	if trimmed == "" {
		return fmt.Sprintf("%s://%s", scheme, host)
	}
	return fmt.Sprintf("%s://%s/%s", scheme, host, trimmed)
}
