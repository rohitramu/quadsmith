package api

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/gen/quadsmith/quadsmithconnect"
)

func getAPIURL() string {
	if url := os.Getenv("QS_API_URL"); url != "" {
		return url
	}
	return "http://127.0.0.1:8080"
}

func TestSandboxBuilds(t *testing.T) {
	apiUrl := getAPIURL()
	// Fail if we can't reach the sandbox
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(apiUrl)
	if err != nil {
		t.Fatalf("Sandbox not running at %s: %v", apiUrl, err)
	}

	buildClient := quadsmithconnect.NewBuildServiceClient(
		http.DefaultClient,
		apiUrl,
	)

	ctx := context.Background()
	req := connect.NewRequest(&pb.ListBuildsRequest{})

	res, err := buildClient.ListBuilds(ctx, req)
	if err != nil {
		t.Fatalf("Failed to list builds: %v", err)
	}

	if res.Msg.Builds == nil {
		t.Log("Got nil builds array (no builds exist yet)")
	} else {
		t.Logf("Got %d builds", len(res.Msg.Builds))
	}
}

func TestSandboxFrames(t *testing.T) {
	apiUrl := getAPIURL()
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(apiUrl)
	if err != nil {
		t.Fatalf("Sandbox not running at %s: %v", apiUrl, err)
	}

	frameClient := quadsmithconnect.NewFrameServiceClient(
		http.DefaultClient,
		apiUrl,
	)

	ctx := context.Background()
	req := connect.NewRequest(&pb.ListFramesRequest{})

	res, err := frameClient.ListFrames(ctx, req)
	if err != nil {
		t.Fatalf("Failed to list frames: %v", err)
	}

	if len(res.Msg.Frames) == 0 {
		t.Log("No frames found")
	} else {
		t.Logf("Found %d frames, e.g. %s", len(res.Msg.Frames), res.Msg.Frames[0].Name)
	}
}

func TestSandboxEvaluateBuild(t *testing.T) {
	apiUrl := getAPIURL()
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(apiUrl)
	if err != nil {
		t.Fatalf("Sandbox not running at %s: %v", apiUrl, err)
	}

	evalClient := quadsmithconnect.NewEvaluatorServiceClient(
		http.DefaultClient,
		apiUrl,
	)
	ctx := context.Background()

	// 1. Missing battery_id should return InvalidArgument
	_, err = evalClient.EvaluateBuild(ctx, connect.NewRequest(&pb.EvaluateBuildRequest{
		BuildSource: &pb.EvaluateBuildRequest_BuildId{BuildId: "ultralight-toothpick"},
	}))
	if err == nil {
		t.Fatal("expected error when battery_id is missing, got nil")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v: %v", connect.CodeOf(err), err)
	}
	if !strings.Contains(err.Error(), "battery_id is required") {
		t.Errorf("expected 'battery_id is required' error, got: %v", err)
	}

	// 2. Non-existent battery should return NotFound
	_, err = evalClient.EvaluateBuild(ctx, connect.NewRequest(&pb.EvaluateBuildRequest{
		BuildSource: &pb.EvaluateBuildRequest_BuildId{BuildId: "ultralight-toothpick"},
		BatteryId:   "non-existent-battery",
	}))
	if err == nil {
		t.Fatal("expected error when battery does not exist, got nil")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v: %v", connect.CodeOf(err), err)
	}

	// 3. Missing build_id should return InvalidArgument
	_, err = evalClient.EvaluateBuild(ctx, connect.NewRequest(&pb.EvaluateBuildRequest{
		BatteryId: "betafpv-lava-1s-300mah-75c-lihv",
	}))
	if err == nil {
		t.Fatal("expected error when build_id is missing, got nil")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v: %v", connect.CodeOf(err), err)
	}
	if !strings.Contains(err.Error(), "either build or build_id must be provided") && !strings.Contains(err.Error(), "build_id is required") {
		t.Errorf("expected build source error, got: %v", err)
	}

	// 4. Non-existent build should return NotFound
	_, err = evalClient.EvaluateBuild(ctx, connect.NewRequest(&pb.EvaluateBuildRequest{
		BuildSource: &pb.EvaluateBuildRequest_BuildId{BuildId: "non-existent-build"},
		BatteryId:   "betafpv-lava-1s-300mah-75c-lihv",
	}))
	if err == nil {
		t.Fatal("expected error when build does not exist, got nil")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("expected CodeNotFound, got %v: %v", connect.CodeOf(err), err)
	}

	// 5. Valid build with battery evaluates successfully
	res, err := evalClient.EvaluateBuild(ctx, connect.NewRequest(&pb.EvaluateBuildRequest{
		BuildSource: &pb.EvaluateBuildRequest_BuildId{BuildId: "ultralight-toothpick"},
		BatteryId:   "betafpv-lava-1s-300mah-75c-lihv",
	}))
	if err != nil {
		t.Fatalf("EvaluateBuild failed: %v", err)
	}
	if res.Msg.BuildId != "ultralight-toothpick" {
		t.Errorf("expected build_id == 'ultralight-toothpick', got: %v", res.Msg.BuildId)
	}
	if res.Msg.BatteryId != "betafpv-lava-1s-300mah-75c-lihv" {
		t.Errorf("expected battery_id == 'betafpv-lava-1s-300mah-75c-lihv', got: %v", res.Msg.BatteryId)
	}
	if res.Msg.AllUpWeightG <= 0 {
		t.Errorf("expected positive all_up_weight_g, got: %v", res.Msg.AllUpWeightG)
	}
	if res.Msg.ThrustToWeightRatio <= 0 {
		t.Errorf("expected positive thrust_to_weight_ratio, got: %v", res.Msg.ThrustToWeightRatio)
	}

	// 6. Evaluate with payload and verify all-up weight includes the payload
	resWithPayload, err := evalClient.EvaluateBuild(ctx, connect.NewRequest(&pb.EvaluateBuildRequest{
		BuildSource:    &pb.EvaluateBuildRequest_BuildId{BuildId: "ultralight-toothpick"},
		BatteryId:      "betafpv-lava-1s-300mah-75c-lihv",
		PayloadWeightG: 25.0,
	}))
	if err != nil {
		t.Fatalf("EvaluateBuild with payload failed: %v", err)
	}
	diff := resWithPayload.Msg.AllUpWeightG - res.Msg.AllUpWeightG
	if diff < 24.9 || diff > 25.1 {
		t.Errorf("expected all-up weight to increase by 25g with payload, got diff=%.2f (before=%.2f, after=%.2f)", diff, res.Msg.AllUpWeightG, resWithPayload.Msg.AllUpWeightG)
	}

	// 7. Evaluate in-memory draft Build object
	resDraft, err := evalClient.EvaluateBuild(ctx, connect.NewRequest(&pb.EvaluateBuildRequest{
		BuildSource: &pb.EvaluateBuildRequest_Build{
			Build: &pb.Build{
				FrameUuid:     "01923019-3008-7001-8001-000000000001",
				MotorUuid:     "01923019-3001-7001-8001-000000000001",
				PropellerUuid: "bac19aec-99fc-43d4-b96d-d1feb30e6b6e",
			},
		},
		BatteryId: "betafpv-lava-1s-300mah-75c-lihv",
	}))
	if err != nil {
		t.Fatalf("EvaluateBuild with in-memory draft build failed: %v", err)
	}
	if resDraft.Msg.AllUpWeightG <= 0 {
		t.Errorf("expected positive all-up weight for in-memory draft, got: %v", resDraft.Msg.AllUpWeightG)
	}
}

func TestSandboxCreateBuildDuplicateID(t *testing.T) {
	apiUrl := getAPIURL()
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(apiUrl)
	if err != nil {
		t.Fatalf("Sandbox not running at %s: %v", apiUrl, err)
	}

	buildClient := quadsmithconnect.NewBuildServiceClient(
		http.DefaultClient,
		apiUrl,
	)
	ctx := context.Background()

	// 1. Create a build whose name generates the same slug as the seeded "bando-basher-5-inch"
	req1 := connect.NewRequest(&pb.CreateBuildRequest{
		Build: &pb.Build{
			Name:                 "Bando Basher 5 inch",
			Description:          "Copy test build 1",
			FrameUuid:            "01923019-3008-7001-8001-000000000001",
			MotorUuid:            "01923019-3001-7001-8001-000000000001",
			PropellerUuid:        "bac19aec-99fc-43d4-b96d-d1feb30e6b6e",
			FlightControllerUuid: "01923019-3006-7001-8001-000000000001",
			VideoTransmitterUuid: "01923019-3004-7001-8001-000000000008",
		},
	})
	res1, err := buildClient.CreateBuild(ctx, req1)
	if err != nil {
		t.Fatalf("Failed to create first duplicate build: %v", err)
	}
	if res1.Msg.Id != "bando-basher-5-inch-copy1" {
		t.Errorf("expected id 'bando-basher-5-inch-copy1', got '%s'", res1.Msg.Id)
	}

	// 2. Create another build with the same name, should auto-increment to -copy2
	req2 := connect.NewRequest(&pb.CreateBuildRequest{
		Build: &pb.Build{
			Name:                 "Bando Basher 5 inch",
			Description:          "Copy test build 2",
			FrameUuid:            "01923019-3008-7001-8001-000000000001",
			MotorUuid:            "01923019-3001-7001-8001-000000000001",
			PropellerUuid:        "bac19aec-99fc-43d4-b96d-d1feb30e6b6e",
			FlightControllerUuid: "01923019-3006-7001-8001-000000000001",
			VideoTransmitterUuid: "01923019-3004-7001-8001-000000000008",
		},
	})
	res2, err := buildClient.CreateBuild(ctx, req2)
	if err != nil {
		t.Fatalf("Failed to create second duplicate build: %v", err)
	}
	if res2.Msg.Id != "bando-basher-5-inch-copy2" {
		t.Errorf("expected id 'bando-basher-5-inch-copy2', got '%s'", res2.Msg.Id)
	}

	// 3. Create a build explicitly providing an ID that already has -copy2 suffix
	req3 := connect.NewRequest(&pb.CreateBuildRequest{
		Build: &pb.Build{
			Id:                   "bando-basher-5-inch-copy2",
			Name:                 "Explicit Copy Build",
			Description:          "Copy test build 3",
			FrameUuid:            "01923019-3008-7001-8001-000000000001",
			MotorUuid:            "01923019-3001-7001-8001-000000000001",
			PropellerUuid:        "bac19aec-99fc-43d4-b96d-d1feb30e6b6e",
			FlightControllerUuid: "01923019-3006-7001-8001-000000000001",
			VideoTransmitterUuid: "01923019-3004-7001-8001-000000000008",
		},
	})
	res3, err := buildClient.CreateBuild(ctx, req3)
	if err != nil {
		t.Fatalf("Failed to create third duplicate build: %v", err)
	}
	if res3.Msg.Id != "bando-basher-5-inch-copy3" {
		t.Errorf("expected id 'bando-basher-5-inch-copy3', got '%s'", res3.Msg.Id)
	}
}

func TestSandboxCreateBuild_RejectsUnspecifiedReferenceLink(t *testing.T) {
	apiUrl := getAPIURL()
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(apiUrl)
	if err != nil {
		t.Fatalf("Sandbox not running at %s: %v", apiUrl, err)
	}

	buildClient := quadsmithconnect.NewBuildServiceClient(
		http.DefaultClient,
		apiUrl,
	)
	ctx := context.Background()

	req := connect.NewRequest(&pb.CreateBuildRequest{
		Build: &pb.Build{
			Name: "Invalid Reference Link Build",
			ReferenceLinks: []*pb.ReferenceLink{
				{
					Types: []pb.ReferenceLinkType{pb.ReferenceLinkType_REFERENCE_LINK_TYPE_UNSPECIFIED},
					Url:   "https://example.com/invalid",
				},
			},
		},
	})
	_, err = buildClient.CreateBuild(ctx, req)
	if err == nil {
		t.Fatal("expected CreateBuild to fail with InvalidArgument when REFERENCE_LINK_TYPE_UNSPECIFIED is provided, but it succeeded")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v: %v", connect.CodeOf(err), err)
	}
	if !strings.Contains(err.Error(), "REFERENCE_LINK_TYPE_UNSPECIFIED is not permitted") {
		t.Errorf("expected error message to mention REFERENCE_LINK_TYPE_UNSPECIFIED is not permitted, got: %v", err)
	}
}

func TestSandboxCreateBuild_RejectsDuplicateReferenceLinkURL(t *testing.T) {
	apiUrl := getAPIURL()
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(apiUrl)
	if err != nil {
		t.Fatalf("Sandbox not running at %s: %v", apiUrl, err)
	}

	buildClient := quadsmithconnect.NewBuildServiceClient(
		http.DefaultClient,
		apiUrl,
	)
	ctx := context.Background()

	req := connect.NewRequest(&pb.CreateBuildRequest{
		Build: &pb.Build{
			Name: "Duplicate URL Reference Link Build",
			ReferenceLinks: []*pb.ReferenceLink{
				{
					Types: []pb.ReferenceLinkType{pb.ReferenceLinkType_REFERENCE_LINK_TYPE_PRODUCT_PAGE},
					Url:   "https://example.com/item",
				},
				{
					Types: []pb.ReferenceLinkType{pb.ReferenceLinkType_REFERENCE_LINK_TYPE_PURCHASE},
					Url:   "https://example.com/item/",
				},
			},
		},
	})
	_, err = buildClient.CreateBuild(ctx, req)
	if err == nil {
		t.Fatal("expected CreateBuild to fail with InvalidArgument when duplicate reference link URL is provided, but it succeeded")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("expected CodeInvalidArgument, got %v: %v", connect.CodeOf(err), err)
	}
	if !strings.Contains(err.Error(), "duplicate reference link URL is not permitted") {
		t.Errorf("expected error message to mention duplicate reference link URL is not permitted, got: %v", err)
	}
}

func TestSandboxAuth_ResendEmailOtpPreservesCodeAndAttempts(t *testing.T) {
	apiUrl := getAPIURL()
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(apiUrl)
	if err != nil {
		t.Fatalf("Sandbox not running at %s: %v", apiUrl, err)
	}

	authClient := quadsmithconnect.NewAuthServiceClient(
		http.DefaultClient,
		apiUrl,
	)
	ctx := context.Background()
	testEmail := "pilot-resend-test@example.com"

	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		dbUrl = "postgres://postgres:postgres@127.0.0.1:5432/quadsmith"
	}
	pool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	// Clean up any existing records for this email
	_, _ = pool.Exec(ctx, `DELETE FROM email_otps WHERE email = $1`, testEmail)
	_, _ = pool.Exec(ctx, `DELETE FROM users WHERE LOWER(email) = LOWER($1) OR LOWER(id) = 'pilot_resend_tester'`, testEmail)

	// 1. Initial send
	res, err := authClient.SendEmailOtp(ctx, connect.NewRequest(&pb.SendEmailOtpRequest{
		Email: testEmail,
	}))
	if err != nil {
		t.Fatalf("SendEmailOtp failed: %v", err)
	}
	if !res.Msg.Success {
		t.Fatalf("expected SendEmailOtp success = true, got %v", res.Msg.Success)
	}

	// Read initial code and attempts from DB
	var code1 string
	var failedAttempts1 int32
	err = pool.QueryRow(ctx, `
		SELECT code, failed_attempts FROM email_otps
		WHERE email = $1 AND used_at IS NULL
		ORDER BY created_at DESC LIMIT 1`, testEmail).Scan(&code1, &failedAttempts1)
	if err != nil {
		t.Fatalf("failed to query otp from DB: %v", err)
	}
	if len(code1) != 6 {
		t.Fatalf("expected 6-digit code stored in DB, got %q", code1)
	}
	if failedAttempts1 != 0 {
		t.Fatalf("expected 0 failed attempts initially, got %d", failedAttempts1)
	}

	// 2. Submit wrong code
	_, err = authClient.VerifyEmailOtp(ctx, connect.NewRequest(&pb.VerifyEmailOtpRequest{
		Email: testEmail,
		Code:  "000000",
	}))
	if err == nil {
		t.Fatal("expected VerifyEmailOtp to fail on incorrect code, got nil")
	}

	// Verify failed_attempts incremented to 1
	var failedAttemptsAfterWrong int32
	err = pool.QueryRow(ctx, `
		SELECT failed_attempts FROM email_otps
		WHERE email = $1 AND used_at IS NULL
		ORDER BY created_at DESC LIMIT 1`, testEmail).Scan(&failedAttemptsAfterWrong)
	if err != nil {
		t.Fatalf("failed to query otp after wrong entry: %v", err)
	}
	if failedAttemptsAfterWrong != 1 {
		t.Fatalf("expected 1 failed attempt, got %d", failedAttemptsAfterWrong)
	}

	// 3. Fast-forward created_at by 25 seconds to bypass cooldown for resend
	_, err = pool.Exec(ctx, `
		UPDATE email_otps
		SET created_at = NOW() - INTERVAL '25 seconds'
		WHERE email = $1 AND used_at IS NULL`, testEmail)
	if err != nil {
		t.Fatalf("failed to adjust created_at for cooldown: %v", err)
	}

	// 4. Resend email OTP
	resResend, err := authClient.SendEmailOtp(ctx, connect.NewRequest(&pb.SendEmailOtpRequest{
		Email: testEmail,
	}))
	if err != nil {
		t.Fatalf("SendEmailOtp resend failed: %v", err)
	}
	if !resResend.Msg.Success {
		t.Fatalf("expected resend success = true, got %v", resResend.Msg.Success)
	}

	// Query DB: code must be identical, and failed_attempts must still be 1!
	var code2 string
	var failedAttempts2 int32
	err = pool.QueryRow(ctx, `
		SELECT code, failed_attempts FROM email_otps
		WHERE email = $1 AND used_at IS NULL
		ORDER BY created_at DESC LIMIT 1`, testEmail).Scan(&code2, &failedAttempts2)
	if err != nil {
		t.Fatalf("failed to query otp after resend: %v", err)
	}
	if code2 != code1 {
		t.Fatalf("expected resent OTP code to be %q, got %q", code1, code2)
	}
	if failedAttempts2 != 1 {
		t.Fatalf("expected failed_attempts to remain 1 after resend, got %d", failedAttempts2)
	}

	// 5. Verify successfully using original code
	verifyRes, err := authClient.VerifyEmailOtp(ctx, connect.NewRequest(&pb.VerifyEmailOtpRequest{
		Email: testEmail,
		Code:  code1,
	}))
	if err != nil {
		t.Fatalf("VerifyEmailOtp with original code failed: %v", err)
	}

	// Because testEmail is a new email, user creation is deferred until Account Setup!
	if !verifyRes.Msg.AccountSetupRequired {
		t.Fatalf("expected AccountSetupRequired = true for new user, got false")
	}
	if verifyRes.Msg.SetupToken == "" {
		t.Fatalf("expected SetupToken to be present for new user, got empty")
	}

	// Verify no user record was prematurely created in users table
	var prematureUserCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE LOWER(email) = LOWER($1)`, testEmail).Scan(&prematureUserCount)
	if err != nil {
		t.Fatalf("failed to query users table: %v", err)
	}
	if prematureUserCount != 0 {
		t.Fatalf("expected 0 user rows in users before Account Setup, got %d", prematureUserCount)
	}

	// Verify challenge is marked used in DB
	var usedAt *time.Time
	err = pool.QueryRow(ctx, `
		SELECT used_at FROM email_otps
		WHERE email = $1
		ORDER BY created_at DESC LIMIT 1`, testEmail).Scan(&usedAt)
	if err != nil {
		t.Fatalf("failed to query used_at: %v", err)
	}
	if usedAt == nil {
		t.Fatal("expected used_at to be non-nil after successful verification")
	}

	// 6. Complete Account Setup
	setupRes, err := authClient.CompleteAccountSetup(ctx, connect.NewRequest(&pb.CompleteAccountSetupRequest{
		SetupToken: verifyRes.Msg.SetupToken,
		Username:   "pilot_resend_tester",
	}))
	if err != nil {
		t.Fatalf("CompleteAccountSetup failed: %v", err)
	}
	if setupRes.Msg.AccessToken == "" {
		t.Fatalf("expected access token after CompleteAccountSetup, got empty")
	}
	if setupRes.Msg.User == nil || setupRes.Msg.User.Id != "pilot_resend_tester" {
		t.Fatalf("expected username pilot_resend_tester, got %v", setupRes.Msg.User)
	}

	// Verify user now exists in users table
	var finalUserCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE LOWER(email) = LOWER($1) AND id = 'pilot_resend_tester'`, testEmail).Scan(&finalUserCount)
	if err != nil {
		t.Fatalf("failed to query users table: %v", err)
	}
	if finalUserCount != 1 {
		t.Fatalf("expected 1 user row in users after CompleteAccountSetup, got %d", finalUserCount)
	}
}
