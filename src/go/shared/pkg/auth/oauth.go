package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	pb "quadsmith/shared/gen/quadsmith/api"
	"quadsmith/shared/gen/quadsmith/config"
)

// OAuthIdentity represents the verified profile and emails returned by an OAuth provider
type OAuthIdentity struct {
	Provider          string
	ProviderAccountID string
	PrimaryEmail      string
	VerifiedEmails    []string
	AvatarURL         string
}

// OAuthProviderToString converts a protobuf OAuthProvider enum to its canonical string
func OAuthProviderToString(p pb.OAuthProvider) string {
	switch p {
	case pb.OAuthProvider_OAUTH_PROVIDER_GOOGLE:
		return "google"
	case pb.OAuthProvider_OAUTH_PROVIDER_GITHUB:
		return "github"
	case pb.OAuthProvider_OAUTH_PROVIDER_DISCORD:
		return "discord"
	default:
		return "unspecified"
	}
}

// StringToOAuthProvider converts a string provider identifier to its protobuf OAuthProvider enum
func StringToOAuthProvider(s string) pb.OAuthProvider {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "google":
		return pb.OAuthProvider_OAUTH_PROVIDER_GOOGLE
	case "github":
		return pb.OAuthProvider_OAUTH_PROVIDER_GITHUB
	case "discord":
		return pb.OAuthProvider_OAUTH_PROVIDER_DISCORD
	default:
		return pb.OAuthProvider_OAUTH_PROVIDER_UNSPECIFIED
	}
}

// OAuthClient provides methods to initiate and complete OAuth flows across providers
type OAuthClient interface {
	GetAuthorizationURL(provider pb.OAuthProvider, state, codeChallenge, redirectURI string) (string, error)
	ExchangeCode(ctx context.Context, provider pb.OAuthProvider, code, codeVerifier, redirectURI string) (*OAuthIdentity, error)
}

// NewOAuthClient constructs an OAuthClient from server AuthOAuthConfig
func NewOAuthClient(cfg *config.AuthOAuthConfig) OAuthClient {
	if cfg == nil || cfg.GetMock() != nil || cfg.GetProviders() == nil {
		return &mockOAuthClient{}
	}
	return &providerOAuthClient{
		cfg:        cfg.GetProviders(),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// ---------------------------------------------------------
// Mock OAuth Client (for sandbox, tests, and CI)
// ---------------------------------------------------------

type mockOAuthClient struct{}

func (m *mockOAuthClient) GetAuthorizationURL(provider pb.OAuthProvider, state, codeChallenge, redirectURI string) (string, error) {
	var mockCode string
	switch provider {
	case pb.OAuthProvider_OAUTH_PROVIDER_GOOGLE:
		mockCode = "mock_google_default"
	case pb.OAuthProvider_OAUTH_PROVIDER_GITHUB:
		mockCode = "mock_github_default"
	case pb.OAuthProvider_OAUTH_PROVIDER_DISCORD:
		mockCode = "mock_discord_default"
	default:
		return "", fmt.Errorf("unsupported oauth provider: %v", provider)
	}

	sep := "?"
	if strings.Contains(redirectURI, "?") {
		sep = "&"
	}
	return fmt.Sprintf("%s%scode=%s&state=%s", redirectURI, sep, mockCode, url.QueryEscape(state)), nil
}

func (m *mockOAuthClient) ExchangeCode(ctx context.Context, provider pb.OAuthProvider, code, codeVerifier, redirectURI string) (*OAuthIdentity, error) {
	if strings.HasPrefix(code, "mock_error_") {
		return nil, fmt.Errorf("mock oauth exchange error: %s", strings.TrimPrefix(code, "mock_error_"))
	}

	// Dynamic mock format: mock:<provider>:<accountId>:<primaryEmail>[:<secondaryEmail>...]
	if strings.HasPrefix(code, "mock:") {
		parts := strings.Split(code, ":")
		if len(parts) >= 4 {
			p := parts[1]
			accID := parts[2]
			primaryEmail := strings.ToLower(parts[3])
			verified := []string{primaryEmail}
			for _, s := range parts[4:] {
				sNorm := strings.ToLower(strings.TrimSpace(s))
				if sNorm != "" && sNorm != primaryEmail {
					verified = append(verified, sNorm)
				}
			}
			return &OAuthIdentity{
				Provider:          p,
				ProviderAccountID: accID,
				PrimaryEmail:      primaryEmail,
				VerifiedEmails:    verified,
				AvatarURL:         fmt.Sprintf("https://example.com/avatar/%s.png", p),
			}, nil
		}
	}

	provStr := OAuthProviderToString(provider)
	switch code {
	case "mock_google_default":
		return &OAuthIdentity{
			Provider:          "google",
			ProviderAccountID: "google_mock_sub_1001",
			PrimaryEmail:      "pilot.google@example.com",
			VerifiedEmails:    []string{"pilot.google@example.com"},
			AvatarURL:         "https://example.com/avatar/google.png",
		}, nil
	case "mock_github_default":
		return &OAuthIdentity{
			Provider:          "github",
			ProviderAccountID: "github_mock_id_2002",
			PrimaryEmail:      "pilot.github@example.com",
			VerifiedEmails:    []string{"pilot.github@example.com", "pilot.secondary@example.com"},
			AvatarURL:         "https://example.com/avatar/github.png",
		}, nil
	case "mock_discord_default":
		return &OAuthIdentity{
			Provider:          "discord",
			ProviderAccountID: "discord_mock_id_3003",
			PrimaryEmail:      "pilot.discord@example.com",
			VerifiedEmails:    []string{"pilot.discord@example.com"},
			AvatarURL:         "https://example.com/avatar/discord.png",
		}, nil
	default:
		// Fallback for custom persona codes like mock_github_user@domain.com
		if strings.HasPrefix(code, "mock_") {
			trimmed := strings.TrimPrefix(code, "mock_")
			if strings.HasPrefix(trimmed, provStr+"_") {
				trimmed = strings.TrimPrefix(trimmed, provStr+"_")
			}
			email := "mock." + trimmed
			if !strings.Contains(trimmed, "@") {
				email = email + "@example.com"
			} else {
				email = trimmed
			}
			email = strings.ToLower(email)
			return &OAuthIdentity{
				Provider:          provStr,
				ProviderAccountID: fmt.Sprintf("%s_mock_%s", provStr, trimmed),
				PrimaryEmail:      email,
				VerifiedEmails:    []string{email},
				AvatarURL:         fmt.Sprintf("https://example.com/avatar/%s.png", provStr),
			}, nil
		}
		return nil, fmt.Errorf("unknown mock authorization code: %s", code)
	}
}

// ---------------------------------------------------------
// Production OAuth Client
// ---------------------------------------------------------

type providerOAuthClient struct {
	cfg        *config.OAuthProvidersConfig
	httpClient *http.Client
}

func (c *providerOAuthClient) GetAuthorizationURL(provider pb.OAuthProvider, state, codeChallenge, redirectURI string) (string, error) {
	switch provider {
	case pb.OAuthProvider_OAUTH_PROVIDER_GOOGLE:
		cred := c.cfg.GetGoogle()
		if cred == nil || cred.GetClientId() == "" {
			return "", errors.New("google oauth client_id not configured")
		}
		v := url.Values{}
		v.Set("client_id", cred.GetClientId())
		v.Set("redirect_uri", redirectURI)
		v.Set("response_type", "code")
		v.Set("scope", "openid email profile")
		v.Set("state", state)
		v.Set("code_challenge", codeChallenge)
		v.Set("code_challenge_method", "S256")
		v.Set("access_type", "offline")
		v.Set("prompt", "select_account")
		return "https://accounts.google.com/o/oauth2/v2/auth?" + v.Encode(), nil

	case pb.OAuthProvider_OAUTH_PROVIDER_GITHUB:
		cred := c.cfg.GetGithub()
		if cred == nil || cred.GetClientId() == "" {
			return "", errors.New("github oauth client_id not configured")
		}
		v := url.Values{}
		v.Set("client_id", cred.GetClientId())
		v.Set("redirect_uri", redirectURI)
		v.Set("scope", "read:user user:email")
		v.Set("state", state)
		return "https://github.com/login/oauth/authorize?" + v.Encode(), nil

	case pb.OAuthProvider_OAUTH_PROVIDER_DISCORD:
		cred := c.cfg.GetDiscord()
		if cred == nil || cred.GetClientId() == "" {
			return "", errors.New("discord oauth client_id not configured")
		}
		v := url.Values{}
		v.Set("client_id", cred.GetClientId())
		v.Set("redirect_uri", redirectURI)
		v.Set("response_type", "code")
		v.Set("scope", "identify email")
		v.Set("state", state)
		v.Set("prompt", "consent")
		return "https://discord.com/api/oauth2/authorize?" + v.Encode(), nil

	default:
		return "", fmt.Errorf("unsupported oauth provider: %v", provider)
	}
}

func (c *providerOAuthClient) ExchangeCode(ctx context.Context, provider pb.OAuthProvider, code, codeVerifier, redirectURI string) (*OAuthIdentity, error) {
	switch provider {
	case pb.OAuthProvider_OAUTH_PROVIDER_GOOGLE:
		return c.exchangeGoogle(ctx, code, codeVerifier, redirectURI)
	case pb.OAuthProvider_OAUTH_PROVIDER_GITHUB:
		return c.exchangeGitHub(ctx, code, redirectURI)
	case pb.OAuthProvider_OAUTH_PROVIDER_DISCORD:
		return c.exchangeDiscord(ctx, code, redirectURI)
	default:
		return nil, fmt.Errorf("unsupported oauth provider: %v", provider)
	}
}

func (c *providerOAuthClient) exchangeGoogle(ctx context.Context, code, codeVerifier, redirectURI string) (*OAuthIdentity, error) {
	cred := c.cfg.GetGoogle()
	if cred == nil || cred.GetClientId() == "" || cred.GetClientSecret() == "" {
		return nil, errors.New("google oauth credentials not configured")
	}

	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", cred.GetClientId())
	data.Set("client_secret", cred.GetClientSecret())
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")
	if codeVerifier != "" {
		data.Set("code_verifier", codeVerifier)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google token endpoint returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tokenRes struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenRes); err != nil {
		return nil, fmt.Errorf("failed to decode google token response: %w", err)
	}

	// Fetch UserInfo
	userReq, err := http.NewRequestWithContext(ctx, "GET", "https://openidconnect.googleapis.com/v1/userinfo", nil)
	if err != nil {
		return nil, err
	}
	userReq.Header.Set("Authorization", "Bearer "+tokenRes.AccessToken)

	userResp, err := c.httpClient.Do(userReq)
	if err != nil {
		return nil, fmt.Errorf("google userinfo request failed: %w", err)
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(userResp.Body)
		return nil, fmt.Errorf("google userinfo endpoint returned HTTP %d: %s", userResp.StatusCode, string(body))
	}

	var userInfo struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Picture       string `json:"picture"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode google userinfo response: %w", err)
	}

	if userInfo.Email == "" || !userInfo.EmailVerified {
		return nil, errors.New("google account does not have a verified email address")
	}

	primaryEmail := strings.ToLower(strings.TrimSpace(userInfo.Email))
	return &OAuthIdentity{
		Provider:          "google",
		ProviderAccountID: userInfo.Sub,
		PrimaryEmail:      primaryEmail,
		VerifiedEmails:    []string{primaryEmail},
		AvatarURL:         userInfo.Picture,
	}, nil
}

func (c *providerOAuthClient) exchangeGitHub(ctx context.Context, code, redirectURI string) (*OAuthIdentity, error) {
	cred := c.cfg.GetGithub()
	if cred == nil || cred.GetClientId() == "" || cred.GetClientSecret() == "" {
		return nil, errors.New("github oauth credentials not configured")
	}

	data := url.Values{}
	data.Set("client_id", cred.GetClientId())
	data.Set("client_secret", cred.GetClientSecret())
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github token endpoint returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tokenRes struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenRes); err != nil {
		return nil, fmt.Errorf("failed to decode github token response: %w", err)
	}
	if tokenRes.Error != "" {
		return nil, fmt.Errorf("github oauth error: %s (%s)", tokenRes.Error, tokenRes.ErrorDesc)
	}

	// Fetch GitHub user profile
	userReq, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	userReq.Header.Set("Authorization", "Bearer "+tokenRes.AccessToken)
	userReq.Header.Set("Accept", "application/vnd.github+json")

	userResp, err := c.httpClient.Do(userReq)
	if err != nil {
		return nil, fmt.Errorf("github user request failed: %w", err)
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(userResp.Body)
		return nil, fmt.Errorf("github user endpoint returned HTTP %d: %s", userResp.StatusCode, string(body))
	}

	var userProfile struct {
		ID        int64  `json:"id"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&userProfile); err != nil {
		return nil, fmt.Errorf("failed to decode github user profile: %w", err)
	}

	// Fetch GitHub user emails
	emailsReq, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return nil, err
	}
	emailsReq.Header.Set("Authorization", "Bearer "+tokenRes.AccessToken)
	emailsReq.Header.Set("Accept", "application/vnd.github+json")

	emailsResp, err := c.httpClient.Do(emailsReq)
	if err != nil {
		return nil, fmt.Errorf("github emails request failed: %w", err)
	}
	defer emailsResp.Body.Close()

	if emailsResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(emailsResp.Body)
		return nil, fmt.Errorf("github emails endpoint returned HTTP %d: %s", emailsResp.StatusCode, string(body))
	}

	type ghEmail struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	var emails []ghEmail
	if err := json.NewDecoder(emailsResp.Body).Decode(&emails); err != nil {
		return nil, fmt.Errorf("failed to decode github user emails: %w", err)
	}

	var verifiedEmails []string
	var primaryEmail string

	for _, item := range emails {
		norm := strings.ToLower(strings.TrimSpace(item.Email))
		if !item.Verified || norm == "" {
			continue
		}
		// Exclude GitHub private noreply masking addresses
		if strings.HasSuffix(norm, "@users.noreply.github.com") {
			continue
		}
		verifiedEmails = append(verifiedEmails, norm)
		if item.Primary && primaryEmail == "" {
			primaryEmail = norm
		}
	}

	if len(verifiedEmails) == 0 {
		return nil, errors.New("no verified real email address found on github account")
	}

	// If primary email was not found among verified non-noreply emails, use the first verified one
	if primaryEmail == "" {
		primaryEmail = verifiedEmails[0]
	}

	return &OAuthIdentity{
		Provider:          "github",
		ProviderAccountID: strconv.FormatInt(userProfile.ID, 10),
		PrimaryEmail:      primaryEmail,
		VerifiedEmails:    verifiedEmails,
		AvatarURL:         userProfile.AvatarURL,
	}, nil
}

func (c *providerOAuthClient) exchangeDiscord(ctx context.Context, code, redirectURI string) (*OAuthIdentity, error) {
	cred := c.cfg.GetDiscord()
	if cred == nil || cred.GetClientId() == "" || cred.GetClientSecret() == "" {
		return nil, errors.New("discord oauth credentials not configured")
	}

	data := url.Values{}
	data.Set("client_id", cred.GetClientId())
	data.Set("client_secret", cred.GetClientSecret())
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://discord.com/api/v10/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("discord token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discord token endpoint returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tokenRes struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenRes); err != nil {
		return nil, fmt.Errorf("failed to decode discord token response: %w", err)
	}

	// Fetch Discord user profile
	userReq, err := http.NewRequestWithContext(ctx, "GET", "https://discord.com/api/v10/users/@me", nil)
	if err != nil {
		return nil, err
	}
	userReq.Header.Set("Authorization", "Bearer "+tokenRes.AccessToken)

	userResp, err := c.httpClient.Do(userReq)
	if err != nil {
		return nil, fmt.Errorf("discord user request failed: %w", err)
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(userResp.Body)
		return nil, fmt.Errorf("discord user endpoint returned HTTP %d: %s", userResp.StatusCode, string(body))
	}

	var discordUser struct {
		ID       string  `json:"id"`
		Email    string  `json:"email"`
		Verified bool    `json:"verified"`
		Avatar   *string `json:"avatar"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&discordUser); err != nil {
		return nil, fmt.Errorf("failed to decode discord user profile: %w", err)
	}

	if discordUser.Email == "" || !discordUser.Verified {
		return nil, errors.New("discord account does not have a verified email address")
	}

	primaryEmail := strings.ToLower(strings.TrimSpace(discordUser.Email))
	avatarURL := ""
	if discordUser.Avatar != nil && *discordUser.Avatar != "" {
		hash := *discordUser.Avatar
		if strings.HasPrefix(hash, "a_") {
			avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.gif", discordUser.ID, hash)
		} else {
			avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", discordUser.ID, hash)
		}
	}

	return &OAuthIdentity{
		Provider:          "discord",
		ProviderAccountID: discordUser.ID,
		PrimaryEmail:      primaryEmail,
		VerifiedEmails:    []string{primaryEmail},
		AvatarURL:         avatarURL,
	}, nil
}
