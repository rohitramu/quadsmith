package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"

	"quadsmith/shared/pkg/auth"

	pb "quadsmith/shared/gen/quadsmith/api"
	"quadsmith/shared/gen/quadsmith/api/apiconnect"
)

func getAPIURL() string {
	if url := os.Getenv("QS_CLI_API_URL"); url != "" {
		return url
	}
	if url := os.Getenv("QS_API_URL"); url != "" {
		return url
	}
	return "http://127.0.0.1:8081"
}

func getDBURL() string {
	if url := os.Getenv("QS_DATABASE_URL"); url != "" {
		return url
	}
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return "postgres://postgres:postgres@127.0.0.1:5432/quadsmith_api"
}

func TestSandboxBuilds(t *testing.T) {
	apiUrl := getAPIURL()
	// Fail if we can't reach the sandbox
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(apiUrl)
	if err != nil {
		t.Fatalf("Sandbox not running at %s: %v", apiUrl, err)
	}

	buildClient := apiconnect.NewBuildServiceClient(
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

	frameClient := apiconnect.NewFrameServiceClient(
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

	evalClient := apiconnect.NewEvaluatorServiceClient(
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

	buildClient := apiconnect.NewBuildServiceClient(
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

	buildClient := apiconnect.NewBuildServiceClient(
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

	buildClient := apiconnect.NewBuildServiceClient(
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

	authClient := apiconnect.NewAuthServiceClient(
		http.DefaultClient,
		apiUrl,
	)
	ctx := context.Background()
	testEmail := "pilot-resend-test@example.com"

	dbUrl := getDBURL()
	pool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	// Clean up any existing records for this email
	_, _ = pool.Exec(ctx, `DELETE FROM email_otps WHERE email = $1`, testEmail)
	_, _ = pool.Exec(ctx, `DELETE FROM users WHERE uuid IN (SELECT user_uuid FROM user_emails WHERE LOWER(email) = LOWER($1)) OR LOWER(id) = 'pilot_resend_tester'`, testEmail)

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

	// Verify no user record was prematurely created in user_emails table
	var prematureUserCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM user_emails WHERE LOWER(email) = LOWER($1)`, testEmail).Scan(&prematureUserCount)
	if err != nil {
		t.Fatalf("failed to query user_emails table: %v", err)
	}
	if prematureUserCount != 0 {
		t.Fatalf("expected 0 user rows in user_emails before Account Setup, got %d", prematureUserCount)
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
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM users u JOIN user_emails ue ON ue.user_uuid = u.uuid WHERE LOWER(ue.email) = LOWER($1) AND u.id = 'pilot_resend_tester'`, testEmail).Scan(&finalUserCount)
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

	authClient := apiconnect.NewAuthServiceClient(
		http.DefaultClient,
		apiUrl,
	)
	ctx := context.Background()
	testIP := "203.0.113.101"
	otherIP := "203.0.113.102"
	testEmail1 := "ip-lockout-test-1@example.com"
	testEmail2 := "ip-lockout-test-2@example.com"

	dbUrl := getDBURL()
	pool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	// Clean up any existing records
	_, _ = pool.Exec(ctx, `DELETE FROM email_otps WHERE LOWER(email) IN (LOWER($1), LOWER($2))`, testEmail1, testEmail2)
	_, _ = pool.Exec(ctx, `DELETE FROM users WHERE uuid IN (SELECT user_uuid FROM user_emails WHERE LOWER(email) IN (LOWER($1), LOWER($2)))`, testEmail1, testEmail2)

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

	authClient := apiconnect.NewAuthServiceClient(
		http.DefaultClient,
		apiUrl,
	)
	ctx := context.Background()

	dbUrl := getDBURL()
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
		INSERT INTO users (uuid, id, role, created_at, updated_at)
		VALUES ($1, $2, 'user', NOW(), NOW())`,
		userAUUID, userAId)
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
		INSERT INTO users (uuid, id, role, created_at, updated_at)
		VALUES ($1, $2, 'user', NOW(), NOW())`,
		userBUUID, userBId)
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
	sendReq := connect.NewRequest(&pb.SendEmailOtpRequest{
		Email:   userBEmail2,
		Purpose: pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_ADD_EMAIL,
	})
	sendReq.Header().Set("Authorization", "Bearer "+tokenA)
	sendReq.Header().Set("X-Forwarded-For", testIP)
	sendRes, err := authClient.SendEmailOtp(ctx, sendReq)
	if err != nil {
		t.Fatalf("SendEmailOtp failed: %v", err)
	}
	challengeId := sendRes.Msg.GetChallengeId()
	if challengeId == "" {
		t.Fatal("expected challenge_id from SendEmailOtp")
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
	verifyReq := connect.NewRequest(&pb.VerifyEmailOtpRequest{
		Email:       userBEmail2,
		Code:        knownCode,
		ChallengeId: &challengeId,
		Purpose:     pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_ADD_EMAIL,
	})
	verifyReq.Header().Set("Authorization", "Bearer "+tokenA)
	verifyReq.Header().Set("X-Forwarded-For", testIP)
	verifyReq.Header().Set("X-Challenge-Id", challengeId)
	verifyRes, err := authClient.VerifyEmailOtp(ctx, verifyReq)
	if err != nil {
		t.Fatalf("VerifyEmailOtp failed: %v", err)
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

func TestSandboxAuth_CrossDeviceMagicLinkApproval(t *testing.T) {
	apiUrl := getAPIURL()
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(apiUrl)
	if err != nil {
		t.Fatalf("Sandbox not running at %s: %v", apiUrl, err)
	}

	authClient := apiconnect.NewAuthServiceClient(
		http.DefaultClient,
		apiUrl,
	)
	ctx := context.Background()

	dbUrl := getDBURL()
	pool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	testEmail := "crossdev_pilot@fpv.build"
	_, _ = pool.Exec(ctx, `DELETE FROM email_otps WHERE email = $1`, testEmail)
	_, _ = pool.Exec(ctx, `DELETE FROM users WHERE uuid IN (SELECT user_uuid FROM user_emails WHERE LOWER(email) = LOWER($1))`, testEmail)

	// 1. Device A sends OTP
	deviceAIP := "192.168.1.105"
	deviceAUA := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	sendReq := connect.NewRequest(&pb.SendEmailOtpRequest{
		Email: testEmail,
	})
	sendReq.Header().Set("X-Forwarded-For", deviceAIP)
	sendReq.Header().Set("User-Agent", deviceAUA)
	sendRes, err := authClient.SendEmailOtp(ctx, sendReq)
	if err != nil {
		t.Fatalf("SendEmailOtp failed: %v", err)
	}
	challengeId := sendRes.Msg.GetChallengeId()
	if challengeId == "" {
		t.Fatal("expected challenge_id from SendEmailOtp")
	}

	// 2. Query DB to get otp UUID and verify metadata
	var otpUUID, status, locationHint, ipAddress string
	err = pool.QueryRow(ctx, `
		SELECT uuid, status, location_hint, ip_address
		FROM email_otps
		WHERE email = $1 AND purpose = 'login'
		ORDER BY created_at DESC LIMIT 1`, testEmail).Scan(&otpUUID, &status, &locationHint, &ipAddress)
	if err != nil {
		t.Fatalf("failed to query email_otps: %v", err)
	}
	if status != "pending" {
		t.Errorf("expected pending status, got %s", status)
	}
	if ipAddress != deviceAIP {
		t.Errorf("expected ip %s, got %s", deviceAIP, ipAddress)
	}
	if locationHint != "Local Network" {
		t.Errorf("expected 'Local Network' location hint for private IP, got %s", locationHint)
	}

	// 3. Device A polls status initially -> PENDING
	statusReq := connect.NewRequest(&pb.GetLoginChallengeStatusRequest{
		ChallengeId: challengeId,
	})
	statusRes, err := authClient.GetLoginChallengeStatus(ctx, statusReq)
	if err != nil {
		t.Fatalf("GetLoginChallengeStatus failed: %v", err)
	}
	if statusRes.Msg.GetStatus() != pb.LoginChallengeStatus_LOGIN_CHALLENGE_STATUS_PENDING {
		t.Errorf("expected PENDING status, got %v", statusRes.Msg.GetStatus())
	}

	// 4. Device B opens magic link token and fetches challenge details
	magicToken, err := auth.SignMagicLinkToken(otpUUID, testEmail, "login", "", time.Now().Add(auth.MagicLinkTTL))
	if err != nil {
		t.Fatalf("failed to sign magic link token: %v", err)
	}

	detailsRes, err := authClient.GetLoginChallengeDetails(ctx, connect.NewRequest(&pb.GetLoginChallengeDetailsRequest{
		Token: magicToken,
	}))
	if err != nil {
		t.Fatalf("GetLoginChallengeDetails failed: %v", err)
	}
	if detailsRes.Msg.GetEmail() != testEmail {
		t.Errorf("expected email %s, got %s", testEmail, detailsRes.Msg.GetEmail())
	}
	if detailsRes.Msg.GetIpAddress() != deviceAIP {
		t.Errorf("expected ip %s, got %s", deviceAIP, detailsRes.Msg.GetIpAddress())
	}
	if detailsRes.Msg.GetLocationHint() != "Local Network" {
		t.Errorf("expected 'Local Network', got %s", detailsRes.Msg.GetLocationHint())
	}
	if detailsRes.Msg.GetDeviceHint() != "Chrome on macOS" {
		t.Errorf("expected 'Chrome on macOS', got %s", detailsRes.Msg.GetDeviceHint())
	}
	if detailsRes.Msg.GetStatus() != pb.LoginChallengeStatus_LOGIN_CHALLENGE_STATUS_PENDING {
		t.Errorf("expected status PENDING, got %v", detailsRes.Msg.GetStatus())
	}

	// 5. Device B approves challenge
	approveRes, err := authClient.ApproveLoginChallenge(ctx, connect.NewRequest(&pb.ApproveLoginChallengeRequest{
		Token: magicToken,
	}))
	if err != nil {
		t.Fatalf("ApproveLoginChallenge failed: %v", err)
	}
	if !approveRes.Msg.GetSuccess() {
		t.Errorf("expected ApproveLoginChallenge success=true, got false")
	}

	// 6. Device B checks details again -> APPROVED
	detailsRes2, err := authClient.GetLoginChallengeDetails(ctx, connect.NewRequest(&pb.GetLoginChallengeDetailsRequest{
		Token: magicToken,
	}))
	if err != nil {
		t.Fatalf("GetLoginChallengeDetails after approval failed: %v", err)
	}
	if detailsRes2.Msg.GetStatus() != pb.LoginChallengeStatus_LOGIN_CHALLENGE_STATUS_APPROVED {
		t.Errorf("expected status APPROVED, got %v", detailsRes2.Msg.GetStatus())
	}

	// 7. Device A polls status -> APPROVED
	statusRes2, err := authClient.GetLoginChallengeStatus(ctx, statusReq)
	if err != nil {
		t.Fatalf("GetLoginChallengeStatus after approval failed: %v", err)
	}
	if statusRes2.Msg.GetStatus() != pb.LoginChallengeStatus_LOGIN_CHALLENGE_STATUS_APPROVED {
		t.Errorf("expected status APPROVED for Device A, got %v", statusRes2.Msg.GetStatus())
	}
	if statusRes2.Msg.GetAuthResponse() == nil {
		t.Fatal("expected AuthResponse in statusRes2, got nil")
	}
	if !statusRes2.Msg.GetAuthResponse().GetAccountSetupRequired() {
		t.Errorf("expected AccountSetupRequired=true for new user, got false")
	}
	if statusRes2.Msg.GetAuthResponse().GetSetupToken() == "" {
		t.Errorf("expected SetupToken to be present, got empty")
	}

	// 8. Device A polling again after consumption -> EXPIRED
	statusRes3, err := authClient.GetLoginChallengeStatus(ctx, statusReq)
	if err != nil {
		t.Fatalf("GetLoginChallengeStatus after consumption failed: %v", err)
	}
	if statusRes3.Msg.GetStatus() != pb.LoginChallengeStatus_LOGIN_CHALLENGE_STATUS_EXPIRED {
		t.Errorf("expected status EXPIRED after consumed, got %v", statusRes3.Msg.GetStatus())
	}

	// 9. Now test Deny flow with a fresh request
	testEmailDeny := "crossdev_deny@fpv.build"
	sendDenyRes, err := authClient.SendEmailOtp(ctx, connect.NewRequest(&pb.SendEmailOtpRequest{
		Email: testEmailDeny,
	}))
	if err != nil {
		t.Fatalf("SendEmailOtp for deny test failed: %v", err)
	}
	challengeIdDeny := sendDenyRes.Msg.GetChallengeId()

	var otpUUIDDeny string
	err = pool.QueryRow(ctx, `
		SELECT uuid FROM email_otps
		WHERE email = $1 AND purpose = 'login'
		ORDER BY created_at DESC LIMIT 1`, testEmailDeny).Scan(&otpUUIDDeny)
	if err != nil {
		t.Fatalf("failed to query email_otps for deny: %v", err)
	}
	magicTokenDeny, err := auth.SignMagicLinkToken(otpUUIDDeny, testEmailDeny, "login", "", time.Now().Add(auth.MagicLinkTTL))
	if err != nil {
		t.Fatalf("failed to sign deny magic link token: %v", err)
	}

	denyRes, err := authClient.DenyLoginChallenge(ctx, connect.NewRequest(&pb.DenyLoginChallengeRequest{
		Token: magicTokenDeny,
	}))
	if err != nil {
		t.Fatalf("DenyLoginChallenge failed: %v", err)
	}
	if !denyRes.Msg.GetSuccess() {
		t.Errorf("expected DenyLoginChallenge success=true, got false")
	}

	// Device A polls status -> DENIED
	statusResDeny, err := authClient.GetLoginChallengeStatus(ctx, connect.NewRequest(&pb.GetLoginChallengeStatusRequest{
		ChallengeId: challengeIdDeny,
	}))
	if err != nil {
		t.Fatalf("GetLoginChallengeStatus for denied challenge failed: %v", err)
	}
	if statusResDeny.Msg.GetStatus() != pb.LoginChallengeStatus_LOGIN_CHALLENGE_STATUS_DENIED {
		t.Errorf("expected status DENIED, got %v", statusResDeny.Msg.GetStatus())
	}

	// 10. Verify server enforcement: Device A tries to request a new code immediately after denial while cooldown is active -> REJECTED by server
	_, prematureErr := authClient.SendEmailOtp(ctx, connect.NewRequest(&pb.SendEmailOtpRequest{
		Email: testEmailDeny,
	}))
	if prematureErr == nil {
		t.Fatal("expected server to reject SendEmailOtp during active cooldown after denial, got nil")
	}
	if connect.CodeOf(prematureErr) != connect.CodeResourceExhausted {
		t.Errorf("expected server to return CodeResourceExhausted, got %v", connect.CodeOf(prematureErr))
	}
	if !strings.Contains(prematureErr.Error(), "seconds before requesting another code") {
		t.Errorf("expected cooldown message from server, got %v", prematureErr)
	}

	// Verify server allows request once cooldown expires (simulated by fast-forwarding created_at by 25s)
	_, err = pool.Exec(ctx, `
		UPDATE email_otps
		SET created_at = NOW() - INTERVAL '25 seconds'
		WHERE email = $1`, testEmailDeny)
	if err != nil {
		t.Fatalf("failed to adjust created_at for cooldown: %v", err)
	}

	resendAfterCooldownRes, err := authClient.SendEmailOtp(ctx, connect.NewRequest(&pb.SendEmailOtpRequest{
		Email: testEmailDeny,
	}))
	if err != nil {
		t.Fatalf("expected SendEmailOtp to succeed after cooldown expired, got %v", err)
	}
	if !resendAfterCooldownRes.Msg.GetSuccess() {
		t.Errorf("expected SendEmailOtp success=true after cooldown expiration")
	}
}

func TestSandboxCreateCollection(t *testing.T) {
	apiUrl := getAPIURL()
	socialClient := apiconnect.NewSocialServiceClient(
		http.DefaultClient,
		apiUrl,
	)

	ctx := context.Background()
	req := connect.NewRequest(&pb.CreateCollectionRequest{
		Name:        "Speed Demons",
		Description: "Collection of fast 5-inch quads",
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

func TestSandboxUpdateCollection(t *testing.T) {
	apiUrl := getAPIURL()
	socialClient := apiconnect.NewSocialServiceClient(
		http.DefaultClient,
		apiUrl,
	)

	ctx := context.Background()
	// 1. Create a collection
	createReq := connect.NewRequest(&pb.CreateCollectionRequest{
		Name:        "Secret Quads",
		Description: "Initial collection",
	})
	createRes, err := socialClient.CreateCollection(ctx, createReq)
	if err != nil {
		t.Fatalf("CreateCollection failed: %v", err)
	}
	colId := createRes.Msg.Id
	if createRes.Msg.Name != "Secret Quads" {
		t.Fatalf("expected collection name to be 'Secret Quads', got '%s'", createRes.Msg.Name)
	}

	// 2. Update collection name and description
	updateReq := connect.NewRequest(&pb.UpdateCollectionRequest{
		CollectionId: colId,
		Name:         "Renamed Quads",
		Description:  "Updated description",
	})
	updateRes, err := socialClient.UpdateCollection(ctx, updateReq)
	if err != nil {
		t.Fatalf("UpdateCollection failed: %v", err)
	}
	if updateRes.Msg.Name != "Renamed Quads" {
		t.Errorf("expected collection name to be 'Renamed Quads', got '%s'", updateRes.Msg.Name)
	}
	if updateRes.Msg.Description != "Updated description" {
		t.Errorf("expected collection description to be 'Updated description', got '%s'", updateRes.Msg.Description)
	}

	// 3. Confirm GetCollection returns updated values
	getReq := connect.NewRequest(&pb.GetCollectionRequest{
		Id: colId,
	})
	getRes, err := socialClient.GetCollection(ctx, getReq)
	if err != nil {
		t.Fatalf("GetCollection failed: %v", err)
	}
	if getRes.Msg.Name != "Renamed Quads" {
		t.Errorf("expected GetCollection name to be 'Renamed Quads', got '%s'", getRes.Msg.Name)
	}
	if getRes.Msg.Description != "Updated description" {
		t.Errorf("expected GetCollection description to be 'Updated description', got '%s'", getRes.Msg.Description)
	}
}

func TestSandboxAuth_AccountDeletionGracePeriodAndResurrection(t *testing.T) {
	apiUrl := getAPIURL()
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(apiUrl)
	if err != nil {
		t.Fatalf("Sandbox not running at %s: %v", apiUrl, err)
	}

	authClient := apiconnect.NewAuthServiceClient(
		http.DefaultClient,
		apiUrl,
	)
	userClient := apiconnect.NewUserServiceClient(
		http.DefaultClient,
		apiUrl,
	)
	ctx := context.Background()

	dbUrl := getDBURL()
	pool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	userUUID := "018f0000-0000-7000-8000-0000000000d1"
	username := "delete_pilot"
	userEmail := "delete_pilot@example.com"

	// Cleanup any previous test data
	_, _ = pool.Exec(ctx, `DELETE FROM email_otps WHERE email = $1 OR requested_by_user_uuid = $2`, userEmail, userUUID)
	_, _ = pool.Exec(ctx, `DELETE FROM user_emails WHERE email = $1`, userEmail)
	_, _ = pool.Exec(ctx, `DELETE FROM user_profiles WHERE user_uuid = $1`, userUUID)
	_, _ = pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_uuid = $1`, userUUID)
	_, _ = pool.Exec(ctx, `DELETE FROM users WHERE uuid = $1 OR id = $2`, userUUID, username)

	// Seed user
	_, err = pool.Exec(ctx, `
		INSERT INTO users (uuid, id, role, created_at, updated_at)
		VALUES ($1, $2, 'USER', NOW(), NOW())`,
		userUUID, username)
	if err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO user_emails (uuid, user_uuid, email, is_primary, verified_via_otp, created_at)
		VALUES ($1, $2, $3, true, true, NOW())`,
		"018f0000-0000-7000-8000-0000000000e1", userUUID, userEmail)
	if err != nil {
		t.Fatalf("failed to seed user email: %v", err)
	}

	token, err := auth.SignAccessToken(userUUID, username, userEmail, "USER", "test-session-d1")
	if err != nil {
		t.Fatalf("failed to sign access token: %v", err)
	}

	unauthClient := apiconnect.NewAuthServiceClient(
		&http.Client{Transport: http.DefaultTransport},
		apiUrl,
	)

	// 1. SendEmailOtp for delete account without auth fails
	badReq := connect.NewRequest(&pb.SendEmailOtpRequest{
		Purpose: pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_DELETE_ACCOUNT,
	})
	_, err = unauthClient.SendEmailOtp(ctx, badReq)
	if err == nil {
		t.Fatal("expected SendEmailOtp for delete account without auth to fail, but it succeeded")
	}

	// 2. SendEmailOtp - correct auth succeeds
	sendReq := connect.NewRequest(&pb.SendEmailOtpRequest{
		Purpose: pb.EmailOtpPurpose_EMAIL_OTP_PURPOSE_DELETE_ACCOUNT,
	})
	sendReq.Header().Set("Authorization", "Bearer "+token)
	sendRes, err := authClient.SendEmailOtp(ctx, sendReq)
	if err != nil {
		t.Fatalf("SendEmailOtp failed: %v", err)
	}
	if !sendRes.Msg.Success || sendRes.Msg.GetChallengeId() == "" {
		t.Fatalf("unexpected SendEmailOtp response: %+v", sendRes.Msg)
	}

	// Query OTP challenge from DB
	var otpUUID string
	var expiresAt time.Time
	err = pool.QueryRow(ctx, `
		SELECT uuid, expires_at FROM email_otps
		WHERE requested_by_user_uuid = $1 AND purpose = 'delete_account' AND used_at IS NULL
		ORDER BY created_at DESC LIMIT 1`, userUUID).Scan(&otpUUID, &expiresAt)
	if err != nil {
		t.Fatalf("failed to query delete_account email_otps: %v", err)
	}

	// 3. DeleteAccount using magic token
	magicToken, err := auth.SignMagicLinkToken(otpUUID, userEmail, "delete_account", userUUID, expiresAt)
	if err != nil {
		t.Fatalf("failed to generate magic link token: %v", err)
	}

	delReq := connect.NewRequest(&pb.DeleteAccountRequest{
		Username:   username,
		MagicToken: &magicToken,
	})
	delReq.Header().Set("Authorization", "Bearer "+token)
	delRes, err := authClient.DeleteAccount(ctx, delReq)
	if err != nil {
		t.Fatalf("DeleteAccount failed: %v", err)
	}
	if !delRes.Msg.Success || delRes.Msg.ScheduledPurgeAt == nil {
		t.Fatalf("unexpected DeleteAccount response: %+v", delRes.Msg)
	}

	// Check DB: user is soft-deleted
	var dbDeletedAt *time.Time
	err = pool.QueryRow(ctx, `SELECT deleted_at FROM users WHERE uuid = $1`, userUUID).Scan(&dbDeletedAt)
	if err != nil || dbDeletedAt == nil {
		t.Fatalf("expected users.deleted_at to be set, err: %v, deleted_at: %v", err, dbDeletedAt)
	}

	// 4. In-Place Handle Locking: CheckUsernameAvailable reports handle UNAVAILABLE
	availRes, err := userClient.CheckUsernameAvailable(ctx, connect.NewRequest(&pb.CheckUsernameAvailableRequest{
		Username: username,
	}))
	if err != nil {
		t.Fatalf("CheckUsernameAvailable failed: %v", err)
	}
	if availRes.Msg.Available || availRes.Msg.Status != pb.UsernameAvailabilityStatus_USERNAME_AVAILABILITY_STATUS_UNAVAILABLE {
		t.Errorf("expected soft-deleted username to remain locked (UNAVAILABLE), got available: %v, status: %v",
			availRes.Msg.Available, availRes.Msg.Status)
	}

	// 5. Login attempt during 30-day grace period returns PENDING_DELETION
	// Age previous email OTP to clear cooldown
	_, _ = pool.Exec(ctx, `UPDATE email_otps SET created_at = NOW() - INTERVAL '30 seconds' WHERE email = $1`, userEmail)
	loginOtpRes, err := authClient.SendEmailOtp(ctx, connect.NewRequest(&pb.SendEmailOtpRequest{
		Email: userEmail,
	}))
	if err != nil {
		t.Fatalf("SendEmailOtp during grace period failed: %v", err)
	}
	if !loginOtpRes.Msg.IsExistingAccount {
		t.Errorf("expected is_existing_account = true for user in grace period")
	}

	var loginOtpUUID string
	var loginExpiresAt time.Time
	err = pool.QueryRow(ctx, `
		SELECT uuid, expires_at FROM email_otps
		WHERE email = $1 AND purpose = 'login' AND used_at IS NULL
		ORDER BY created_at DESC LIMIT 1`, userEmail).Scan(&loginOtpUUID, &loginExpiresAt)
	if err != nil {
		t.Fatalf("failed to query login OTP from DB: %v", err)
	}

	loginMagicToken, err := auth.SignMagicLinkToken(loginOtpUUID, userEmail, "login", "", loginExpiresAt)
	if err != nil {
		t.Fatalf("failed to sign login magic token: %v", err)
	}

	verifyRes, err := authClient.VerifyMagicLink(ctx, connect.NewRequest(&pb.VerifyMagicLinkRequest{
		Token: loginMagicToken,
	}))
	if err != nil {
		t.Fatalf("VerifyMagicLink failed during grace period: %v", err)
	}
	if verifyRes.Msg.GetAccountStatus() != "PENDING_DELETION" {
		t.Errorf("expected account_status 'PENDING_DELETION', got '%s'", verifyRes.Msg.GetAccountStatus())
	}
	if verifyRes.Msg.ScheduledPurgeAt == nil {
		t.Errorf("expected scheduled_purge_at to be populated in AuthResponse")
	}

	// 6. CancelAccountDeletion restores the account
	resurrectReq := connect.NewRequest(&pb.CancelAccountDeletionRequest{})
	resurrectReq.Header().Set("Authorization", "Bearer "+verifyRes.Msg.AccessToken)
	resurrectRes, err := authClient.CancelAccountDeletion(ctx, resurrectReq)
	if err != nil {
		t.Fatalf("CancelAccountDeletion failed: %v", err)
	}
	if !resurrectRes.Msg.Success {
		t.Errorf("expected CancelAccountDeletion success = true")
	}

	// Check DB: deleted_at is cleared!
	err = pool.QueryRow(ctx, `SELECT deleted_at FROM users WHERE uuid = $1`, userUUID).Scan(&dbDeletedAt)
	if err != nil {
		t.Fatalf("failed to query user after resurrection: %v", err)
	}
	if dbDeletedAt != nil {
		t.Errorf("expected deleted_at to be NULL after resurrection, got %v", dbDeletedAt)
	}

	// 7. Scrubber test: simulate account past 30 days and verify purge
	_, _ = pool.Exec(ctx, `UPDATE users SET deleted_at = NOW() - INTERVAL '31 days' WHERE uuid = $1`, userUUID)
	purgedCount, err := auth.HandlePurgeSoftDeletedAccounts(ctx, pool)
	if err != nil {
		t.Fatalf("HandlePurgeSoftDeletedAccounts failed: %v", err)
	}
	if purgedCount != 1 {
		t.Errorf("expected 1 user purged, got %d", purgedCount)
	}

	// Verify handle released and anonymized
	var currentId string
	err = pool.QueryRow(ctx, `SELECT id FROM users WHERE uuid = $1`, userUUID).Scan(&currentId)
	if err != nil {
		t.Fatalf("failed to query purged user: %v", err)
	}
	if !strings.HasPrefix(currentId, "deleted-") {
		t.Errorf("expected purged user id to start with 'deleted-', got '%s'", currentId)
	}

	// Verify original username is now available to new users
	availAfterPurge, err := userClient.CheckUsernameAvailable(ctx, connect.NewRequest(&pb.CheckUsernameAvailableRequest{
		Username: username,
	}))
	if err != nil {
		t.Fatalf("CheckUsernameAvailable after purge failed: %v", err)
	}
	if !availAfterPurge.Msg.Available {
		t.Errorf("expected username to be available after purge, got: %+v", availAfterPurge.Msg)
	}
}

func TestOAuthIntegration(t *testing.T) {
	apiUrl := getAPIURL()
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(apiUrl)
	if err != nil {
		t.Fatalf("Sandbox not running at %s: %v", apiUrl, err)
	}

	ctx := context.Background()
	authClient := apiconnect.NewAuthServiceClient(http.DefaultClient, apiUrl)

	dbUrl := getDBURL()
	pool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	// Clean up any leftover test emails/users
	_, _ = pool.Exec(ctx, `DELETE FROM user_emails WHERE email IN ('pilot.github@example.com', 'pilot.secondary@example.com', 'pilot.discord@example.com', 'other.oauth@example.com')`)
	_, _ = pool.Exec(ctx, `DELETE FROM user_profiles WHERE user_uuid IN (SELECT uuid FROM users WHERE id IN ('oauth_gh_pilot', 'oauth_other_pilot'))`)
	_, _ = pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_uuid IN (SELECT uuid FROM users WHERE id IN ('oauth_gh_pilot', 'oauth_other_pilot'))`)
	_, _ = pool.Exec(ctx, `DELETE FROM oauth_accounts WHERE user_uuid IN (SELECT uuid FROM users WHERE id IN ('oauth_gh_pilot', 'oauth_other_pilot'))`)
	_, _ = pool.Exec(ctx, `DELETE FROM username_histories WHERE user_uuid IN (SELECT uuid FROM users WHERE id IN ('oauth_gh_pilot', 'oauth_other_pilot'))`)
	_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id IN ('oauth_gh_pilot', 'oauth_other_pilot')`)

	// 1. Unauthenticated StartOAuth (GitHub Login) through NewAuthInterceptor
	startRes, err := authClient.StartOAuth(ctx, connect.NewRequest(&pb.StartOAuthRequest{
		Provider:     pb.OAuthProvider_OAUTH_PROVIDER_GITHUB,
		Purpose:      pb.OAuthPurpose_OAUTH_PURPOSE_LOGIN,
		RedirectPath: "/login",
	}))
	if err != nil {
		t.Fatalf("StartOAuth failed: %v", err)
	}
	if startRes.Msg.State == "" || !strings.Contains(startRes.Msg.AuthorizationUrl, "code=mock_github_default") {
		t.Fatalf("unexpected StartOAuth response: %+v", startRes.Msg)
	}

	// 2. Unauthenticated CompleteOAuth (Case A: Brand-new user -> AccountSetupRequired)
	completeRes, err := authClient.CompleteOAuth(ctx, connect.NewRequest(&pb.CompleteOAuthRequest{
		Code:  "mock_github_default",
		State: startRes.Msg.State,
	}))
	if err != nil {
		t.Fatalf("CompleteOAuth failed: %v", err)
	}
	if !completeRes.Msg.Success || completeRes.Msg.AuthResponse == nil || !completeRes.Msg.AuthResponse.AccountSetupRequired {
		t.Fatalf("expected AccountSetupRequired = true, got: %+v", completeRes.Msg)
	}

	// 3. CompleteAccountSetup creates user + links GitHub + both verified emails
	setupRes, err := authClient.CompleteAccountSetup(ctx, connect.NewRequest(&pb.CompleteAccountSetupRequest{
		SetupToken: completeRes.Msg.AuthResponse.SetupToken,
		Username:   "oauth_gh_pilot",
	}))
	if err != nil {
		t.Fatalf("CompleteAccountSetup failed: %v", err)
	}
	accessToken := setupRes.Msg.AccessToken
	if accessToken == "" {
		t.Fatalf("expected accessToken from CompleteAccountSetup")
	}

	// 4. ListUserEmails verifies both primary and secondary GitHub emails and OAuth provenance
	listReq := connect.NewRequest(&pb.ListUserEmailsRequest{})
	listReq.Header().Set("Authorization", "Bearer "+accessToken)
	listRes, err := authClient.ListUserEmails(ctx, listReq)
	if err != nil {
		t.Fatalf("ListUserEmails failed: %v", err)
	}
	if len(listRes.Msg.Emails) != 2 {
		t.Fatalf("expected 2 emails from GitHub OAuth signup, got %d", len(listRes.Msg.Emails))
	}
	for _, em := range listRes.Msg.Emails {
		if em.VerifiedViaOtp {
			t.Errorf("expected VerifiedViaOtp = false for OAuth-created email %s", em.Email)
		}
		if len(em.OauthProviders) != 1 || em.OauthProviders[0] != "github" {
			t.Errorf("expected OauthProviders = [github] for %s, got %v", em.Email, em.OauthProviders)
		}
	}

	// 5. UnlinkOAuth(GITHUB) must be blocked by primary email safeguard
	unlinkReq := connect.NewRequest(&pb.UnlinkOAuthRequest{
		Provider: pb.OAuthProvider_OAUTH_PROVIDER_GITHUB,
	})
	unlinkReq.Header().Set("Authorization", "Bearer "+accessToken)
	_, err = authClient.UnlinkOAuth(ctx, unlinkReq)
	if err == nil || connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("expected FailedPrecondition when unlinking sole primary email provider, got: %v", err)
	}

	// 6. Link Discord (OAUTH_PURPOSE_ADD_EMAIL) and then Unlink Discord (prunes orphaned secondary email)
	startDiscordReq := connect.NewRequest(&pb.StartOAuthRequest{
		Provider:     pb.OAuthProvider_OAUTH_PROVIDER_DISCORD,
		Purpose:      pb.OAuthPurpose_OAUTH_PURPOSE_ADD_EMAIL,
		RedirectPath: "/account",
	})
	startDiscordReq.Header().Set("Authorization", "Bearer "+accessToken)
	startDiscordRes, err := authClient.StartOAuth(ctx, startDiscordReq)
	if err != nil {
		t.Fatalf("StartOAuth Discord ADD_EMAIL failed: %v", err)
	}

	compDiscordReq := connect.NewRequest(&pb.CompleteOAuthRequest{
		Code:  "mock_discord_default",
		State: startDiscordRes.Msg.State,
	})
	compDiscordReq.Header().Set("Authorization", "Bearer "+accessToken)
	compDiscordRes, err := authClient.CompleteOAuth(ctx, compDiscordReq)
	if err != nil || !compDiscordRes.Msg.Success {
		t.Fatalf("CompleteOAuth Discord ADD_EMAIL failed: %v, res: %+v", err, compDiscordRes)
	}

	unlinkDiscordReq := connect.NewRequest(&pb.UnlinkOAuthRequest{
		Provider: pb.OAuthProvider_OAUTH_PROVIDER_DISCORD,
	})
	unlinkDiscordReq.Header().Set("Authorization", "Bearer "+accessToken)
	if _, err := authClient.UnlinkOAuth(ctx, unlinkDiscordReq); err != nil {
		t.Fatalf("UnlinkOAuth Discord failed: %v", err)
	}

	listResAfterUnlink, err := authClient.ListUserEmails(ctx, listReq)
	if err != nil {
		t.Fatalf("ListUserEmails after unlink failed: %v", err)
	}
	if len(listResAfterUnlink.Msg.Emails) != 2 {
		t.Fatalf("expected orphaned Discord secondary email to be pruned leaving 2 emails, got %d", len(listResAfterUnlink.Msg.Emails))
	}

	// 7. Seed a second account with collections and questions, and test OAuth-triggered MergeAccount preserving OAuth provenance and FK tables
	otherUserUUID := "018f0000-0000-7000-8000-0000000000f1"
	_, err = pool.Exec(ctx, `
		INSERT INTO users (uuid, id, role, created_at, updated_at)
		VALUES ($1, 'oauth_other_pilot', 'USER', NOW(), NOW())`, otherUserUUID)
	if err != nil {
		t.Fatalf("failed to seed oauth_other_pilot: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO user_emails (uuid, user_uuid, email, is_primary, verified_via_otp, created_at)
		VALUES ('018f0000-0000-7000-8000-0000000000f2', $1, 'other.oauth@example.com', true, true, NOW())`, otherUserUUID)
	if err != nil {
		t.Fatalf("failed to seed other.oauth@example.com: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO collections (uuid, id, user_uuid, name, description, is_system_collection, created_at, updated_at)
		VALUES ('018f0000-0000-7000-8000-0000000000f3', 'other_favorites', $1, 'Favorites', 'System favorites', true, NOW(), NOW())`, otherUserUUID)
	if err != nil {
		t.Fatalf("failed to seed collection for otherUserUUID: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO user_reputations (uuid, user_uuid, score, updated_at)
		VALUES ('018f0000-0000-7000-8000-0000000000f4', $1, 42, NOW())`, otherUserUUID)
	if err != nil {
		t.Fatalf("failed to seed user_reputations for otherUserUUID: %v", err)
	}

	startGoogleReq := connect.NewRequest(&pb.StartOAuthRequest{
		Provider:     pb.OAuthProvider_OAUTH_PROVIDER_GOOGLE,
		Purpose:      pb.OAuthPurpose_OAUTH_PURPOSE_ADD_EMAIL,
		RedirectPath: "/account",
	})
	startGoogleReq.Header().Set("Authorization", "Bearer "+accessToken)
	startGoogleRes, err := authClient.StartOAuth(ctx, startGoogleReq)
	if err != nil {
		t.Fatalf("StartOAuth Google ADD_EMAIL failed: %v", err)
	}

	compGoogleReq := connect.NewRequest(&pb.CompleteOAuthRequest{
		Code:  "mock:google:g_acc_999:other.oauth@example.com",
		State: startGoogleRes.Msg.State,
	})
	compGoogleReq.Header().Set("Authorization", "Bearer "+accessToken)
	compGoogleRes, err := authClient.CompleteOAuth(ctx, compGoogleReq)
	if err != nil {
		t.Fatalf("CompleteOAuth Google conflict check failed: %v", err)
	}
	if !compGoogleRes.Msg.MergeRequired || compGoogleRes.Msg.GetMergeTicket() == "" {
		t.Fatalf("expected MergeRequired = true with MergeTicket, got: %+v", compGoogleRes.Msg)
	}

	mergeTicket := compGoogleRes.Msg.GetMergeTicket()
	mergeReq := connect.NewRequest(&pb.MergeAccountRequest{
		Email:       "other.oauth@example.com",
		MergeTicket: &mergeTicket,
	})
	mergeReq.Header().Set("Authorization", "Bearer "+accessToken)
	mergeRes, err := authClient.MergeAccount(ctx, mergeReq)
	if err != nil || !mergeRes.Msg.Success {
		t.Fatalf("MergeAccount with OAuth merge ticket failed: %v", err)
	}

	listResAfterMerge, err := authClient.ListUserEmails(ctx, listReq)
	if err != nil {
		t.Fatalf("ListUserEmails after merge failed: %v", err)
	}
	foundMergedEmail := false
	for _, em := range listResAfterMerge.Msg.Emails {
		if em.Email == "other.oauth@example.com" {
			foundMergedEmail = true
			if !em.VerifiedViaOtp {
				t.Errorf("expected merged email other.oauth@example.com to preserve VerifiedViaOtp = true")
			}
			if len(em.OauthProviders) != 1 || em.OauthProviders[0] != "google" {
				t.Errorf("expected merged email other.oauth@example.com to have OauthProviders = [google], got %v", em.OauthProviders)
			}
		}
	}
	if !foundMergedEmail {
		t.Errorf("expected other.oauth@example.com to be present on account after merge")
	}

	// 8. Test CLI OAuth Login flow (OAUTH_PURPOSE_CLI_LOGIN -> CompleteOAuth -> GetLoginChallengeStatus)
	startCliRes, err := authClient.StartOAuth(ctx, connect.NewRequest(&pb.StartOAuthRequest{
		Provider:     pb.OAuthProvider_OAUTH_PROVIDER_GITHUB,
		Purpose:      pb.OAuthPurpose_OAUTH_PURPOSE_CLI_LOGIN,
		RedirectPath: "/login",
	}))
	if err != nil {
		t.Fatalf("StartOAuth CLI_LOGIN failed: %v", err)
	}
	if startCliRes.Msg.GetChallengeId() == "" {
		t.Fatalf("expected non-empty challenge_id from StartOAuth CLI_LOGIN")
	}

	compCliRes, err := authClient.CompleteOAuth(ctx, connect.NewRequest(&pb.CompleteOAuthRequest{
		Code:  "mock_github_default",
		State: startCliRes.Msg.State,
	}))
	if err != nil || !compCliRes.Msg.Success {
		t.Fatalf("CompleteOAuth CLI_LOGIN failed: %v", err)
	}

	cliStatusRes, err := authClient.GetLoginChallengeStatus(ctx, connect.NewRequest(&pb.GetLoginChallengeStatusRequest{
		ChallengeId: startCliRes.Msg.GetChallengeId(),
	}))
	if err != nil {
		t.Fatalf("GetLoginChallengeStatus for CLI_LOGIN failed: %v", err)
	}
	if cliStatusRes.Msg.Status != pb.LoginChallengeStatus_LOGIN_CHALLENGE_STATUS_APPROVED {
		t.Fatalf("expected CLI login challenge status APPROVED, got %v", cliStatusRes.Msg.Status)
	}
	if cliStatusRes.Msg.GetAuthResponse() == nil || cliStatusRes.Msg.GetAuthResponse().GetAccessToken() == "" {
		t.Fatalf("expected valid AuthResponse with AccessToken from CLI login challenge")
	}
	if cliStatusRes.Msg.GetAuthResponse().GetUser().GetEmail() != "pilot.github@example.com" {
		t.Errorf("expected primary email pilot.github@example.com on CLI session user, got %s", cliStatusRes.Msg.GetAuthResponse().GetUser().GetEmail())
	}
}
