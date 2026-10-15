package main

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "quadsmith/shared/gen/quadsmith/api"
	"quadsmith/shared/gen/quadsmith/api/apiconnect"
)

type mockUserService struct {
	apiconnect.UnimplementedUserServiceHandler
	banUserFunc          func(ctx context.Context, req *connect.Request[pb.BanUserRequest]) (*connect.Response[pb.BanUserResponse], error)
	unbanUserFunc        func(ctx context.Context, req *connect.Request[pb.UnbanUserRequest]) (*connect.Response[pb.UnbanUserResponse], error)
	updateBanFunc        func(ctx context.Context, req *connect.Request[pb.UpdateBanRequest]) (*connect.Response[pb.UpdateBanResponse], error)
	listBansFunc         func(ctx context.Context, req *connect.Request[pb.ListBansRequest]) (*connect.Response[pb.ListBansResponse], error)
	getUserBanStatusFunc func(ctx context.Context, req *connect.Request[pb.GetUserBanStatusRequest]) (*connect.Response[pb.GetUserBanStatusResponse], error)
}

func (m *mockUserService) BanUser(ctx context.Context, req *connect.Request[pb.BanUserRequest]) (*connect.Response[pb.BanUserResponse], error) {
	if m.banUserFunc != nil {
		return m.banUserFunc(ctx, req)
	}
	return m.UnimplementedUserServiceHandler.BanUser(ctx, req)
}

func (m *mockUserService) UnbanUser(ctx context.Context, req *connect.Request[pb.UnbanUserRequest]) (*connect.Response[pb.UnbanUserResponse], error) {
	if m.unbanUserFunc != nil {
		return m.unbanUserFunc(ctx, req)
	}
	return m.UnimplementedUserServiceHandler.UnbanUser(ctx, req)
}

func (m *mockUserService) UpdateBan(ctx context.Context, req *connect.Request[pb.UpdateBanRequest]) (*connect.Response[pb.UpdateBanResponse], error) {
	if m.updateBanFunc != nil {
		return m.updateBanFunc(ctx, req)
	}
	return m.UnimplementedUserServiceHandler.UpdateBan(ctx, req)
}

func (m *mockUserService) ListBans(ctx context.Context, req *connect.Request[pb.ListBansRequest]) (*connect.Response[pb.ListBansResponse], error) {
	if m.listBansFunc != nil {
		return m.listBansFunc(ctx, req)
	}
	return m.UnimplementedUserServiceHandler.ListBans(ctx, req)
}

func (m *mockUserService) GetUserBanStatus(ctx context.Context, req *connect.Request[pb.GetUserBanStatusRequest]) (*connect.Response[pb.GetUserBanStatusResponse], error) {
	if m.getUserBanStatusFunc != nil {
		return m.getUserBanStatusFunc(ctx, req)
	}
	return m.UnimplementedUserServiceHandler.GetUserBanStatus(ctx, req)
}

func TestParseDurationOrDays(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		{input: "1d", expected: 24 * time.Hour, hasError: false},
		{input: "7d", expected: 7 * 24 * time.Hour, hasError: false},
		{input: "30D", expected: 30 * 24 * time.Hour, hasError: false},
		{input: "24h", expected: 24 * time.Hour, hasError: false},
		{input: "1h30m", expected: 90 * time.Minute, hasError: false},
		{input: "invalid", expected: 0, hasError: true},
		{input: "xd", expected: 0, hasError: true},
	}

	for _, tt := range tests {
		got, err := parseDurationOrDays(tt.input)
		if tt.hasError {
			if err == nil {
				t.Errorf("parseDurationOrDays(%q) expected error, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("parseDurationOrDays(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.expected {
				t.Errorf("parseDurationOrDays(%q) = %v, expected %v", tt.input, got, tt.expected)
			}
		}
	}
}

func TestCLI_UserBans_CreateBothOrderings(t *testing.T) {
	calledTarget := ""
	calledReason := ""

	mock := &mockUserService{
		banUserFunc: func(ctx context.Context, req *connect.Request[pb.BanUserRequest]) (*connect.Response[pb.BanUserResponse], error) {
			calledTarget = req.Msg.UserId
			calledReason = req.Msg.Reason
			return connect.NewResponse(&pb.BanUserResponse{
				Success: true,
				Message: "User @" + req.Msg.UserId + " successfully banned",
				Ban: &pb.Ban{
					Uuid:     "ban-1234",
					UserUuid: "user-5678",
					Reason:   req.Msg.Reason,
				},
			}), nil
		},
	}
	setupMockServer(t, func(mux *http.ServeMux) {
		mux.Handle(apiconnect.NewUserServiceHandler(mock))
	})

	// Style 1: qs user bans <username|uuid> create --reason="..."
	cmd1 := newRootCmd()
	var outBuf1 bytes.Buffer
	cmd1.SetOut(&outBuf1)
	cmd1.SetArgs([]string{"user", "bans", "fpv_pilot", "create", "--reason", "Spamming reviews"})
	err := cmd1.Execute()
	if err != nil {
		t.Fatalf("style 1 failed: %v", err)
	}
	if calledTarget != "fpv_pilot" || calledReason != "Spamming reviews" {
		t.Errorf("unexpected call args in style 1: target=%q, reason=%q", calledTarget, calledReason)
	}
	if !strings.Contains(outBuf1.String(), "successfully banned") {
		t.Errorf("expected success message in style 1, got: %s", outBuf1.String())
	}

	// Reset
	calledTarget = ""
	calledReason = ""

	// Style 2: qs user bans create <username|uuid> --reason="..."
	cmd2 := newRootCmd()
	var outBuf2 bytes.Buffer
	cmd2.SetOut(&outBuf2)
	cmd2.SetArgs([]string{"user", "bans", "create", "fpv_pilot", "--reason", "Spamming reviews"})
	err = cmd2.Execute()
	if err != nil {
		t.Fatalf("style 2 failed: %v", err)
	}
	if calledTarget != "fpv_pilot" || calledReason != "Spamming reviews" {
		t.Errorf("unexpected call args in style 2: target=%q, reason=%q", calledTarget, calledReason)
	}
	if !strings.Contains(outBuf2.String(), "successfully banned") {
		t.Errorf("expected success message in style 2, got: %s", outBuf2.String())
	}
}

func TestCLI_UserBans_GetBothOrderings(t *testing.T) {
	mock := &mockUserService{
		getUserBanStatusFunc: func(ctx context.Context, req *connect.Request[pb.GetUserBanStatusRequest]) (*connect.Response[pb.GetUserBanStatusResponse], error) {
			return connect.NewResponse(&pb.GetUserBanStatusResponse{
				IsBanned: true,
				ActiveBan: &pb.Ban{
					Uuid:      "ban-9999",
					UserUuid:  "user-8888",
					Reason:    "Repeated community violations",
					CreatedAt: timestamppb.Now(),
				},
			}), nil
		},
	}
	setupMockServer(t, func(mux *http.ServeMux) {
		mux.Handle(apiconnect.NewUserServiceHandler(mock))
	})

	// Style 1: qs user bans <username|uuid> get
	cmd1 := newRootCmd()
	var outBuf1 bytes.Buffer
	cmd1.SetOut(&outBuf1)
	cmd1.SetArgs([]string{"user", "bans", "fpv_pilot", "get"})
	if err := cmd1.Execute(); err != nil {
		t.Fatalf("style 1 failed: %v", err)
	}
	if !strings.Contains(outBuf1.String(), "Repeated community violations") {
		t.Errorf("expected ban details in style 1, got:\n%s", outBuf1.String())
	}

	// Style 2: qs user bans get <username|uuid>
	cmd2 := newRootCmd()
	var outBuf2 bytes.Buffer
	cmd2.SetOut(&outBuf2)
	cmd2.SetArgs([]string{"user", "bans", "get", "fpv_pilot"})
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("style 2 failed: %v", err)
	}
	if !strings.Contains(outBuf2.String(), "Repeated community violations") {
		t.Errorf("expected ban details in style 2, got:\n%s", outBuf2.String())
	}
}

func TestCLI_UserBans_UpdateBothOrderings(t *testing.T) {
	calledTarget := ""
	mock := &mockUserService{
		updateBanFunc: func(ctx context.Context, req *connect.Request[pb.UpdateBanRequest]) (*connect.Response[pb.UpdateBanResponse], error) {
			calledTarget = req.Msg.UserId
			return connect.NewResponse(&pb.UpdateBanResponse{
				Success: true,
				Message: "Ban on @" + req.Msg.UserId + " successfully updated",
				Ban: &pb.Ban{
					Uuid:     "ban-1234",
					UserUuid: "user-5678",
					Reason:   "Updated reason",
				},
			}), nil
		},
	}
	setupMockServer(t, func(mux *http.ServeMux) {
		mux.Handle(apiconnect.NewUserServiceHandler(mock))
	})

	// Style 1: qs user bans <username|uuid> update
	cmd1 := newRootCmd()
	var outBuf1 bytes.Buffer
	cmd1.SetOut(&outBuf1)
	cmd1.SetArgs([]string{"user", "bans", "fpv_pilot", "update", "--expires-in", "7d"})
	if err := cmd1.Execute(); err != nil {
		t.Fatalf("style 1 failed: %v", err)
	}
	if calledTarget != "fpv_pilot" {
		t.Errorf("unexpected target in style 1: %q", calledTarget)
	}

	calledTarget = ""

	// Style 2: qs user bans update <username|uuid>
	cmd2 := newRootCmd()
	var outBuf2 bytes.Buffer
	cmd2.SetOut(&outBuf2)
	cmd2.SetArgs([]string{"user", "bans", "update", "fpv_pilot", "--expires-in", "7d"})
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("style 2 failed: %v", err)
	}
	if calledTarget != "fpv_pilot" {
		t.Errorf("unexpected target in style 2: %q", calledTarget)
	}
}

func TestCLI_UserBans_DeleteBothOrderings(t *testing.T) {
	calledTarget := ""
	mock := &mockUserService{
		unbanUserFunc: func(ctx context.Context, req *connect.Request[pb.UnbanUserRequest]) (*connect.Response[pb.UnbanUserResponse], error) {
			calledTarget = req.Msg.UserId
			return connect.NewResponse(&pb.UnbanUserResponse{
				Success: true,
				Message: "User @" + req.Msg.UserId + " successfully unbanned",
				Ban: &pb.Ban{
					Uuid:     "ban-1234",
					UserUuid: "user-5678",
				},
			}), nil
		},
	}
	setupMockServer(t, func(mux *http.ServeMux) {
		mux.Handle(apiconnect.NewUserServiceHandler(mock))
	})

	// Style 1: qs user bans <username|uuid> delete
	cmd1 := newRootCmd()
	var outBuf1 bytes.Buffer
	cmd1.SetOut(&outBuf1)
	cmd1.SetArgs([]string{"user", "bans", "fpv_pilot", "delete", "--reason", "Appeal granted"})
	if err := cmd1.Execute(); err != nil {
		t.Fatalf("style 1 failed: %v", err)
	}
	if calledTarget != "fpv_pilot" {
		t.Errorf("unexpected target in style 1: %q", calledTarget)
	}

	calledTarget = ""

	// Style 2: qs user bans delete <username|uuid>
	cmd2 := newRootCmd()
	var outBuf2 bytes.Buffer
	cmd2.SetOut(&outBuf2)
	cmd2.SetArgs([]string{"user", "bans", "delete", "fpv_pilot", "--reason", "Appeal granted"})
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("style 2 failed: %v", err)
	}
	if calledTarget != "fpv_pilot" {
		t.Errorf("unexpected target in style 2: %q", calledTarget)
	}
}
