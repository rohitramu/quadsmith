package ops

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"quadsmith/shared/pkg/opsauth"

	pb "quadsmith/shared/gen/quadsmith/ops"
)

func TestSetUserRole_Validation(t *testing.T) {
	svc := &OpsService{}
	ctx := context.Background()

	// 1. Unauthenticated (no operator context)
	_, err := svc.SetUserRole(ctx, connect.NewRequest(&pb.SetUserRoleRequest{
		UserId: "test_user",
		Role:   "ADMIN",
	}))
	if err == nil {
		t.Fatal("expected unauthenticated error, got nil")
	}
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", connect.CodeOf(err))
	}

	// 2. Non-HUMAN operator (ROBOT) rejected
	robotCtx := opsauth.WithOperatorContext(ctx, &opsauth.OperatorContext{
		Operator:     "test-robot",
		OperatorType: "ROBOT",
		GroupName:    "robots",
	})
	_, err = svc.SetUserRole(robotCtx, connect.NewRequest(&pb.SetUserRoleRequest{
		UserId: "test_user",
		Role:   "ADMIN",
	}))
	if err == nil {
		t.Fatal("expected permission denied error for ROBOT operator, got nil")
	}
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("expected CodePermissionDenied, got %v", connect.CodeOf(err))
	}

	// HUMAN operator context
	humanCtx := opsauth.WithOperatorContext(ctx, &opsauth.OperatorContext{
		Operator:     "test-human",
		OperatorType: "HUMAN",
		GroupName:    "operators",
	})

	// 3. Missing user ID
	_, err = svc.SetUserRole(humanCtx, connect.NewRequest(&pb.SetUserRoleRequest{
		UserId: "   ",
		Role:   "ADMIN",
	}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument for empty user_id, got %v", err)
	}

	// 4. Invalid role
	for _, badRole := range []string{"", "SUPERADMIN", "MODERATOR", "admin123"} {
		_, err = svc.SetUserRole(humanCtx, connect.NewRequest(&pb.SetUserRoleRequest{
			UserId: "test_user",
			Role:   badRole,
		}))
		if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("expected CodeInvalidArgument for invalid role %q, got %v", badRole, err)
		}
	}
}

func TestTaskCatalog_SetUserRole(t *testing.T) {
	found := false
	for _, task := range taskCatalog {
		if task.Name == "set_user_role" {
			found = true
			if task.Procedure != "SetUserRole" {
				t.Errorf("expected procedure SetUserRole, got %s", task.Procedure)
			}
			if task.DefaultInterval != "manual" {
				t.Errorf("expected manual interval, got %s", task.DefaultInterval)
			}
		}
	}
	if !found {
		t.Fatal("task 'set_user_role' not found in taskCatalog")
	}

	if proc := taskToProcedure("set_user_role"); proc != "SetUserRole" {
		t.Errorf("expected taskToProcedure(set_user_role) == SetUserRole, got %s", proc)
	}
	if task := procedureToTask("SetUserRole"); task != "set_user_role" {
		t.Errorf("expected procedureToTask(SetUserRole) == set_user_role, got %s", task)
	}
}

func TestListTasks_FilterAndSort(t *testing.T) {
	svc := &OpsService{}
	ctx := context.Background()

	filter := `name.startsWith("cleanup_") && default_interval != "manual"`
	resp, err := svc.ListTasks(ctx, connect.NewRequest(&pb.ListTasksRequest{
		Filter: &filter,
		Sort:   []string{"^name"},
	}))
	if err != nil {
		t.Fatalf("unexpected error from ListTasks: %v", err)
	}
	tasks := resp.Msg.GetTasks()
	if len(tasks) != 3 {
		t.Fatalf("expected 3 cleanup tasks, got %d", len(tasks))
	}
	if tasks[0].GetName() < tasks[1].GetName() {
		t.Errorf("expected descending sort by name, got %s then %s", tasks[0].GetName(), tasks[1].GetName())
	}

	// Invalid CEL expression returns CodeInvalidArgument
	badFilter := `unknown_field == "foo"`
	_, err = svc.ListTasks(ctx, connect.NewRequest(&pb.ListTasksRequest{
		Filter: &badFilter,
	}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument for invalid filter, got %v", err)
	}

	// Invalid sort column returns CodeInvalidArgument
	_, err = svc.ListTasks(ctx, connect.NewRequest(&pb.ListTasksRequest{
		Sort: []string{"non_existent_col"},
	}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument for invalid sort column, got %v", err)
	}
}

func TestFilterAndSortHelpers(t *testing.T) {
	routes := []*pb.RouteMetricStat{
		{ServiceName: "server-api", Route: "/a", TotalRequests: 10, ErrorCount: 0, P95RttMs: 3.5},
		{ServiceName: "server-ops-api", Route: "/b", TotalRequests: 50, ErrorCount: 2, P95RttMs: 18.2},
	}
	filteredRoutes, err := filterAndSortRouteMetrics(routes, `error_count > 0 || p95_rtt_ms > 10.0`, []string{"^total_requests"})
	if err != nil {
		t.Fatalf("unexpected error filtering routes: %v", err)
	}
	if len(filteredRoutes) != 1 || filteredRoutes[0].GetRoute() != "/b" {
		t.Fatalf("unexpected filtered routes: %+v", filteredRoutes)
	}

	tables := []*pb.TableStat{
		{TableName: "motors", RowCount: 25, TotalSizeBytes: 65536},
		{TableName: "empty_tbl", RowCount: 0, TotalSizeBytes: 8192},
	}
	filteredTables, err := filterAndSortTableStats(tables, `row_count > 0`, []string{"table_name"})
	if err != nil {
		t.Fatalf("unexpected error filtering tables: %v", err)
	}
	if len(filteredTables) != 1 || filteredTables[0].GetTableName() != "motors" {
		t.Fatalf("unexpected filtered tables: %+v", filteredTables)
	}
}

func TestGetTrace_Validation(t *testing.T) {
	svc := &OpsService{}
	ctx := context.Background()

	_, err := svc.GetTrace(ctx, connect.NewRequest(&pb.GetTraceRequest{
		TraceId: "   ",
	}))
	if err == nil || connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument for empty trace_id, got %v", err)
	}
}
