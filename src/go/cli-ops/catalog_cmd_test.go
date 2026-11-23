package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	apipb "quadsmith/shared/gen/quadsmith/api"
	pb "quadsmith/shared/gen/quadsmith/ops"
	"quadsmith/shared/gen/quadsmith/ops/opsconnect"
)

type mockMotorOpsService struct {
	opsconnect.UnimplementedMotorOpsServiceHandler
	lastAuthHeader string
	listFunc       func(ctx context.Context, req *connect.Request[pb.ListMotorsOpsRequest]) (*connect.Response[pb.ListMotorsOpsResponse], error)
	getFunc        func(ctx context.Context, req *connect.Request[pb.GetMotorOpsRequest]) (*connect.Response[apipb.Motor], error)
	createFunc     func(ctx context.Context, req *connect.Request[pb.CreateMotorOpsRequest]) (*connect.Response[apipb.Motor], error)
	updateFunc     func(ctx context.Context, req *connect.Request[pb.UpdateMotorOpsRequest]) (*connect.Response[apipb.Motor], error)
	deleteFunc     func(ctx context.Context, req *connect.Request[pb.DeleteMotorOpsRequest]) (*connect.Response[pb.DeleteMotorOpsResponse], error)
	restoreFunc    func(ctx context.Context, req *connect.Request[pb.RestoreMotorOpsRequest]) (*connect.Response[apipb.Motor], error)
}

func (m *mockMotorOpsService) ListMotors(ctx context.Context, req *connect.Request[pb.ListMotorsOpsRequest]) (*connect.Response[pb.ListMotorsOpsResponse], error) {
	m.lastAuthHeader = req.Header().Get("Authorization")
	if m.listFunc != nil {
		return m.listFunc(ctx, req)
	}
	return m.UnimplementedMotorOpsServiceHandler.ListMotors(ctx, req)
}

func (m *mockMotorOpsService) GetMotor(ctx context.Context, req *connect.Request[pb.GetMotorOpsRequest]) (*connect.Response[apipb.Motor], error) {
	m.lastAuthHeader = req.Header().Get("Authorization")
	if m.getFunc != nil {
		return m.getFunc(ctx, req)
	}
	return m.UnimplementedMotorOpsServiceHandler.GetMotor(ctx, req)
}

func (m *mockMotorOpsService) CreateMotor(ctx context.Context, req *connect.Request[pb.CreateMotorOpsRequest]) (*connect.Response[apipb.Motor], error) {
	m.lastAuthHeader = req.Header().Get("Authorization")
	if m.createFunc != nil {
		return m.createFunc(ctx, req)
	}
	return m.UnimplementedMotorOpsServiceHandler.CreateMotor(ctx, req)
}

func (m *mockMotorOpsService) UpdateMotor(ctx context.Context, req *connect.Request[pb.UpdateMotorOpsRequest]) (*connect.Response[apipb.Motor], error) {
	m.lastAuthHeader = req.Header().Get("Authorization")
	if m.updateFunc != nil {
		return m.updateFunc(ctx, req)
	}
	return m.UnimplementedMotorOpsServiceHandler.UpdateMotor(ctx, req)
}

func (m *mockMotorOpsService) DeleteMotor(ctx context.Context, req *connect.Request[pb.DeleteMotorOpsRequest]) (*connect.Response[pb.DeleteMotorOpsResponse], error) {
	m.lastAuthHeader = req.Header().Get("Authorization")
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, req)
	}
	return m.UnimplementedMotorOpsServiceHandler.DeleteMotor(ctx, req)
}

func (m *mockMotorOpsService) RestoreMotor(ctx context.Context, req *connect.Request[pb.RestoreMotorOpsRequest]) (*connect.Response[apipb.Motor], error) {
	m.lastAuthHeader = req.Header().Get("Authorization")
	if m.restoreFunc != nil {
		return m.restoreFunc(ctx, req)
	}
	return m.UnimplementedMotorOpsServiceHandler.RestoreMotor(ctx, req)
}

func setupOpsMockServer(t *testing.T, register func(*http.ServeMux)) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	register(mux)
	server := httptest.NewServer(mux)
	overrideOpsServerURL = server.URL
	t.Cleanup(func() {
		overrideOpsServerURL = ""
		server.Close()
	})
	return server
}

func TestOpsCatalogSubcommands(t *testing.T) {
	rootCmd := newRootCmd()

	componentPaths := [][]string{
		{"components", "hardware", "antennas"},
		{"components", "hardware", "batteries"},
		{"components", "hardware", "cameras"},
		{"components", "hardware", "electronic-speed-controllers"},
		{"components", "hardware", "flight-controllers"},
		{"components", "hardware", "frames"},
		{"components", "hardware", "gps-receivers"},
		{"components", "hardware", "motors"},
		{"components", "hardware", "propellers"},
		{"components", "hardware", "receivers"},
		{"components", "hardware", "video-transmitters"},
		{"components", "gear", "transmitters"},
		{"components", "gear", "goggles"},
		{"components", "gear", "chargers"},
		{"components", "gear", "field-tools"},
		{"components", "software", "fc-firmwares"},
		{"components", "software", "esc-firmwares"},
		{"components", "software", "vtx-firmwares"},
		{"components", "software", "radio-firmwares"},
		{"components", "software", "software-tools"},
	}

	for _, p := range componentPaths {
		pathStr := strings.Join(p, "/")
		t.Run(pathStr, func(t *testing.T) {
			cmd, _, err := rootCmd.Find(p)
			if err != nil {
				t.Fatalf("could not find %s: %v", pathStr, err)
			}
			subcommands := make(map[string]bool)
			for _, sub := range cmd.Commands() {
				subcommands[sub.Name()] = true
			}
			for _, expected := range []string{"list", "get", "create", "update", "delete", "restore"} {
				if !subcommands[expected] {
					t.Errorf("%s missing %q subcommand", pathStr, expected)
				}
			}
		})
	}
}

func TestOpsCatalogLifecycleAndFlags(t *testing.T) {
	var sawIncludeArchived bool
	mock := &mockMotorOpsService{
		listFunc: func(ctx context.Context, req *connect.Request[pb.ListMotorsOpsRequest]) (*connect.Response[pb.ListMotorsOpsResponse], error) {
			sawIncludeArchived = req.Msg.GetIncludeArchived()
			return connect.NewResponse(&pb.ListMotorsOpsResponse{
				Motors: []*apipb.Motor{
					{Id: "tmotor-f60", Name: "F60 Pro V", Manufacturer: "T-Motor", Kv: 1950, IsArchived: true},
				},
			}), nil
		},
		getFunc: func(ctx context.Context, req *connect.Request[pb.GetMotorOpsRequest]) (*connect.Response[apipb.Motor], error) {
			return connect.NewResponse(&apipb.Motor{
				Uuid:         "018f0000-0000-7000-8000-000000000999",
				Id:           req.Msg.GetId(),
				Name:         "F60 Pro V",
				Manufacturer: "T-Motor",
				Kv:           1950,
				WeightG:      32.5,
			}), nil
		},
		createFunc: func(ctx context.Context, req *connect.Request[pb.CreateMotorOpsRequest]) (*connect.Response[apipb.Motor], error) {
			m := req.Msg.GetMotor()
			m.Uuid = "018f0000-0000-7000-8000-000000000999"
			if m.Id == "" {
				m.Id = "tmotor-f60-pro-v"
			}
			return connect.NewResponse(m), nil
		},
		updateFunc: func(ctx context.Context, req *connect.Request[pb.UpdateMotorOpsRequest]) (*connect.Response[apipb.Motor], error) {
			return connect.NewResponse(req.Msg.GetMotor()), nil
		},
		deleteFunc: func(ctx context.Context, req *connect.Request[pb.DeleteMotorOpsRequest]) (*connect.Response[pb.DeleteMotorOpsResponse], error) {
			return connect.NewResponse(&pb.DeleteMotorOpsResponse{IsArchived: true}), nil
		},
		restoreFunc: func(ctx context.Context, req *connect.Request[pb.RestoreMotorOpsRequest]) (*connect.Response[apipb.Motor], error) {
			return connect.NewResponse(&apipb.Motor{
				Id:         req.Msg.GetId(),
				Name:       "F60 Pro V",
				IsArchived: false,
			}), nil
		},
	}

	setupOpsMockServer(t, func(mux *http.ServeMux) {
		mux.Handle(opsconnect.NewMotorOpsServiceHandler(mock))
	})

	// 1. List with --include-archived and --token
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--token", "ops-test-token", "--json", "components", "hardware", "motors", "list", "--include-archived"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if !sawIncludeArchived {
		t.Error("expected IncludeArchived=true in ListMotorsOpsRequest")
	}
	if mock.lastAuthHeader != "Bearer ops-test-token" {
		t.Errorf("expected Bearer ops-test-token, got %q", mock.lastAuthHeader)
	}

	// 2. Create with typed flags
	cmd = newRootCmd()
	out.Reset()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--token", "ops-test-token", "--json", "components", "hw", "motor", "create", "--name", "F60 Pro V", "--manufacturer", "T-Motor", "--kv", "2020", "--weight-g", "33.5"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	var created map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &created); err != nil {
		t.Fatalf("failed to unmarshal create response: %v\nOutput: %s", err, out.String())
	}
	if created["name"] != "F60 Pro V" || created["manufacturer"] != "T-Motor" || created["kv"] != float64(2020) {
		t.Errorf("unexpected created motor: %+v", created)
	}

	// 3. Update with typed flag only (merges onto existing record)
	cmd = newRootCmd()
	out.Reset()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--token", "ops-test-token", "--json", "components", "hardware", "motors", "update", "tmotor-f60", "--kv", "2150"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	var updated map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &updated); err != nil {
		t.Fatalf("failed to unmarshal update response: %v", err)
	}
	if updated["name"] != "F60 Pro V" || updated["kv"] != float64(2150) {
		t.Errorf("expected merged name 'F60 Pro V' and updated kv 2150, got: %+v", updated)
	}

	// 4. Delete (soft-archive) and Restore
	cmd = newRootCmd()
	out.Reset()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--token", "ops-test-token", "components", "hardware", "motors", "delete", "tmotor-f60"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if !strings.Contains(out.String(), "archived/delisted") {
		t.Errorf("expected archived message, got: %s", out.String())
	}

	cmd = newRootCmd()
	out.Reset()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--token", "ops-test-token", "--json", "components", "hardware", "motors", "restore", "tmotor-f60"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("restore failed: %v", err)
	}
	var restored map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &restored); err != nil {
		t.Fatalf("failed to unmarshal restore response: %v", err)
	}
	if restored["id"] != "tmotor-f60" || restored["is_archived"] != false {
		t.Errorf("unexpected restore output: %+v", restored)
	}
}

func TestOpsBlobUploadCmd(t *testing.T) {
	var gotAuth string
	setupOpsMockServer(t, func(mux *http.ServeMux) {
		mux.HandleFunc("/api/ops/blobs", func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(opsBlobUploadResponse{
				ID:        "blob-test-123",
				URL:       "/blobs/blob-test-123",
				Filename:  "motor.png",
				MIMEType:  "image/png",
				SizeBytes: 8,
			})
		})
	})

	tmpFile := filepath.Join(t.TempDir(), "motor.png")
	if err := os.WriteFile(tmpFile, []byte("fakepng!"), 0600); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--token", "ops-jit-token", "--json", "blob", "upload", tmpFile})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("blob upload failed: %v", err)
	}
	if gotAuth != "Bearer ops-jit-token" {
		t.Errorf("expected Bearer ops-jit-token, got %q", gotAuth)
	}
	if !strings.Contains(out.String(), "blob-test-123") {
		t.Errorf("unexpected output: %s", out.String())
	}
}
