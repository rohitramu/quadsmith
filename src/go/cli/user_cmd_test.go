package main

import (
	"testing"
	"time"
)

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
