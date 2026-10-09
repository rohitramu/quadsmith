package search

import (
	"testing"

	pb "quadsmith/api/gen/quadsmith"
)

func TestResolveTargets_EmptySelectors(t *testing.T) {
	targets, err := ResolveTargets(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(targets) != len(pb.AllSearchCollections) {
		t.Fatalf("expected %d targets, got %d", len(pb.AllSearchCollections), len(targets))
	}

	for _, target := range targets {
		if target.Filter != "" {
			t.Errorf("expected empty filter for default target %s, got %q", target.Collection.CanonicalPath, target.Filter)
		}
	}
}

func TestResolveTargets_SingleSelector(t *testing.T) {
	selectors := []*pb.SearchSelector{
		{
			Path:   "components/hardware/batteries",
			Filter: protoOptString("cell_count_s == 6"),
		},
	}

	targets, err := ResolveTargets(selectors)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}

	if targets[0].Collection.CanonicalPath != "components/hardware/batteries" {
		t.Errorf("expected path components/hardware/batteries, got %s", targets[0].Collection.CanonicalPath)
	}

	if targets[0].Filter != "cell_count_s == 6" {
		t.Errorf("expected filter 'cell_count_s == 6', got %q", targets[0].Filter)
	}
}

func TestResolveTargets_DuplicatePaths_ANDMergedWithParentheses(t *testing.T) {
	selectors := []*pb.SearchSelector{
		{
			Path:   "components/hardware/batteries",
			Filter: protoOptString("cell_count_s == 6 || cell_count_s == 4"),
		},
		{
			Path:   "batteries", // Alias for components/hardware/batteries
			Filter: protoOptString("capacity_mah >= 1000"),
		},
	}

	targets, err := ResolveTargets(selectors)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(targets) != 1 {
		t.Fatalf("expected 1 target after merging duplicates, got %d", len(targets))
	}

	expectedFilter := "(cell_count_s == 6 || cell_count_s == 4) && (capacity_mah >= 1000)"
	if targets[0].Filter != expectedFilter {
		t.Errorf("expected merged filter %q, got %q", expectedFilter, targets[0].Filter)
	}
}

func TestResolveTargets_DuplicatePaths_OneEmptyFilter(t *testing.T) {
	selectors := []*pb.SearchSelector{
		{
			Path:   "components/hardware/frames",
			Filter: protoOptString("wheelbase_mm >= 210"),
		},
		{
			Path:   "frames",
			Filter: nil, // empty filter
		},
	}

	targets, err := ResolveTargets(selectors)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}

	expectedFilter := "wheelbase_mm >= 210"
	if targets[0].Filter != expectedFilter {
		t.Errorf("expected filter %q, got %q", expectedFilter, targets[0].Filter)
	}
}

func TestResolveTargets_DuplicatePaths_BothEmptyFilters(t *testing.T) {
	selectors := []*pb.SearchSelector{
		{
			Path:   "components/hardware/motors",
			Filter: nil,
		},
		{
			Path:   "motors",
			Filter: protoOptString(""),
		},
	}

	targets, err := ResolveTargets(selectors)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}

	if targets[0].Filter != "" {
		t.Errorf("expected empty filter, got %q", targets[0].Filter)
	}
}

func TestResolveTargets_MultipleDistinctCollections(t *testing.T) {
	selectors := []*pb.SearchSelector{
		{
			Path:   "escs",
			Filter: protoOptString("motor_current_max_a >= 45.0"),
		},
		{
			Path:   "builds",
			Filter: nil,
		},
		{
			Path:   "motors",
			Filter: protoOptString("kv_rating > 2000"),
		},
	}

	targets, err := ResolveTargets(selectors)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(targets) != 3 {
		t.Fatalf("expected 3 targets, got %d", len(targets))
	}

	if targets[0].Collection.CanonicalPath != "components/hardware/electronic-speed-controllers" {
		t.Errorf("expected electronic-speed-controllers, got %s", targets[0].Collection.CanonicalPath)
	}
	if targets[1].Collection.CanonicalPath != "builds" {
		t.Errorf("expected builds, got %s", targets[1].Collection.CanonicalPath)
	}
	if targets[2].Collection.CanonicalPath != "components/hardware/motors" {
		t.Errorf("expected motors, got %s", targets[2].Collection.CanonicalPath)
	}
}

func TestResolveTargets_UnknownPath(t *testing.T) {
	selectors := []*pb.SearchSelector{
		{
			Path: "unknown/collection/path",
		},
	}

	_, err := ResolveTargets(selectors)
	if err == nil {
		t.Fatal("expected error for unknown collection path, got nil")
	}
}

func TestResolveTargets_EmptyPath(t *testing.T) {
	selectors := []*pb.SearchSelector{
		{
			Path: "   ",
		},
	}

	_, err := ResolveTargets(selectors)
	if err == nil {
		t.Fatal("expected error for empty path, got nil")
	}
}

func protoOptString(s string) *string {
	return &s
}
