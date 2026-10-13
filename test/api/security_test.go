package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"quadsmith/shared/pkg/auth"

	pb "quadsmith/shared/gen/quadsmith/api"
	"quadsmith/shared/gen/quadsmith/api/apiconnect"
)

func TestSandboxSecurity_SQLInjectionRejectedOnColumns(t *testing.T) {
	apiUrl := getAPIURL()
	client := &http.Client{Timeout: 2 * time.Second}
	if _, err := client.Get(apiUrl); err != nil {
		t.Fatalf("Sandbox not running at %s: %v", apiUrl, err)
	}

	buildClient := apiconnect.NewBuildServiceClient(
		http.DefaultClient,
		apiUrl,
	)
	ctx := context.Background()

	// 1. GetBuild with injected SQL in columns
	_, err := buildClient.GetBuild(ctx, connect.NewRequest(&pb.GetBuildRequest{
		Id:      "bando-basher-5-inch",
		Columns: []string{"id", "name; DROP TABLE builds;--"},
	}))
	if err == nil {
		t.Fatal("expected GetBuild to fail when column contains SQL injection, got nil")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v: %v", connect.CodeOf(err), err)
	}

	// 2. ListBuilds with injected SQL in columns
	_, err = buildClient.ListBuilds(ctx, connect.NewRequest(&pb.ListBuildsRequest{
		Columns: []string{"(SELECT 'injected')"},
	}))
	if err == nil {
		t.Fatal("expected ListBuilds to fail when column contains SQL injection, got nil")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v: %v", connect.CodeOf(err), err)
	}
}

func TestSandboxSecurity_CreateBuildOwnershipEnforcement(t *testing.T) {
	apiUrl := getAPIURL()
	client := &http.Client{Timeout: 2 * time.Second}
	if _, err := client.Get(apiUrl); err != nil {
		t.Fatalf("Sandbox not running at %s: %v", apiUrl, err)
	}

	buildClient := apiconnect.NewBuildServiceClient(
		http.DefaultClient,
		apiUrl,
	)
	ctx := context.Background()

	attackerUserUUID := "01920000-0000-7000-8000-000000000999"
	attackerUsername := "victim_target"
	clientProvidedUUID := "01920000-0000-7000-8000-111111111111"

	token, err := auth.SignAccessToken("018f0000-0000-7000-8000-000000000001", "test_pilot", "pilot@example.com", "user", "session-1")
	if err != nil {
		t.Fatalf("SignAccessToken failed: %v", err)
	}

	// Attempt to create a build claiming to belong to another user while authenticated as test_pilot
	req := connect.NewRequest(&pb.CreateBuildRequest{
		Build: &pb.Build{
			Uuid:                 clientProvidedUUID,
			UserUuid:             &attackerUserUUID,
			AuthorUsername:       &attackerUsername,
			Name:                 "Ownership Security Test Build",
			Description:          "Testing ownership enforcement",
			FrameUuid:            "01923019-3008-7001-8001-000000000001",
			MotorUuid:            "01923019-3001-7001-8001-000000000001",
			PropellerUuid:        "bac19aec-99fc-43d4-b96d-d1feb30e6b6e",
			FlightControllerUuid: "01923019-3006-7001-8001-000000000001",
			VideoTransmitterUuid: "01923019-3004-7001-8001-000000000008",
		},
	})
	req.Header().Set("Authorization", "Bearer "+token)

	res, err := buildClient.CreateBuild(ctx, req)
	if err != nil {
		t.Fatalf("CreateBuild failed: %v", err)
	}

	// Server must overwrite client-provided UUID, UserUuid, and AuthorUsername
	if res.Msg.Uuid == clientProvidedUUID {
		t.Errorf("expected server to assign fresh UUID v7, but client-provided UUID was accepted")
	}
	if res.Msg.GetUserUuid() == attackerUserUUID {
		t.Errorf("vulnerability: attacker was able to spoof user_uuid!")
	}
	if res.Msg.GetAuthorUsername() == attackerUsername {
		t.Errorf("vulnerability: attacker was able to spoof author_username!")
	}
	if res.Msg.GetUserUuid() != "018f0000-0000-7000-8000-000000000001" {
		t.Errorf("expected user_uuid to match authenticated session ('018f0000-0000-7000-8000-000000000001'), got '%s'", res.Msg.GetUserUuid())
	}
	if res.Msg.GetAuthorUsername() != "test_pilot" {
		t.Errorf("expected author_username to match authenticated session ('test_pilot'), got '%s'", res.Msg.GetAuthorUsername())
	}

	// Also verify unauthenticated build creation cannot spoof owner
	unauthClient := &http.Client{Transport: http.DefaultTransport}
	unauthBuildClient := apiconnect.NewBuildServiceClient(unauthClient, apiUrl)
	unauthReq := connect.NewRequest(&pb.CreateBuildRequest{
		Build: &pb.Build{
			UserUuid:             &attackerUserUUID,
			AuthorUsername:       &attackerUsername,
			Name:                 "Unauthenticated Ownership Test",
			FrameUuid:            "01923019-3008-7001-8001-000000000001",
			MotorUuid:            "01923019-3001-7001-8001-000000000001",
			PropellerUuid:        "bac19aec-99fc-43d4-b96d-d1feb30e6b6e",
			FlightControllerUuid: "01923019-3006-7001-8001-000000000001",
			VideoTransmitterUuid: "01923019-3004-7001-8001-000000000008",
		},
	})
	_, err = unauthBuildClient.CreateBuild(ctx, unauthReq)
	if err == nil {
		t.Fatal("expected unauthenticated CreateBuild to fail, got nil")
	}
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("expected CodeUnauthenticated, got %v: %v", connect.CodeOf(err), err)
	}
}

func TestSandboxSecurity_CreateBuildRejectsJavascriptSchemeAndLongURLs(t *testing.T) {
	apiUrl := getAPIURL()
	client := &http.Client{Timeout: 2 * time.Second}
	if _, err := client.Get(apiUrl); err != nil {
		t.Fatalf("Sandbox not running at %s: %v", apiUrl, err)
	}

	buildClient := apiconnect.NewBuildServiceClient(
		http.DefaultClient,
		apiUrl,
	)
	ctx := context.Background()

	// 1. Javascript scheme
	_, err := buildClient.CreateBuild(ctx, connect.NewRequest(&pb.CreateBuildRequest{
		Build: &pb.Build{
			Name: "XSS Test Build",
			ReferenceLinks: []*pb.ReferenceLink{
				{
					Url:   "javascript:alert(document.cookie)",
					Types: []pb.ReferenceLinkType{pb.ReferenceLinkType_REFERENCE_LINK_TYPE_PRODUCT_PAGE},
				},
			},
		},
	}))
	if err == nil {
		t.Fatal("expected CreateBuild to reject javascript: scheme, but succeeded")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v", connect.CodeOf(err))
	}

	// 2. Excessively long URL (> 2048 chars)
	longURL := "https://example.com/" + strings.Repeat("a", 2100)
	_, err = buildClient.CreateBuild(ctx, connect.NewRequest(&pb.CreateBuildRequest{
		Build: &pb.Build{
			Name: "Long URL Test Build",
			ReferenceLinks: []*pb.ReferenceLink{
				{
					Url:   longURL,
					Types: []pb.ReferenceLinkType{pb.ReferenceLinkType_REFERENCE_LINK_TYPE_PRODUCT_PAGE},
				},
			},
		},
	}))
	if err == nil {
		t.Fatal("expected CreateBuild to reject URL exceeding 2048 chars, but succeeded")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v", connect.CodeOf(err))
	}
}

func TestSandboxSecurity_GetUserRequiresAuthorization(t *testing.T) {
	apiUrl := getAPIURL()
	client := &http.Client{Timeout: 2 * time.Second}
	if _, err := client.Get(apiUrl); err != nil {
		t.Fatalf("Sandbox not running at %s: %v", apiUrl, err)
	}

	userClient := apiconnect.NewUserServiceClient(
		http.DefaultClient,
		apiUrl,
	)
	ctx := context.Background()

	// 1. Unauthenticated GetUser must be rejected with CodeUnauthenticated
	unauthClient := &http.Client{Transport: http.DefaultTransport}
	unauthUserClient := apiconnect.NewUserServiceClient(unauthClient, apiUrl)
	_, err := unauthUserClient.GetUser(ctx, connect.NewRequest(&pb.GetUserRequest{
		Id: "018f0000-0000-7000-8000-000000000001",
	}))
	if err == nil {
		t.Fatal("expected unauthenticated GetUser to fail, got nil")
	}
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("expected CodeUnauthenticated for unauthenticated GetUser, got %v: %v", connect.CodeOf(err), err)
	}

	token, err := auth.SignAccessToken("018f0000-0000-7000-8000-000000000001", "test_pilot", "pilot@example.com", "user", "session-1")
	if err != nil {
		t.Fatalf("SignAccessToken failed: %v", err)
	}

	// 2. Authenticated as test_pilot (018f0000-0000-7000-8000-000000000001).
	// Attempting to query another user's private User record must be rejected with PermissionDenied.
	crossReq := connect.NewRequest(&pb.GetUserRequest{
		Id: "01923019-3000-7000-8000-000000000001",
	})
	crossReq.Header().Set("Authorization", "Bearer "+token)
	_, err = userClient.GetUser(ctx, crossReq)
	if err == nil {
		t.Fatal("expected GetUser for another user to be rejected with PermissionDenied, got nil")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied && connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("expected CodePermissionDenied or CodeNotFound, got %v: %v", connect.CodeOf(err), err)
	}

	// 3. Authenticated GetUser for oneself should succeed (or return NotFound if user not yet created)
	selfReq := connect.NewRequest(&pb.GetUserRequest{
		Id: "018f0000-0000-7000-8000-000000000001",
	})
	selfReq.Header().Set("Authorization", "Bearer "+token)
	_, err = userClient.GetUser(ctx, selfReq)
	// Must NOT return PermissionDenied or Unauthenticated
	if connect.CodeOf(err) == connect.CodePermissionDenied || connect.CodeOf(err) == connect.CodeUnauthenticated {
		t.Errorf("expected authorized self-lookup, got auth error: %v", err)
	}
}
