package cel2sql

import (
	"fmt"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/ast"
)

// Compile parses a CEL expression and generates a SQL WHERE clause and its arguments.
// Examples:
// - `weight_g < 30.0` => `(weight_g < $1)`, [30.0]
// - `manufacturer == "T-Motor"` => `(manufacturer = $1)`, ["T-Motor"]
func Compile(filter string) (string, []any, error) {
	if filter == "" {
		return "", nil, nil
	}

	env, err := cel.NewEnv()
	if err != nil {
		return "", nil, err
	}

	parsed, issues := env.Parse(filter)
	if issues != nil && issues.Err() != nil {
		return "", nil, issues.Err()
	}

	return walk(parsed.NativeRep().Expr(), 1)
}

func walk(node ast.Expr, argIdx int) (string, []any, error) {
	switch node.Kind() {
	case ast.IdentKind:
		ident := node.AsIdent()
		// Basic sanitization to prevent SQL injection via identifier names
		for _, c := range ident {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
				return "", nil, fmt.Errorf("invalid identifier: %s", ident)
			}
		}
		return ident, nil, nil

	case ast.LiteralKind:
		val := node.AsLiteral().Value()
		return fmt.Sprintf("$%d", argIdx), []any{val}, nil

	case ast.CallKind:
		call := node.AsCall()

		if call.IsMemberFunction() {
			target, targetArgs, err := walk(call.Target(), argIdx)
			if err != nil {
				return "", nil, err
			}

			if len(call.Args()) != 1 {
				return "", nil, fmt.Errorf("unsupported member function arg count: %s", call.FunctionName())
			}

			argNode := call.Args()[0]
			if argNode.Kind() != ast.LiteralKind {
				return "", nil, fmt.Errorf("function arguments must be literals")
			}
			val := argNode.AsLiteral().Value()

			nextArgIdx := argIdx + len(targetArgs)
			switch call.FunctionName() {
			case "contains":
				return fmt.Sprintf("(%s LIKE $%d)", target, nextArgIdx), append(targetArgs, fmt.Sprintf("%%%v%%", val)), nil
			case "startsWith":
				return fmt.Sprintf("(%s LIKE $%d)", target, nextArgIdx), append(targetArgs, fmt.Sprintf("%v%%", val)), nil
			case "endsWith":
				return fmt.Sprintf("(%s LIKE $%d)", target, nextArgIdx), append(targetArgs, fmt.Sprintf("%%%v", val)), nil
			default:
				return "", nil, fmt.Errorf("unsupported member function: %s", call.FunctionName())
			}
		}

		if len(call.Args()) == 2 {
			lhs, lhsArgs, err := walk(call.Args()[0], argIdx)
			if err != nil {
				return "", nil, err
			}

			rhs, rhsArgs, err := walk(call.Args()[1], argIdx+len(lhsArgs))
			if err != nil {
				return "", nil, err
			}

			op := ""
			switch call.FunctionName() {
			case "_==_":
				op = "="
			case "_!=_":
				op = "!="
			case "_<_":
				op = "<"
			case "_<=_":
				op = "<="
			case "_>_":
				op = ">"
			case "_>=_":
				op = ">="
			case "_&&_":
				op = "AND"
			case "_||_":
				op = "OR"
			default:
				return "", nil, fmt.Errorf("unsupported binary function: %s", call.FunctionName())
			}

			sql := fmt.Sprintf("(%s %s %s)", lhs, op, rhs)
			args := append(lhsArgs, rhsArgs...)
			return sql, args, nil
		}

		if len(call.Args()) == 1 && call.FunctionName() == "!_" {
			expr, args, err := walk(call.Args()[0], argIdx)
			if err != nil {
				return "", nil, err
			}
			return fmt.Sprintf("NOT %s", expr), args, nil
		}

		return "", nil, fmt.Errorf("unsupported call: %s", call.FunctionName())

	default:
		return "", nil, fmt.Errorf("unsupported AST node kind: %v", node.Kind())
	}
}
