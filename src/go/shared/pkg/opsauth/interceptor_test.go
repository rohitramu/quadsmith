package opsauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	configpb "quadsmith/shared/gen/quadsmith/config"
)

func TestIsTaskMethod_SetUserRole(t *testing.T) {
	if !isTaskMethod("SetUserRole") {
		t.Error("expected isTaskMethod('SetUserRole') to be true")
	}
	if isTaskMethod("GetCurrentOperator") {
		t.Error("expected isTaskMethod('GetCurrentOperator') to be false")
	}
}

func TestOpsInterceptor_SensitiveMethodRequiresHuman(t *testing.T) {
	interceptor := NewOpsInterceptor(
		nil,
		[]*configpb.BootstrapTokenConfig{
			{
				Token:     proto.String("robot-token"),
				Type:      proto.String("ROBOT"),
				GroupName: proto.String("robots"),
			},
		},
		[]*configpb.InternalGroupConfig{
			{
				Name:           proto.String("robots"),
				AllowedMethods: []string{"*"},
			},
		},
		&configpb.JitPolicyConfig{
			Enabled:          proto.Bool(true),
			SensitiveMethods: []string{"SetUserRole", "RevokeInternalToken"},
		},
	)

	handler := connect.NewUnaryHandler(
		"/quadsmith.ops.OpsService/SetUserRole",
		func(ctx context.Context, r *connect.Request[configpb.BootstrapTokenConfig]) (*connect.Response[configpb.BootstrapTokenConfig], error) {
			return connect.NewResponse(&configpb.BootstrapTokenConfig{}), nil
		},
		connect.WithInterceptors(interceptor),
	)

	// 1. Robot token on sensitive method -> CodePermissionDenied (403)
	httpReq := httptest.NewRequest(http.MethodPost, "/quadsmith.ops.OpsService/SetUserRole", strings.NewReader("{}"))
	httpReq.Header.Set("Authorization", "Bearer robot-token")
	httpReq.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httpReq)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected HTTP 403 Forbidden for ROBOT operator on sensitive method, got %d (body: %s)", rec.Code, rec.Body.String())
	}
}
