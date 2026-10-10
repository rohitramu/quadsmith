package schemacheck

import (
	"testing"

	_ "quadsmith/api/gen/quadsmith"
)

func TestCheckAllProtoRegexPatterns(t *testing.T) {
	verified, err := CheckAllProtoRegexPatterns()
	if err != nil {
		t.Fatalf("CheckAllProtoRegexPatterns failed: %v", err)
	}

	if len(verified) == 0 {
		t.Fatalf("expected at least 1 validated regex in proto schema, found 0")
	}

	t.Logf("Successfully verified %d regex patterns in Protobuf schema:", len(verified))
	for _, v := range verified {
		t.Logf("  ✓ %s: %s", v.Field, v.Pattern)
	}
}
