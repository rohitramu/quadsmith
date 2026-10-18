package cel2sql

import (
	"fmt"
	"sort"
	"strings"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/ast"
)

// CompileFilter parses and validates a CEL filter expression against allowedFields,
// returning a predicate function that evaluates a record map.
func CompileFilter(filter string, allowedFields map[string]bool) (func(map[string]any) bool, error) {
	trimmed := strings.TrimSpace(filter)
	if trimmed == "" {
		return func(_ map[string]any) bool { return true }, nil
	}

	env, err := cel.NewEnv()
	if err != nil {
		return nil, err
	}

	parsed, issues := env.Parse(trimmed)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("invalid CEL filter syntax: %w", issues.Err())
	}

	root := parsed.NativeRep().Expr()
	if err := validateExpr(root, allowedFields); err != nil {
		return nil, err
	}

	return func(rec map[string]any) bool {
		val, err := evalExpr(root, rec)
		if err != nil {
			return false
		}
		b, ok := val.(bool)
		return ok && b
	}, nil
}

func validateExpr(node ast.Expr, allowedFields map[string]bool) error {
	switch node.Kind() {
	case ast.LiteralKind:
		return nil
	case ast.IdentKind:
		ident := node.AsIdent()
		if len(allowedFields) > 0 && !allowedFields[ident] {
			return fmt.Errorf("unknown filter field %q", ident)
		}
		return nil
	case ast.SelectKind:
		path, err := extractPath(node)
		if err != nil {
			return err
		}
		if len(allowedFields) > 0 && !allowedFields[path] {
			return fmt.Errorf("unknown filter field %q", path)
		}
		return nil
	case ast.ListKind:
		list := node.AsList()
		for _, elem := range list.Elements() {
			if err := validateExpr(elem, allowedFields); err != nil {
				return err
			}
		}
		return nil
	case ast.CallKind:
		call := node.AsCall()
		if call.IsMemberFunction() {
			if err := validateExpr(call.Target(), allowedFields); err != nil {
				return err
			}
			switch call.FunctionName() {
			case "contains", "startsWith", "endsWith":
				if len(call.Args()) != 1 {
					return fmt.Errorf("%s() requires 1 argument", call.FunctionName())
				}
				return validateExpr(call.Args()[0], allowedFields)
			default:
				return fmt.Errorf("unsupported member function %q", call.FunctionName())
			}
		}

		for _, arg := range call.Args() {
			if err := validateExpr(arg, allowedFields); err != nil {
				return err
			}
		}

		switch call.FunctionName() {
		case "_&&_", "_||_", "_==_", "_!=_", "_<_", "_<=_", "_>_", "_>=_", "@in", "_+_", "_-_", "_*_", "_/_":
			if len(call.Args()) != 2 && !(call.FunctionName() == "_-_" && len(call.Args()) == 1) {
				return fmt.Errorf("operator %s expects 2 arguments", call.FunctionName())
			}
			return nil
		case "!_", "-_":
			if len(call.Args()) != 1 {
				return fmt.Errorf("operator %s expects 1 argument", call.FunctionName())
			}
			return nil
		default:
			return fmt.Errorf("unsupported CEL function %q", call.FunctionName())
		}
	default:
		return fmt.Errorf("unsupported CEL expression kind: %v", node.Kind())
	}
}

func evalExpr(node ast.Expr, rec map[string]any) (any, error) {
	switch node.Kind() {
	case ast.LiteralKind:
		return node.AsLiteral().Value(), nil
	case ast.IdentKind:
		return rec[node.AsIdent()], nil
	case ast.SelectKind:
		path, err := extractPath(node)
		if err != nil {
			return nil, err
		}
		return rec[path], nil
	case ast.ListKind:
		var out []any
		for _, elem := range node.AsList().Elements() {
			v, err := evalExpr(elem, rec)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	case ast.CallKind:
		call := node.AsCall()
		if call.IsMemberFunction() {
			targetVal, err := evalExpr(call.Target(), rec)
			if err != nil {
				return nil, err
			}
			argVal, err := evalExpr(call.Args()[0], rec)
			if err != nil {
				return nil, err
			}
			s := fmt.Sprintf("%v", targetVal)
			sub := fmt.Sprintf("%v", argVal)
			switch call.FunctionName() {
			case "contains":
				return strings.Contains(s, sub), nil
			case "startsWith":
				return strings.HasPrefix(s, sub), nil
			case "endsWith":
				return strings.HasSuffix(s, sub), nil
			}
		}

		fn := call.FunctionName()
		if fn == "!_" {
			v, err := evalExpr(call.Args()[0], rec)
			if err != nil {
				return nil, err
			}
			b, ok := v.(bool)
			if !ok {
				return false, nil
			}
			return !b, nil
		}
		if (fn == "-_" || fn == "_-_") && len(call.Args()) == 1 {
			v, err := evalExpr(call.Args()[0], rec)
			if err != nil {
				return nil, err
			}
			f, ok := toFloat64(v)
			if !ok {
				return 0.0, nil
			}
			return -f, nil
		}

		if fn == "_&&_" {
			l, err := evalExpr(call.Args()[0], rec)
			if err != nil {
				return false, err
			}
			lb, _ := l.(bool)
			if !lb {
				return false, nil
			}
			r, err := evalExpr(call.Args()[1], rec)
			if err != nil {
				return false, err
			}
			rb, _ := r.(bool)
			return rb, nil
		}
		if fn == "_||_" {
			l, err := evalExpr(call.Args()[0], rec)
			if err != nil {
				return false, err
			}
			lb, _ := l.(bool)
			if lb {
				return true, nil
			}
			r, err := evalExpr(call.Args()[1], rec)
			if err != nil {
				return false, err
			}
			rb, _ := r.(bool)
			return rb, nil
		}

		lhs, err := evalExpr(call.Args()[0], rec)
		if err != nil {
			return nil, err
		}
		rhs, err := evalExpr(call.Args()[1], rec)
		if err != nil {
			return nil, err
		}

		switch fn {
		case "@in":
			slice, ok := rhs.([]any)
			if !ok {
				return false, nil
			}
			for _, item := range slice {
				if compareValues(lhs, item) == 0 {
					return true, nil
				}
			}
			return false, nil
		case "_==_":
			return compareValues(lhs, rhs) == 0, nil
		case "_!=_":
			return compareValues(lhs, rhs) != 0, nil
		case "_<_":
			return compareValues(lhs, rhs) < 0, nil
		case "_<=_":
			return compareValues(lhs, rhs) <= 0, nil
		case "_>_":
			return compareValues(lhs, rhs) > 0, nil
		case "_>=_":
			return compareValues(lhs, rhs) >= 0, nil
		case "_+_":
			lf, lok := toFloat64(lhs)
			rf, rok := toFloat64(rhs)
			if lok && rok {
				return lf + rf, nil
			}
			return fmt.Sprintf("%v%v", lhs, rhs), nil
		case "_-_":
			lf, _ := toFloat64(lhs)
			rf, _ := toFloat64(rhs)
			return lf - rf, nil
		case "_*_":
			lf, _ := toFloat64(lhs)
			rf, _ := toFloat64(rhs)
			return lf * rf, nil
		case "_/_":
			lf, _ := toFloat64(lhs)
			rf, _ := toFloat64(rhs)
			if rf == 0 {
				return 0.0, nil
			}
			return lf / rf, nil
		}
	}
	return nil, fmt.Errorf("unsupported expression")
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

func compareValues(a, b any) int {
	af, aNum := toFloat64(a)
	bf, bNum := toFloat64(b)
	if aNum && bNum {
		if af < bf {
			return -1
		}
		if af > bf {
			return 1
		}
		return 0
	}

	ab, aBool := a.(bool)
	bb, bBool := b.(bool)
	if aBool && bBool {
		if ab == bb {
			return 0
		}
		if !ab && bb {
			return -1
		}
		return 1
	}

	as := fmt.Sprintf("%v", a)
	bs := fmt.Sprintf("%v", b)
	return strings.Compare(as, bs)
}

// FilterAndSort filters a slice of items using a CEL expression and sorts them using sortSpecs
// (where each spec is "field" for ascending or "^field" / "-field" for descending).
func FilterAndSort[T any](
	items []T,
	filter string,
	sortSpecs []string,
	allowedFields map[string]bool,
	toRecord func(T) map[string]any,
) ([]T, error) {
	predicate, err := CompileFilter(filter, allowedFields)
	if err != nil {
		return nil, err
	}

	type parsedSort struct {
		field string
		desc  bool
	}
	var sorts []parsedSort
	for _, rawSpec := range sortSpecs {
		for _, part := range strings.Split(rawSpec, ",") {
			s := strings.TrimSpace(part)
			if s == "" {
				continue
			}
			desc := false
			if strings.HasPrefix(s, "^") || strings.HasPrefix(s, "-") {
				desc = true
				s = strings.TrimSpace(s[1:])
			}
			if len(allowedFields) > 0 && !allowedFields[s] {
				return nil, fmt.Errorf("invalid sort column %q", s)
			}
			sorts = append(sorts, parsedSort{field: s, desc: desc})
		}
	}

	var filtered []T
	for _, item := range items {
		rec := toRecord(item)
		if predicate(rec) {
			filtered = append(filtered, item)
		}
	}

	if len(sorts) > 0 {
		sort.SliceStable(filtered, func(i, j int) bool {
			recI := toRecord(filtered[i])
			recJ := toRecord(filtered[j])
			for _, sp := range sorts {
				cmp := compareValues(recI[sp.field], recJ[sp.field])
				if cmp != 0 {
					if sp.desc {
						return cmp > 0
					}
					return cmp < 0
				}
			}
			return false
		})
	}

	return filtered, nil
}
