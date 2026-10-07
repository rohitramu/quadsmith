package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestParsePageOffset(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected int
	}{
		{
			name:     "empty token",
			token:    "",
			expected: 0,
		},
		{
			name:     "invalid base64",
			token:    "not-valid-base64!@#$",
			expected: 0,
		},
		{
			name:     "valid json with offset 20",
			token:    "eyJvZmZzZXQiOjIwfQ", // base64 of {"offset":20}
			expected: 20,
		},
		{
			name:     "valid json with offset 40",
			token:    "eyJvZmZzZXQiOjQwfQ", // base64 of {"offset":40}
			expected: 40,
		},
		{
			name:     "negative offset ignored",
			token:    "eyJvZmZzZXQiOi01fQ", // base64 of {"offset":-5}
			expected: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := parsePageOffset(tc.token)
			if actual != tc.expected {
				t.Errorf("parsePageOffset(%q) = %d, expected %d", tc.token, actual, tc.expected)
			}
		})
	}
}

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

	// First line should start with #
	if !strings.HasPrefix(lines[0], "#") {
		t.Errorf("expected header to start with #, got: %s", lines[0])
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
