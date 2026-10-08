package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/gen/quadsmith/quadsmithconnect"
)

type mockMotorService struct {
	quadsmithconnect.UnimplementedMotorServiceHandler
	listMotorsFunc func(ctx context.Context, req *connect.Request[pb.ListMotorsRequest]) (*connect.Response[pb.ListMotorsResponse], error)
	getMotorFunc   func(ctx context.Context, req *connect.Request[pb.GetMotorRequest]) (*connect.Response[pb.Motor], error)
}

func (m *mockMotorService) ListMotors(ctx context.Context, req *connect.Request[pb.ListMotorsRequest]) (*connect.Response[pb.ListMotorsResponse], error) {
	if m.listMotorsFunc != nil {
		return m.listMotorsFunc(ctx, req)
	}
	return m.UnimplementedMotorServiceHandler.ListMotors(ctx, req)
}

func (m *mockMotorService) GetMotor(ctx context.Context, req *connect.Request[pb.GetMotorRequest]) (*connect.Response[pb.Motor], error) {
	if m.getMotorFunc != nil {
		return m.getMotorFunc(ctx, req)
	}
	return m.UnimplementedMotorServiceHandler.GetMotor(ctx, req)
}

type mockBuildService struct {
	quadsmithconnect.UnimplementedBuildServiceHandler
	listBuildsFunc func(ctx context.Context, req *connect.Request[pb.ListBuildsRequest]) (*connect.Response[pb.ListBuildsResponse], error)
	getBuildFunc   func(ctx context.Context, req *connect.Request[pb.GetBuildRequest]) (*connect.Response[pb.Build], error)
}

func (m *mockBuildService) ListBuilds(ctx context.Context, req *connect.Request[pb.ListBuildsRequest]) (*connect.Response[pb.ListBuildsResponse], error) {
	if m.listBuildsFunc != nil {
		return m.listBuildsFunc(ctx, req)
	}
	return m.UnimplementedBuildServiceHandler.ListBuilds(ctx, req)
}

func (m *mockBuildService) GetBuild(ctx context.Context, req *connect.Request[pb.GetBuildRequest]) (*connect.Response[pb.Build], error) {
	if m.getBuildFunc != nil {
		return m.getBuildFunc(ctx, req)
	}
	return m.UnimplementedBuildServiceHandler.GetBuild(ctx, req)
}

type mockEvaluatorService struct {
	quadsmithconnect.UnimplementedEvaluatorServiceHandler
	evaluateBuildFunc func(ctx context.Context, req *connect.Request[pb.EvaluateBuildRequest]) (*connect.Response[pb.EvaluateBuildResponse], error)
}

func (m *mockEvaluatorService) EvaluateBuild(ctx context.Context, req *connect.Request[pb.EvaluateBuildRequest]) (*connect.Response[pb.EvaluateBuildResponse], error) {
	if m.evaluateBuildFunc != nil {
		return m.evaluateBuildFunc(ctx, req)
	}
	return m.UnimplementedEvaluatorServiceHandler.EvaluateBuild(ctx, req)
}

func setupMockServer(t *testing.T, registerHandlers ...func(*http.ServeMux)) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for _, reg := range registerHandlers {
		reg(mux)
	}
	server := httptest.NewServer(mux)
	t.Cleanup(func() {
		server.Close()
	})
	t.Setenv("QS_API_URL", server.URL)
	return server
}
