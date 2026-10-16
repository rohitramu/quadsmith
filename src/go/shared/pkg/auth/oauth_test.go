package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	pb "quadsmith/shared/gen/quadsmith/api"
	"quadsmith/shared/gen/quadsmith/api/apiconnect"
	"quadsmith/shared/gen/quadsmith/config"
)

func TestOAuthPublicProcedures(t *testing.T) {
	if !IsPublicProcedure(apiconnect.AuthServiceStartOAuthProcedure) {
		t.Errorf("expected StartOAuth to be a public procedure")
	}
	if !IsPublicProcedure(apiconnect.AuthServiceCompleteOAuthProcedure) {
		t.Errorf("expected CompleteOAuth to be a public procedure")
	}
	if !IsAllowedBannedProcedure(apiconnect.AuthServiceStartOAuthProcedure) {
		t.Errorf("expected StartOAuth to be an allowed banned procedure")
	}
	if !IsAllowedBannedProcedure(apiconnect.AuthServiceCompleteOAuthProcedure) {
		t.Errorf("expected CompleteOAuth to be an allowed banned procedure")
	}
}

func TestOAuthProviderHelpers(t *testing.T) {
	cases := []struct {
		enum pb.OAuthProvider
		str  string
	}{
		{pb.OAuthProvider_OAUTH_PROVIDER_GOOGLE, "google"},
		{pb.OAuthProvider_OAUTH_PROVIDER_GITHUB, "github"},
		{pb.OAuthProvider_OAUTH_PROVIDER_DISCORD, "discord"},
		{pb.OAuthProvider_OAUTH_PROVIDER_UNSPECIFIED, "unspecified"},
	}

	for _, tc := range cases {
		if got := OAuthProviderToString(tc.enum); got != tc.str {
			t.Errorf("OAuthProviderToString(%v) = %s, want %s", tc.enum, got, tc.str)
		}
		if tc.enum != pb.OAuthProvider_OAUTH_PROVIDER_UNSPECIFIED {
			if got := StringToOAuthProvider(tc.str); got != tc.enum {
				t.Errorf("StringToOAuthProvider(%s) = %v, want %v", tc.str, got, tc.enum)
			}
		}
	}
}

func TestMockOAuthClient(t *testing.T) {
	client := NewOAuthClient(&config.AuthOAuthConfig{
		Mode: &config.AuthOAuthConfig_Mock{Mock: &config.MockOAuthConfig{}},
	})

	// Test Google auth URL & exchange
	authURL, err := client.GetAuthorizationURL(pb.OAuthProvider_OAUTH_PROVIDER_GOOGLE, "state123", "challenge", "http://localhost:5173/login")
	if err != nil {
		t.Fatalf("GetAuthorizationURL google failed: %v", err)
	}
	if !strings.Contains(authURL, "code=mock_google_default") || !strings.Contains(authURL, "state=state123") {
		t.Fatalf("unexpected authURL: %s", authURL)
	}

	ident, err := client.ExchangeCode(context.Background(), pb.OAuthProvider_OAUTH_PROVIDER_GOOGLE, "mock_google_default", "verifier", "http://localhost:5173/login")
	if err != nil {
		t.Fatalf("ExchangeCode google failed: %v", err)
	}
	if ident.Provider != "google" || ident.PrimaryEmail != "pilot.google@example.com" {
		t.Errorf("unexpected identity: %+v", ident)
	}

	// Test GitHub auth URL & exchange (multi-email)
	authURL, err = client.GetAuthorizationURL(pb.OAuthProvider_OAUTH_PROVIDER_GITHUB, "state456", "challenge", "http://localhost:5173/login")
	if err != nil {
		t.Fatalf("GetAuthorizationURL github failed: %v", err)
	}
	if !strings.Contains(authURL, "code=mock_github_default") {
		t.Fatalf("unexpected authURL: %s", authURL)
	}

	ident, err = client.ExchangeCode(context.Background(), pb.OAuthProvider_OAUTH_PROVIDER_GITHUB, "mock_github_default", "verifier", "http://localhost:5173/login")
	if err != nil {
		t.Fatalf("ExchangeCode github failed: %v", err)
	}
	if ident.Provider != "github" || ident.PrimaryEmail != "pilot.github@example.com" || len(ident.VerifiedEmails) != 2 {
		t.Errorf("unexpected github identity: %+v", ident)
	}

	// Test Discord auth URL & exchange
	authURL, err = client.GetAuthorizationURL(pb.OAuthProvider_OAUTH_PROVIDER_DISCORD, "state789", "challenge", "http://localhost:5173/login")
	if err != nil {
		t.Fatalf("GetAuthorizationURL discord failed: %v", err)
	}
	if !strings.Contains(authURL, "code=mock_discord_default") {
		t.Fatalf("unexpected authURL: %s", authURL)
	}

	ident, err = client.ExchangeCode(context.Background(), pb.OAuthProvider_OAUTH_PROVIDER_DISCORD, "mock_discord_default", "verifier", "http://localhost:5173/login")
	if err != nil {
		t.Fatalf("ExchangeCode discord failed: %v", err)
	}
	if ident.Provider != "discord" || ident.PrimaryEmail != "pilot.discord@example.com" {
		t.Errorf("unexpected discord identity: %+v", ident)
	}

	// Test dynamic mock code format: mock:<provider>:<accountId>:<primaryEmail>[:<secondaryEmail>...]
	dynamicCode := "mock:github:gh_100:lead@example.com:sec@example.com"
	ident, err = client.ExchangeCode(context.Background(), pb.OAuthProvider_OAUTH_PROVIDER_GITHUB, dynamicCode, "", "")
	if err != nil {
		t.Fatalf("ExchangeCode dynamic failed: %v", err)
	}
	if ident.Provider != "github" || ident.ProviderAccountID != "gh_100" || ident.PrimaryEmail != "lead@example.com" {
		t.Errorf("unexpected dynamic identity: %+v", ident)
	}
	if len(ident.VerifiedEmails) != 2 || ident.VerifiedEmails[1] != "sec@example.com" {
		t.Errorf("unexpected dynamic verified emails: %v", ident.VerifiedEmails)
	}
}

type rewriteHostRoundTripper struct {
	targetURL string
	rt        http.RoundTripper
}

func (r *rewriteHostRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = "http"
	cloned.URL.Host = strings.TrimPrefix(r.targetURL, "http://")
	return r.rt.RoundTrip(cloned)
}

func TestGitHubEmailFilteringLogic(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/oauth/access_token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"gho_dummy_token"}`))
		case "/user":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":123456,"avatar_url":"https://github.com/avatar.png"}`))
		case "/user/emails":
			w.Header().Set("Content-Type", "application/json")
			// One noreply email (primary), one verified real email, one unverified email
			_, _ = w.Write([]byte(`[
				{"email":"123456+user@users.noreply.github.com","primary":true,"verified":true},
				{"email":"unverified@example.com","primary":false,"verified":false},
				{"email":"real_verified@example.com","primary":false,"verified":true}
			]`))
		default:
			http.NotFound(w, r)
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	httpClient := server.Client()
	httpClient.Transport = &rewriteHostRoundTripper{
		targetURL: server.URL,
		rt:        http.DefaultTransport,
	}

	client := &providerOAuthClient{
		cfg: &config.OAuthProvidersConfig{
			Github: &config.OAuthProviderCredentials{
				ClientId:     proto.String("gh_client_id"),
				ClientSecret: proto.String("gh_client_secret"),
			},
		},
		httpClient: httpClient,
	}

	ident, err := client.ExchangeCode(context.Background(), pb.OAuthProvider_OAUTH_PROVIDER_GITHUB, "gh_code_123", "pkce_verifier", "http://localhost:5173/login")
	if err != nil {
		t.Fatalf("ExchangeCode github against mock HTTP server failed: %v", err)
	}
	if ident.Provider != "github" || ident.ProviderAccountID != "123456" {
		t.Errorf("unexpected github identity metadata: %+v", ident)
	}
	if ident.PrimaryEmail != "real_verified@example.com" {
		t.Errorf("expected noreply primary email to be filtered out and fallback to real_verified@example.com, got %s", ident.PrimaryEmail)
	}
	if len(ident.VerifiedEmails) != 1 || ident.VerifiedEmails[0] != "real_verified@example.com" {
		t.Errorf("expected only [real_verified@example.com], got %v", ident.VerifiedEmails)
	}
}
