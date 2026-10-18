package static

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"quadsmith/shared/pkg/telemetry"
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
	crawlerRegex  = regexp.MustCompile(`(?i)(bot|crawler|spider|crawling|facebookexternalhit|facebot|slackbot|telegrambot|whatsapp|twitterbot|discordbot|applebot|pinterest|linkedinbot)`)

	crawlerHTTPClient = &http.Client{
		Timeout: 3 * time.Second,
	}

	collectionServiceMap = map[string]string{
		"motors":                       "MotorService/GetMotor",
		"frames":                       "FrameService/GetFrame",
		"batteries":                    "BatteryService/GetBattery",
		"electronic-speed-controllers": "ElectronicSpeedControllerService/GetElectronicSpeedController",
		"electronic_speed_controllers": "ElectronicSpeedControllerService/GetElectronicSpeedController",
		"flight-controllers":           "FlightControllerService/GetFlightController",
		"flight_controllers":           "FlightControllerService/GetFlightController",
		"cameras":                      "CameraService/GetCamera",
		"video-transmitters":           "VideoTransmitterService/GetVideoTransmitter",
		"video_transmitters":           "VideoTransmitterService/GetVideoTransmitter",
		"receivers":                    "ReceiverService/GetReceiver",
		"antennas":                     "AntennaService/GetAntenna",
		"propellers":                   "PropellerService/GetPropeller",
		"gps-receivers":                "GpsReceiverService/GetGpsReceiver",
		"gps_receivers":                "GpsReceiverService/GetGpsReceiver",
	}
)

// IsSocialCrawler detects if a request is coming from a social media crawler
func IsSocialCrawler(userAgent string) bool {
	return crawlerRegex.MatchString(userAgent)
}

func precomputeETags(dir string) map[string]string {
	etags := make(map[string]string)
	if strings.TrimSpace(dir) == "" {
		return etags
	}
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		cleanRel := "/" + filepath.ToSlash(rel)
		if cleanRel == "/index.html" {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		h := sha256.New()
		if _, err := io.Copy(h, f); err == nil {
			etags[cleanRel] = fmt.Sprintf(`"%s"`, hex.EncodeToString(h.Sum(nil)[:16]))
		}
		return nil
	})
	return etags
}

// ServeSPA serves a Single Page Application, dynamically injecting Open Graph and Twitter
// card tags into index.html for crawlers and browser visits.
func ServeSPA(dir string, apiURL string) http.HandlerFunc {
	etags := precomputeETags(dir)
	fileServer := http.FileServer(http.Dir(dir))

	return func(w http.ResponseWriter, r *http.Request) {
		cleanPath := filepath.ToSlash(filepath.Clean("/" + r.URL.Path))
		if cleanPath != "/index.html" {
			fullPath := filepath.Join(dir, filepath.FromSlash(cleanPath))
			info, err := os.Stat(fullPath)
			if err == nil && !info.IsDir() {
				// Static asset exists on disk (JS, CSS, WASM, images, etc.)
				if strings.HasPrefix(cleanPath, "/assets/") || r.URL.Query().Get("v") != "" {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				} else {
					w.Header().Set("Cache-Control", "public, max-age=3600, stale-while-revalidate=86400")
				}
				if strings.HasSuffix(cleanPath, ".wasm") {
					w.Header().Set("Content-Type", "application/wasm")
				}
				if etag, ok := etags[cleanPath]; ok {
					w.Header().Set("ETag", etag)
				}
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// Read base index.html
		indexPath := filepath.Join(dir, "index.html")
		indexBytes, err := os.ReadFile(indexPath)
		if err != nil {
			http.Error(w, "index.html not found", http.StatusNotFound)
			return
		}

		var meta PageMeta
		if IsSocialCrawler(r.UserAgent()) && apiURL != "" {
			meta = ResolvePageMetaFromAPI(r.Context(), apiURL, r.URL.Path)
		} else {
			meta = ResolvePageMeta(r.Context(), r.URL.Path)
		}

		meta.Canonical = resolveAbsoluteURL(r, r.URL.Path)
		meta.Image = resolveAbsoluteURL(r, meta.Image)

		renderedHTML := InjectMetaTags(string(indexBytes), meta)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(renderedHTML))
	}
}

// ResolvePageMeta resolves fallback metadata for the requested URL path.
func ResolvePageMeta(ctx context.Context, path string) PageMeta {
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

// ResolvePageMetaFromAPI queries server-api over HTTP to populate rich Open Graph and
// Twitter metadata for social crawlers requesting specific users, builds, or components.
func ResolvePageMetaFromAPI(ctx context.Context, apiURL string, path string) PageMeta {
	fallback := ResolvePageMeta(ctx, path)
	if strings.TrimSpace(apiURL) == "" {
		return fallback
	}

	trimmed := strings.Trim(strings.TrimSpace(path), "/")
	if trimmed == "" {
		return fallback
	}
	parts := strings.Split(trimmed, "/")

	baseURL := strings.TrimRight(apiURL, "/")

	// Route: /users/:id
	if len(parts) == 2 && parts[0] == "users" {
		userId := parts[1]
		endpoint := baseURL + "/quadsmith.api.UserService/GetPublicProfile"
		reqBody, _ := json.Marshal(map[string]string{"id": userId})
		respBytes, err := fetchConnectRPC(ctx, endpoint, reqBody)
		if err == nil {
			var profile struct {
				Id        string `json:"id"`
				Bio       string `json:"bio"`
				AvatarUrl string `json:"avatar_url"`
			}
			if err := json.Unmarshal(respBytes, &profile); err == nil && profile.Id != "" {
				fallback.Title = fmt.Sprintf("@%s — User Profile — Quadsmith", profile.Id)
				if strings.TrimSpace(profile.Bio) != "" {
					fallback.Description = profile.Bio
				} else {
					fallback.Description = fmt.Sprintf("View FPV drone builds and user stats for @%s on Quadsmith.", profile.Id)
				}
				if strings.TrimSpace(profile.AvatarUrl) != "" {
					fallback.Image = profile.AvatarUrl
				}
				fallback.Type = "profile"
				return fallback
			}
		}
	}

	// Route: /builds/:buildId
	if len(parts) == 2 && parts[0] == "builds" {
		buildId := parts[1]
		endpoint := baseURL + "/quadsmith.api.BuildService/GetBuild"
		reqBody, _ := json.Marshal(map[string]string{"id": buildId})
		respBytes, err := fetchConnectRPC(ctx, endpoint, reqBody)
		if err == nil {
			var b struct {
				Name                string `json:"name"`
				Description         string `json:"description"`
				PrimaryDisplayImage string `json:"primary_display_image"`
			}
			if err := json.Unmarshal(respBytes, &b); err == nil && b.Name != "" {
				fallback.Title = fmt.Sprintf("%s — FPV Drone Build — Quadsmith", b.Name)
				if strings.TrimSpace(b.Description) != "" {
					fallback.Description = b.Description
				}
				if strings.TrimSpace(b.PrimaryDisplayImage) != "" {
					fallback.Image = b.PrimaryDisplayImage
				}
				fallback.Type = "article"
				return fallback
			}
		}
	}

	// Route: /components/:categoryId/:collectionId/:productId
	if len(parts) >= 4 && parts[0] == "components" {
		collectionSlug := parts[2]
		productId := parts[3]
		serviceMethod, exists := collectionServiceMap[collectionSlug]
		if exists {
			endpoint := fmt.Sprintf("%s/quadsmith.api.%s", baseURL, serviceMethod)
			reqBody, _ := json.Marshal(map[string]string{"id": productId})
			respBytes, err := fetchConnectRPC(ctx, endpoint, reqBody)
			if err == nil {
				var comp struct {
					Manufacturer        string `json:"manufacturer"`
					Name                string `json:"name"`
					Description         string `json:"description"`
					PrimaryDisplayImage string `json:"primary_display_image"`
				}
				if err := json.Unmarshal(respBytes, &comp); err == nil && comp.Name != "" {
					if comp.Manufacturer != "" {
						fallback.Title = fmt.Sprintf("%s %s — Quadsmith", comp.Manufacturer, comp.Name)
					} else {
						fallback.Title = fmt.Sprintf("%s — Quadsmith", comp.Name)
					}
					if strings.TrimSpace(comp.Description) != "" {
						fallback.Description = comp.Description
					}
					if strings.TrimSpace(comp.PrimaryDisplayImage) != "" {
						fallback.Image = comp.PrimaryDisplayImage
					}
					fallback.Type = "article"
					return fallback
				}
			}
		}
	}

	return fallback
}

func fetchConnectRPC(ctx context.Context, endpoint string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	traceID := telemetry.TraceIDFromContext(ctx)
	if !telemetry.IsValidUUID(traceID) {
		traceID = telemetry.NewTraceID()
	}
	req.Header.Set(telemetry.TraceIDHeader, traceID)
	req.Header.Set(telemetry.InternalHopTokenHeader, telemetry.InternalHopToken())

	resp, err := crawlerHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, endpoint)
	}

	return io.ReadAll(resp.Body)
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
		host = "quadsmith.com"
	}
	trimmed := strings.TrimPrefix(rawURL, "/")
	if trimmed == "" {
		return fmt.Sprintf("%s://%s", scheme, host)
	}
	return fmt.Sprintf("%s://%s/%s", scheme, host, trimmed)
}
