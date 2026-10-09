package schemacheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSQL_QuadsmithSchema(t *testing.T) {
	// Find schema.sql relative to repo
	possiblePaths := []string{
		"../../../db/schema.sql",
		"../../../../src/backend/db/schema.sql",
		"src/backend/db/schema.sql",
	}

	var schemaPath string
	for _, p := range possiblePaths {
		if _, err := os.Stat(p); err == nil {
			schemaPath = p
			break
		}
	}

	if schemaPath == "" {
		t.Skip("Could not find schema.sql, skipping live schema parse test")
	}

	content, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("Failed to read schema.sql: %v", err)
	}

	schema, err := ParseSQL(string(content))
	if err != nil {
		t.Fatalf("Failed to parse schema.sql: %v", err)
	}

	expectedTables := []string{
		"antennas", "batteries", "builds", "cameras",
		"electronic_speed_controllers", "flight_controllers",
		"frames", "gps_receivers", "motors", "propellers",
		"receivers", "video_transmitters",
	}

	for _, tbl := range expectedTables {
		if _, ok := schema.Tables[tbl]; !ok {
			t.Errorf("Expected table %q not found in parsed schema", tbl)
		}
	}

	// Verify motors table has expected columns
	motors, ok := schema.Tables["motors"]
	if !ok {
		t.Fatal("motors table not found")
	}

	if motors.PrimaryKey != "uuid" {
		t.Errorf("Expected motors primary key to be 'uuid', got %q", motors.PrimaryKey)
	}

	expectedMotorCols := []string{"uuid", "id", "manufacturer", "name", "weight_g", "stator_diameter_mm", "stator_height_mm", "kv", "description", "reference_links", "primary_display_image", "media"}
	for _, c := range expectedMotorCols {
		if _, hasCol := motors.Columns[c]; !hasCol {
			t.Errorf("Expected motors column %q not found", c)
		}
	}

	// Verify Foreign Keys
	if len(schema.ForeignKeys) == 0 {
		t.Errorf("Expected foreign keys to be parsed, found 0")
	}
}

func TestCompare_IdenticalSchemas(t *testing.T) {
	sql := `
CREATE TABLE users (
    uuid UUID NOT NULL,
    PRIMARY KEY (uuid),
    id TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    age INTEGER
);
`
	base, err := ParseSQL(sql)
	if err != nil {
		t.Fatal(err)
	}
	curr, err := ParseSQL(sql)
	if err != nil {
		t.Fatal(err)
	}

	changes := Compare(base, curr)
	if len(changes) != 0 {
		t.Errorf("Expected 0 breaking changes for identical schemas, got %d: %v", len(changes), changes)
	}
}

func TestCompare_SafeAdditions(t *testing.T) {
	baseSQL := `
CREATE TABLE users (
    uuid UUID NOT NULL,
    PRIMARY KEY (uuid),
    id TEXT UNIQUE NOT NULL
);
`
	currSQL := `
CREATE TABLE users (
    uuid UUID NOT NULL,
    PRIMARY KEY (uuid),
    id TEXT UNIQUE NOT NULL,
    email TEXT,
    bio TEXT DEFAULT ''
);
CREATE TABLE products (
    uuid UUID NOT NULL,
    PRIMARY KEY (uuid),
    title TEXT NOT NULL
);
`
	base, err := ParseSQL(baseSQL)
	if err != nil {
		t.Fatal(err)
	}
	curr, err := ParseSQL(currSQL)
	if err != nil {
		t.Fatal(err)
	}

	changes := Compare(base, curr)
	if len(changes) != 0 {
		t.Errorf("Expected 0 breaking changes for safe additions, got %d: %v", len(changes), changes)
	}
}

func TestCompare_TableDropped(t *testing.T) {
	baseSQL := `
CREATE TABLE users (uuid UUID PRIMARY KEY);
CREATE TABLE orders (uuid UUID PRIMARY KEY);
`
	currSQL := `
CREATE TABLE users (uuid UUID PRIMARY KEY);
`
	base, _ := ParseSQL(baseSQL)
	curr, _ := ParseSQL(currSQL)

	changes := Compare(base, curr)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 breaking change, got %d", len(changes))
	}
	if changes[0].Type != ChangeTableDropped || changes[0].Table != "orders" {
		t.Errorf("Expected ChangeTableDropped for orders, got %v", changes[0])
	}
}

func TestCompare_ColumnDropped(t *testing.T) {
	baseSQL := `
CREATE TABLE users (
    uuid UUID PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT
);
`
	currSQL := `
CREATE TABLE users (
    uuid UUID PRIMARY KEY,
    name TEXT NOT NULL
);
`
	base, _ := ParseSQL(baseSQL)
	curr, _ := ParseSQL(currSQL)

	changes := Compare(base, curr)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 breaking change, got %d", len(changes))
	}
	if changes[0].Type != ChangeColumnDropped || changes[0].Column != "email" {
		t.Errorf("Expected ChangeColumnDropped for email, got %v", changes[0])
	}
}

func TestCompare_ColumnTypeChanged(t *testing.T) {
	baseSQL := `
CREATE TABLE items (
    uuid UUID PRIMARY KEY,
    price DECIMAL NOT NULL
);
`
	currSQL := `
CREATE TABLE items (
    uuid UUID PRIMARY KEY,
    price INTEGER NOT NULL
);
`
	base, _ := ParseSQL(baseSQL)
	curr, _ := ParseSQL(currSQL)

	changes := Compare(base, curr)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 breaking change, got %d", len(changes))
	}
	if changes[0].Type != ChangeTypeChanged || changes[0].Column != "price" {
		t.Errorf("Expected ChangeTypeChanged for price, got %v", changes[0])
	}
}

func TestCompare_ColumnMadeNotNull(t *testing.T) {
	baseSQL := `
CREATE TABLE items (
    uuid UUID PRIMARY KEY,
    description TEXT
);
`
	currSQL := `
CREATE TABLE items (
    uuid UUID PRIMARY KEY,
    description TEXT NOT NULL
);
`
	base, _ := ParseSQL(baseSQL)
	curr, _ := ParseSQL(currSQL)

	changes := Compare(base, curr)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 breaking change, got %d", len(changes))
	}
	if changes[0].Type != ChangeColumnMadeNotNull || changes[0].Column != "description" {
		t.Errorf("Expected ChangeColumnMadeNotNull for description, got %v", changes[0])
	}
}

func TestCompare_NewColumnNotNullWithoutDefault(t *testing.T) {
	baseSQL := `
CREATE TABLE items (
    uuid UUID PRIMARY KEY
);
`
	currSQL := `
CREATE TABLE items (
    uuid UUID PRIMARY KEY,
    required_code TEXT NOT NULL
);
`
	base, _ := ParseSQL(baseSQL)
	curr, _ := ParseSQL(currSQL)

	changes := Compare(base, curr)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 breaking change, got %d", len(changes))
	}
	if changes[0].Type != ChangeNewColumnNotNull || changes[0].Column != "required_code" {
		t.Errorf("Expected ChangeNewColumnNotNull for required_code, got %v", changes[0])
	}
}

func TestCompare_UniqueConstraintAdded(t *testing.T) {
	baseSQL := `
CREATE TABLE items (
    uuid UUID PRIMARY KEY,
    sku TEXT NOT NULL
);
`
	currSQL := `
CREATE TABLE items (
    uuid UUID PRIMARY KEY,
    sku TEXT UNIQUE NOT NULL
);
`
	base, _ := ParseSQL(baseSQL)
	curr, _ := ParseSQL(currSQL)

	changes := Compare(base, curr)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 breaking change, got %d", len(changes))
	}
	if changes[0].Type != ChangeUniqueAdded || changes[0].Column != "sku" {
		t.Errorf("Expected ChangeUniqueAdded for sku, got %v", changes[0])
	}
}

func TestCompare_PrimaryKeyModified(t *testing.T) {
	baseSQL := `
CREATE TABLE items (
    uuid UUID NOT NULL,
    PRIMARY KEY (uuid),
    id TEXT NOT NULL
);
`
	currSQL := `
CREATE TABLE items (
    uuid UUID NOT NULL,
    id TEXT NOT NULL,
    PRIMARY KEY (id)
);
`
	base, _ := ParseSQL(baseSQL)
	curr, _ := ParseSQL(currSQL)

	changes := Compare(base, curr)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 breaking change, got %d", len(changes))
	}
	if changes[0].Type != ChangePrimaryKeyModified {
		t.Errorf("Expected ChangePrimaryKeyModified, got %v", changes[0])
	}
}

func TestCompare_ForeignKeyDropped(t *testing.T) {
	baseSQL := `
CREATE TABLE builds (uuid UUID PRIMARY KEY, motor_uuid UUID NOT NULL);
CREATE TABLE motors (uuid UUID PRIMARY KEY);
ALTER TABLE builds ADD CONSTRAINT fk_builds_motor_uuid FOREIGN KEY (motor_uuid) REFERENCES motors (uuid);
`
	currSQL := `
CREATE TABLE builds (uuid UUID PRIMARY KEY, motor_uuid UUID NOT NULL);
CREATE TABLE motors (uuid UUID PRIMARY KEY);
`
	base, _ := ParseSQL(baseSQL)
	curr, _ := ParseSQL(currSQL)

	changes := Compare(base, curr)
	if len(changes) != 1 {
		t.Fatalf("Expected 1 breaking change, got %d", len(changes))
	}
	if changes[0].Type != ChangeForeignKeyDropped {
		t.Errorf("Expected ChangeForeignKeyDropped, got %v", changes[0])
	}
}

func TestBreakingChange_DetailedString(t *testing.T) {
	bc := BreakingChange{
		Type:        ChangeColumnDropped,
		Table:       "motors",
		Column:      "kv",
		Description: `Column "kv" was dropped from table "motors".`,
		Impact:      "Permanently deletes data stored in this column across all records. Breaks queries selecting this column.",
		Remediation: "Retain the column, deprecate it in proto, or write a pre-migration script in src/backend/db/migrations/ to archive data.",
		Severity:    SeverityBreaking,
	}

	detail := bc.DetailedString()
	if !strings.Contains(detail, "[BREAKING] COLUMN_DROPPED (motors.kv)") {
		t.Errorf("Unexpected header in DetailedString: %s", detail)
	}
	if !strings.Contains(detail, "• Impact: Permanently deletes data") {
		t.Errorf("Missing impact in DetailedString: %s", detail)
	}
	if !strings.Contains(detail, "• Remediation: Retain the column") {
		t.Errorf("Missing remediation in DetailedString: %s", detail)
	}
}

// Ensure unused import compiles cleanly
var _ = filepath.Join
