package buildid

import (
	"testing"
)

func TestNextCopyID(t *testing.T) {
	tests := []struct {
		name       string
		candidate  string
		existing   map[string]bool
		expectedID string
	}{
		{
			name:       "no existing suffix generates copy1",
			candidate:  "bando-basher-5-inch",
			existing:   map[string]bool{"bando-basher-5-inch": true},
			expectedID: "bando-basher-5-inch-copy1",
		},
		{
			name:      "copy1 already taken generates copy2",
			candidate: "bando-basher-5-inch",
			existing: map[string]bool{
				"bando-basher-5-inch":       true,
				"bando-basher-5-inch-copy1": true,
			},
			expectedID: "bando-basher-5-inch-copy2",
		},
		{
			name:      "candidate has copy1 suffix, increments to copy2",
			candidate: "bando-basher-5-inch-copy1",
			existing: map[string]bool{
				"bando-basher-5-inch-copy1": true,
			},
			expectedID: "bando-basher-5-inch-copy2",
		},
		{
			name:      "candidate has copy1 suffix, copy2 taken, increments to copy3",
			candidate: "bando-basher-5-inch-copy1",
			existing: map[string]bool{
				"bando-basher-5-inch-copy1": true,
				"bando-basher-5-inch-copy2": true,
			},
			expectedID: "bando-basher-5-inch-copy3",
		},
		{
			name:      "candidate has copy99 suffix, increments to copy100",
			candidate: "my-drone-copy99",
			existing: map[string]bool{
				"my-drone-copy99": true,
			},
			expectedID: "my-drone-copy100",
		},
		{
			name:      "candidate has -copy with no digits, generates copy1",
			candidate: "custom-drone-copy",
			existing: map[string]bool{
				"custom-drone-copy": true,
			},
			expectedID: "custom-drone-copy1",
		},
		{
			name:      "gaps in copies uses first available number",
			candidate: "bando-basher-5-inch",
			existing: map[string]bool{
				"bando-basher-5-inch":       true,
				"bando-basher-5-inch-copy1": true,
				"bando-basher-5-inch-copy3": true,
			},
			expectedID: "bando-basher-5-inch-copy2",
		},
		{
			name:      "case-insensitive copy detection",
			candidate: "Toothpick-COPY2",
			existing: map[string]bool{
				"Toothpick-COPY2": true,
			},
			expectedID: "Toothpick-copy3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isTaken := func(id string) (bool, error) {
				return tt.existing[id], nil
			}
			got, err := NextCopyID(tt.candidate, isTaken)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expectedID {
				t.Errorf("NextCopyID(%q) = %q, want %q", tt.candidate, got, tt.expectedID)
			}
		})
	}
}
