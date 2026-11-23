package ops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"

	"quadsmith/shared/pkg/auth"
	"quadsmith/shared/pkg/opsauth"
	"quadsmith/shared/pkg/telemetry"

	"google.golang.org/protobuf/types/known/timestamppb"
	configpb "quadsmith/shared/gen/quadsmith/config"
	pb "quadsmith/shared/gen/quadsmith/ops"
)

const (
	defaultGitHubOAuthBaseURL = "https://github.com"
	defaultGitHubAPIBaseURL   = "https://api.github.com"
	opsGitHubOAuthScopes      = "read:user,read:org"
	maxGitHubTeamPages        = 5
)

type githubTeamMembership struct {
	Org  string
	Slug string
}

func (s *OpsService) getOAuthHTTPClient() *http.Client {
	if s.oauthHTTPClient != nil {
		return s.oauthHTTPClient
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func (s *OpsService) getGitHubOAuthBaseURL() string {
	if strings.TrimSpace(s.githubOAuthBaseURL) != "" {
		return strings.TrimRight(strings.TrimSpace(s.githubOAuthBaseURL), "/")
	}
	return defaultGitHubOAuthBaseURL
}

func (s *OpsService) getGitHubAPIBaseURL() string {
	if strings.TrimSpace(s.githubAPIBaseURL) != "" {
		return strings.TrimRight(strings.TrimSpace(s.githubAPIBaseURL), "/")
	}
	return defaultGitHubAPIBaseURL
}

func (s *OpsService) GetOAuthConfig(_ context.Context, _ *connect.Request[pb.GetOAuthConfigRequest]) (*connect.Response[pb.GetOAuthConfigResponse], error) {
	env := "sandbox"
	clientID := ""
	authURL := "?code=dev-sandbox-code"

	if s.cfg != nil {
		if s.cfg.GetEnvironment() != "" {
			env = s.cfg.GetEnvironment()
		}
		if s.cfg.Oauth != nil {
			if gh := s.cfg.Oauth.GetGithub(); gh != nil {
				clientID = strings.TrimSpace(gh.GetClientId())
				authURL = fmt.Sprintf(
					"%s/login/oauth/authorize?client_id=%s&scope=%s",
					s.getGitHubOAuthBaseURL(),
					url.QueryEscape(clientID),
					opsGitHubOAuthScopes,
				)
			} else if s.cfg.Oauth.GetMock() != nil {
				clientID = "github-sandbox-mock"
				authURL = "?code=dev-sandbox-code"
			}
		}
	}

	return connect.NewResponse(&pb.GetOAuthConfigResponse{
		Provider:     "github",
		ClientId:     clientID,
		Environment:  env,
		Region:       telemetry.CurrentRegion(),
		AuthorizeUrl: authURL,
	}), nil
}

func (s *OpsService) StartOAuthDeviceFlow(ctx context.Context, _ *connect.Request[pb.StartOAuthDeviceFlowRequest]) (*connect.Response[pb.StartOAuthDeviceFlowResponse], error) {
	if s.cfg == nil || s.cfg.Oauth == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("ops OAuth is not configured"))
	}

	if s.cfg.Oauth.GetMock() != nil {
		return connect.NewResponse(&pb.StartOAuthDeviceFlowResponse{
			DeviceCode:       "mock-device-code",
			UserCode:         "QSOP-1234",
			VerificationUri:  "http://127.0.0.1:8083",
			ExpiresInSeconds: 900,
			IntervalSeconds:  1,
		}), nil
	}

	gh := s.cfg.Oauth.GetGithub()
	if gh == nil || strings.TrimSpace(gh.GetClientId()) == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("ops GitHub OAuth client_id is not configured"))
	}

	form := url.Values{}
	form.Set("client_id", strings.TrimSpace(gh.GetClientId()))
	form.Set("scope", opsGitHubOAuthScopes)

	endpoint := s.getGitHubOAuthBaseURL() + "/login/device/code"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to construct GitHub device flow request: %w", err))
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "quadsmith-ops-api")

	resp, err := s.getOAuthHTTPClient().Do(httpReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("failed to contact GitHub device authorization endpoint: %w", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read GitHub device flow response: %w", err))
	}

	var payload struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		ExpiresIn       int32  `json:"expires_in"`
		Interval        int32  `json:"interval"`
		Error           string `json:"error"`
		ErrorDesc       string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to decode GitHub device flow response: %w", err))
	}

	if payload.Error != "" {
		if payload.Error == "device_flow_disabled" {
			return nil, connect.NewError(
				connect.CodeFailedPrecondition,
				errors.New("GitHub Device Flow is disabled on this OAuth App; enable 'Device Flow' in the GitHub OAuth App settings"),
			)
		}
		detail := payload.ErrorDesc
		if detail == "" {
			detail = payload.Error
		}
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("GitHub device flow initiation failed: %s", detail))
	}
	if strings.TrimSpace(payload.DeviceCode) == "" || strings.TrimSpace(payload.UserCode) == "" {
		return nil, connect.NewError(connect.CodeInternal, errors.New("GitHub device flow returned incomplete payload"))
	}

	expiresIn := payload.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 900
	}
	interval := payload.Interval
	if interval <= 0 {
		interval = 5
	}
	verifyURI := strings.TrimSpace(payload.VerificationURI)
	if verifyURI == "" {
		verifyURI = "https://github.com/login/device"
	}

	return connect.NewResponse(&pb.StartOAuthDeviceFlowResponse{
		DeviceCode:       payload.DeviceCode,
		UserCode:         payload.UserCode,
		VerificationUri:  verifyURI,
		ExpiresInSeconds: expiresIn,
		IntervalSeconds:  interval,
	}), nil
}

func (s *OpsService) ExchangeOAuthCode(ctx context.Context, req *connect.Request[pb.ExchangeOAuthCodeRequest]) (*connect.Response[pb.ExchangeOAuthCodeResponse], error) {
	code := strings.TrimSpace(req.Msg.GetCode())
	deviceCode := strings.TrimSpace(req.Msg.GetDeviceCode())
	if code == "" && deviceCode == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("missing authorization code or device_code"))
	}

	if s.cfg == nil || s.cfg.Oauth == nil || len(s.cfg.Oauth.GetTeamMappings()) == 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("ops OAuth team_mappings are not configured"))
	}

	var githubLogin string
	var userTeams []githubTeamMembership

	if s.cfg.Oauth.GetMock() != nil {
		login, teams, err := resolveMockOAuthIdentity(code, deviceCode)
		if err != nil {
			return nil, err
		}
		githubLogin = login
		userTeams = teams
	} else if gh := s.cfg.Oauth.GetGithub(); gh != nil {
		accessToken, pendingResp, err := s.exchangeGitHubAccessToken(
			ctx,
			gh,
			code,
			strings.TrimSpace(req.Msg.GetRedirectUri()),
			strings.TrimSpace(req.Msg.GetCodeVerifier()),
			deviceCode,
		)
		if err != nil {
			return nil, err
		}
		if pendingResp != nil {
			return connect.NewResponse(pendingResp), nil
		}

		login, err := s.fetchGitHubUserLogin(ctx, accessToken)
		if err != nil {
			return nil, err
		}
		teams, err := s.fetchGitHubUserTeams(ctx, accessToken)
		if err != nil {
			return nil, err
		}
		githubLogin = login
		userTeams = teams
	} else {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("ops OAuth mode is not configured"))
	}

	groupName, matched := resolveGroupFromGitHubTeams(s.cfg.Oauth.GetTeamMappings(), userTeams)
	if !matched {
		return nil, connect.NewError(
			connect.CodePermissionDenied,
			fmt.Errorf("GitHub user @%s is not an active member of any authorized GitHub team", githubLogin),
		)
	}

	if !s.isConfiguredGroup(groupName) {
		return nil, connect.NewError(
			connect.CodePermissionDenied,
			fmt.Errorf("mapped IAM group %q is not defined on server-ops-api", groupName),
		)
	}

	tokenSecret := generateCryptoToken("qs_op_")
	tokenHash := opsauth.HashToken(tokenSecret)
	tokenUUID := auth.NewUUIDv7()
	tokenID := fmt.Sprintf("tok_%d", time.Now().UnixNano())
	expiresAt := time.Now().Add(24 * time.Hour)

	if s.opsDb != nil {
		_, err := s.opsDb.Exec(ctx, `
			INSERT INTO internal_tokens (
				uuid, id, name, type, group_name, token_hash, expires_at, created_at
			) VALUES (
				$1, $2, $3, 'HUMAN', $4, $5, $6, NOW()
			)`, tokenUUID, tokenID, githubLogin, groupName, tokenHash, expiresAt)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to issue session token: %w", err))
		}
	}

	return connect.NewResponse(&pb.ExchangeOAuthCodeResponse{
		Token:     tokenSecret,
		ExpiresAt: timestamppb.New(expiresAt),
		Operator:  githubLogin,
		GroupName: groupName,
	}), nil
}

func resolveMockOAuthIdentity(code, deviceCode string) (string, []githubTeamMembership, error) {
	switch {
	case code == "dev-sandbox-code" || code == "mock-operator-code" || deviceCode == "mock-device-code":
		return "sandbox-operator", []githubTeamMembership{
			{Org: "quadsmith", Slug: "operators"},
		}, nil
	case code == "mock-viewer-code":
		return "sandbox-viewer", []githubTeamMembership{
			{Org: "quadsmith", Slug: "viewers"},
		}, nil
	case code == "mock-dual-team-code":
		return "sandbox-dual", []githubTeamMembership{
			{Org: "quadsmith", Slug: "viewers"},
			{Org: "quadsmith", Slug: "operators"},
		}, nil
	case code == "mock-outsider-code":
		return "sandbox-outsider", []githubTeamMembership{
			{Org: "external-org", Slug: "operators"},
		}, nil
	default:
		return "", nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid or expired mock OAuth code"))
	}
}

func resolveGroupFromGitHubTeams(mappings []*configpb.GitHubTeamMapping, teams []githubTeamMembership) (string, bool) {
	membershipSet := make(map[string]bool, len(teams))
	for _, t := range teams {
		org := strings.ToLower(strings.TrimSpace(t.Org))
		slug := strings.ToLower(strings.TrimSpace(t.Slug))
		if org != "" && slug != "" {
			membershipSet[org+"/"+slug] = true
		}
	}

	for _, m := range mappings {
		org := strings.ToLower(strings.TrimSpace(m.GetOrg()))
		slug := strings.ToLower(strings.TrimSpace(m.GetTeamSlug()))
		group := strings.TrimSpace(m.GetGroupName())
		if org != "" && slug != "" && group != "" && membershipSet[org+"/"+slug] {
			return group, true
		}
	}

	return "", false
}

func (s *OpsService) isConfiguredGroup(groupName string) bool {
	if s.cfg == nil {
		return false
	}
	for _, g := range s.cfg.GetGroups() {
		if g.GetName() == groupName {
			return true
		}
	}
	return false
}

func (s *OpsService) exchangeGitHubAccessToken(
	ctx context.Context,
	gh *configpb.OpsGitHubOAuthConfig,
	code, redirectURI, codeVerifier, deviceCode string,
) (string, *pb.ExchangeOAuthCodeResponse, error) {
	form := url.Values{}
	form.Set("client_id", strings.TrimSpace(gh.GetClientId()))
	form.Set("client_secret", strings.TrimSpace(gh.GetClientSecret()))
	if deviceCode != "" {
		form.Set("device_code", deviceCode)
		form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	} else {
		form.Set("code", code)
		if redirectURI != "" {
			form.Set("redirect_uri", redirectURI)
		}
		if codeVerifier != "" {
			form.Set("code_verifier", codeVerifier)
		}
	}

	endpoint := s.getGitHubOAuthBaseURL() + "/login/oauth/access_token"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create GitHub OAuth token request: %w", err))
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "quadsmith-ops-api")

	resp, err := s.getOAuthHTTPClient().Do(httpReq)
	if err != nil {
		return "", nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("failed to contact GitHub OAuth token endpoint: %w", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read GitHub OAuth token response: %w", err))
	}

	var tokenPayload struct {
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
		Interval    int32  `json:"interval"`
	}
	if err := json.Unmarshal(body, &tokenPayload); err != nil {
		return "", nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to decode GitHub OAuth token response: %w", err))
	}

	if tokenPayload.Error != "" {
		if deviceCode != "" {
			switch tokenPayload.Error {
			case "authorization_pending":
				return "", &pb.ExchangeOAuthCodeResponse{
					Pending:           true,
					RetryAfterSeconds: 5,
				}, nil
			case "slow_down":
				waitSec := tokenPayload.Interval
				if waitSec <= 0 {
					waitSec = 10
				}
				return "", &pb.ExchangeOAuthCodeResponse{
					Pending:           true,
					RetryAfterSeconds: waitSec,
				}, nil
			}
		}
		detail := tokenPayload.ErrorDesc
		if detail == "" {
			detail = tokenPayload.Error
		}
		return "", nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("GitHub OAuth token exchange failed: %s", detail))
	}

	accessToken := strings.TrimSpace(tokenPayload.AccessToken)
	if accessToken == "" {
		return "", nil, connect.NewError(connect.CodeUnauthenticated, errors.New("GitHub OAuth token exchange returned an empty access_token"))
	}

	return accessToken, nil, nil
}

func (s *OpsService) fetchGitHubUserLogin(ctx context.Context, accessToken string) (string, error) {
	endpoint := s.getGitHubAPIBaseURL() + "/user"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create GitHub /user request: %w", err))
	}
	httpReq.Header.Set("Authorization", "Bearer "+accessToken)
	httpReq.Header.Set("Accept", "application/vnd.github+json")
	httpReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	httpReq.Header.Set("User-Agent", "quadsmith-ops-api")

	resp, err := s.getOAuthHTTPClient().Do(httpReq)
	if err != nil {
		return "", connect.NewError(connect.CodeUnavailable, fmt.Errorf("failed to query GitHub /user API: %w", err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read GitHub /user response: %w", err))
	}
	if resp.StatusCode != http.StatusOK {
		return "", connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("GitHub /user API returned status %d", resp.StatusCode))
	}

	var userPayload struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal(body, &userPayload); err != nil {
		return "", connect.NewError(connect.CodeInternal, fmt.Errorf("failed to decode GitHub /user response: %w", err))
	}
	login := strings.TrimSpace(userPayload.Login)
	if login == "" {
		return "", connect.NewError(connect.CodeUnauthenticated, errors.New("GitHub /user response missing login handle"))
	}
	return login, nil
}

func (s *OpsService) fetchGitHubUserTeams(ctx context.Context, accessToken string) ([]githubTeamMembership, error) {
	var allTeams []githubTeamMembership

	for page := 1; page <= maxGitHubTeamPages; page++ {
		endpoint := fmt.Sprintf("%s/user/teams?per_page=100&page=%d", s.getGitHubAPIBaseURL(), page)
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create GitHub /user/teams request: %w", err))
		}
		httpReq.Header.Set("Authorization", "Bearer "+accessToken)
		httpReq.Header.Set("Accept", "application/vnd.github+json")
		httpReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		httpReq.Header.Set("User-Agent", "quadsmith-ops-api")

		resp, err := s.getOAuthHTTPClient().Do(httpReq)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("failed to query GitHub /user/teams API: %w", err))
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		resp.Body.Close()
		if readErr != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read GitHub /user/teams response: %w", readErr))
		}
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
			return nil, connect.NewError(
				connect.CodePermissionDenied,
				fmt.Errorf("GitHub /user/teams API returned status %d (ensure read:org scope and GitHub OAuth App organization access are granted)", resp.StatusCode),
			)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("GitHub /user/teams API returned status %d", resp.StatusCode))
		}

		var pageItems []struct {
			Slug         string `json:"slug"`
			Organization struct {
				Login string `json:"login"`
			} `json:"organization"`
		}
		if err := json.Unmarshal(body, &pageItems); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to decode GitHub /user/teams response: %w", err))
		}

		for _, item := range pageItems {
			allTeams = append(allTeams, githubTeamMembership{
				Org:  item.Organization.Login,
				Slug: item.Slug,
			})
		}

		if len(pageItems) < 100 {
			break
		}
	}

	return allTeams, nil
}
