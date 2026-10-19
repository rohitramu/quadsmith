package api

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"

	"quadsmith/shared/pkg/auth"

	apipb "quadsmith/shared/gen/quadsmith/api"
	"quadsmith/shared/gen/quadsmith/api/apiconnect"
	pb "quadsmith/shared/gen/quadsmith/ops"
	"quadsmith/shared/gen/quadsmith/ops/opsconnect"
)

func getOpsURL() string {
	if url := os.Getenv("QS_OPS_SERVER_URL"); url != "" {
		return url
	}
	if url := os.Getenv("QS_OPS_URL"); url != "" {
		return url
	}
	return "http://127.0.0.1:8082"
}

func getOpsDBURL() string {
	if url := os.Getenv("QS_INTERNAL_DATABASE_URL"); url != "" {
		return url
	}
	return "postgres://postgres:postgres@127.0.0.1:5432/quadsmith_ops"
}

func newOpsClient(token string) opsconnect.OpsServiceClient {
	transport := &authRoundTripper{
		base:  http.DefaultTransport,
		token: token,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}
	return opsconnect.NewOpsServiceClient(client, getOpsURL())
}

func resolveQSOpsPath(t *testing.T) string {
	t.Helper()
	for _, p := range []string{"../../bin/qsops", "../bin/qsops", "bin/qsops"} {
		abs, err := filepath.Abs(p)
		if err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs
			}
		}
	}
	t.Fatalf("Failed to resolve qsops binary path (checked ../../bin/qsops, ../bin/qsops, bin/qsops)")
	return ""
}

func TestOpsServer_HealthAndDiagnostics(t *testing.T) {
	opsUrl := getOpsURL()
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(opsUrl + "/healthz")
	if err != nil {
		t.Fatalf("Ops server not running at %s: %v", opsUrl, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK from /healthz, got %d", resp.StatusCode)
	}

	opsClient := newOpsClient("qs_op_sandbox_token_12345")
	ctx := context.Background()

	diagRes, err := opsClient.GetSystemDiagnostics(ctx, connect.NewRequest(&pb.GetSystemDiagnosticsRequest{}))
	if err != nil {
		t.Fatalf("GetSystemDiagnostics failed: %v", err)
	}

	diag := diagRes.Msg
	if diag.Version == "" {
		t.Errorf("Expected non-empty version")
	}
	if diag.Environment == "" {
		t.Errorf("Expected non-empty environment")
	}
	if diag.NumGoroutines <= 0 {
		t.Errorf("Expected positive goroutine count, got %d", diag.NumGoroutines)
	}
}

func TestOpsServer_AuthenticationAndRBAC(t *testing.T) {
	ctx := context.Background()

	// 1. Unauthenticated request (no token)
	unauthClient := newOpsClient("")
	_, err := unauthClient.GetSystemDiagnostics(ctx, connect.NewRequest(&pb.GetSystemDiagnosticsRequest{}))
	if err == nil {
		t.Fatalf("Expected unauthenticated request to fail, got success")
	}
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("Expected CodeUnauthenticated, got %v: %v", connect.CodeOf(err), err)
	}

	// 2. Invalid token
	badClient := newOpsClient("invalid-bogus-token-xyz")
	_, err = badClient.GetSystemDiagnostics(ctx, connect.NewRequest(&pb.GetSystemDiagnosticsRequest{}))
	if err == nil {
		t.Fatalf("Expected invalid token request to fail, got success")
	}
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("Expected CodeUnauthenticated, got %v: %v", connect.CodeOf(err), err)
	}

	// 3. Viewer token: allowed for Get* / List*
	viewerClient := newOpsClient("qs_viewer_sandbox_token_99999")
	_, err = viewerClient.GetSystemDiagnostics(ctx, connect.NewRequest(&pb.GetSystemDiagnosticsRequest{}))
	if err != nil {
		// Fallback to test viewer token if configured
		viewerClient = newOpsClient("qs_viewer_test_token_99999")
		_, err = viewerClient.GetSystemDiagnostics(ctx, connect.NewRequest(&pb.GetSystemDiagnosticsRequest{}))
	}
	if err != nil {
		t.Fatalf("Viewer token expected to access GetSystemDiagnostics: %v", err)
	}

	// Viewer token trying to mutate/create token -> PermissionDenied
	_, err = viewerClient.CreateInternalToken(ctx, connect.NewRequest(&pb.CreateInternalTokenRequest{
		Name:         "unauthorized-token",
		Type:         "HUMAN",
		GroupName:    "operators",
		DurationDays: 30,
	}))
	if err == nil {
		t.Fatalf("Expected viewer token to be denied CreateInternalToken")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Errorf("Expected CodePermissionDenied for viewer mutation, got %v: %v", connect.CodeOf(err), err)
	}
}

func TestOpsServer_JITElevation(t *testing.T) {
	ctx := context.Background()
	operatorClient := newOpsClient("qs_op_sandbox_token_12345")

	// Create a temporary token so we have a valid target for RevokeInternalToken
	createTokRes, err := operatorClient.CreateInternalToken(ctx, connect.NewRequest(&pb.CreateInternalTokenRequest{
		Name:         "jit-elevation-test-token",
		Type:         "ROBOT",
		GroupName:    "robots",
		DurationDays: 1,
	}))
	if err != nil {
		t.Fatalf("CreateInternalToken failed: %v", err)
	}
	targetTokenID := createTokRes.Msg.Record.Id

	// 1. RevokeInternalToken is a sensitive method requiring JIT elevation
	_, err = operatorClient.RevokeInternalToken(ctx, connect.NewRequest(&pb.RevokeInternalTokenRequest{
		Id: targetTokenID,
	}))
	if err == nil {
		t.Fatalf("Expected RevokeInternalToken to fail without active JIT elevation")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Errorf("Expected CodePermissionDenied for JIT-required method, got %v: %v", connect.CodeOf(err), err)
	}

	// 2. Acquire JIT Elevation (auto-approved in sandbox/test)
	grantRes, err := operatorClient.AcquireJitGrant(ctx, connect.NewRequest(&pb.AcquireJitGrantRequest{
		Hours:     2,
		Reason:    "E2E Integration Test Revoke Internal Token",
		Operation: "*",
	}))
	if err != nil {
		t.Fatalf("AcquireJitGrant failed: %v", err)
	}

	grant := grantRes.Msg.Grant
	if grant.Id == "" {
		t.Fatalf("Expected non-empty JIT grant ID")
	}
	if grant.Status != "ACTIVE" {
		t.Errorf("Expected grant status ACTIVE, got %q", grant.Status)
	}

	// 3. Verify operator has active JIT in GetCurrentOperator
	meRes, err := operatorClient.GetCurrentOperator(ctx, connect.NewRequest(&pb.GetCurrentOperatorRequest{}))
	if err != nil {
		t.Fatalf("GetCurrentOperator failed: %v", err)
	}
	if meRes.Msg.ActiveJitGrant == nil || meRes.Msg.ActiveJitGrant.Id != grant.Id {
		t.Errorf("Expected active JIT grant %s in GetCurrentOperator", grant.Id)
	}

	// 4. Now RevokeInternalToken succeeds with active JIT!
	revTokRes, err := operatorClient.RevokeInternalToken(ctx, connect.NewRequest(&pb.RevokeInternalTokenRequest{
		Id: targetTokenID,
	}))
	if err != nil {
		t.Fatalf("RevokeInternalToken failed despite active JIT: %v", err)
	}
	if !revTokRes.Msg.Success {
		t.Errorf("Expected token revocation success")
	}

	// Verify audit record stores jit_grant_uuid and projects jit_grant_id via LEFT JOIN
	time.Sleep(50 * time.Millisecond)
	auditRes, err := operatorClient.ListInternalApiCalls(ctx, connect.NewRequest(&pb.ListInternalApiCallsRequest{
		MethodFilter: "RevokeInternalToken",
		Limit:        5,
	}))
	if err != nil {
		t.Fatalf("ListInternalApiCalls failed: %v", err)
	}
	foundJitCall := false
	for _, c := range auditRes.Msg.GetCalls() {
		if c.GetJitGrantUuid() == grant.Uuid && c.GetJitGrantId() == grant.Id {
			foundJitCall = true
			break
		}
	}
	if !foundJitCall {
		t.Errorf("Expected audit record for RevokeInternalToken with jit_grant_uuid=%s and jit_grant_id=%s", grant.Uuid, grant.Id)
	}

	// 5. Extend JIT Grant
	extRes, err := operatorClient.ExtendJitGrant(ctx, connect.NewRequest(&pb.ExtendJitGrantRequest{
		GrantId: grant.Id,
		Hours:   1,
	}))
	if err != nil {
		t.Fatalf("ExtendJitGrant failed: %v", err)
	}
	if extRes.Msg.Grant.Hours != 3 {
		t.Errorf("Expected total 3 hours after extension, got %d", extRes.Msg.Grant.Hours)
	}

	// 6. Revoke JIT Grant
	revokeRes, err := operatorClient.RevokeJitGrant(ctx, connect.NewRequest(&pb.RevokeJitGrantRequest{
		GrantId: grant.Id,
	}))
	if err != nil {
		t.Fatalf("RevokeJitGrant failed: %v", err)
	}
	if !revokeRes.Msg.Success {
		t.Errorf("Expected revoke success")
	}

	// 7. Verify subsequent sensitive method fails again
	_, err = operatorClient.RevokeInternalToken(ctx, connect.NewRequest(&pb.RevokeInternalTokenRequest{
		Id: targetTokenID,
	}))
	if err == nil {
		t.Fatalf("Expected RevokeInternalToken to fail after JIT revocation")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Errorf("Expected CodePermissionDenied after JIT revocation, got %v: %v", connect.CodeOf(err), err)
	}
}

func TestOpsServer_DatabaseStats(t *testing.T) {
	ctx := context.Background()
	operatorClient := newOpsClient("qs_op_sandbox_token_12345")

	statsRes, err := operatorClient.GetDatabaseStats(ctx, connect.NewRequest(&pb.GetDatabaseStatsRequest{}))
	if err != nil {
		t.Fatalf("GetDatabaseStats failed: %v", err)
	}

	stats := statsRes.Msg
	if stats.DatabaseName != "quadsmith_api" {
		t.Errorf("Expected database_name 'quadsmith_api', got %q", stats.DatabaseName)
	}
	if len(stats.Tables) == 0 {
		t.Errorf("Expected table stats in database stats response, got 0 tables")
	}

	foundUsers := false
	for _, tbl := range stats.Tables {
		if tbl.TableName == "users" {
			foundUsers = true
			break
		}
	}
	if !foundUsers {
		t.Errorf("Expected 'users' table in database stats")
	}
}

func TestOpsServer_TasksJobsAndAudit(t *testing.T) {
	ctx := context.Background()
	operatorClient := newOpsClient("qs_op_sandbox_token_12345")

	// 1. List Tasks on OpsService
	tasksRes, err := operatorClient.ListTasks(ctx, connect.NewRequest(&pb.ListTasksRequest{}))
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}

	expectedTasks := []string{
		"cleanup_expired_otps",
		"cleanup_expired_tokens",
		"cleanup_orphaned_blobs",
		"purge_soft_deleted_accounts",
		"send_username_release_warnings",
		"set_user_role",
	}

	for _, exp := range expectedTasks {
		found := false
		for _, task := range tasksRes.Msg.Tasks {
			if task.Name == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected task %q in registered tasks list", exp)
		}
	}

	// 2. Get Task details
	getTaskRes, err := operatorClient.GetTask(ctx, connect.NewRequest(&pb.GetTaskRequest{
		TaskName: "cleanup_expired_otps",
	}))
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if getTaskRes.Msg.Task.Procedure != "CleanupExpiredOtps" {
		t.Errorf("Expected procedure CleanupExpiredOtps, got %s", getTaskRes.Msg.Task.Procedure)
	}

	// 3. Trigger maintenance tasks directly over dedicated typed RPCs
	otpsRes, err := operatorClient.CleanupExpiredOtps(ctx, connect.NewRequest(&pb.CleanupExpiredOtpsRequest{}))
	if err != nil {
		t.Fatalf("CleanupExpiredOtps failed: %v", err)
	}
	if otpsRes.Msg.CleanedCount < 0 {
		t.Errorf("Expected non-negative cleaned count, got %d", otpsRes.Msg.CleanedCount)
	}
	otpsJobID := otpsRes.Header().Get("X-Job-Id")
	if otpsJobID == "" {
		t.Errorf("Expected non-empty X-Job-Id header on CleanupExpiredOtps response")
	}

	purgeRes, err := operatorClient.PurgeSoftDeletedAccounts(ctx, connect.NewRequest(&pb.PurgeSoftDeletedAccountsRequest{}))
	if err != nil {
		t.Fatalf("PurgeSoftDeletedAccounts failed: %v", err)
	}
	if purgeRes.Msg.PurgedCount < 0 {
		t.Errorf("Expected non-negative purged count, got %d", purgeRes.Msg.PurgedCount)
	}

	warnRes, err := operatorClient.SendUsernameReleaseWarnings(ctx, connect.NewRequest(&pb.SendUsernameReleaseWarningsRequest{}))
	if err != nil {
		t.Fatalf("SendUsernameReleaseWarnings failed: %v", err)
	}
	if warnRes.Msg.WarningCount < 0 {
		t.Errorf("Expected non-negative warning count, got %d", warnRes.Msg.WarningCount)
	}

	// 4. Query Job History on OpsService
	limitVal := int32(20)
	jobsRes, err := operatorClient.ListJobs(ctx, connect.NewRequest(&pb.ListJobsRequest{Limit: &limitVal}))
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}
	if jobsRes.Header().Get("X-Job-Id") != "" {
		t.Errorf("Expected no X-Job-Id header on read-only ListJobs response, got %q", jobsRes.Header().Get("X-Job-Id"))
	}
	if len(jobsRes.Msg.Jobs) == 0 {
		t.Fatalf("Expected recorded jobs in OpsService after triggering tasks, got 0")
	}
	firstJob := jobsRes.Msg.Jobs[0]
	if firstJob.GetUuid() == "" {
		t.Errorf("Expected non-empty uuid on JobRecord")
	}
	getJobRes, err := operatorClient.GetJob(ctx, connect.NewRequest(&pb.GetJobRequest{JobId: firstJob.JobId}))
	if err != nil {
		t.Fatalf("GetJob failed for %s: %v", firstJob.JobId, err)
	}
	if getJobRes.Msg.Job.JobId != firstJob.JobId {
		t.Errorf("Expected job ID %s, got %s", firstJob.JobId, getJobRes.Msg.Job.JobId)
	}
	if getJobRes.Msg.Job.GetUuid() != firstJob.GetUuid() {
		t.Errorf("Expected job UUID %s, got %s", firstJob.GetUuid(), getJobRes.Msg.Job.GetUuid())
	}

	// Verify GetJob also resolves by UUID fallback
	getJobByUUIDRes, err := operatorClient.GetJob(ctx, connect.NewRequest(&pb.GetJobRequest{JobId: firstJob.GetUuid()}))
	if err != nil {
		t.Fatalf("GetJob by UUID failed for %s: %v", firstJob.GetUuid(), err)
	}
	if getJobByUUIDRes.Msg.Job.JobId != firstJob.JobId {
		t.Errorf("Expected job ID %s when querying by UUID, got %s", firstJob.JobId, getJobByUUIDRes.Msg.Job.JobId)
	}

	// Verify ListJobs filtering by valid task_name and rejection of invalid task_name
	taskFilter := "cleanup_expired_otps"
	filteredJobsRes, err := operatorClient.ListJobs(ctx, connect.NewRequest(&pb.ListJobsRequest{
		Limit:    &limitVal,
		TaskName: &taskFilter,
	}))
	if err != nil {
		t.Fatalf("ListJobs with task_name=%q failed: %v", taskFilter, err)
	}
	for _, j := range filteredJobsRes.Msg.GetJobs() {
		if j.GetTaskName() != "cleanup_expired_otps" {
			t.Errorf("Expected filtered job task_name 'cleanup_expired_otps', got %q", j.GetTaskName())
		}
	}

	badTask := "nonexistent_task"
	_, err = operatorClient.ListJobs(ctx, connect.NewRequest(&pb.ListJobsRequest{
		TaskName: &badTask,
	}))
	if err == nil {
		t.Errorf("Expected error for unknown task_name in ListJobs, got nil")
	}

	// Verify unauthenticated call to ListJobs is rejected
	unauthOpsClient := newOpsClient("")
	_, err = unauthOpsClient.ListJobs(ctx, connect.NewRequest(&pb.ListJobsRequest{}))
	if err == nil {
		t.Errorf("Expected error for unauthenticated ListJobs, got nil")
	}

	// Verify GetJob rejects non-existent job ID
	_, err = operatorClient.GetJob(ctx, connect.NewRequest(&pb.GetJobRequest{JobId: "nonexistent-job-id"}))
	if err == nil {
		t.Errorf("Expected error for non-existent job ID, got nil")
	}

	// 5. Verify Audit Log recorded in internal_api_calls
	auditRes, err := operatorClient.ListInternalApiCalls(ctx, connect.NewRequest(&pb.ListInternalApiCallsRequest{
		Limit: 20,
	}))
	if err != nil {
		t.Fatalf("ListInternalApiCalls failed: %v", err)
	}
	if len(auditRes.Msg.Calls) == 0 {
		t.Fatalf("Expected recorded internal API calls in audit log, got 0")
	}

	// Verify DB Isolation: audit table exists in quadsmith_ops, NOT in quadsmith_api
	opsDB := getOpsDBURL()
	pool, err := pgxpool.New(ctx, opsDB)
	if err == nil {
		defer pool.Close()
		var count int
		_ = pool.QueryRow(ctx, "SELECT count(*) FROM internal_api_calls").Scan(&count)
		if count == 0 {
			t.Errorf("Expected records in quadsmith_ops.internal_api_calls, got 0")
		}
	}

	publicDB := getDBURL()
	pubPool, err := pgxpool.New(ctx, publicDB)
	if err == nil {
		defer pubPool.Close()
		var tableCount int
		_ = pubPool.QueryRow(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'internal_api_calls'").Scan(&tableCount)
		if tableCount != 0 {
			t.Errorf("Expected internal_api_calls table to NOT exist in quadsmith_api, found %d", tableCount)
		}
	}
}

func TestQSOpsCLIExecution(t *testing.T) {
	qsopsPath := resolveQSOpsPath(t)

	// 1. qsops system diag
	cmd := exec.Command(qsopsPath, "system", "diag", "--token", "qs_op_sandbox_token_12345", "--json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("qsops system diag failed: %v, output: %s", err, string(out))
	}
	outStr := string(out)
	if !containsString(outStr, "environment") && !containsString(outStr, "version") {
		t.Errorf("Expected json diagnostics in output: %s", outStr)
	}

	// 2. qsops db stats
	cmd = exec.Command(qsopsPath, "db", "stats", "--token", "qs_op_sandbox_token_12345", "--json")
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("qsops db stats failed: %v, output: %s", err, string(out))
	}
	outStr = string(out)
	if !containsString(outStr, "quadsmith_api") {
		t.Errorf("Expected 'quadsmith_api' in db stats output: %s", outStr)
	}

	// 3. qsops tasks list
	cmd = exec.Command(qsopsPath, "tasks", "list", "--token", "qs_op_sandbox_token_12345", "--json")
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("qsops tasks list failed: %v, output: %s", err, string(out))
	}
	outStr = string(out)
	if !containsString(outStr, "cleanup_expired_otps") {
		t.Errorf("Expected 'cleanup_expired_otps' in tasks list output: %s", outStr)
	}

	// 4. qsops tasks get cleanup_expired_otps
	cmd = exec.Command(qsopsPath, "tasks", "get", "cleanup_expired_otps", "--token", "qs_op_sandbox_token_12345")
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("qsops tasks get failed: %v, output: %s", err, string(out))
	}
	outStr = string(out)
	if !containsString(outStr, "CleanupExpiredOtps") {
		t.Errorf("Expected 'CleanupExpiredOtps' in tasks get output: %s", outStr)
	}

	// 5. qsops tasks trigger cleanup_expired_otps
	cmd = exec.Command(qsopsPath, "tasks", "trigger", "cleanup_expired_otps", "--token", "qs_op_sandbox_token_12345")
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("qsops tasks trigger failed: %v, output: %s", err, string(out))
	}
	outStr = string(out)
	if !containsString(outStr, "Cleaned") || !containsString(outStr, "Job ID:") {
		t.Errorf("Expected 'Cleaned' and 'Job ID:' in tasks trigger output: %s", outStr)
	}

	// 6. qsops jobs list
	cmd = exec.Command(qsopsPath, "jobs", "list", "--task", "cleanup_expired_otps", "--token", "qs_op_sandbox_token_12345")
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("qsops jobs list failed: %v, output: %s", err, string(out))
	}
	outStr = string(out)
	if !containsString(outStr, "cleanup_expired_otps") || !containsString(outStr, "CALLER") {
		t.Errorf("Expected 'cleanup_expired_otps' and 'CALLER' header in jobs list output: %s", outStr)
	}

	// 7. qsops jobs get <job-id>
	opsClient := newOpsClient("qs_op_sandbox_token_12345")
	jobsRes, err := opsClient.ListJobs(context.Background(), connect.NewRequest(&pb.ListJobsRequest{}))
	if err != nil || len(jobsRes.Msg.GetJobs()) == 0 {
		t.Fatalf("ListJobs for CLI get test failed: %v", err)
	}
	targetJobID := jobsRes.Msg.GetJobs()[0].GetJobId()

	cmd = exec.Command(qsopsPath, "jobs", "get", targetJobID, "--token", "qs_op_sandbox_token_12345")
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("qsops jobs get %s failed: %v, output: %s", targetJobID, err, string(out))
	}
	outStr = string(out)
	if !containsString(outStr, targetJobID) || !containsString(outStr, "UUID:") {
		t.Errorf("Expected job details in jobs get output: %s", outStr)
	}
}

func TestOpsServer_SetUserRoleAndBanRoleDemotion(t *testing.T) {
	ctx := context.Background()
	pubPool, err := pgxpool.New(ctx, getDBURL())
	if err != nil {
		t.Fatalf("Failed to connect to quadsmith_api DB: %v", err)
	}
	defer pubPool.Close()

	user1UUID := "018f0000-0000-7000-8000-000000000901"
	user1Handle := "role_test_admin1"
	user1Email := "role_admin1@quadsmith.dev"

	user2UUID := "018f0000-0000-7000-8000-000000000902"
	user2Handle := "role_test_admin2"
	user2Email := "role_admin2@quadsmith.dev"

	cleanup := func() {
		_, _ = pubPool.Exec(ctx, `DELETE FROM banned_emails WHERE email IN ($1, $2)`, user1Email, user2Email)
		_, _ = pubPool.Exec(ctx, `DELETE FROM bans WHERE user_uuid IN ($1::uuid, $2::uuid) OR banned_by_user_uuid IN ($1::uuid, $2::uuid)`, user1UUID, user2UUID)
		_, _ = pubPool.Exec(ctx, `DELETE FROM user_emails WHERE user_uuid IN ($1::uuid, $2::uuid)`, user1UUID, user2UUID)
		_, _ = pubPool.Exec(ctx, `DELETE FROM user_profiles WHERE user_uuid IN ($1::uuid, $2::uuid)`, user1UUID, user2UUID)
		_, _ = pubPool.Exec(ctx, `DELETE FROM users WHERE uuid IN ($1::uuid, $2::uuid)`, user1UUID, user2UUID)
	}
	cleanup()
	defer cleanup()

	// 1. Seed both users without specifying role (relies on PostgreSQL DEFAULT 'USER')
	for _, u := range []struct {
		uuid, handle, email string
	}{
		{user1UUID, user1Handle, user1Email},
		{user2UUID, user2Handle, user2Email},
	} {
		if _, err := pubPool.Exec(ctx, `INSERT INTO users (uuid, id, created_at, updated_at) VALUES ($1, $2, NOW(), NOW())`, u.uuid, u.handle); err != nil {
			t.Fatalf("Failed to seed user %s: %v", u.handle, err)
		}
		if _, err := pubPool.Exec(ctx, `INSERT INTO user_emails (uuid, user_uuid, email, is_primary, created_at) VALUES ($1, $2, $3, true, NOW())`, auth.NewUUIDv7(), u.uuid, u.email); err != nil {
			t.Fatalf("Failed to seed email for %s: %v", u.handle, err)
		}
	}

	// 2. Verify PostgreSQL CHECK (role IN ('USER', 'ADMIN')) constraint rejects invalid roles
	_, err = pubPool.Exec(ctx, `UPDATE users SET role = 'SUPERADMIN' WHERE uuid = $1::uuid`, user1UUID)
	if err == nil {
		t.Fatalf("Expected PostgreSQL CHECK constraint to reject role='SUPERADMIN'")
	}

	// 3. Verify SetUserRole requires active JIT grant and HUMAN operator
	operatorClient := newOpsClient("qs_op_sandbox_token_12345")
	robotClient := newOpsClient("qs_robot_sandbox_token_67890")

	// Revoke any pre-existing active JIT grants on sandbox operator
	if grantsRes, err := operatorClient.ListJitGrants(ctx, connect.NewRequest(&pb.ListJitGrantsRequest{ActiveOnly: true})); err == nil {
		for _, g := range grantsRes.Msg.GetGrants() {
			_, _ = operatorClient.RevokeJitGrant(ctx, connect.NewRequest(&pb.RevokeJitGrantRequest{GrantId: g.GetId()}))
		}
	}

	// Without JIT grant -> PermissionDenied
	_, err = operatorClient.SetUserRole(ctx, connect.NewRequest(&pb.SetUserRoleRequest{
		UserId: user1Handle,
		Role:   "ADMIN",
	}))
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("Expected CodePermissionDenied when calling SetUserRole without JIT grant, got %v", err)
	}

	// With ROBOT token -> PermissionDenied
	_, err = robotClient.SetUserRole(ctx, connect.NewRequest(&pb.SetUserRoleRequest{
		UserId: user1Handle,
		Role:   "ADMIN",
	}))
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("Expected CodePermissionDenied when calling SetUserRole with ROBOT token, got %v", err)
	}

	// 4. Verify user1's JWT initially does NOT have admin access on server-api
	token1, err := auth.SignAccessToken(user1UUID, user1Handle, user1Email, "session-admin1")
	if err != nil {
		t.Fatalf("Failed to sign token for user1: %v", err)
	}
	user1ApiClient := apiconnect.NewUserServiceClient(&http.Client{
		Transport: &authRoundTripper{base: http.DefaultTransport, token: token1},
		Timeout:   5 * time.Second,
	}, getAPIURL())

	_, err = user1ApiClient.ListBans(ctx, connect.NewRequest(&apipb.ListBansRequest{}))
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("Expected CodePermissionDenied for ListBans before promotion, got %v", err)
	}

	// 5. Acquire JIT grant for HUMAN operator
	grantRes, err := operatorClient.AcquireJitGrant(ctx, connect.NewRequest(&pb.AcquireJitGrantRequest{
		Hours:     1,
		Reason:    "E2E test for SetUserRole",
		Operation: "SetUserRole",
	}))
	if err != nil {
		t.Fatalf("AcquireJitGrant failed: %v", err)
	}
	grantID := grantRes.Msg.GetGrant().GetId()
	defer func() {
		_, _ = operatorClient.RevokeJitGrant(ctx, connect.NewRequest(&pb.RevokeJitGrantRequest{GrantId: grantID}))
	}()

	// 6. Promote user1 to ADMIN via OpsService.SetUserRole
	setRes, err := operatorClient.SetUserRole(ctx, connect.NewRequest(&pb.SetUserRoleRequest{
		UserId: user1Handle,
		Role:   "ADMIN",
	}))
	if err != nil {
		t.Fatalf("SetUserRole(user1 -> ADMIN) failed: %v", err)
	}
	if setRes.Msg.GetPreviousRole() != "USER" || setRes.Msg.GetRole() != "ADMIN" {
		t.Errorf("Unexpected SetUserRole response: prev=%q, role=%q", setRes.Msg.GetPreviousRole(), setRes.Msg.GetRole())
	}
	jobID := setRes.Header().Get("X-Job-Id")
	if jobID == "" {
		t.Errorf("Expected non-empty X-Job-Id header on SetUserRole response")
	}

	// Verify job is inspectable via GetJob
	jobRes, err := operatorClient.GetJob(ctx, connect.NewRequest(&pb.GetJobRequest{JobId: jobID}))
	if err != nil {
		t.Fatalf("GetJob(%s) for SetUserRole failed: %v", jobID, err)
	}
	if jobRes.Msg.GetJob().GetTaskName() != "set_user_role" || jobRes.Msg.GetJob().GetProcedure() != "SetUserRole" {
		t.Errorf("Unexpected job record: task=%q, proc=%q", jobRes.Msg.GetJob().GetTaskName(), jobRes.Msg.GetJob().GetProcedure())
	}

	// 7. Verify user1's existing JWT immediately has ADMIN privileges on server-api (live DB role hydration)
	if _, err := user1ApiClient.ListBans(ctx, connect.NewRequest(&apipb.ListBansRequest{})); err != nil {
		t.Fatalf("Expected ListBans to succeed immediately after promoting user1 to ADMIN, got %v", err)
	}

	// 8. Promote user2 to ADMIN via qsops CLI
	qsopsPath := resolveQSOpsPath(t)

	// Verify alias "setuserrole" is rejected by Cobra (only "set_user_role" is accepted)
	cmd := exec.Command(qsopsPath, "tasks", "trigger", "setuserrole", "--token", "qs_op_sandbox_token_12345")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("Expected qsops tasks trigger setuserrole to fail, but succeeded: %s", string(out))
	}
	if !containsString(string(out), `unknown task subcommand "setuserrole"`) && !containsString(string(out), `unknown command "setuserrole"`) {
		t.Errorf("Expected 'unknown command/subcommand' error for setuserrole, got: %s", string(out))
	}

	// Verify strict username match in qsops: leading "@" is NOT stripped -> fails
	cmd = exec.Command(qsopsPath, "tasks", "trigger", "set_user_role", "--user", "@"+user2Handle, "--role", "ADMIN", "--token", "qs_op_sandbox_token_12345")
	out, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("Expected qsops set_user_role with '@' to fail under strict matching, but succeeded: %s", string(out))
	}

	// Verify strict username match in API: uppercase handle fails (no lowercase conversion)
	_, err = operatorClient.SetUserRole(ctx, connect.NewRequest(&pb.SetUserRoleRequest{
		UserId: strings.ToUpper(user2Handle),
		Role:   "ADMIN",
	}))
	if err == nil || connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("Expected CodeNotFound when calling SetUserRole with uppercase handle, got %v", err)
	}

	// Strict match with exact username succeeds
	cmd = exec.Command(qsopsPath, "tasks", "trigger", "set_user_role", "--user", user2Handle, "--role", "ADMIN", "--token", "qs_op_sandbox_token_12345")
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("qsops tasks trigger set_user_role failed: %v, output: %s", err, string(out))
	}
	if !containsString(string(out), "from USER to ADMIN") || !containsString(string(out), "Job ID:") {
		t.Errorf("Unexpected qsops set_user_role output: %s", string(out))
	}

	var user2Role string
	if err := pubPool.QueryRow(ctx, `SELECT role FROM users WHERE uuid = $1::uuid`, user2UUID).Scan(&user2Role); err != nil || user2Role != "ADMIN" {
		t.Fatalf("Expected user2 role in DB to be ADMIN, got %q (err=%v)", user2Role, err)
	}

	// 9. Ban user2 (who is currently ADMIN) using user1's admin client -> must succeed and demote user2 to USER
	banRes, err := user1ApiClient.BanUser(ctx, connect.NewRequest(&apipb.BanUserRequest{
		UserId: user2Handle,
		Reason: "Testing automatic role demotion on ban",
	}))
	if err != nil || !banRes.Msg.GetSuccess() {
		t.Fatalf("BanUser on admin user2 failed: %v", err)
	}
	if err := pubPool.QueryRow(ctx, `SELECT role FROM users WHERE uuid = $1::uuid`, user2UUID).Scan(&user2Role); err != nil || user2Role != "USER" {
		t.Fatalf("Expected banned user2 role to automatically revert to USER, got %q (err=%v)", user2Role, err)
	}

	// 10. Attempting to promote actively banned user2 back to ADMIN via SetUserRole must fail with CodeFailedPrecondition
	_, err = operatorClient.SetUserRole(ctx, connect.NewRequest(&pb.SetUserRoleRequest{
		UserId: user2UUID,
		Role:   "ADMIN",
	}))
	if err == nil || connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("Expected CodeFailedPrecondition when promoting actively banned user to ADMIN, got %v", err)
	}

	// 11. Revoke the ban on user2 -> role must remain USER (not restored to ADMIN)
	revokeFlag := true
	revokeReason := "Lifting test ban"
	if _, err := user1ApiClient.UpdateBan(ctx, connect.NewRequest(&apipb.UpdateBanRequest{
		UserId:       user2Handle,
		Revoke:       &revokeFlag,
		RevokeReason: &revokeReason,
	})); err != nil {
		t.Fatalf("UpdateBan(revoke) failed: %v", err)
	}
	if err := pubPool.QueryRow(ctx, `SELECT role FROM users WHERE uuid = $1::uuid`, user2UUID).Scan(&user2Role); err != nil || user2Role != "USER" {
		t.Fatalf("Expected user2 role to remain USER after ban revocation, got %q (err=%v)", user2Role, err)
	}

	// 12. Soft-delete user2 -> SetUserRole must reject soft-deleted user with CodeFailedPrecondition
	if _, err := pubPool.Exec(ctx, `UPDATE users SET deleted_at = NOW() WHERE uuid = $1::uuid`, user2UUID); err != nil {
		t.Fatalf("Failed to soft-delete user2: %v", err)
	}
	_, err = operatorClient.SetUserRole(ctx, connect.NewRequest(&pb.SetUserRoleRequest{
		UserId: user2Handle,
		Role:   "ADMIN",
	}))
	if err == nil || connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("Expected CodeFailedPrecondition when changing role of soft-deleted user, got %v", err)
	}

	// 13. Demote user1 back to USER via qsops CLI -> existing JWT immediately loses ADMIN privileges
	cmd = exec.Command(qsopsPath, "tasks", "trigger", "set_user_role", "--user", user1UUID, "--role", "USER", "--token", "qs_op_sandbox_token_12345")
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("qsops tasks trigger set_user_role (demote) failed: %v, output: %s", err, string(out))
	}
	if !containsString(string(out), "from ADMIN to USER") {
		t.Errorf("Unexpected qsops demote output: %s", string(out))
	}
	_, err = user1ApiClient.ListBans(ctx, connect.NewRequest(&apipb.ListBansRequest{}))
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("Expected CodePermissionDenied for ListBans immediately after demoting user1 to USER, got %v", err)
	}
}

func TestOpsServer_ServiceHealthAndObservability(t *testing.T) {
	ctx := context.Background()
	client := &http.Client{Timeout: 5 * time.Second}

	services := []struct {
		name string
		url  string
	}{
		{"server-api", getAPIURL()},
		{"server-ops-api", getOpsURL()},
		{"server-ops-web", "http://127.0.0.1:8083"},
		{"server-cron", "http://127.0.0.1:8084"},
		{"server-web", "http://127.0.0.1:8080"},
	}

	// 1. Verify /healthz returns structured JSON with status=HEALTHY, service, region, and X-Trace-Id header,
	//    and /metrics exposes quadsmith_service_info + quadsmith_http_requests_total across all 5 services.
	for _, svc := range services {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, svc.url+"/healthz", nil)
		// Attempt to spoof X-Trace-Id without hop token; service must ignore it and mint a fresh UUIDv7
		req.Header.Set("X-Trace-Id", "spoofed-client-trace-id")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("GET %s/healthz failed: %v", svc.url, err)
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200 from %s/healthz, got %d", svc.name, resp.StatusCode)
		}
		traceID := resp.Header.Get("X-Trace-Id")
		if traceID == "" || traceID == "spoofed-client-trace-id" {
			t.Errorf("Expected fresh server-minted X-Trace-Id on %s/healthz, got %q", svc.name, traceID)
		}
		body := string(bodyBytes)
		if !containsString(body, `"status":"HEALTHY"`) || !containsString(body, `"service":"`+svc.name+`"`) || !containsString(body, `"region":`) {
			t.Errorf("Unexpected %s/healthz body: %s", svc.name, body)
		}

		// Verify /metrics
		mResp, err := client.Get(svc.url + "/metrics")
		if err != nil {
			t.Fatalf("GET %s/metrics failed: %v", svc.url, err)
		}
		mBytes, _ := io.ReadAll(mResp.Body)
		mResp.Body.Close()
		if mResp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200 from %s/metrics, got %d", svc.name, mResp.StatusCode)
		}
		mBody := string(mBytes)
		if !containsString(mBody, `quadsmith_health_status{`) || !containsString(mBody, `service="`+svc.name+`"`) {
			t.Errorf("Expected quadsmith_health_status for %s in /metrics output", svc.name)
		}
	}

	// 2. Verify Nginx blocks external /metrics with 403 Forbidden
	nginxMetricsResp, err := client.Get("http://127.0.0.1:9000/metrics")
	if err == nil {
		nginxMetricsResp.Body.Close()
		if nginxMetricsResp.StatusCode != http.StatusForbidden {
			t.Errorf("Expected 403 Forbidden from Nginx /metrics, got %d", nginxMetricsResp.StatusCode)
		}
	}

	// 3. Verify GetSystemHealth, GetSystemMetrics, and GetOAuthConfig RPCs on server-ops-api
	operatorClient := newOpsClient("qs_op_sandbox_token_12345")

	healthRes, err := operatorClient.GetSystemHealth(ctx, connect.NewRequest(&pb.GetSystemHealthRequest{}))
	if err != nil {
		t.Fatalf("GetSystemHealth failed: %v", err)
	}
	if len(healthRes.Msg.GetServices()) < 5 {
		t.Errorf("Expected at least 5 services in GetSystemHealth, got %d", len(healthRes.Msg.GetServices()))
	}
	for _, sh := range healthRes.Msg.GetServices() {
		if sh.GetServiceName() == "server-ops-api" && sh.GetStatus() != "HEALTHY" {
			t.Errorf("Expected server-ops-api to be HEALTHY in GetSystemHealth, got %q", sh.GetStatus())
		}
	}

	win := "1h"
	metricsRes, err := operatorClient.GetSystemMetrics(ctx, connect.NewRequest(&pb.GetSystemMetricsRequest{
		Window: &win,
	}))
	if err != nil {
		t.Fatalf("GetSystemMetrics failed: %v", err)
	}
	if metricsRes.Msg.GetWindow() != "1h" {
		t.Errorf("Expected window '1h', got %q", metricsRes.Msg.GetWindow())
	}

	oauthRes, err := operatorClient.GetOAuthConfig(ctx, connect.NewRequest(&pb.GetOAuthConfigRequest{}))
	if err != nil {
		t.Fatalf("GetOAuthConfig failed: %v", err)
	}
	if oauthRes.Msg == nil {
		t.Errorf("Expected non-nil GetOAuthConfigResponse")
	}

	// 4. Verify trace_id and region are recorded on InternalApiCallRecord and JobRecord
	otpRes, err := operatorClient.CleanupExpiredOtps(ctx, connect.NewRequest(&pb.CleanupExpiredOtpsRequest{}))
	if err != nil {
		t.Fatalf("CleanupExpiredOtps failed: %v", err)
	}
	jobID := otpRes.Header().Get("X-Job-Id")
	jobRes, err := operatorClient.GetJob(ctx, connect.NewRequest(&pb.GetJobRequest{JobId: jobID}))
	if err != nil {
		t.Fatalf("GetJob(%s) failed: %v", jobID, err)
	}
	if jobRes.Msg.GetJob().GetTraceId() == "" || jobRes.Msg.GetJob().GetRegion() == "" {
		t.Errorf("Expected non-empty trace_id and region on JobRecord, got trace_id=%q region=%q",
			jobRes.Msg.GetJob().GetTraceId(), jobRes.Msg.GetJob().GetRegion())
	}

	time.Sleep(50 * time.Millisecond)
	callsRes, err := operatorClient.ListInternalApiCalls(ctx, connect.NewRequest(&pb.ListInternalApiCallsRequest{
		MethodFilter: "CleanupExpiredOtps",
		Limit:        5,
	}))
	if err != nil || len(callsRes.Msg.GetCalls()) == 0 {
		t.Fatalf("ListInternalApiCalls(CleanupExpiredOtps) failed: %v", err)
	}
	latestCall := callsRes.Msg.GetCalls()[0]
	if latestCall.GetTraceId() == "" || latestCall.GetRegion() == "" {
		t.Errorf("Expected non-empty trace_id and region on InternalApiCallRecord, got trace_id=%q region=%q",
			latestCall.GetTraceId(), latestCall.GetRegion())
	}

	// 5. Verify qsops system health and qsops system metrics CLI commands
	qsopsPath := resolveQSOpsPath(t)
	cmdHealth := exec.Command(qsopsPath, "system", "health", "--token", "qs_op_sandbox_token_12345")
	outHealth, err := cmdHealth.CombinedOutput()
	if err != nil {
		t.Fatalf("qsops system health failed: %v, output: %s", err, string(outHealth))
	}
	if !containsString(string(outHealth), "server-ops-api") || !containsString(string(outHealth), "HEALTHY") {
		t.Errorf("Expected 'server-ops-api' and 'HEALTHY' in qsops system health output: %s", string(outHealth))
	}

	cmdMetrics := exec.Command(qsopsPath, "system", "metrics", "--window", "1h", "--token", "qs_op_sandbox_token_12345")
	outMetrics, err := cmdMetrics.CombinedOutput()
	if err != nil {
		t.Fatalf("qsops system metrics failed: %v, output: %s", err, string(outMetrics))
	}
	if !containsString(string(outMetrics), "SERVICE") || !containsString(string(outMetrics), "AVAILABILITY") {
		t.Errorf("Expected 'SERVICE' and 'AVAILABILITY' in qsops system metrics output: %s", string(outMetrics))
	}

	// 6. Verify GetTrace RPC and qsops system trace CLI command for correlated trace_id
	traceID := latestCall.GetTraceId()
	traceRes, err := operatorClient.GetTrace(ctx, connect.NewRequest(&pb.GetTraceRequest{
		TraceId: traceID,
	}))
	if err != nil {
		t.Fatalf("GetTrace(%s) failed: %v", traceID, err)
	}
	if traceRes.Msg.GetTraceId() != traceID {
		t.Errorf("Expected trace_id %q, got %q", traceID, traceRes.Msg.GetTraceId())
	}
	if len(traceRes.Msg.GetSpans()) == 0 {
		t.Errorf("Expected at least 1 span in GetTrace(%s), got 0", traceID)
	}
	if len(traceRes.Msg.GetLogs()) == 0 {
		t.Errorf("Expected at least 1 log entry in GetTrace(%s), got 0", traceID)
	}
	foundDebugLog := false
	for _, l := range traceRes.Msg.GetLogs() {
		if l.GetLevel() == "DEBUG" {
			foundDebugLog = true
			break
		}
	}
	if !foundDebugLog {
		t.Errorf("Expected at least 1 DEBUG log entry in GetTrace(%s), got logs: %+v", traceID, traceRes.Msg.GetLogs())
	}

	cmdTrace := exec.Command(qsopsPath, "system", "trace", traceID, "--token", "qs_op_sandbox_token_12345")
	outTrace, err := cmdTrace.CombinedOutput()
	if err != nil {
		t.Fatalf("qsops system trace %s failed: %v, output: %s", traceID, err, string(outTrace))
	}
	if !containsString(string(outTrace), traceID) || !containsString(string(outTrace), "CleanupExpiredOtps") {
		t.Errorf("Expected trace_id %q and 'CleanupExpiredOtps' in qsops system trace output: %s", traceID, string(outTrace))
	}
}

func TestOpsWeb_GrafanaProxyJITGating(t *testing.T) {
	ctx := context.Background()
	opsWebURL := "http://127.0.0.1:8083"
	client := &http.Client{Timeout: 5 * time.Second}
	operatorClient := newOpsClient("qs_op_sandbox_token_12345")

	// Revoke any pre-existing active JIT grants on sandbox operator
	if grantsRes, err := operatorClient.ListJitGrants(ctx, connect.NewRequest(&pb.ListJitGrantsRequest{ActiveOnly: true})); err == nil {
		for _, g := range grantsRes.Msg.GetGrants() {
			_, _ = operatorClient.RevokeJitGrant(ctx, connect.NewRequest(&pb.RevokeJitGrantRequest{GrantId: g.GetId()}))
		}
	}

	// 1. Unauthenticated /grafana/* -> 403 Forbidden
	resp1, err := client.Get(opsWebURL + "/grafana/api/health")
	if err != nil {
		t.Fatalf("GET /grafana/api/health failed: %v", err)
	}
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden for unauthenticated /grafana/*, got %d", resp1.StatusCode)
	}

	// 2. Authenticated operator WITHOUT active JIT -> 403 Forbidden
	req2, _ := http.NewRequestWithContext(ctx, http.MethodGet, opsWebURL+"/grafana/api/health", nil)
	req2.Header.Set("Authorization", "Bearer qs_op_sandbox_token_12345")
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("Authenticated GET /grafana/api/health failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden without active JIT grant, got %d (%s)", resp2.StatusCode, string(body2))
	}

	// 3. Acquire JIT grant -> /grafana/* is allowed past the JIT gate (not 401/403)
	grantRes, err := operatorClient.AcquireJitGrant(ctx, connect.NewRequest(&pb.AcquireJitGrantRequest{
		Hours:     1,
		Reason:    "E2E test Grafana JIT proxy gating",
		Operation: "*",
	}))
	if err != nil {
		t.Fatalf("AcquireJitGrant failed: %v", err)
	}
	grantID := grantRes.Msg.GetGrant().GetId()
	defer func() {
		_, _ = operatorClient.RevokeJitGrant(ctx, connect.NewRequest(&pb.RevokeJitGrantRequest{GrantId: grantID}))
	}()

	req3, _ := http.NewRequestWithContext(ctx, http.MethodGet, opsWebURL+"/grafana/api/health", nil)
	req3.Header.Set("Authorization", "Bearer qs_op_sandbox_token_12345")
	resp3, err := client.Do(req3)
	if err != nil {
		t.Fatalf("JIT-elevated GET /grafana/api/health failed: %v", err)
	}
	resp3.Body.Close()
	if resp3.StatusCode == http.StatusUnauthorized || resp3.StatusCode == http.StatusForbidden {
		t.Errorf("Expected JIT-elevated request to pass auth/JIT gate, got %d", resp3.StatusCode)
	}

	// 4. Verify /services/<service>/healthz and /services/<service>/metrics proxy on server-ops-web
	reqCronHealth, _ := http.NewRequestWithContext(ctx, http.MethodGet, opsWebURL+"/services/server-cron/healthz", nil)
	reqCronHealth.Header.Set("Authorization", "Bearer qs_op_sandbox_token_12345")
	respCronHealth, err := client.Do(reqCronHealth)
	if err != nil {
		t.Fatalf("GET /services/server-cron/healthz failed: %v", err)
	}
	cronHealthBody, _ := io.ReadAll(respCronHealth.Body)
	respCronHealth.Body.Close()
	if respCronHealth.StatusCode != http.StatusOK || !strings.Contains(string(cronHealthBody), `"service":"server-cron"`) {
		t.Errorf("Expected 200 OK from /services/server-cron/healthz, got %d: %s", respCronHealth.StatusCode, string(cronHealthBody))
	}

	reqCronMetrics, _ := http.NewRequestWithContext(ctx, http.MethodGet, opsWebURL+"/services/server-cron/metrics", nil)
	reqCronMetrics.Header.Set("Authorization", "Bearer qs_op_sandbox_token_12345")
	respCronMetrics, err := client.Do(reqCronMetrics)
	if err != nil {
		t.Fatalf("GET /services/server-cron/metrics failed: %v", err)
	}
	cronMetricsBody, _ := io.ReadAll(respCronMetrics.Body)
	respCronMetrics.Body.Close()
	if respCronMetrics.StatusCode != http.StatusOK || !strings.Contains(string(cronMetricsBody), "quadsmith_health_status") {
		t.Errorf("Expected 200 OK from /services/server-cron/metrics, got %d: %s", respCronMetrics.StatusCode, string(cronMetricsBody))
	}
}

func TestOpsService_CELFilterAndSorting(t *testing.T) {
	ctx := context.Background()
	operatorClient := newOpsClient("qs_op_sandbox_token_12345")

	// 1. ListTasks with CEL filter & descending sort
	taskFilter := `name.startsWith("cleanup_")`
	tasksRes, err := operatorClient.ListTasks(ctx, connect.NewRequest(&pb.ListTasksRequest{
		Filter: &taskFilter,
		Sort:   []string{"^name"},
	}))
	if err != nil {
		t.Fatalf("ListTasks with CEL filter failed: %v", err)
	}
	if len(tasksRes.Msg.GetTasks()) != 3 {
		t.Fatalf("Expected 3 cleanup tasks, got %d", len(tasksRes.Msg.GetTasks()))
	}
	if tasksRes.Msg.GetTasks()[0].GetName() < tasksRes.Msg.GetTasks()[1].GetName() {
		t.Errorf("Expected descending sort order on tasks, got %s before %s",
			tasksRes.Msg.GetTasks()[0].GetName(), tasksRes.Msg.GetTasks()[1].GetName())
	}

	// 2. GetDatabaseStats with CEL filter & ascending table_name sort
	dbFilter := `table_name.contains("motor") || table_name == "frames"`
	dbRes, err := operatorClient.GetDatabaseStats(ctx, connect.NewRequest(&pb.GetDatabaseStatsRequest{
		Filter: &dbFilter,
		Sort:   []string{"table_name"},
	}))
	if err != nil {
		t.Fatalf("GetDatabaseStats with CEL filter failed: %v", err)
	}
	if len(dbRes.Msg.GetTables()) < 2 {
		t.Fatalf("Expected at least 2 tables matching motor/frames filter, got %d", len(dbRes.Msg.GetTables()))
	}
	if dbRes.Msg.GetTables()[0].GetTableName() > dbRes.Msg.GetTables()[1].GetTableName() {
		t.Errorf("Expected ascending table_name sort, got %s before %s",
			dbRes.Msg.GetTables()[0].GetTableName(), dbRes.Msg.GetTables()[1].GetTableName())
	}

	// 3. Invalid CEL filter returns CodeInvalidArgument
	invalidFilter := `unknown_column == 123`
	_, err = operatorClient.GetDatabaseStats(ctx, connect.NewRequest(&pb.GetDatabaseStatsRequest{
		Filter: &invalidFilter,
	}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("Expected CodeInvalidArgument for invalid CEL filter, got %v", err)
	}

	// 4. Verify qsops CLI --filter and --sort flags
	qsopsPath := resolveQSOpsPath(t)
	cmdTasks := exec.Command(
		qsopsPath, "tasks", "list",
		"-f", `name.startsWith("cleanup_")`,
		"-s", "^name",
		"--token", "qs_op_sandbox_token_12345",
	)
	outTasks, err := cmdTasks.CombinedOutput()
	if err != nil {
		t.Fatalf("qsops tasks list -f -s failed: %v, output: %s", err, string(outTasks))
	}
	if !containsString(string(outTasks), "cleanup_orphaned_blobs") || containsString(string(outTasks), "set_user_role") {
		t.Errorf("Unexpected qsops tasks list filtered output: %s", string(outTasks))
	}
}

func TestOpsServer_LoginAsQuadsmithAndSystemAccountProtections(t *testing.T) {
	ctx := context.Background()
	pubPool, err := pgxpool.New(ctx, getDBURL())
	if err != nil {
		t.Fatalf("Failed to connect to quadsmith_api DB: %v", err)
	}
	defer pubPool.Close()

	// 1. Verify @quadsmith system user exists with role = 'SYSTEM' and DB CHECK constraints hold
	var sysUUID, sysRole string
	if err := pubPool.QueryRow(ctx, `SELECT uuid::text, role FROM users WHERE id = 'quadsmith'`).Scan(&sysUUID, &sysRole); err != nil {
		t.Fatalf("Expected @quadsmith system user to exist in quadsmith_api.users: %v", err)
	}
	if sysUUID != auth.SystemUserUUID || sysRole != "SYSTEM" {
		t.Fatalf("Expected @quadsmith uuid=%s role=SYSTEM, got uuid=%s role=%s", auth.SystemUserUUID, sysUUID, sysRole)
	}

	// PostgreSQL CHECK constraint must reject changing @quadsmith role or soft-deleting @quadsmith
	if _, err := pubPool.Exec(ctx, `UPDATE users SET role = 'ADMIN' WHERE id = 'quadsmith'`); err == nil {
		t.Fatalf("Expected PostgreSQL CHECK constraint to reject changing @quadsmith role to ADMIN")
	}
	if _, err := pubPool.Exec(ctx, `UPDATE users SET deleted_at = NOW() WHERE id = 'quadsmith'`); err == nil {
		t.Fatalf("Expected PostgreSQL CHECK constraint to reject soft-deleting @quadsmith")
	}

	operatorClient := newOpsClient("qs_op_sandbox_token_12345")
	robotClient := newOpsClient("qs_robot_sandbox_token_67890")

	// Revoke any pre-existing active JIT grants on sandbox operator
	if grantsRes, err := operatorClient.ListJitGrants(ctx, connect.NewRequest(&pb.ListJitGrantsRequest{ActiveOnly: true})); err == nil {
		for _, g := range grantsRes.Msg.GetGrants() {
			_, _ = operatorClient.RevokeJitGrant(ctx, connect.NewRequest(&pb.RevokeJitGrantRequest{GrantId: g.GetId()}))
		}
	}

	// 2. Without JIT grant -> PermissionDenied; with ROBOT token -> PermissionDenied
	if _, err := operatorClient.LoginAsQuadsmith(ctx, connect.NewRequest(&pb.LoginAsQuadsmithRequest{})); err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("Expected CodePermissionDenied when calling LoginAsQuadsmith without JIT grant, got %v", err)
	}
	if _, err := robotClient.LoginAsQuadsmith(ctx, connect.NewRequest(&pb.LoginAsQuadsmithRequest{})); err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("Expected CodePermissionDenied when calling LoginAsQuadsmith with ROBOT token, got %v", err)
	}

	// 3. Acquire JIT grant for HUMAN operator
	grantRes, err := operatorClient.AcquireJitGrant(ctx, connect.NewRequest(&pb.AcquireJitGrantRequest{
		Hours:     1,
		Reason:    "E2E test for LoginAsQuadsmith",
		Operation: "*",
	}))
	if err != nil {
		t.Fatalf("AcquireJitGrant failed: %v", err)
	}
	grantID := grantRes.Msg.GetGrant().GetId()
	defer func() {
		_, _ = operatorClient.RevokeJitGrant(ctx, connect.NewRequest(&pb.RevokeJitGrantRequest{GrantId: grantID}))
	}()

	// 4. Call LoginAsQuadsmith via OpsService
	loginRes, err := operatorClient.LoginAsQuadsmith(ctx, connect.NewRequest(&pb.LoginAsQuadsmithRequest{}))
	if err != nil {
		t.Fatalf("LoginAsQuadsmith failed: %v", err)
	}
	if loginRes.Msg.GetUsername() != "quadsmith" || loginRes.Msg.GetUserUuid() != auth.SystemUserUUID || loginRes.Msg.GetRole() != "SYSTEM" {
		t.Fatalf("Unexpected LoginAsQuadsmith user fields: username=%q uuid=%q role=%q",
			loginRes.Msg.GetUsername(), loginRes.Msg.GetUserUuid(), loginRes.Msg.GetRole())
	}
	if loginRes.Msg.GetAccessToken() == "" || loginRes.Msg.GetRefreshToken() == "" || loginRes.Msg.GetExpiresAt() == nil {
		t.Fatalf("Expected non-empty access_token, refresh_token, and expires_at from LoginAsQuadsmith")
	}

	// 5. Authenticate to public AuthService & UserService as @quadsmith
	qsHTTPClient := &http.Client{
		Transport: &authRoundTripper{base: http.DefaultTransport, token: loginRes.Msg.GetAccessToken()},
		Timeout:   5 * time.Second,
	}
	qsAuthClient := apiconnect.NewAuthServiceClient(qsHTTPClient, getAPIURL())
	qsUserClient := apiconnect.NewUserServiceClient(qsHTTPClient, getAPIURL())

	meRes, err := qsAuthClient.GetCurrentUser(ctx, connect.NewRequest(&apipb.GetCurrentUserRequest{}))
	if err != nil {
		t.Fatalf("GetCurrentUser as @quadsmith failed: %v", err)
	}
	if meRes.Msg.GetId() != "quadsmith" || meRes.Msg.GetRole() != "SYSTEM" {
		t.Errorf("Expected @quadsmith role=SYSTEM in GetCurrentUser, got id=%q role=%q",
			meRes.Msg.GetId(), meRes.Msg.GetRole())
	}

	// Verify @quadsmith has full admin privileges (e.g., ListBans)
	if _, err := qsUserClient.ListBans(ctx, connect.NewRequest(&apipb.ListBansRequest{})); err != nil {
		t.Fatalf("Expected @quadsmith SYSTEM role to have admin access to ListBans: %v", err)
	}

	// Verify @quadsmith cannot rename handle, delete account, or be banned/role-modified
	if _, err := qsAuthClient.ChangeUsername(ctx, connect.NewRequest(&apipb.ChangeUsernameRequest{NewUsername: "not_quadsmith"})); err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Errorf("Expected ChangeUsername to be rejected with CodePermissionDenied for @quadsmith, got %v", err)
	}
	if _, err := qsAuthClient.DeleteAccount(ctx, connect.NewRequest(&apipb.DeleteAccountRequest{Username: "quadsmith"})); err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Errorf("Expected DeleteAccount to be rejected with CodePermissionDenied for @quadsmith, got %v", err)
	}
	if _, err := qsUserClient.BanUser(ctx, connect.NewRequest(&apipb.BanUserRequest{UserId: "quadsmith", Reason: "attempting to ban system"})); err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Errorf("Expected BanUser targeting @quadsmith to be rejected with CodePermissionDenied, got %v", err)
	}
	if _, err := operatorClient.SetUserRole(ctx, connect.NewRequest(&pb.SetUserRoleRequest{UserId: "quadsmith", Role: "USER"})); err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Errorf("Expected SetUserRole targeting @quadsmith to be rejected with CodePermissionDenied, got %v", err)
	}

	// Verify RefreshToken works with the minted refresh_token (used by app-web ops_refresh_token handoff)
	// and preserves the fixed JIT grant expires_at ceiling instead of sliding out to 5 days.
	refRes, err := qsAuthClient.RefreshToken(ctx, connect.NewRequest(&apipb.RefreshTokenRequest{
		RefreshToken: loginRes.Msg.GetRefreshToken(),
	}))
	if err != nil {
		t.Fatalf("RefreshToken for @quadsmith failed: %v", err)
	}
	if refRes.Msg.GetUser().GetId() != "quadsmith" || refRes.Msg.GetUser().GetRole() != "SYSTEM" || refRes.Msg.GetAccessToken() == "" {
		t.Errorf("Unexpected RefreshToken response for @quadsmith: %+v", refRes.Msg)
	}
	op1AccessToken := refRes.Msg.GetAccessToken()
	op1RefreshToken := loginRes.Msg.GetRefreshToken()
	if refRes.Msg.GetRefreshToken() != "" {
		op1RefreshToken = refRes.Msg.GetRefreshToken()
	}
	qsHTTPClient = &http.Client{
		Transport: &authRoundTripper{base: http.DefaultTransport, token: op1AccessToken},
		Timeout:   5 * time.Second,
	}
	qsAuthClient = apiconnect.NewAuthServiceClient(qsHTTPClient, getAPIURL())

	var op1SessionExpiresAt time.Time
	var op1JitGrantUUID string
	if err := pubPool.QueryRow(ctx, `
		SELECT expires_at, COALESCE(jit_grant_uuid::text, '')
		FROM refresh_tokens
		WHERE token_hash = encode(sha256($1::bytea), 'hex')
	`, op1RefreshToken).Scan(&op1SessionExpiresAt, &op1JitGrantUUID); err != nil {
		t.Fatalf("Failed to query operator1 @quadsmith refresh_tokens row: %v", err)
	}
	if op1JitGrantUUID == "" {
		t.Fatalf("Expected non-empty jit_grant_uuid on @quadsmith refresh_tokens row")
	}
	if time.Until(op1SessionExpiresAt) > 2*time.Hour {
		t.Fatalf("Expected @quadsmith session expires_at to be capped at 1h JIT grant (~1h), got %v", time.Until(op1SessionExpiresAt))
	}

	// 6. Verify qsops login-quadsmith CLI command writes valid qs config (creates a 2nd session under grant1)
	qsopsPath := resolveQSOpsPath(t)
	tmpConfig := filepath.Join(t.TempDir(), "config.json")
	cmd := exec.Command(qsopsPath, "login-quadsmith", "--qs-config", tmpConfig, "--token", "qs_op_sandbox_token_12345")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("qsops login-quadsmith failed: %v, output: %s", err, string(out))
	}
	if !containsString(string(out), "Authenticated public 'qs' CLI as @quadsmith (role: SYSTEM)") {
		t.Errorf("Unexpected qsops login-quadsmith output: %s", string(out))
	}
	cfgBytes, err := os.ReadFile(tmpConfig)
	if err != nil {
		t.Fatalf("Failed to read written qs config file: %v", err)
	}
	if !containsString(string(cfgBytes), `"user_id": "quadsmith"`) || !containsString(string(cfgBytes), `"access_token":`) {
		t.Errorf("Expected user_id=quadsmith and access_token in written config.json, got: %s", string(cfgBytes))
	}

	// 7. Create a second HUMAN operator token (operator2) to verify parallel independent @quadsmith session lifecycles
	op2TokenRes, err := operatorClient.CreateInternalToken(ctx, connect.NewRequest(&pb.CreateInternalTokenRequest{
		Name:         "parallel-op2-" + time.Now().Format("150405.000"),
		Type:         "HUMAN",
		GroupName:    "operators",
		DurationDays: 1,
	}))
	if err != nil {
		t.Fatalf("CreateInternalToken for operator2 failed: %v", err)
	}
	op2TokenID := op2TokenRes.Msg.GetTokenId()
	op2RawToken := op2TokenRes.Msg.GetTokenSecret()
	defer func() {
		_, _ = operatorClient.RevokeInternalToken(ctx, connect.NewRequest(&pb.RevokeInternalTokenRequest{
			Id: op2TokenID,
		}))
	}()

	operator2Client := newOpsClient(op2RawToken)
	grant2Res, err := operator2Client.AcquireJitGrant(ctx, connect.NewRequest(&pb.AcquireJitGrantRequest{
		Hours:     1,
		Reason:    "Parallel operator2 LoginAsQuadsmith test",
		Operation: "*",
	}))
	if err != nil {
		t.Fatalf("AcquireJitGrant for operator2 failed: %v", err)
	}
	grant2ID := grant2Res.Msg.GetGrant().GetId()
	defer func() {
		_, _ = operator2Client.RevokeJitGrant(ctx, connect.NewRequest(&pb.RevokeJitGrantRequest{GrantId: grant2ID}))
	}()

	login2Res, err := operator2Client.LoginAsQuadsmith(ctx, connect.NewRequest(&pb.LoginAsQuadsmithRequest{}))
	if err != nil {
		t.Fatalf("LoginAsQuadsmith for operator2 failed: %v", err)
	}
	op2AccessToken := login2Res.Msg.GetAccessToken()
	op2RefreshToken := login2Res.Msg.GetRefreshToken()
	qs2HTTPClient := &http.Client{
		Transport: &authRoundTripper{base: http.DefaultTransport, token: op2AccessToken},
		Timeout:   5 * time.Second,
	}
	qs2AuthClient := apiconnect.NewAuthServiceClient(qs2HTTPClient, getAPIURL())

	var op2SessionExpiresAt time.Time
	var op2JitGrantUUID string
	if err := pubPool.QueryRow(ctx, `
		SELECT expires_at, COALESCE(jit_grant_uuid::text, '')
		FROM refresh_tokens
		WHERE token_hash = encode(sha256($1::bytea), 'hex')
	`, op2RefreshToken).Scan(&op2SessionExpiresAt, &op2JitGrantUUID); err != nil {
		t.Fatalf("Failed to query operator2 @quadsmith refresh_tokens row: %v", err)
	}
	if op2JitGrantUUID == "" || op2JitGrantUUID == op1JitGrantUUID {
		t.Fatalf("Expected distinct jit_grant_uuid for operator2, got op1=%s op2=%s", op1JitGrantUUID, op2JitGrantUUID)
	}

	// 8. Verify ExtendJitGrant on grant1 extends operator1's @quadsmith session without changing operator2's session
	if _, err := operatorClient.ExtendJitGrant(ctx, connect.NewRequest(&pb.ExtendJitGrantRequest{
		GrantId: grantID,
		Hours:   1,
	})); err != nil {
		t.Fatalf("ExtendJitGrant for operator1 failed: %v", err)
	}
	var op1ExtendedExpiresAt, op2UnchangedExpiresAt time.Time
	if err := pubPool.QueryRow(ctx, `SELECT expires_at FROM refresh_tokens WHERE token_hash = encode(sha256($1::bytea), 'hex')`, op1RefreshToken).Scan(&op1ExtendedExpiresAt); err != nil {
		t.Fatalf("Failed to query extended operator1 session: %v", err)
	}
	if err := pubPool.QueryRow(ctx, `SELECT expires_at FROM refresh_tokens WHERE token_hash = encode(sha256($1::bytea), 'hex')`, op2RefreshToken).Scan(&op2UnchangedExpiresAt); err != nil {
		t.Fatalf("Failed to query operator2 session after operator1 extension: %v", err)
	}
	if !op1ExtendedExpiresAt.After(op1SessionExpiresAt.Add(30 * time.Minute)) {
		t.Errorf("Expected operator1 @quadsmith session expires_at to be extended by ~1h, before=%v after=%v", op1SessionExpiresAt, op1ExtendedExpiresAt)
	}
	if !op2UnchangedExpiresAt.Equal(op2SessionExpiresAt) {
		t.Errorf("Expected operator2 @quadsmith session expires_at to remain unchanged (%v), got %v", op2SessionExpiresAt, op2UnchangedExpiresAt)
	}

	// 9. Verify ListActiveSessions and RevokeOtherSessions isolate @quadsmith sessions by jit_grant_uuid
	op1SessionsRes, err := qsAuthClient.ListActiveSessions(ctx, connect.NewRequest(&apipb.ListActiveSessionsRequest{}))
	if err != nil {
		t.Fatalf("ListActiveSessions for operator1 failed: %v", err)
	}
	if len(op1SessionsRes.Msg.GetSessions()) != 2 {
		t.Fatalf("Expected operator1 to see exactly 2 sessions under grant1 (RPC + CLI), got %d", len(op1SessionsRes.Msg.GetSessions()))
	}
	revOtherRes, err := qsAuthClient.RevokeOtherSessions(ctx, connect.NewRequest(&apipb.RevokeOtherSessionsRequest{}))
	if err != nil {
		t.Fatalf("RevokeOtherSessions for operator1 failed: %v", err)
	}
	if revOtherRes.Msg.GetRevokedCount() != 1 {
		t.Errorf("Expected RevokeOtherSessions to revoke 1 sibling session under grant1, got %d", revOtherRes.Msg.GetRevokedCount())
	}
	// Operator2's session must still be active
	if _, err := qs2AuthClient.GetCurrentUser(ctx, connect.NewRequest(&apipb.GetCurrentUserRequest{})); err != nil {
		t.Fatalf("Expected operator2's @quadsmith session to remain active after operator1 RevokeOtherSessions: %v", err)
	}

	// 10. Verify RevokeJitGrant on grant1 immediately logs out operator1's @quadsmith session while operator2 stays logged in
	if _, err := operatorClient.RevokeJitGrant(ctx, connect.NewRequest(&pb.RevokeJitGrantRequest{GrantId: grantID})); err != nil {
		t.Fatalf("RevokeJitGrant for operator1 failed: %v", err)
	}
	if _, err := qsAuthClient.GetCurrentUser(ctx, connect.NewRequest(&apipb.GetCurrentUserRequest{})); err == nil || connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("Expected operator1 GetCurrentUser as @quadsmith to fail with CodeUnauthenticated immediately after JIT revocation, got %v", err)
	}
	if _, err := qsAuthClient.RefreshToken(ctx, connect.NewRequest(&apipb.RefreshTokenRequest{RefreshToken: op1RefreshToken})); err == nil || connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("Expected operator1 RefreshToken as @quadsmith to fail with CodeUnauthenticated immediately after JIT revocation, got %v", err)
	}

	// Meanwhile, operator2's @quadsmith session (access token + refresh token) must still succeed!
	if _, err := qs2AuthClient.GetCurrentUser(ctx, connect.NewRequest(&apipb.GetCurrentUserRequest{})); err != nil {
		t.Fatalf("Expected operator2 GetCurrentUser as @quadsmith to remain authenticated after operator1 JIT revocation: %v", err)
	}
	ref2Res, err := qs2AuthClient.RefreshToken(ctx, connect.NewRequest(&apipb.RefreshTokenRequest{RefreshToken: op2RefreshToken}))
	if err != nil {
		t.Fatalf("Expected operator2 RefreshToken as @quadsmith to remain valid after operator1 JIT revocation: %v", err)
	}
	if ref2Res.Msg.GetRefreshToken() != "" {
		op2RefreshToken = ref2Res.Msg.GetRefreshToken()
	}

	// 11. Verify RevokeInternalToken on operator2 immediately revokes grant2 and logs out operator2's @quadsmith session
	if _, err := operator2Client.RevokeInternalToken(ctx, connect.NewRequest(&pb.RevokeInternalTokenRequest{
		Id: op2TokenID,
	})); err != nil {
		t.Fatalf("RevokeInternalToken for operator2 failed: %v", err)
	}
	if _, err := qs2AuthClient.GetCurrentUser(ctx, connect.NewRequest(&apipb.GetCurrentUserRequest{})); err == nil || connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("Expected operator2 GetCurrentUser as @quadsmith to fail with CodeUnauthenticated after internal token revocation, got %v", err)
	}
	if _, err := qs2AuthClient.RefreshToken(ctx, connect.NewRequest(&apipb.RefreshTokenRequest{RefreshToken: op2RefreshToken})); err == nil || connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("Expected operator2 RefreshToken as @quadsmith to fail with CodeUnauthenticated after internal token revocation, got %v", err)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
