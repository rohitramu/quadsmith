package config

import (
	"testing"
	"time"
)

func TestDurationToCron(t *testing.T) {
	tests := []struct {
		d        time.Duration
		expected string
	}{
		{5 * time.Minute, "*/5 * * * *"},
		{1 * time.Minute, "* * * * *"},
		{10 * time.Minute, "*/10 * * * *"},
		{15 * time.Minute, "*/15 * * * *"},
		{30 * time.Minute, "*/30 * * * *"},
		{1 * time.Hour, "0 * * * *"},
		{2 * time.Hour, "0 0,2,4,6,8,10,12,14,16,18,20,22 * * *"},
		{6 * time.Hour, "0 0,6,12,18 * * *"},
		{8 * time.Hour, "0 0,8,16 * * *"},
		{12 * time.Hour, "0 0,12 * * *"},
		{24 * time.Hour, "0 0 * * *"},
		{7 * 24 * time.Hour, "0 0 * * 4"},
	}

	for _, tt := range tests {
		got, err := DurationToCron(tt.d)
		if err != nil {
			t.Errorf("DurationToCron(%v) unexpected error: %v", tt.d, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("DurationToCron(%v) = %q, want %q", tt.d, got, tt.expected)
		}
		if err := ValidateCronExpression(got); err != nil {
			t.Errorf("ValidateCronExpression(%q) failed: %v", got, err)
		}
	}
}
