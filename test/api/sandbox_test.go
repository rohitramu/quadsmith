package api

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/gen/quadsmith/quadsmithconnect"
	"quadsmith/api/pkg/auth"
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

func TestSandboxAuth_ResendEmailOtpGeneratesFreshCodeAndBindsChallenge(t *testing.T) {
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
	challengeId1 := res.Msg.GetChallengeId()
	if challengeId1 == "" {
		t.Fatal("expected SendEmailOtp to return challenge_id, got empty")
	}

	// Read initial code_hash and attempts from DB
	var codeHash1, challengeHash1 string
	var failedAttempts1 int32
	err = pool.QueryRow(ctx, `
		SELECT code_hash, challenge_id_hash, failed_attempts FROM email_otps
		WHERE email = $1 AND used_at IS NULL
		ORDER BY created_at DESC LIMIT 1`, testEmail).Scan(&codeHash1, &challengeHash1, &failedAttempts1)
	if err != nil {
		t.Fatalf("failed to query otp from DB: %v", err)
	}
	if auth.HashToken(challengeId1) != challengeHash1 {
		t.Fatalf("challenge token hash mismatch in DB")
	}
	if failedAttempts1 != 0 {
		t.Fatalf("expected 0 failed attempts initially, got %d", failedAttempts1)
	}

	// 2. Submit wrong code with challenge ID
	_, err = authClient.VerifyEmailOtp(ctx, connect.NewRequest(&pb.VerifyEmailOtpRequest{
		Email:       testEmail,
		Code:        "000000",
		ChallengeId: &challengeId1,
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

	// 4. Resend email OTP (Option A: generates a fresh code and binds to new challenge)
	resResend, err := authClient.SendEmailOtp(ctx, connect.NewRequest(&pb.SendEmailOtpRequest{
		Email: testEmail,
	}))
	if err != nil {
		t.Fatalf("SendEmailOtp resend failed: %v", err)
	}
	if !resResend.Msg.Success {
		t.Fatalf("expected resend success = true, got %v", resResend.Msg.Success)
	}
	challengeId2 := resResend.Msg.GetChallengeId()
	if challengeId2 == "" || challengeId2 == challengeId1 {
		t.Fatalf("expected new distinct challenge_id on resend, got %q", challengeId2)
	}

	// Query DB: old challenge must be invalidated (used_at NOT NULL), new challenge has new code_hash!
	var codeHash2 string
	var failedAttempts2 int32
	err = pool.QueryRow(ctx, `
		SELECT code_hash, failed_attempts FROM email_otps
		WHERE email = $1 AND used_at IS NULL
		ORDER BY created_at DESC LIMIT 1`, testEmail).Scan(&codeHash2, &failedAttempts2)
	if err != nil {
		t.Fatalf("failed to query otp after resend: %v", err)
	}
	if codeHash2 == codeHash1 {
		t.Fatalf("expected fresh code_hash on resend, got identical hash")
	}
	if failedAttempts2 != 0 {
		t.Fatalf("expected fresh failed_attempts count of 0 on new challenge, got %d", failedAttempts2)
	}

	// 5. Attempting to verify using the superseded challengeId1 must fail
	_, err = authClient.VerifyEmailOtp(ctx, connect.NewRequest(&pb.VerifyEmailOtpRequest{
		Email:       testEmail,
		Code:        "123456",
		ChallengeId: &challengeId1,
	}))
	if err == nil {
		t.Fatal("expected verification with superseded challengeId1 to fail, got nil")
	}

	// 6. Find 6-digit code matching codeHash2 for verification
	var validCode string
	for i := 0; i < 1000000; i++ {
		candidate := fmt.Sprintf("%06d", i)
		if auth.VerifyOTP(candidate, codeHash2) {
			validCode = candidate
			break
		}
	}
	if validCode == "" {
		t.Fatal("failed to find valid OTP matching codeHash2")
	}

	// 7. Verify successfully using validCode and challengeId2
	verifyRes, err := authClient.VerifyEmailOtp(ctx, connect.NewRequest(&pb.VerifyEmailOtpRequest{
		Email:       testEmail,
		Code:        validCode,
		ChallengeId: &challengeId2,
	}))
	if err != nil {
		t.Fatalf("VerifyEmailOtp with new challenge failed: %v", err)
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

func TestSandboxAuth_LockoutAfter3FailedCodesPerIP(t *testing.T) {
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
	testIP := "203.0.113.101"
	otherIP := "203.0.113.102"
	testEmail1 := "ip-lockout-test-1@example.com"
	testEmail2 := "ip-lockout-test-2@example.com"

	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		dbUrl = "postgres://postgres:postgres@127.0.0.1:5432/quadsmith"
	}
	pool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	// Clean up any existing records
	_, _ = pool.Exec(ctx, `DELETE FROM email_otps WHERE LOWER(email) IN (LOWER($1), LOWER($2))`, testEmail1, testEmail2)
	_, _ = pool.Exec(ctx, `DELETE FROM users WHERE LOWER(email) IN (LOWER($1), LOWER($2))`, testEmail1, testEmail2)

	failOneCodeChallenge := func(ip string, email string, codeNum int) (bool, int32) {
		reqSend := connect.NewRequest(&pb.SendEmailOtpRequest{
			Email: email,
		})
		reqSend.Header().Set("X-Forwarded-For", ip)
		sendRes, err := authClient.SendEmailOtp(ctx, reqSend)
		if err != nil {
			t.Fatalf("SendEmailOtp for code #%d failed: %v", codeNum, err)
		}
		chId := sendRes.Msg.GetChallengeId()

		// Attempt 1: wrong
		reqV1 := connect.NewRequest(&pb.VerifyEmailOtpRequest{
			Email:       email,
			Code:        "000000",
			ChallengeId: &chId,
		})
		reqV1.Header().Set("X-Forwarded-For", ip)
		if _, err := authClient.VerifyEmailOtp(ctx, reqV1); err == nil {
			t.Fatalf("expected attempt 1 to fail")
		}

		// Attempt 2: wrong
		reqV2 := connect.NewRequest(&pb.VerifyEmailOtpRequest{
			Email:       email,
			Code:        "000000",
			ChallengeId: &chId,
		})
		reqV2.Header().Set("X-Forwarded-For", ip)
		if _, err := authClient.VerifyEmailOtp(ctx, reqV2); err == nil {
			t.Fatalf("expected attempt 2 to fail")
		}

		// Attempt 3: wrong (exhausts this code challenge)
		reqV3 := connect.NewRequest(&pb.VerifyEmailOtpRequest{
			Email:       email,
			Code:        "000000",
			ChallengeId: &chId,
		})
		reqV3.Header().Set("X-Forwarded-For", ip)
		_, err = authClient.VerifyEmailOtp(ctx, reqV3)
		if err == nil {
			t.Fatalf("expected attempt 3 to fail")
		}

		var isLockout bool
		var cooldownSec int32
		if connectErr, ok := err.(*connect.Error); ok {
			if connectErr.Code() == connect.CodeResourceExhausted {
				isLockout = true
			}
			for _, detail := range connectErr.Details() {
				msg, valErr := detail.Value()
				if valErr == nil {
					if otpErr, ok := msg.(*pb.OtpVerificationError); ok {
						cooldownSec = otpErr.CooldownSeconds
					}
				}
			}
		}
		return isLockout, cooldownSec
	}

	// 1. Code 1 fails 3 times -> 15s exhausted cooldown, NOT 60-minute lockout
	isLockout1, cd1 := failOneCodeChallenge(testIP, testEmail1, 1)
	if isLockout1 {
		t.Fatalf("Code 1 exhaustion must NOT trigger 60-minute lockout, got cooldown %d", cd1)
	}
	if cd1 != 15 {
		t.Errorf("expected 15s exhausted cooldown for Code 1, got %d", cd1)
	}

	// Fast-forward 16 seconds to bypass exhausted cooldown
	_, _ = pool.Exec(ctx, `UPDATE email_otps SET created_at = NOW() - INTERVAL '16 seconds', used_at = NOW() - INTERVAL '16 seconds' WHERE email = $1`, testEmail1)

	// 2. Code 2 fails 3 times -> must NOT trigger 60-minute lockout (verifies bug fix!)
	isLockout2, cd2 := failOneCodeChallenge(testIP, testEmail1, 2)
	if isLockout2 {
		t.Fatalf("Lockout happened after only 2 failed codes! Expected lockout only after 3 failed codes.")
	}
	if cd2 != 15 {
		t.Errorf("expected 15s exhausted cooldown for Code 2, got %d", cd2)
	}

	// Fast-forward 16 seconds
	_, _ = pool.Exec(ctx, `UPDATE email_otps SET created_at = NOW() - INTERVAL '16 seconds', used_at = NOW() - INTERVAL '16 seconds' WHERE email = $1`, testEmail1)

	// 3. Code 3 fails 3 times -> triggers 60-minute lockout
	isLockout3, cd3 := failOneCodeChallenge(testIP, testEmail1, 3)
	if !isLockout3 {
		t.Fatalf("expected 3rd failed code to trigger 60-minute lockout, got isLockout=false, cooldown=%d", cd3)
	}
	if cd3 < 3500 || cd3 > 3600 {
		t.Errorf("expected ~3600s lockout cooldown for 3rd failed code, got %d", cd3)
	}

	// 4. Calling SendEmailOtp from testIP (even for a DIFFERENT email) is blocked due to per-IP isolation
	reqBlocked := connect.NewRequest(&pb.SendEmailOtpRequest{
		Email: testEmail2,
	})
	reqBlocked.Header().Set("X-Forwarded-For", testIP)
	_, err = authClient.SendEmailOtp(ctx, reqBlocked)
	if err == nil {
		t.Fatal("expected SendEmailOtp from locked IP to fail, but it succeeded")
	}
	if connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Errorf("expected CodeResourceExhausted for locked IP, got %v: %v", connect.CodeOf(err), err)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "too many failed verification attempts") && !strings.Contains(err.Error(), "minutes") {
		t.Errorf("expected lockout message, got: %v", err)
	}

	// 5. Calling SendEmailOtp from a DIFFERENT IP address must succeed without restriction
	reqAllowed := connect.NewRequest(&pb.SendEmailOtpRequest{
		Email: testEmail2,
	})
	reqAllowed.Header().Set("X-Forwarded-For", otherIP)
	unlockedRes, err := authClient.SendEmailOtp(ctx, reqAllowed)
	if err != nil {
		t.Fatalf("different IP address must NOT be locked out, failed with: %v", err)
	}
	if !unlockedRes.Msg.Success {
		t.Fatalf("expected SendEmailOtp success = true for different IP, got %v", unlockedRes.Msg.Success)
	}
}

func TestSandboxAuth_MergeAccountWithMultipleEmails(t *testing.T) {
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

	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		dbUrl = "postgres://postgres:postgres@127.0.0.1:5432/quadsmith"
	}
	pool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	userAUUID := "018f0000-0000-7000-8000-0000000000a1"
	userAId := "pilot_target_a"
	userAEmail := "pilot_target_a@example.com"

	userBUUID := "018f0000-0000-7000-8000-0000000000b1"
	userBId := "pilot_source_b"
	userBEmail1 := "pilot_source_b1@example.com"
	userBEmail2 := "pilot_source_b2@example.com"

	// Cleanup any previous data
	_, _ = pool.Exec(ctx, `DELETE FROM email_otps WHERE email IN ($1, $2, $3) OR requested_by_user_uuid IN ($4, $5)`,
		userAEmail, userBEmail1, userBEmail2, userAUUID, userBUUID)
	_, _ = pool.Exec(ctx, `DELETE FROM user_emails WHERE email IN ($1, $2, $3)`, userAEmail, userBEmail1, userBEmail2)
	_, _ = pool.Exec(ctx, `DELETE FROM user_profiles WHERE user_uuid IN ($1, $2)`, userAUUID, userBUUID)
	_, _ = pool.Exec(ctx, `DELETE FROM users WHERE uuid IN ($1, $2) OR id IN ($3, $4)`, userAUUID, userBUUID, userAId, userBId)

	// Seed User A
	_, err = pool.Exec(ctx, `
		INSERT INTO users (uuid, id, email, role, created_at, updated_at)
		VALUES ($1, $2, $3, 'user', NOW(), NOW())`,
		userAUUID, userAId, userAEmail)
	if err != nil {
		t.Fatalf("failed to seed user A: %v", err)
	}
	_, _ = pool.Exec(ctx, `
		INSERT INTO user_profiles (uuid, user_uuid, avatar_url, bio, updated_at)
		VALUES ($1, $2, '', 'Target User', NOW())`,
		"018f0000-0000-7000-8000-0000000000a2", userAUUID)
	_, _ = pool.Exec(ctx, `
		INSERT INTO user_emails (uuid, user_uuid, email, is_primary, created_at)
		VALUES ($1, $2, $3, true, NOW())`,
		"018f0000-0000-7000-8000-0000000000a3", userAUUID, userAEmail)

	// Seed User B with 2 emails and an email_otp referencing user_b
	_, err = pool.Exec(ctx, `
		INSERT INTO users (uuid, id, email, role, created_at, updated_at)
		VALUES ($1, $2, $3, 'user', NOW(), NOW())`,
		userBUUID, userBId, userBEmail1)
	if err != nil {
		t.Fatalf("failed to seed user B: %v", err)
	}
	_, _ = pool.Exec(ctx, `
		INSERT INTO user_profiles (uuid, user_uuid, avatar_url, bio, updated_at)
		VALUES ($1, $2, '', 'Source User to merge', NOW())`,
		"018f0000-0000-7000-8000-0000000000b2", userBUUID)
	_, _ = pool.Exec(ctx, `
		INSERT INTO user_emails (uuid, user_uuid, email, is_primary, created_at)
		VALUES ($1, $2, $3, true, NOW())`,
		"018f0000-0000-7000-8000-0000000000b3", userBUUID, userBEmail1)
	_, _ = pool.Exec(ctx, `
		INSERT INTO user_emails (uuid, user_uuid, email, is_primary, created_at)
		VALUES ($1, $2, $3, false, NOW())`,
		"018f0000-0000-7000-8000-0000000000b4", userBUUID, userBEmail2)

	// Add an OTP record requested by user B to verify fk constraint handling
	_, _ = pool.Exec(ctx, `
		INSERT INTO email_otps (uuid, email, code_hash, expires_at, created_at, challenge_id_hash, purpose, requested_by_user_uuid)
		VALUES ($1, $2, 'dummyhash', NOW() + INTERVAL '10 minutes', NOW() - INTERVAL '1 hour', 'cidhash', 'add_email', $3)`,
		"018f0000-0000-7000-8000-0000000000b5", userBEmail1, userBUUID)

	// Generate access token for User A
	tokenA, err := auth.SignAccessToken(userAUUID, userAId, userAEmail, "user", "session-a")
	if err != nil {
		t.Fatalf("failed to sign token for user A: %v", err)
	}

	testIP := "198.51.100.99"

	// 1. User A initiates Add Email for userBEmail2 (the secondary email)
	sendReq := connect.NewRequest(&pb.SendAddEmailOtpRequest{
		Email: userBEmail2,
	})
	sendReq.Header().Set("Authorization", "Bearer "+tokenA)
	sendReq.Header().Set("X-Forwarded-For", testIP)
	sendRes, err := authClient.SendAddEmailOtp(ctx, sendReq)
	if err != nil {
		t.Fatalf("SendAddEmailOtp failed: %v", err)
	}
	challengeId := sendRes.Msg.GetChallengeId()
	if challengeId == "" {
		t.Fatal("expected challenge_id from SendAddEmailOtp")
	}

	// Update OTP code to a known value for verification
	knownCode := "654321"
	_, err = pool.Exec(ctx, `
		UPDATE email_otps
		SET code_hash = $1
		WHERE email = $2 AND purpose = 'add_email' AND requested_by_user_uuid = $3`,
		auth.HashOTP(knownCode), userBEmail2, userAUUID)
	if err != nil {
		t.Fatalf("failed to update otp code: %v", err)
	}

	// 2. User A verifies Add Email OTP
	verifyReq := connect.NewRequest(&pb.VerifyAddEmailOtpRequest{
		Email:       userBEmail2,
		Code:        knownCode,
		ChallengeId: &challengeId,
	})
	verifyReq.Header().Set("Authorization", "Bearer "+tokenA)
	verifyReq.Header().Set("X-Forwarded-For", testIP)
	verifyReq.Header().Set("X-Challenge-Id", challengeId)
	verifyRes, err := authClient.VerifyAddEmailOtp(ctx, verifyReq)
	if err != nil {
		t.Fatalf("VerifyAddEmailOtp failed: %v", err)
	}
	if !verifyRes.Msg.MergeRequired {
		t.Fatalf("expected MergeRequired=true, got false")
	}
	if verifyRes.Msg.OtherUserUsername != userBId {
		t.Errorf("expected other username %s, got %s", userBId, verifyRes.Msg.OtherUserUsername)
	}
	// Verify that OtherEmails contains BOTH emails of User B
	if len(verifyRes.Msg.OtherEmails) != 2 {
		t.Errorf("expected 2 other emails, got %v", verifyRes.Msg.OtherEmails)
	}
	ticket := verifyRes.Msg.GetMergeTicket()
	if ticket == "" {
		t.Fatal("expected non-empty merge ticket")
	}

	// 3. User A calls MergeAccount
	mergeReq := connect.NewRequest(&pb.MergeAccountRequest{
		Email:       userBEmail2,
		MergeTicket: &ticket,
	})
	mergeReq.Header().Set("Authorization", "Bearer "+tokenA)
	mergeReq.Header().Set("X-Forwarded-For", testIP)
	mergeRes, err := authClient.MergeAccount(ctx, mergeReq)
	if err != nil {
		t.Fatalf("MergeAccount failed: %v", err)
	}
	if !mergeRes.Msg.Success {
		t.Errorf("expected MergeAccount success=true, got false")
	}
	if len(mergeRes.Msg.MergedEmails) != 2 {
		t.Errorf("expected 2 merged emails in response, got %v", mergeRes.Msg.MergedEmails)
	}

	// 4. Verify in DB that User B is deleted
	var bCount int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE uuid = $1`, userBUUID).Scan(&bCount)
	if bCount != 0 {
		t.Errorf("expected User B to be deleted, found count %d", bCount)
	}

	// 5. Verify User A now has all 3 emails in user_emails
	var userEmails []string
	rows, err := pool.Query(ctx, `SELECT email FROM user_emails WHERE user_uuid = $1 ORDER BY email ASC`, userAUUID)
	if err != nil {
		t.Fatalf("failed to query user_emails: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var em string
		if err := rows.Scan(&em); err == nil {
			userEmails = append(userEmails, em)
		}
	}
	if len(userEmails) != 3 {
		t.Fatalf("expected User A to have 3 emails, got %d: %v", len(userEmails), userEmails)
	}

	// 6. User A calls ListUserEmails via RPC
	listReq := connect.NewRequest(&pb.ListUserEmailsRequest{})
	listReq.Header().Set("Authorization", "Bearer "+tokenA)
	listRes, err := authClient.ListUserEmails(ctx, listReq)
	if err != nil {
		t.Fatalf("ListUserEmails failed: %v", err)
	}
	if len(listRes.Msg.Emails) != 3 {
		t.Errorf("expected 3 emails from ListUserEmails RPC, got %d", len(listRes.Msg.Emails))
	}
}

func TestSandboxCreateCollection(t *testing.T) {
	apiUrl := getAPIURL()
	socialClient := quadsmithconnect.NewSocialServiceClient(
		http.DefaultClient,
		apiUrl,
	)

	ctx := context.Background()
	req := connect.NewRequest(&pb.CreateCollectionRequest{
		Name:        "Speed Demons",
		Description: "Collection of fast 5-inch quads",
		IsPublic:    true,
	})

	res, err := socialClient.CreateCollection(ctx, req)
	if err != nil {
		t.Fatalf("CreateCollection failed: %v", err)
	}

	if res.Msg == nil {
		t.Fatal("expected non-nil Collection response")
	}

	if res.Msg.Name != "Speed Demons" {
		t.Errorf("expected collection name 'Speed Demons', got '%s'", res.Msg.Name)
	}

	// Verify collection can be listed
	listReq := connect.NewRequest(&pb.ListUserCollectionsRequest{
		UserId: "test_pilot",
	})
	listRes, err := socialClient.ListUserCollections(ctx, listReq)
	if err != nil {
		t.Fatalf("ListUserCollections failed: %v", err)
	}

	found := false
	for _, col := range listRes.Msg.Collections {
		if col.Id == res.Msg.Id {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("created collection id %s not found in user collections", res.Msg.Id)
	}
}
