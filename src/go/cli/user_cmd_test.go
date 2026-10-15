package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"sigs.k8s.io/yaml"

	pb "quadsmith/shared/gen/quadsmith/api"
	"quadsmith/shared/gen/quadsmith/api/apiconnect"
)

type mockUserService struct {
	apiconnect.UnimplementedUserServiceHandler
	banUserFunc          func(ctx context.Context, req *connect.Request[pb.BanUserRequest]) (*connect.Response[pb.BanUserResponse], error)
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

func TestCLI_UserBans_New(t *testing.T) {
	calledTarget := ""
	calledReason := ""
	calledNotes := ""

	mock := &mockUserService{
		banUserFunc: func(ctx context.Context, req *connect.Request[pb.BanUserRequest]) (*connect.Response[pb.BanUserResponse], error) {
			calledTarget = req.Msg.UserId
			calledReason = req.Msg.Reason
			calledNotes = req.Msg.InternalNotes
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

	// 1. Primary command 'new'
	cmd := newRootCmd()
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"user", "bans", "new", "fpv_pilot", "--reason", "Spamming reviews", "--notes", "Repeated report"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("new failed: %v", err)
	}
	if calledTarget != "fpv_pilot" || calledReason != "Spamming reviews" || calledNotes != "Repeated report" {
		t.Errorf("unexpected call args: target=%q, reason=%q, notes=%q", calledTarget, calledReason, calledNotes)
	}
	if !strings.Contains(outBuf.String(), "successfully banned") {
		t.Errorf("expected success message, got: %s", outBuf.String())
	}

	// 2. Alias command 'create'
	cmd2 := newRootCmd()
	var outBuf2 bytes.Buffer
	cmd2.SetOut(&outBuf2)
	cmd2.SetArgs([]string{"user", "bans", "create", "fpv_pilot_2", "--reason", "Abusive language"})
	err2 := cmd2.Execute()
	if err2 != nil {
		t.Fatalf("create alias failed: %v", err2)
	}
	if calledTarget != "fpv_pilot_2" || calledReason != "Abusive language" {
		t.Errorf("unexpected call args on create alias: target=%q, reason=%q", calledTarget, calledReason)
	}
}

func TestCLI_UserBans_Get(t *testing.T) {
	mock := &mockUserService{
		getUserBanStatusFunc: func(ctx context.Context, req *connect.Request[pb.GetUserBanStatusRequest]) (*connect.Response[pb.GetUserBanStatusResponse], error) {
			if req.Msg.UserId == "clean_user" {
				return connect.NewResponse(&pb.GetUserBanStatusResponse{
					IsBanned: false,
				}), nil
			}
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

	// 1. Get banned user
	cmd1 := newRootCmd()
	var outBuf1 bytes.Buffer
	cmd1.SetOut(&outBuf1)
	cmd1.SetArgs([]string{"user", "bans", "get", "fpv_pilot"})
	if err := cmd1.Execute(); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !strings.Contains(outBuf1.String(), "Repeated community violations") {
		t.Errorf("expected ban details, got:\n%s", outBuf1.String())
	}

	// 2. Get with column projection
	cmd2 := newRootCmd()
	var outBuf2 bytes.Buffer
	cmd2.SetOut(&outBuf2)
	cmd2.SetArgs([]string{"user", "bans", "get", "fpv_pilot", "-c", "reason"})
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("get with columns failed: %v", err)
	}
	if !strings.Contains(outBuf2.String(), "Repeated community violations") {
		t.Errorf("expected reason column, got:\n%s", outBuf2.String())
	}

	// 3. Get clean user
	cmd3 := newRootCmd()
	var outBuf3 bytes.Buffer
	cmd3.SetOut(&outBuf3)
	cmd3.SetArgs([]string{"user", "bans", "get", "clean_user"})
	if err := cmd3.Execute(); err != nil {
		t.Fatalf("get clean user failed: %v", err)
	}
	if !strings.Contains(outBuf3.String(), "not currently banned") {
		t.Errorf("expected not banned message, got:\n%s", outBuf3.String())
	}
}

func TestCLI_UserBans_Update(t *testing.T) {
	calledTarget := ""
	calledReason := ""
	calledRevoke := false

	mock := &mockUserService{
		updateBanFunc: func(ctx context.Context, req *connect.Request[pb.UpdateBanRequest]) (*connect.Response[pb.UpdateBanResponse], error) {
			calledTarget = req.Msg.UserId
			if req.Msg.Reason != nil {
				calledReason = *req.Msg.Reason
			}
			if req.Msg.Revoke != nil {
				calledRevoke = *req.Msg.Revoke
			}
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

	// 1. Update reason and expiry
	cmd1 := newRootCmd()
	var outBuf1 bytes.Buffer
	cmd1.SetOut(&outBuf1)
	cmd1.SetArgs([]string{"user", "bans", "update", "fpv_pilot", "--reason", "Updated reason", "--expires-in", "7d"})
	if err := cmd1.Execute(); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if calledTarget != "fpv_pilot" || calledReason != "Updated reason" {
		t.Errorf("unexpected update args: target=%q, reason=%q", calledTarget, calledReason)
	}

	// 2. Revoke active ban
	cmd2 := newRootCmd()
	var outBuf2 bytes.Buffer
	cmd2.SetOut(&outBuf2)
	cmd2.SetArgs([]string{"user", "bans", "update", "fpv_pilot", "--revoke", "--reason", "Appeal granted"})
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("update revoke failed: %v", err)
	}
	if !calledRevoke || calledReason != "Appeal granted" {
		t.Errorf("unexpected revoke call: revoke=%v, reason=%q", calledRevoke, calledReason)
	}
}

func TestCLI_UserBans_List_Flags(t *testing.T) {
	var capturedReq *pb.ListBansRequest

	mock := &mockUserService{
		listBansFunc: func(ctx context.Context, req *connect.Request[pb.ListBansRequest]) (*connect.Response[pb.ListBansResponse], error) {
			capturedReq = req.Msg
			return connect.NewResponse(&pb.ListBansResponse{
				Bans: []*pb.Ban{
					{
						Uuid:      "ban-1",
						UserUuid:  "user-1",
						Reason:    "Spamming",
						CreatedAt: timestamppb.Now(),
					},
					{
						Uuid:      "ban-2",
						UserUuid:  "user-2",
						Reason:    "Abuse",
						CreatedAt: timestamppb.Now(),
					},
				},
			}), nil
		},
	}
	setupMockServer(t, func(mux *http.ServeMux) {
		mux.Handle(apiconnect.NewUserServiceHandler(mock))
	})

	// 1. List with standard flags
	cmd1 := newRootCmd()
	var outBuf1 bytes.Buffer
	cmd1.SetOut(&outBuf1)
	cmd1.SetArgs([]string{
		"user", "bans", "list", "fpv_pilot",
		"-f", "reason == 'Spamming'",
		"-s", "^created_at",
		"-c", "uuid,reason",
		"-l", "25",
		"--active-only",
	})
	if err := cmd1.Execute(); err != nil {
		t.Fatalf("list with flags failed: %v", err)
	}

	if capturedReq == nil {
		t.Fatal("expected listBans to be called, got nil")
	}
	if capturedReq.UserId == nil || *capturedReq.UserId != "fpv_pilot" {
		t.Errorf("expected user_id filter fpv_pilot, got %v", capturedReq.UserId)
	}
	if capturedReq.Filter != "reason == 'Spamming'" {
		t.Errorf("expected filter flag, got %q", capturedReq.Filter)
	}
	if len(capturedReq.Sort) != 1 || capturedReq.Sort[0] != "^created_at" {
		t.Errorf("expected sort flag [^created_at], got %v", capturedReq.Sort)
	}
	if !capturedReq.ActiveOnly {
		t.Errorf("expected active-only flag true")
	}

	// Output should include header columns UUID and REASON
	out1 := outBuf1.String()
	if !strings.Contains(out1, "UUID") || !strings.Contains(out1, "REASON") {
		t.Errorf("expected column headers UUID and REASON, got:\n%s", out1)
	}

	// 2. List with --json
	cmd2 := newRootCmd()
	var outBuf2 bytes.Buffer
	cmd2.SetOut(&outBuf2)
	cmd2.SetArgs([]string{"user", "bans", "list", "--json"})
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("list --json failed: %v", err)
	}
	var jsonResults []map[string]interface{}
	if err := json.Unmarshal(outBuf2.Bytes(), &jsonResults); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput:\n%s", err, outBuf2.String())
	}
	if len(jsonResults) != 2 {
		t.Fatalf("expected 2 JSON ban records, got %d", len(jsonResults))
	}

	// 3. List with --yaml
	cmd3 := newRootCmd()
	var outBuf3 bytes.Buffer
	cmd3.SetOut(&outBuf3)
	cmd3.SetArgs([]string{"user", "bans", "list", "--yaml"})
	if err := cmd3.Execute(); err != nil {
		t.Fatalf("list --yaml failed: %v", err)
	}
	var yamlResults []map[string]interface{}
	if err := yaml.Unmarshal(outBuf3.Bytes(), &yamlResults); err != nil {
		t.Fatalf("failed to parse YAML output: %v\nOutput:\n%s", err, outBuf3.String())
	}
	if len(yamlResults) != 2 {
		t.Fatalf("expected 2 YAML ban records, got %d", len(yamlResults))
	}
}

func TestCLI_UserBans_NoDeleteSubcommand(t *testing.T) {
	cmd := newRootCmd()
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"user", "bans", "delete", "fpv_pilot"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error executing 'delete' subcommand on bans, got nil")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("expected unknown command error for delete, got: %v", err)
	}
}

func TestCLI_UserBans_TargetMustFollowOperation(t *testing.T) {
	cmd := newRootCmd()
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"user", "bans", "fpv_pilot", "create"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error executing 'user bans fpv_pilot create' (reversed order), got nil")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("expected unknown command error for 'fpv_pilot', got: %v", err)
	}
}
