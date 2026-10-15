package api

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"

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

	// 1. RunDatabaseMaintenance is a sensitive method requiring JIT elevation
	_, err := operatorClient.RunDatabaseMaintenance(ctx, connect.NewRequest(&pb.RunDatabaseMaintenanceRequest{
		FullVacuum: false,
	}))
	if err == nil {
		t.Fatalf("Expected RunDatabaseMaintenance to fail without active JIT elevation")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Errorf("Expected CodePermissionDenied for JIT-required method, got %v: %v", connect.CodeOf(err), err)
	}

	// 2. Acquire JIT Elevation (auto-approved in sandbox/test)
	grantRes, err := operatorClient.AcquireJitGrant(ctx, connect.NewRequest(&pb.AcquireJitGrantRequest{
		Hours:     2,
		Reason:    "E2E Integration Test Database Maintenance",
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

	// 4. Now RunDatabaseMaintenance succeeds with active JIT!
	maintRes, err := operatorClient.RunDatabaseMaintenance(ctx, connect.NewRequest(&pb.RunDatabaseMaintenanceRequest{
		FullVacuum: false,
	}))
	if err != nil {
		t.Fatalf("RunDatabaseMaintenance failed despite active JIT: %v", err)
	}
	if !maintRes.Msg.Success {
		t.Errorf("Expected maintenance success: %s", maintRes.Msg.Message)
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
	_, err = operatorClient.RunDatabaseMaintenance(ctx, connect.NewRequest(&pb.RunDatabaseMaintenanceRequest{
		FullVacuum: false,
	}))
	if err == nil {
		t.Fatalf("Expected RunDatabaseMaintenance to fail after JIT revocation")
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

func TestOpsServer_ScheduledJobsAndAudit(t *testing.T) {
	ctx := context.Background()
	operatorClient := newOpsClient("qs_op_sandbox_token_12345")

	// 1. List Scheduled Jobs
	jobsRes, err := operatorClient.ListScheduledJobs(ctx, connect.NewRequest(&pb.ListScheduledJobsRequest{}))
	if err != nil {
		t.Fatalf("ListScheduledJobs failed: %v", err)
	}

	expectedJobs := []string{
		"cleanup_expired_otps",
		"cleanup_expired_tokens",
		"cleanup_orphaned_blobs",
	}

	for _, exp := range expectedJobs {
		found := false
		for _, j := range jobsRes.Msg.Jobs {
			if j.Name == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected job %q in scheduled jobs list", exp)
		}
	}

	// 2. Trigger Job
	trigRes, err := operatorClient.TriggerJob(ctx, connect.NewRequest(&pb.TriggerJobRequest{
		JobName:     "cleanup_expired_otps",
		PayloadJson: "{}",
	}))
	if err != nil {
		t.Fatalf("TriggerJob failed: %v", err)
	}
	if !trigRes.Msg.Success {
		t.Errorf("Expected TriggerJob to report success: %s", trigRes.Msg.Message)
	}

	// 3. Verify Audit Log recorded in internal_api_calls
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

	// 3. qsops jobs list
	cmd = exec.Command(qsopsPath, "jobs", "list", "--token", "qs_op_sandbox_token_12345", "--json")
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("qsops jobs list failed: %v, output: %s", err, string(out))
	}
	outStr = string(out)
	if !containsString(outStr, "cleanup_expired_otps") {
		t.Errorf("Expected 'cleanup_expired_otps' in jobs list output: %s", outStr)
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
