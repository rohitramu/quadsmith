package schemacheck

import (
	"testing"

	_ "quadsmith/shared/gen/quadsmith/api"
	_ "quadsmith/shared/gen/quadsmith/data"
	_ "quadsmith/shared/gen/quadsmith/ops"
	_ "quadsmith/shared/gen/quadsmith/store"
)

func TestCheckAllProtoDataClassification(t *testing.T) {
	verified, err := CheckAllProtoDataClassification()
	if err != nil {
		t.Fatalf("CheckAllProtoDataClassification failed: %v", err)
	}

	if len(verified) == 0 {
		t.Fatalf("expected verified table data, found 0 tables")
	}

	t.Logf("Successfully verified %d SQL tables for GDPR quadsmith.data coverage:", len(verified))
	for _, v := range verified {
		t.Logf("  ✓ %s (table: %s, domain: %s, fields: %d)", v.MessageName, v.TableName, v.Domain, v.FieldCount)
	}
}
