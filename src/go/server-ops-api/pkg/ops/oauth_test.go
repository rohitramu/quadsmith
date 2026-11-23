package ops

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	"quadsmith/shared/pkg/config"

	configpb "quadsmith/shared/gen/quadsmith/config"
	pb "quadsmith/shared/gen/quadsmith/ops"
)

func TestOpsOAuth_GitHubTeamMappingsAndDenyByDefault(t *testing.T) {
	ctx := context.Background()

	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/login/device/code":
			_ = r.ParseForm()
			if r.Form.Get("client_id") != "gh-client-123" || r.Form.Get("scope") != "read:user,read:org" {
				http.Error(w, `{"error":"invalid_client"}`, http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"device_code":      "dev_code_live_1",
				"user_code":        "ABCD-9876",
				"verification_uri": "https://github.com/login/device",
				"expires_in":       900,
				"interval":         5,
			})

		case "/login/oauth/access_token":
			_ = r.ParseForm()
			if r.Form.Get("device_code") == "dev_code_pending" {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": "authorization_pending",
				})
				return
			}
			if r.Form.Get("device_code") == "dev_code_live_1" {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"access_token": "gho_operator_token",
					"scope":        "read:user,read:org",
				})
				return
			}
			switch r.Form.Get("code") {
			case "code_operator":
				_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "gho_operator_token"})
			case "code_viewer":
				_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "gho_viewer_token"})
			case "code_both_teams":
				_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "gho_both_token"})
			case "code_org_member_no_team":
				_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "gho_no_team_token"})
			default:
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error":             "bad_verification_code",
					"error_description": "The code passed is incorrect or expired.",
				})
			}

		case "/user":
			switch r.Header.Get("Authorization") {
			case "Bearer gho_operator_token":
				_ = json.NewEncoder(w).Encode(map[string]any{"login": "alice-ops"})
			case "Bearer gho_viewer_token":
				_ = json.NewEncoder(w).Encode(map[string]any{"login": "bob-viewer"})
			case "Bearer gho_both_token":
				_ = json.NewEncoder(w).Encode(map[string]any{"login": "carol-lead"})
			case "Bearer gho_no_team_token":
				_ = json.NewEncoder(w).Encode(map[string]any{"login": "dave-contractor"})
			default:
				w.WriteHeader(http.StatusUnauthorized)
			}

		case "/user/teams":
			switch r.Header.Get("Authorization") {
			case "Bearer gho_operator_token":
				_ = json.NewEncoder(w).Encode([]map[string]any{
					{"slug": "operators", "organization": map[string]any{"login": "Quadsmith"}},
				})
			case "Bearer gho_viewer_token":
				_ = json.NewEncoder(w).Encode([]map[string]any{
					{"slug": "viewers", "organization": map[string]any{"login": "quadsmith"}},
				})
			case "Bearer gho_both_token":
				// Even when GitHub returns viewers before operators, config-order precedence selects operators.
				_ = json.NewEncoder(w).Encode([]map[string]any{
					{"slug": "viewers", "organization": map[string]any{"login": "quadsmith"}},
					{"slug": "operators", "organization": map[string]any{"login": "quadsmith"}},
				})
			case "Bearer gho_no_team_token":
				// Member of unrelated team/org -> must be denied by default!
				_ = json.NewEncoder(w).Encode([]map[string]any{
					{"slug": "contributors", "organization": map[string]any{"login": "quadsmith"}},
				})
			default:
				w.WriteHeader(http.StatusUnauthorized)
			}

		default:
			http.NotFound(w, r)
		}
	}))
	defer ghServer.Close()

	cfg := config.NewTestOpsApiServerConfig(t, func(c *configpb.OpsApiServerConfig) {
		c.Environment = proto.String("prod")
		c.Oauth.Mode = &configpb.OAuthConfig_Github{
			Github: &configpb.OpsGitHubOAuthConfig{
				ClientId:     proto.String("gh-client-123"),
				ClientSecret: proto.String("gh-secret-456"),
			},
		}
	})

	svc := NewOpsService(nil, nil, cfg)
	svc.oauthHTTPClient = ghServer.Client()
	svc.githubOAuthBaseURL = ghServer.URL
	svc.githubAPIBaseURL = ghServer.URL

	// 1. GetOAuthConfig includes read:user,read:org scope
	cfgRes, err := svc.GetOAuthConfig(ctx, connect.NewRequest(&pb.GetOAuthConfigRequest{}))
	if err != nil {
		t.Fatalf("GetOAuthConfig failed: %v", err)
	}
	if cfgRes.Msg.GetClientId() != "gh-client-123" {
		t.Errorf("expected client_id gh-client-123, got %q", cfgRes.Msg.GetClientId())
	}
	if want := ghServer.URL + "/login/oauth/authorize?client_id=gh-client-123&scope=read:user,read:org"; cfgRes.Msg.GetAuthorizeUrl() != want {
		t.Errorf("expected authorize_url %q, got %q", want, cfgRes.Msg.GetAuthorizeUrl())
	}

	// 2. StartOAuthDeviceFlow & pending/completed device polling
	devStart, err := svc.StartOAuthDeviceFlow(ctx, connect.NewRequest(&pb.StartOAuthDeviceFlowRequest{}))
	if err != nil {
		t.Fatalf("StartOAuthDeviceFlow failed: %v", err)
	}
	if devStart.Msg.GetUserCode() != "ABCD-9876" || devStart.Msg.GetDeviceCode() != "dev_code_live_1" {
		t.Fatalf("unexpected device flow response: %+v", devStart.Msg)
	}

	pendingRes, err := svc.ExchangeOAuthCode(ctx, connect.NewRequest(&pb.ExchangeOAuthCodeRequest{
		DeviceCode: "dev_code_pending",
	}))
	if err != nil {
		t.Fatalf("expected nil error on authorization_pending, got %v", err)
	}
	if !pendingRes.Msg.GetPending() || pendingRes.Msg.GetToken() != "" {
		t.Fatalf("expected pending=true and empty token, got %+v", pendingRes.Msg)
	}

	devComplete, err := svc.ExchangeOAuthCode(ctx, connect.NewRequest(&pb.ExchangeOAuthCodeRequest{
		DeviceCode: "dev_code_live_1",
	}))
	if err != nil {
		t.Fatalf("device flow completion failed: %v", err)
	}
	if devComplete.Msg.GetOperator() != "alice-ops" || devComplete.Msg.GetGroupName() != "operators" || devComplete.Msg.GetToken() == "" {
		t.Errorf("unexpected device completion payload: %+v", devComplete.Msg)
	}

	// 3. @quadsmith/viewers maps to viewers
	viewerRes, err := svc.ExchangeOAuthCode(ctx, connect.NewRequest(&pb.ExchangeOAuthCodeRequest{
		Code: "code_viewer",
	}))
	if err != nil {
		t.Fatalf("viewer code exchange failed: %v", err)
	}
	if viewerRes.Msg.GetOperator() != "bob-viewer" || viewerRes.Msg.GetGroupName() != "viewers" {
		t.Errorf("expected bob-viewer in viewers, got %+v", viewerRes.Msg)
	}

	// 4. Member of both viewers and operators gets operators (first team_mappings match wins)
	bothRes, err := svc.ExchangeOAuthCode(ctx, connect.NewRequest(&pb.ExchangeOAuthCodeRequest{
		Code: "code_both_teams",
	}))
	if err != nil {
		t.Fatalf("both-teams code exchange failed: %v", err)
	}
	if bothRes.Msg.GetOperator() != "carol-lead" || bothRes.Msg.GetGroupName() != "operators" {
		t.Errorf("expected carol-lead in operators, got %+v", bothRes.Msg)
	}

	// 5. Deny-by-default: org member not in @quadsmith/operators or @quadsmith/viewers is rejected!
	_, err = svc.ExchangeOAuthCode(ctx, connect.NewRequest(&pb.ExchangeOAuthCodeRequest{
		Code: "code_org_member_no_team",
	}))
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied for unmapped team member, got %v", err)
	}

	// 6. Invalid verification code returns CodeUnauthenticated
	_, err = svc.ExchangeOAuthCode(ctx, connect.NewRequest(&pb.ExchangeOAuthCodeRequest{
		Code: "bad_code",
	}))
	if err == nil || connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated for invalid code, got %v", err)
	}
}
