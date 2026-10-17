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
