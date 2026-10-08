package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/gen/quadsmith/quadsmithconnect"
)

func TestPrintTableTo_SliceWithRowNumbers(t *testing.T) {
	data := []map[string]interface{}{
		{"id": "m1", "name": "Motor 1", "kv": 19000},
		{"id": "m2", "name": "Motor 2", "kv": 21000},
	}
	cols := []string{"id", "name", "kv"}

	var buf bytes.Buffer
	printTableTo(&buf, data, cols)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d:\n%s", len(lines), output)
	}

	// Header should not contain '#'
	if strings.Contains(lines[0], "#") {
		t.Errorf("header should not contain #, got: %s", lines[0])
	}
	if !strings.Contains(lines[0], "ID") || !strings.Contains(lines[0], "NAME") || !strings.Contains(lines[0], "KV") {
		t.Errorf("header missing expected columns: %s", lines[0])
	}

	// Second line should start with 1
	if !strings.HasPrefix(lines[1], "1 ") {
		t.Errorf("expected first row to start with '1 ', got: %s", lines[1])
	}
	if !strings.Contains(lines[1], "m1") {
		t.Errorf("expected first row to contain 'm1', got: %s", lines[1])
	}

	// Third line should start with 2
	if !strings.HasPrefix(lines[2], "2 ") {
		t.Errorf("expected second row to start with '2 ', got: %s", lines[2])
	}
	if !strings.Contains(lines[2], "m2") {
		t.Errorf("expected second row to contain 'm2', got: %s", lines[2])
	}
}

func TestPrintTableTo_RightJustifiedRowNumbers(t *testing.T) {
	var data []map[string]interface{}
	for i := 1; i <= 10; i++ {
		data = append(data, map[string]interface{}{"id": fmt.Sprintf("m%d", i)})
	}
	var buf bytes.Buffer
	printTableTo(&buf, data, []string{"id"})

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 11 {
		t.Fatalf("expected 11 lines, got %d:\n%s", len(lines), buf.String())
	}
	// Header should not contain '#'
	if strings.Contains(lines[0], "#") {
		t.Errorf("header should not contain '#', got: %s", lines[0])
	}
	// Row 1 should have a leading space for right justification (" 1 ")
	if !strings.HasPrefix(lines[1], " 1 ") {
		t.Errorf("expected row 1 to start with ' 1 ', got: %q", lines[1])
	}
	// Row 10 should have no leading space ("10 ")
	if !strings.HasPrefix(lines[10], "10 ") {
		t.Errorf("expected row 10 to start with '10 ', got: %q", lines[10])
	}
}

func TestPrintTableTo_WithOffset(t *testing.T) {
	data := []map[string]interface{}{
		{"id": "m21", "name": "Motor 21"},
		{"id": "m22", "name": "Motor 22"},
	}
	cols := []string{"id", "name"}

	var buf bytes.Buffer
	printTableTo(&buf, data, cols, 20)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d:\n%s", len(lines), output)
	}

	if !strings.HasPrefix(lines[1], "21 ") {
		t.Errorf("expected row to start with '21 ', got: %s", lines[1])
	}
	if !strings.HasPrefix(lines[2], "22 ") {
		t.Errorf("expected row to start with '22 ', got: %s", lines[2])
	}
}

func TestPrintTableTo_EmptySlice(t *testing.T) {
	var data []map[string]interface{}

	var buf bytes.Buffer
	printTableTo(&buf, data, nil)

	output := strings.TrimSpace(buf.String())
	if output != "No records found." {
		t.Errorf("expected 'No records found.', got %q", output)
	}
}

func TestPrintTableTo_Map(t *testing.T) {
	data := map[string]interface{}{
		"id":   "motor-1",
		"name": "Test Motor",
	}

	var buf bytes.Buffer
	printTableTo(&buf, data, nil)

	output := buf.String()
	if strings.Contains(output, "#") {
		t.Errorf("map view should not contain '#', got: %s", output)
	}
	if !strings.Contains(output, "motor-1") || !strings.Contains(output, "Test Motor") {
		t.Errorf("map output missing fields: %s", output)
	}
}

func TestSilenceUsage_OnConnectionError(t *testing.T) {
	t.Setenv("QS_API_URL", "http://127.0.0.1:54321")
	cmd := newRootCmd()
	var errBuf bytes.Buffer
	var outBuf bytes.Buffer
	cmd.SetErr(&errBuf)
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"motors", "list"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}

	combined := errBuf.String() + outBuf.String()
	if strings.Contains(combined, "Usage:") {
		t.Errorf("connection error should not show usage text, got:\n%s", combined)
	}
}

func TestSilenceUsage_OnMissingArgs(t *testing.T) {
	cmd := newRootCmd()
	var errBuf bytes.Buffer
	var outBuf bytes.Buffer
	cmd.SetErr(&errBuf)
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"motors", "get"}) // requires 1 arg

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	combined := errBuf.String() + outBuf.String()
	if !strings.Contains(combined, "Usage:") {
		t.Errorf("missing argument error should show usage text, got:\n%s", combined)
	}
}

func TestSilenceUsage_OnUnknownFlag(t *testing.T) {
	cmd := newRootCmd()
	var errBuf bytes.Buffer
	var outBuf bytes.Buffer
	cmd.SetErr(&errBuf)
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"motors", "list", "--nonexistent-flag"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	combined := errBuf.String() + outBuf.String()
	if !strings.Contains(combined, "Usage:") {
		t.Errorf("unknown flag error should show usage text, got:\n%s", combined)
	}
}

func TestList_LimitNegative(t *testing.T) {
	cmd := newRootCmd()
	var errBuf bytes.Buffer
	var outBuf bytes.Buffer
	cmd.SetErr(&errBuf)
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"motors", "list", "--limit", "-1"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	combined := errBuf.String() + outBuf.String()
	if !strings.Contains(combined, "limit cannot be negative") {
		t.Errorf("expected negative limit error, got:\n%s", combined)
	}
	if !strings.Contains(combined, "Usage:") {
		t.Errorf("flag error should show usage, got:\n%s", combined)
	}
}

func TestList_LimitApplied(t *testing.T) {
	mock := &mockMotorService{
		listMotorsFunc: func(ctx context.Context, req *connect.Request[pb.ListMotorsRequest]) (*connect.Response[pb.ListMotorsResponse], error) {
			motors := []*pb.Motor{
				{Id: "m1", Name: "Motor 1", Kv: 19000},
				{Id: "m2", Name: "Motor 2", Kv: 21000},
				{Id: "m3", Name: "Motor 3", Kv: 23000},
				{Id: "m4", Name: "Motor 4", Kv: 25000},
				{Id: "m5", Name: "Motor 5", Kv: 27000},
			}
			limit := int(req.Msg.PageSize)
			if limit > len(motors) {
				limit = len(motors)
			}
			return connect.NewResponse(&pb.ListMotorsResponse{
				Motors: motors[:limit],
			}), nil
		},
	}
	setupMockServer(t, func(mux *http.ServeMux) {
		mux.Handle(quadsmithconnect.NewMotorServiceHandler(mock))
	})

	cmd := newRootCmd()
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"motors", "list", "-l", "3"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(outBuf.String()), "\n")
	// Header + 3 records = 4 lines
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines (header + 3 records), got %d:\n%s", len(lines), outBuf.String())
	}
}

func TestList_LimitJSON(t *testing.T) {
	mock := &mockMotorService{
		listMotorsFunc: func(ctx context.Context, req *connect.Request[pb.ListMotorsRequest]) (*connect.Response[pb.ListMotorsResponse], error) {
			motors := []*pb.Motor{
				{Id: "m1", Name: "Motor 1", Kv: 19000},
				{Id: "m2", Name: "Motor 2", Kv: 21000},
				{Id: "m3", Name: "Motor 3", Kv: 23000},
			}
			limit := int(req.Msg.PageSize)
			if limit > len(motors) {
				limit = len(motors)
			}
			return connect.NewResponse(&pb.ListMotorsResponse{
				Motors: motors[:limit],
			}), nil
		},
	}
	setupMockServer(t, func(mux *http.ServeMux) {
		mux.Handle(quadsmithconnect.NewMotorServiceHandler(mock))
	})

	cmd := newRootCmd()
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"motors", "list", "--json", "--limit", "2"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var results []map[string]interface{}
	if err := json.Unmarshal(outBuf.Bytes(), &results); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v\nOutput was:\n%s", err, outBuf.String())
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 items in JSON array, got %d", len(results))
	}
}

func TestList_RemovedPageTokenFlag(t *testing.T) {
	cmd := newRootCmd()
	var errBuf bytes.Buffer
	var outBuf bytes.Buffer
	cmd.SetErr(&errBuf)
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"motors", "list", "--page-token", "some-token"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error with unknown flag --page-token, got nil")
	}

	combined := errBuf.String() + outBuf.String()
	if !strings.Contains(combined, "unknown flag: --page-token") {
		t.Errorf("expected unknown flag error, got:\n%s", combined)
	}
}

func TestList_RemovedPageSizeFlag(t *testing.T) {
	cmd := newRootCmd()
	var errBuf bytes.Buffer
	var outBuf bytes.Buffer
	cmd.SetErr(&errBuf)
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"motors", "list", "--page-size", "10"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error with unknown flag --page-size, got nil")
	}

	combined := errBuf.String() + outBuf.String()
	if !strings.Contains(combined, "unknown flag: --page-size") {
		t.Errorf("expected unknown flag error, got:\n%s", combined)
	}
}

func TestList_AutoPaging(t *testing.T) {
	mock := &mockMotorService{
		listMotorsFunc: func(ctx context.Context, req *connect.Request[pb.ListMotorsRequest]) (*connect.Response[pb.ListMotorsResponse], error) {
			if req.Msg.PageToken == "" {
				var motors []*pb.Motor
				for i := 1; i <= 100; i++ {
					motors = append(motors, &pb.Motor{Id: fmt.Sprintf("m%d", i), Name: fmt.Sprintf("Motor %d", i)})
				}
				return connect.NewResponse(&pb.ListMotorsResponse{
					Motors:        motors,
					NextPageToken: "page-2",
				}), nil
			}
			if req.Msg.PageToken == "page-2" {
				var motors []*pb.Motor
				for i := 101; i <= 120; i++ {
					motors = append(motors, &pb.Motor{Id: fmt.Sprintf("m%d", i), Name: fmt.Sprintf("Motor %d", i)})
				}
				return connect.NewResponse(&pb.ListMotorsResponse{
					Motors:        motors,
					NextPageToken: "",
				}), nil
			}
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unexpected token: %s", req.Msg.PageToken))
		},
	}
	setupMockServer(t, func(mux *http.ServeMux) {
		mux.Handle(quadsmithconnect.NewMotorServiceHandler(mock))
	})

	cmd := newRootCmd()
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"motors", "list", "--json"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var results []map[string]interface{}
	if err := json.Unmarshal(outBuf.Bytes(), &results); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v\nOutput was:\n%s", err, outBuf.String())
	}
	// Total motors is 120, confirming it paged across 100-item page size
	if len(results) != 120 {
		t.Fatalf("expected 120 items from auto-paging, got %d", len(results))
	}
}

func TestCompletion_NoFlagDescriptions(t *testing.T) {
	cmd := newRootCmd()
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"__complete", "motors", "list", "-"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(outBuf.String(), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, ":") || strings.TrimSpace(line) == "" {
			continue
		}
		if strings.Contains(line, "\t") {
			t.Errorf("completion line contained description: %q", line)
		}
	}
}

func TestCompletionScript_BashNoDesc(t *testing.T) {
	cmd := newRootCmd()
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"completion", "bash"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	script := outBuf.String()
	if !strings.Contains(script, "__completeNoDesc") {
		t.Errorf("expected bash completion script to use __completeNoDesc, got:\n%s", script)
	}
}
