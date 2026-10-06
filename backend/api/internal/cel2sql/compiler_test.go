package cel2sql

import (
	"reflect"
	"testing"
)

func TestCompile(t *testing.T) {
	tests := []struct {
		name        string
		filter      string
		wantSql     string
		wantArgs    []any
		expectError bool
	}{
		{
			name:     "empty filter",
			filter:   "",
			wantSql:  "",
			wantArgs: nil,
		},
		{
			name:     "simple equality",
			filter:   `manufacturer == "T-Motor"`,
			wantSql:  `(manufacturer = $1)`,
			wantArgs: []any{"T-Motor"},
		},
		{
			name:     "numeric comparison",
			filter:   `weight_g < 30.0`,
			wantSql:  `(weight_g < $1)`,
			wantArgs: []any{30.0},
		},
		{
			name:     "logical AND",
			filter:   `weight_g <= 30.5 && manufacturer == "SpeedyBee"`,
			wantSql:  `((weight_g <= $1) AND (manufacturer = $2))`,
			wantArgs: []any{30.5, "SpeedyBee"},
		},
		{
			name:     "member function startsWith",
			filter:   `model.startsWith("F80")`,
			wantSql:  `(model LIKE $1)`,
			wantArgs: []any{"F80%"},
		},
		{
			name:     "member function contains",
			filter:   `firmware.contains("Bluejay")`,
			wantSql:  `(firmware LIKE $1)`,
			wantArgs: []any{"%Bluejay%"},
		},
		{
			name:     "complex query",
			filter:   `(kv > 1900 && kv < 2500) || (stator_diameter_mm == 22 && stator_height_mm >= 7)`,
			wantSql:  `(((kv > $1) AND (kv < $2)) OR ((stator_diameter_mm = $3) AND (stator_height_mm >= $4)))`,
			wantArgs: []any{int64(1900), int64(2500), int64(22), int64(7)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSql, gotArgs, err := Compile(tt.filter)
			
			if tt.expectError && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			
			if gotSql != tt.wantSql {
				t.Errorf("Compile() gotSql = %v, want %v", gotSql, tt.wantSql)
			}
			if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
				t.Errorf("Compile() gotArgs = %v, want %v", gotArgs, tt.wantArgs)
			}
		})
	}
}
