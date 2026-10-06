package cel2sql_test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"cel.dev/cel-go/cel"
	"quadsmith/api/pkg/cel2sql"
)

func TestCompiler_Extensive(t *testing.T) {
	env, err := cel.NewEnv(
		cel.Variable("motor.kv_rating", cel.IntType),
		cel.Variable("motor.stator_width_mm", cel.DoubleType),
		cel.Variable("frame.is_ducted", cel.BoolType),
		cel.Variable("frame.wheelbase_mm", cel.IntType),
		cel.Variable("vtx.protocol", cel.StringType),
		cel.Variable("mount_patterns", cel.ListType(cel.StringType)),
		cel.Variable("error.trigger", cel.IntType),
		cel.Function("unsupportedFunc",
			cel.MemberOverload("string_unsupportedFunc", []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType),
		),
	)
	if err != nil {
		t.Fatalf("Failed to create CEL env: %v", err)
	}

	compiler := cel2sql.NewCompiler(cel2sql.ConvertOptions{
		FieldMapper: func(path string) (string, error) {
			if path == "error.trigger" {
				return "", errors.New("simulated mapping error")
			}
			return fmt.Sprintf("db_%s", path), nil
		},
	})

	tests := []struct {
		name      string
		cel       string
		wantSql   string
		wantArgs  []any
		wantError bool
	}{
		{"Equals String", `vtx.protocol == "avatar"`, `(db_vtx.protocol = $1)`, []any{"avatar"}, false},
		{"Not Equals Int", `motor.kv_rating != 1800`, `(db_motor.kv_rating != $1)`, []any{int64(1800)}, false},
		{"Greater Than Float", `motor.stator_width_mm > 22.5`, `(db_motor.stator_width_mm > $1)`, []any{float64(22.5)}, false},
		{"Less Than or Equal", `frame.wheelbase_mm <= 250`, `(db_frame.wheelbase_mm <= $1)`, []any{int64(250)}, false},
		{"Boolean True", `frame.is_ducted == true`, `(db_frame.is_ducted = $1)`, []any{true}, false},
		{"Implicit Boolean", `frame.is_ducted`, `db_frame.is_ducted`, nil, false},
		{"Negation", `!frame.is_ducted`, `NOT (db_frame.is_ducted)`, nil, false},
		{"Complex Grouping", `(motor.kv_rating > 2000 && frame.is_ducted) || (frame.wheelbase_mm < 100)`,
			`(((db_motor.kv_rating > $1) AND db_frame.is_ducted) OR (db_frame.wheelbase_mm < $2))`,
			[]any{int64(2000), int64(100)}, false},
		// CEL automatically optimizes double negation !!x to x during AST generation!
		{"Double Negation", `!!frame.is_ducted`, `db_frame.is_ducted`, nil, false},
		{"Addition", `motor.kv_rating + 500 > 2500`, `((db_motor.kv_rating + $1) > $2)`, []any{int64(500), int64(2500)}, false},
		{"Subtraction", `frame.wheelbase_mm - 10 == 90`, `((db_frame.wheelbase_mm - $1) = $2)`, []any{int64(10), int64(90)}, false},
		{"Multiplication", `motor.stator_width_mm * 2.0 < 50.0`, `((db_motor.stator_width_mm * $1) < $2)`, []any{float64(2.0), float64(50.0)}, false},
		{"Division", `motor.kv_rating / 10 >= 100`, `((db_motor.kv_rating / $1) >= $2)`, []any{int64(10), int64(100)}, false},
		// CEL represents unary minus as "-_"
		{"Unary Minus", `-motor.kv_rating < -1000`, `(-(db_motor.kv_rating) < $1)`, []any{int64(-1000)}, false},
		{"IN operator", `"16x16" in mount_patterns`, `$1 = ANY(db_mount_patterns)`, []any{"16x16"}, false},
		{"Unsupported Function", `vtx.protocol.unsupportedFunc("dji")`, "", nil, true},
		{"Mapping Error", `error.trigger == 1`, "", nil, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ast, issues := env.Compile(tc.cel)
			if issues != nil && issues.Err() != nil {
				t.Fatalf("CEL compile error on '%s': %v", tc.cel, issues.Err())
			}

			sql, args, err := compiler.Compile(ast)

			if tc.wantError {
				if err == nil {
					t.Fatalf("Expected error, but compilation succeeded for '%s'", tc.cel)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected compilation error for '%s': %v", tc.cel, err)
			}
			if sql != tc.wantSql {
				t.Errorf("\nExpected SQL: %s\nGot SQL:      %s", tc.wantSql, sql)
			}
			if len(args) != len(tc.wantArgs) {
				t.Fatalf("Expected %d args, got %d", len(tc.wantArgs), len(args))
			}
			for i := range args {
				if !reflect.DeepEqual(args[i], tc.wantArgs[i]) {
					t.Errorf("Arg %d: expected %v (%T), got %v (%T)", i, tc.wantArgs[i], tc.wantArgs[i], args[i], args[i])
				}
			}
		})
	}
}

func TestCompiler_NilSafety(t *testing.T) {
	compiler := cel2sql.NewCompiler(cel2sql.ConvertOptions{})
	_, _, err := compiler.Compile(nil)
	if err == nil {
		t.Errorf("Expected error when passing nil AST")
	}
}
