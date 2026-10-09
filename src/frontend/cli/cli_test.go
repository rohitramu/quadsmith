package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"sigs.k8s.io/yaml"

	pb "quadsmith/api/gen/quadsmith"
	"quadsmith/api/gen/quadsmith/quadsmithconnect"
)

// --- OUTPUT FORMAT TESTS ---

func TestList_YAML(t *testing.T) {
	mock := &mockMotorService{
		listMotorsFunc: func(ctx context.Context, req *connect.Request[pb.ListMotorsRequest]) (*connect.Response[pb.ListMotorsResponse], error) {
			return connect.NewResponse(&pb.ListMotorsResponse{
				Motors: []*pb.Motor{
					{Id: "motor-alpha", Name: "Alpha Motor", Kv: 19500},
					{Id: "motor-beta", Name: "Beta Motor", Kv: 22000},
				},
			}), nil
		},
	}
	setupMockServer(t, func(mux *http.ServeMux) {
		mux.Handle(quadsmithconnect.NewMotorServiceHandler(mock))
	})

	cmd := newRootCmd()
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"components", "motors", "list", "--yaml"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var results []map[string]interface{}
	if err := yaml.Unmarshal(outBuf.Bytes(), &results); err != nil {
		t.Fatalf("failed to unmarshal YAML output: %v\nOutput was:\n%s", err, outBuf.String())
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 items, got %d", len(results))
	}
	if results[0]["id"] != "motor-alpha" || results[1]["id"] != "motor-beta" {
		t.Errorf("unexpected results: %+v", results)
	}
}

func TestList_EmptySlice_JSON(t *testing.T) {
	mock := &mockMotorService{
		listMotorsFunc: func(ctx context.Context, req *connect.Request[pb.ListMotorsRequest]) (*connect.Response[pb.ListMotorsResponse], error) {
			return connect.NewResponse(&pb.ListMotorsResponse{
				Motors: []*pb.Motor{},
			}), nil
		},
	}
	setupMockServer(t, func(mux *http.ServeMux) {
		mux.Handle(quadsmithconnect.NewMotorServiceHandler(mock))
	})

	cmd := newRootCmd()
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"components", "motors", "list", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	trimmed := strings.TrimSpace(outBuf.String())
	if trimmed != "[]" {
		t.Errorf("expected empty JSON array '[]', got %q", trimmed)
	}
}

func TestList_EmptySlice_YAML(t *testing.T) {
	mock := &mockMotorService{
		listMotorsFunc: func(ctx context.Context, req *connect.Request[pb.ListMotorsRequest]) (*connect.Response[pb.ListMotorsResponse], error) {
			return connect.NewResponse(&pb.ListMotorsResponse{
				Motors: []*pb.Motor{},
			}), nil
		},
	}
	setupMockServer(t, func(mux *http.ServeMux) {
		mux.Handle(quadsmithconnect.NewMotorServiceHandler(mock))
	})

	cmd := newRootCmd()
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"components", "motors", "list", "--yaml"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	trimmed := strings.TrimSpace(outBuf.String())
	if trimmed != "[]" {
		t.Errorf("expected empty YAML array '[]', got %q", trimmed)
	}
}

func TestList_EmptySlice_Table(t *testing.T) {
	mock := &mockMotorService{
		listMotorsFunc: func(ctx context.Context, req *connect.Request[pb.ListMotorsRequest]) (*connect.Response[pb.ListMotorsResponse], error) {
			return connect.NewResponse(&pb.ListMotorsResponse{
				Motors: []*pb.Motor{},
			}), nil
		},
	}
	setupMockServer(t, func(mux *http.ServeMux) {
		mux.Handle(quadsmithconnect.NewMotorServiceHandler(mock))
	})

	cmd := newRootCmd()
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"components", "motors", "list"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	trimmed := strings.TrimSpace(outBuf.String())
	if trimmed != "No records found." {
		t.Errorf("expected 'No records found.', got %q", trimmed)
	}
}

func TestGet_JSON(t *testing.T) {
	mock := &mockMotorService{
		getMotorFunc: func(ctx context.Context, req *connect.Request[pb.GetMotorRequest]) (*connect.Response[pb.Motor], error) {
			if req.Msg.Id == "motor-1" {
				return connect.NewResponse(&pb.Motor{
					Id:   "motor-1",
					Name: "Test Motor 1",
					Kv:   25000,
				}), nil
			}
			return nil, connect.NewError(connect.CodeNotFound, errors.New("not found"))
		},
	}
	setupMockServer(t, func(mux *http.ServeMux) {
		mux.Handle(quadsmithconnect.NewMotorServiceHandler(mock))
	})

	cmd := newRootCmd()
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"components", "motors", "get", "motor-1", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var res map[string]interface{}
	if err := json.Unmarshal(outBuf.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if res["id"] != "motor-1" || res["name"] != "Test Motor 1" {
		t.Errorf("unexpected output: %+v", res)
	}
}

func TestGet_NoYAMLFlag(t *testing.T) {
	cmd := newRootCmd()
	var errBuf bytes.Buffer
	var outBuf bytes.Buffer
	cmd.SetErr(&errBuf)
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"components", "motors", "get", "motor-1", "--yaml"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when passing --yaml to get command, since YAML is default")
	}
	combined := errBuf.String() + outBuf.String()
	if !strings.Contains(combined, "unknown flag: --yaml") {
		t.Errorf("expected 'unknown flag: --yaml', got: %s", combined)
	}
}

func TestGet_DefaultYAML(t *testing.T) {
	mock := &mockMotorService{
		getMotorFunc: func(ctx context.Context, req *connect.Request[pb.GetMotorRequest]) (*connect.Response[pb.Motor], error) {
			return connect.NewResponse(&pb.Motor{
				Id:   "motor-table",
				Name: "Table Motor",
			}), nil
		},
	}
	setupMockServer(t, func(mux *http.ServeMux) {
		mux.Handle(quadsmithconnect.NewMotorServiceHandler(mock))
	})

	cmd := newRootCmd()
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"components", "motors", "get", "motor-table"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var res map[string]interface{}
	if err := yaml.Unmarshal(outBuf.Bytes(), &res); err != nil {
		t.Fatalf("expected valid YAML by default, got: %v\nOutput:\n%s", err, outBuf.String())
	}
	if res["id"] != "motor-table" || res["name"] != "Table Motor" {
		t.Errorf("expected motor-table and Table Motor, got: %+v", res)
	}
}

// --- VALUE FORMATTING TESTS ---

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"nil", nil, "-"},
		{"empty string", "", "-"},
		{"empty slice", []interface{}{}, "-"},
		{"non-empty string", "test", "test"},
		{"integer", 42, "42"},
		{"float", 3.14, "3.14"},
		{"boolean", true, "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatValue(tt.input)
			if got != tt.expected {
				t.Errorf("formatValue(%v) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// --- COMMAND ALIASES AND SUBCOMMAND TESTS ---

func TestCommandAliases(t *testing.T) {
	rootCmd := newRootCmd()

	// Root-level commands & aliases
	rootTests := []struct {
		alias        string
		expectedRoot string
	}{
		{"build", "builds"},
		{"builds", "builds"},
		{"component", "components"},
		{"components", "components"},
	}

	for _, tt := range rootTests {
		t.Run(tt.alias, func(t *testing.T) {
			cmd, _, err := rootCmd.Find([]string{tt.alias})
			if err != nil {
				t.Fatalf("failed to find command for alias %q: %v", tt.alias, err)
			}
			if cmd.Name() != tt.expectedRoot {
				t.Errorf("alias %q resolved to %q, want %q", tt.alias, cmd.Name(), tt.expectedRoot)
			}
		})
	}

	// Builds subcommands & aliases
	buildTests := []struct {
		alias    string
		expected string
	}{
		{"eval", "evaluate"},
		{"evaluate", "evaluate"},
	}
	for _, tt := range buildTests {
		t.Run("builds/"+tt.alias, func(t *testing.T) {
			cmd, _, err := rootCmd.Find([]string{"builds", tt.alias})
			if err != nil {
				t.Fatalf("failed to find builds alias %q: %v", tt.alias, err)
			}
			if cmd.Name() != tt.expected {
				t.Errorf("builds alias %q resolved to %q, want %q", tt.alias, cmd.Name(), tt.expected)
			}
		})
	}

	// Component collection aliases under components
	componentTests := []struct {
		alias        string
		expectedRoot string
	}{
		{"fc", "flight-controllers"},
		{"fcs", "flight-controllers"},
		{"flightcontrollers", "flight-controllers"},
		{"flight-controller", "flight-controllers"},
		{"vtx", "video-transmitters"},
		{"vtxs", "video-transmitters"},
		{"videotransmitters", "video-transmitters"},
		{"video-transmitter", "video-transmitters"},
		{"esc", "electronic-speed-controllers"},
		{"escs", "electronic-speed-controllers"},
		{"electronic-speed-controller", "electronic-speed-controllers"},
		{"rx", "receivers"},
		{"rxs", "receivers"},
		{"receiver", "receivers"},
		{"gps", "gps-receivers"},
		{"gpsreceivers", "gps-receivers"},
		{"gps-receiver", "gps-receivers"},
		{"motors", "motors"},
		{"motor", "motors"},
		{"frames", "frames"},
		{"frame", "frames"},
		{"antennas", "antennas"},
		{"antenna", "antennas"},
		{"batteries", "batteries"},
		{"battery", "batteries"},
		{"propellers", "propellers"},
		{"propeller", "propellers"},
		{"props", "propellers"},
		{"prop", "propellers"},
	}

	for _, tt := range componentTests {
		t.Run("components/"+tt.alias, func(t *testing.T) {
			cmd, _, err := rootCmd.Find([]string{"components", tt.alias})
			if err != nil {
				t.Fatalf("failed to find command for alias %q: %v", tt.alias, err)
			}
			if cmd.Name() != tt.expectedRoot {
				t.Errorf("alias %q resolved to %q, want %q", tt.alias, cmd.Name(), tt.expectedRoot)
			}
		})
	}
}

func TestDomainSubcommands(t *testing.T) {
	rootCmd := newRootCmd()

	// Verify builds subcommand under rootCmd
	t.Run("builds", func(t *testing.T) {
		cmd, _, err := rootCmd.Find([]string{"builds"})
		if err != nil {
			t.Fatalf("could not find builds subcommand: %v", err)
		}
		subcommands := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			subcommands[sub.Name()] = true
		}
		for _, expected := range []string{"list", "get", "evaluate"} {
			if !subcommands[expected] {
				t.Errorf("builds missing %q subcommand", expected)
			}
		}
	})

	// Verify components subcommand and its 11 component collections
	componentCollections := []string{
		"antennas",
		"batteries",
		"cameras",
		"electronic-speed-controllers",
		"flight-controllers",
		"frames",
		"gps-receivers",
		"motors",
		"propellers",
		"receivers",
		"video-transmitters",
	}

	for _, c := range componentCollections {
		t.Run("components/"+c, func(t *testing.T) {
			cmd, _, err := rootCmd.Find([]string{"components", c})
			if err != nil {
				t.Fatalf("could not find component collection components/%s: %v", c, err)
			}

			subcommands := make(map[string]bool)
			for _, sub := range cmd.Commands() {
				subcommands[sub.Name()] = true
			}

			if !subcommands["list"] {
				t.Errorf("components/%s missing 'list' subcommand", c)
			}
			if !subcommands["get"] {
				t.Errorf("components/%s missing 'get' subcommand", c)
			}
		})
	}
}

// --- PROTOBUF REFLECTION & COLUMNS TESTS ---

func TestGetColumns(t *testing.T) {
	motorCols := GetColumns(&pb.Motor{})
	if len(motorCols) == 0 {
		t.Fatal("expected non-empty columns for Motor")
	}

	colMap := make(map[string]bool)
	for _, c := range motorCols {
		colMap[c] = true
		if strings.HasPrefix(c, "XXX_") {
			t.Errorf("found unexported protobuf field in columns: %s", c)
		}
	}

	for _, expected := range []string{"id", "name", "kv", "weight_g"} {
		if !colMap[expected] {
			t.Errorf("expected Motor columns to contain %q, got: %v", expected, motorCols)
		}
	}

	frameCols := GetColumns(&pb.Frame{})
	frameMap := make(map[string]bool)
	for _, c := range frameCols {
		frameMap[c] = true
	}
	for _, expected := range []string{"id", "name", "wheelbase_mm"} {
		if !frameMap[expected] {
			t.Errorf("expected Frame columns to contain %q, got: %v", expected, frameCols)
		}
	}
}

func TestGetDefaultColumns(t *testing.T) {
	defCols := GetDefaultColumns(&pb.Motor{})
	if len(defCols) == 0 {
		t.Fatal("expected non-empty default columns for Motor")
	}

	colMap := make(map[string]bool)
	for _, c := range defCols {
		colMap[c] = true
	}

	// manufacturer, name, and kv should be present in default columns
	if !colMap["manufacturer"] || !colMap["name"] || !colMap["kv"] {
		t.Errorf("expected 'manufacturer', 'name', and 'kv' in default columns, got: %v", defCols)
	}
}

func TestList_CustomColumns(t *testing.T) {
	mock := &mockMotorService{
		listMotorsFunc: func(ctx context.Context, req *connect.Request[pb.ListMotorsRequest]) (*connect.Response[pb.ListMotorsResponse], error) {
			return connect.NewResponse(&pb.ListMotorsResponse{
				Motors: []*pb.Motor{
					{Id: "m1", Name: "Motor One", Kv: 19000, WeightG: 3.5},
				},
			}), nil
		},
	}
	setupMockServer(t, func(mux *http.ServeMux) {
		mux.Handle(quadsmithconnect.NewMotorServiceHandler(mock))
	})

	cmd := newRootCmd()
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"components", "motors", "list", "-c", "id,kv"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(outBuf.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected header and row, got: %v", lines)
	}

	header := lines[0]
	if !strings.Contains(header, "ID") || !strings.Contains(header, "KV") {
		t.Errorf("expected header to contain ID and KV, got: %s", header)
	}
	if strings.Contains(header, "WEIGHT_G") {
		t.Errorf("header should not contain unrequested column WEIGHT_G, got: %s", header)
	}
}

// --- SHELL AUTOCOMPLETION TESTS ---

func TestCompletion_ColumnFlag(t *testing.T) {
	rootCmd := newRootCmd()
	cmd, _, err := rootCmd.Find([]string{"components", "motors", "list"})
	if err != nil {
		t.Fatalf("failed to find motors list: %v", err)
	}

	fn, ok := cmd.GetFlagCompletionFunc("column")
	if !ok || fn == nil {
		t.Fatal("expected flag completion function for --column")
	}

	// Completing "k" should suggest "kv"
	completions, _ := fn(cmd, nil, "k")
	foundKv := false
	for _, c := range completions {
		if c == "kv" {
			foundKv = true
			break
		}
	}
	if !foundKv {
		t.Errorf("expected 'kv' in column completions for 'k', got: %v", completions)
	}

	// When "id" is already specified, it should not suggest "id" again
	_ = cmd.Flags().Set("column", "id")
	completions, _ = fn(cmd, nil, "i")
	for _, c := range completions {
		if c == "id" {
			t.Errorf("'id' was already selected and should not be suggested again: %v", completions)
		}
	}
}

func TestCompletion_SortFlag(t *testing.T) {
	rootCmd := newRootCmd()
	cmd, _, err := rootCmd.Find([]string{"components", "motors", "list"})
	if err != nil {
		t.Fatalf("failed to find motors list: %v", err)
	}

	fn, ok := cmd.GetFlagCompletionFunc("sort")
	if !ok || fn == nil {
		t.Fatal("expected flag completion function for --sort")
	}

	// Completing empty string should suggest ascending and descending options
	completions, _ := fn(cmd, nil, "")
	foundAsc := false
	foundDesc := false
	for _, c := range completions {
		if c == "kv" {
			foundAsc = true
		}
		if c == "^kv" {
			foundDesc = true
		}
	}
	if !foundAsc || !foundDesc {
		t.Errorf("expected both 'kv' and '^kv' in sort completions, got: %v", completions)
	}

	// Completing "^k" should only suggest descending
	completions, _ = fn(cmd, nil, "^k")
	if len(completions) == 0 || completions[0] != "^kv" {
		t.Errorf("expected ['^kv'] for '^k', got: %v", completions)
	}
}

func TestCompletion_FilterFlag(t *testing.T) {
	rootCmd := newRootCmd()
	cmd, _, err := rootCmd.Find([]string{"components", "motors", "list"})
	if err != nil {
		t.Fatalf("failed to find motors list: %v", err)
	}

	fn, ok := cmd.GetFlagCompletionFunc("filter")
	if !ok || fn == nil {
		t.Fatal("expected flag completion function for --filter")
	}

	// Completing "weight_g > 10 && k" should complete to "weight_g > 10 && kv"
	completions, _ := fn(cmd, nil, "weight_g > 10 && k")
	found := false
	for _, c := range completions {
		if c == "weight_g > 10 && kv" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'weight_g > 10 && kv' in filter completions, got: %v", completions)
	}
}

func TestCompletion_GetValidArgs(t *testing.T) {
	mock := &mockMotorService{
		listMotorsFunc: func(ctx context.Context, req *connect.Request[pb.ListMotorsRequest]) (*connect.Response[pb.ListMotorsResponse], error) {
			return connect.NewResponse(&pb.ListMotorsResponse{
				Motors: []*pb.Motor{
					{Id: "m-first"},
					{Id: "m-second"},
				},
			}), nil
		},
	}
	setupMockServer(t, func(mux *http.ServeMux) {
		mux.Handle(quadsmithconnect.NewMotorServiceHandler(mock))
	})

	rootCmd := newRootCmd()
	cmd, _, err := rootCmd.Find([]string{"components", "motors", "get"})
	if err != nil {
		t.Fatalf("failed to find motors get: %v", err)
	}

	if cmd.ValidArgsFunction == nil {
		t.Fatal("expected ValidArgsFunction on motors get")
	}

	comps, directive := cmd.ValidArgsFunction(cmd, nil, "")
	if len(comps) != 2 || comps[0] != "m-first" || comps[1] != "m-second" {
		t.Errorf("unexpected completion results: %v, directive: %v", comps, directive)
	}

	// If an arg is already provided, it should return nil
	comps, _ = cmd.ValidArgsFunction(cmd, []string{"m-first"}, "")
	if len(comps) != 0 {
		t.Errorf("expected no completions when arg is already provided, got: %v", comps)
	}
}

// --- EVALUATE COMMAND TESTS ---

func TestEvaluate_Success(t *testing.T) {
	mockBuild := &mockBuildService{
		getBuildFunc: func(ctx context.Context, req *connect.Request[pb.GetBuildRequest]) (*connect.Response[pb.Build], error) {
			if req.Msg.Id == "build-1" {
				return connect.NewResponse(&pb.Build{
					Id:   "build-1",
					Name: "Freestyle 5 inch",
				}), nil
			}
			return nil, connect.NewError(connect.CodeNotFound, errors.New("build not found"))
		},
	}

	mockEval := &mockEvaluatorService{
		getBuildElectricalLimitsFunc: func(ctx context.Context, req *connect.Request[pb.GetBuildElectricalLimitsRequest]) (*connect.Response[pb.GetBuildElectricalLimitsResponse], error) {
			return connect.NewResponse(&pb.GetBuildElectricalLimitsResponse{
				DefaultBatteryId: "battery-1",
			}), nil
		},
		evaluateBuildFunc: func(ctx context.Context, req *connect.Request[pb.EvaluateBuildRequest]) (*connect.Response[pb.EvaluateBuildResponse], error) {
			if req.Msg.BatteryId == "" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("battery_id is required"))
			}
			if req.Msg.Build.Id != "build-1" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("mismatched build"))
			}
			return connect.NewResponse(&pb.EvaluateBuildResponse{
				BuildId:              req.Msg.Build.Id,
				PayloadWeightG:       req.Msg.PayloadWeightG,
				BatteryId:            req.Msg.BatteryId,
				TotalWeightG:         350.5,
				HoverThrottlePercent: 28.4,
				ThrustToWeightRatio:  7.2,
				MinFlightTimeMin:     4.5,
				MaxFlightTimeMin:     8.5,
				SystemMessages: []*pb.SystemMessage{
					{
						Severity: pb.SystemMessageSeverity_SYSTEM_MESSAGE_SEVERITY_WARNING,
						Message:  "High KV for battery voltage",
					},
				},
			}), nil
		},
	}

	setupMockServer(t, func(mux *http.ServeMux) {
		mux.Handle(quadsmithconnect.NewBuildServiceHandler(mockBuild))
		mux.Handle(quadsmithconnect.NewEvaluatorServiceHandler(mockEval))
	})

	cmd := newRootCmd()
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"builds", "evaluate", "build-1", "--payload", "30", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var res map[string]interface{}
	if err := json.Unmarshal(outBuf.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v\nOutput: %s", err, outBuf.String())
	}

	if res["build_id"] != "build-1" {
		t.Errorf("expected build_id: 'build-1', got: %v", res["build_id"])
	}
	if res["payload_weight_g"] != float64(30) {
		t.Errorf("expected payload_weight_g: 30, got: %v", res["payload_weight_g"])
	}
	if res["battery_id"] != "battery-1" {
		t.Errorf("expected battery_id: 'battery-1', got: %v", res["battery_id"])
	}
	if res["system_messages"] == nil {
		t.Errorf("expected system_messages in output: %+v", res)
	}
}

func TestEvaluate_DefaultYAML(t *testing.T) {
	mockBuild := &mockBuildService{
		getBuildFunc: func(ctx context.Context, req *connect.Request[pb.GetBuildRequest]) (*connect.Response[pb.Build], error) {
			if req.Msg.Id == "build-1" {
				return connect.NewResponse(&pb.Build{
					Id:   "build-1",
					Name: "Freestyle 5 inch",
				}), nil
			}
			return nil, connect.NewError(connect.CodeNotFound, errors.New("build not found"))
		},
	}

	mockEval := &mockEvaluatorService{
		getBuildElectricalLimitsFunc: func(ctx context.Context, req *connect.Request[pb.GetBuildElectricalLimitsRequest]) (*connect.Response[pb.GetBuildElectricalLimitsResponse], error) {
			return connect.NewResponse(&pb.GetBuildElectricalLimitsResponse{
				DefaultBatteryId: "default-battery-id",
			}), nil
		},
		evaluateBuildFunc: func(ctx context.Context, req *connect.Request[pb.EvaluateBuildRequest]) (*connect.Response[pb.EvaluateBuildResponse], error) {
			if req.Msg.BatteryId == "" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("battery_id is required"))
			}
			return connect.NewResponse(&pb.EvaluateBuildResponse{
				BuildId:             req.Msg.Build.Id,
				PayloadWeightG:      req.Msg.PayloadWeightG,
				BatteryId:           req.Msg.BatteryId,
				TotalWeightG:        350.5,
				ThrustToWeightRatio: 7.2,
				SystemMessages: []*pb.SystemMessage{
					{
						Severity: pb.SystemMessageSeverity_SYSTEM_MESSAGE_SEVERITY_WARNING,
						Message:  "High KV for battery voltage",
					},
				},
			}), nil
		},
	}

	setupMockServer(t, func(mux *http.ServeMux) {
		mux.Handle(quadsmithconnect.NewBuildServiceHandler(mockBuild))
		mux.Handle(quadsmithconnect.NewEvaluatorServiceHandler(mockEval))
	})

	cmd := newRootCmd()
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"builds", "evaluate", "build-1"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var res map[string]interface{}
	if err := yaml.Unmarshal(outBuf.Bytes(), &res); err != nil {
		t.Fatalf("expected valid YAML by default, got: %v\nOutput:\n%s", err, outBuf.String())
	}

	if res["build_id"] != "build-1" {
		t.Errorf("expected build_id: 'build-1', got: %v", res["build_id"])
	}
	if res["battery_id"] != "default-battery-id" {
		t.Errorf("expected battery_id: 'default-battery-id', got: %v", res["battery_id"])
	}
	if res["thrust_to_weight_ratio"] != float64(7.2) {
		t.Errorf("expected thrust_to_weight_ratio: 7.2, got: %v", res["thrust_to_weight_ratio"])
	}
	if res["system_messages"] == nil {
		t.Errorf("expected system_messages in output: %+v", res)
	}
}

func TestEvaluate_WithBatteryOverride(t *testing.T) {
	mockBuild := &mockBuildService{
		getBuildFunc: func(ctx context.Context, req *connect.Request[pb.GetBuildRequest]) (*connect.Response[pb.Build], error) {
			return connect.NewResponse(&pb.Build{
				Id:   "build-1",
				Name: "Freestyle 5 inch",
			}), nil
		},
	}

	mockEval := &mockEvaluatorService{
		evaluateBuildFunc: func(ctx context.Context, req *connect.Request[pb.EvaluateBuildRequest]) (*connect.Response[pb.EvaluateBuildResponse], error) {
			if req.Msg.BatteryId != "custom-bat" {
				return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("expected custom-bat, got %s", req.Msg.BatteryId))
			}
			return connect.NewResponse(&pb.EvaluateBuildResponse{
				BuildId:        req.Msg.Build.Id,
				PayloadWeightG: req.Msg.PayloadWeightG,
				BatteryId:      req.Msg.BatteryId,
				TotalWeightG:   320.0,
			}), nil
		},
	}

	setupMockServer(t, func(mux *http.ServeMux) {
		mux.Handle(quadsmithconnect.NewBuildServiceHandler(mockBuild))
		mux.Handle(quadsmithconnect.NewEvaluatorServiceHandler(mockEval))
	})

	cmd := newRootCmd()
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"builds", "evaluate", "build-1", "--battery", "custom-bat", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var res map[string]interface{}
	if err := json.Unmarshal(outBuf.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if res["battery_id"] != "custom-bat" {
		t.Errorf("expected battery_id: 'custom-bat', got: %v", res["battery_id"])
	}
}

func TestEvaluate_NoYAMLFlag(t *testing.T) {
	cmd := newRootCmd()
	var errBuf bytes.Buffer
	var outBuf bytes.Buffer
	cmd.SetErr(&errBuf)
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"builds", "evaluate", "build-1", "--yaml"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when passing --yaml to evaluate command, since YAML is default")
	}
	combined := errBuf.String() + outBuf.String()
	if !strings.Contains(combined, "unknown flag: --yaml") {
		t.Errorf("expected 'unknown flag: --yaml', got: %s", combined)
	}
}

func TestEvaluate_AliasEval(t *testing.T) {
	mockBuild := &mockBuildService{
		getBuildFunc: func(ctx context.Context, req *connect.Request[pb.GetBuildRequest]) (*connect.Response[pb.Build], error) {
			if req.Msg.Id == "build-1" {
				return connect.NewResponse(&pb.Build{
					Id:   "build-1",
					Name: "Freestyle 5 inch",
				}), nil
			}
			return nil, connect.NewError(connect.CodeNotFound, errors.New("build not found"))
		},
	}

	mockEval := &mockEvaluatorService{
		getBuildElectricalLimitsFunc: func(ctx context.Context, req *connect.Request[pb.GetBuildElectricalLimitsRequest]) (*connect.Response[pb.GetBuildElectricalLimitsResponse], error) {
			return connect.NewResponse(&pb.GetBuildElectricalLimitsResponse{
				DefaultBatteryId: "battery-1",
			}), nil
		},
		evaluateBuildFunc: func(ctx context.Context, req *connect.Request[pb.EvaluateBuildRequest]) (*connect.Response[pb.EvaluateBuildResponse], error) {
			if req.Msg.BatteryId == "" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("battery_id is required"))
			}
			return connect.NewResponse(&pb.EvaluateBuildResponse{
				TotalWeightG: 350.5,
			}), nil
		},
	}

	setupMockServer(t, func(mux *http.ServeMux) {
		mux.Handle(quadsmithconnect.NewBuildServiceHandler(mockBuild))
		mux.Handle(quadsmithconnect.NewEvaluatorServiceHandler(mockEval))
	})

	cmd := newRootCmd()
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"builds", "eval", "build-1", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEvaluate_NoCompatibleBatteryError(t *testing.T) {
	mockBuild := &mockBuildService{
		getBuildFunc: func(ctx context.Context, req *connect.Request[pb.GetBuildRequest]) (*connect.Response[pb.Build], error) {
			return connect.NewResponse(&pb.Build{
				Id:   "build-1",
				Name: "Freestyle 5 inch",
			}), nil
		},
	}

	mockEval := &mockEvaluatorService{
		getBuildElectricalLimitsFunc: func(ctx context.Context, req *connect.Request[pb.GetBuildElectricalLimitsRequest]) (*connect.Response[pb.GetBuildElectricalLimitsResponse], error) {
			return connect.NewResponse(&pb.GetBuildElectricalLimitsResponse{
				DefaultBatteryId: "", // No compatible battery found
			}), nil
		},
	}

	setupMockServer(t, func(mux *http.ServeMux) {
		mux.Handle(quadsmithconnect.NewBuildServiceHandler(mockBuild))
		mux.Handle(quadsmithconnect.NewEvaluatorServiceHandler(mockEval))
	})

	cmd := newRootCmd()
	var errBuf bytes.Buffer
	var outBuf bytes.Buffer
	cmd.SetErr(&errBuf)
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"builds", "evaluate", "build-1"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no compatible battery found, but got none")
	}
	if !strings.Contains(err.Error(), "no compatible battery found for build; specify one using --battery") {
		t.Errorf("expected 'no compatible battery found' error, got: %v", err)
	}
}

func TestEvaluate_AutoSelectLightestBattery(t *testing.T) {
	mockBuild := &mockBuildService{
		getBuildFunc: func(ctx context.Context, req *connect.Request[pb.GetBuildRequest]) (*connect.Response[pb.Build], error) {
			return connect.NewResponse(&pb.Build{
				Id:   "build-1",
				Name: "Freestyle 5 inch",
			}), nil
		},
	}

	limitsCalled := false
	var evaluatedBattery string
	mockEval := &mockEvaluatorService{
		getBuildElectricalLimitsFunc: func(ctx context.Context, req *connect.Request[pb.GetBuildElectricalLimitsRequest]) (*connect.Response[pb.GetBuildElectricalLimitsResponse], error) {
			limitsCalled = true
			return connect.NewResponse(&pb.GetBuildElectricalLimitsResponse{
				DefaultBatteryId: "lightest-compatible-battery",
			}), nil
		},
		evaluateBuildFunc: func(ctx context.Context, req *connect.Request[pb.EvaluateBuildRequest]) (*connect.Response[pb.EvaluateBuildResponse], error) {
			evaluatedBattery = req.Msg.BatteryId
			return connect.NewResponse(&pb.EvaluateBuildResponse{
				BuildId:      req.Msg.Build.Id,
				BatteryId:    req.Msg.BatteryId,
				TotalWeightG: 350.5,
			}), nil
		},
	}

	setupMockServer(t, func(mux *http.ServeMux) {
		mux.Handle(quadsmithconnect.NewBuildServiceHandler(mockBuild))
		mux.Handle(quadsmithconnect.NewEvaluatorServiceHandler(mockEval))
	})

	cmd := newRootCmd()
	var outBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"builds", "evaluate", "build-1", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !limitsCalled {
		t.Errorf("expected GetBuildElectricalLimits to be called when --battery is omitted")
	}
	if evaluatedBattery != "lightest-compatible-battery" {
		t.Errorf("expected evaluatedBattery to be 'lightest-compatible-battery', got: %s", evaluatedBattery)
	}
}

func TestEvaluate_MissingArg(t *testing.T) {
	cmd := newRootCmd()
	var errBuf bytes.Buffer
	var outBuf bytes.Buffer
	cmd.SetErr(&errBuf)
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"builds", "evaluate"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when evaluate is called without build ID")
	}

	combined := errBuf.String() + outBuf.String()
	if !strings.Contains(combined, "accepts 1 arg(s), received 0") {
		t.Errorf("expected 'accepts 1 arg(s), received 0', got:\n%s", combined)
	}
}

func TestEvaluate_BuildNotFound(t *testing.T) {
	mockBuild := &mockBuildService{
		getBuildFunc: func(ctx context.Context, req *connect.Request[pb.GetBuildRequest]) (*connect.Response[pb.Build], error) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("build does not exist"))
		},
	}
	setupMockServer(t, func(mux *http.ServeMux) {
		mux.Handle(quadsmithconnect.NewBuildServiceHandler(mockBuild))
	})

	cmd := newRootCmd()
	var errBuf bytes.Buffer
	var outBuf bytes.Buffer
	cmd.SetErr(&errBuf)
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{"builds", "evaluate", "missing-build"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent build")
	}

	if !strings.Contains(err.Error(), "failed to fetch build") {
		t.Errorf("expected 'failed to fetch build' error, got: %v", err)
	}
}
