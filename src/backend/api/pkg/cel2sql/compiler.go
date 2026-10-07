package cel2sql

import (
	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/ast"
	"cel.dev/cel-go/common/types/ref"
	"fmt"
)

type ConvertOptions struct {
	FieldMapper func(path string) (string, error)
}

type Compiler struct {
	options ConvertOptions
}

func NewCompiler(opts ConvertOptions) *Compiler {
	if opts.FieldMapper == nil {
		opts.FieldMapper = func(p string) (string, error) { return p, nil }
	}
	return &Compiler{options: opts}
}

func (c *Compiler) Compile(celAst *cel.Ast) (string, []any, error) {
	if celAst == nil {
		return "", nil, fmt.Errorf("ast is nil")
	}

	nativeAst := celAst.NativeRep()
	if nativeAst == nil {
		return "", nil, fmt.Errorf("native AST is nil")
	}

	expr := nativeAst.Expr()
	var args []any
	sql, err := c.visit(expr, &args)
	if err != nil {
		return "", nil, err
	}

	return sql, args, nil
}

func (c *Compiler) visit(expr ast.Expr, args *[]any) (string, error) {
	switch expr.Kind() {
	case ast.LiteralKind:
		return c.visitLiteral(expr.AsLiteral(), args)
	case ast.IdentKind:
		return c.visitIdent(expr.AsIdent(), args)
	case ast.SelectKind:
		path, err := extractPath(expr)
		if err != nil {
			return "", err
		}
		return c.options.FieldMapper(path)
	case ast.CallKind:
		return c.visitCall(expr.AsCall(), args)
	default:
		return "", fmt.Errorf("unsupported CEL expression kind: %v", expr.Kind())
	}
}

func (c *Compiler) visitLiteral(lit ref.Val, args *[]any) (string, error) {
	*args = append(*args, lit.Value())
	return fmt.Sprintf("$%d", len(*args)), nil
}

func (c *Compiler) visitIdent(ident string, args *[]any) (string, error) {
	return c.options.FieldMapper(ident)
}

func (c *Compiler) visitCall(call ast.CallExpr, args *[]any) (string, error) {
	funcName := call.FunctionName()
	switch funcName {
	case "_&&_":
		return c.visitBinaryOp("AND", call, args)
	case "_||_":
		return c.visitBinaryOp("OR", call, args)
	case "_==_":
		return c.visitBinaryOp("=", call, args)
	case "_!=_":
		return c.visitBinaryOp("!=", call, args)
	case "_<_":
		return c.visitBinaryOp("<", call, args)
	case "_<=_":
		return c.visitBinaryOp("<=", call, args)
	case "_>_":
		return c.visitBinaryOp(">", call, args)
	case "_>=_":
		return c.visitBinaryOp(">=", call, args)
	case "_+_":
		return c.visitBinaryOp("+", call, args)
	case "_-_", "-_":
		// Overloaded: can be binary subtraction or unary negation
		if len(call.Args()) == 1 {
			op, err := c.visit(call.Args()[0], args)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("-(%s)", op), nil
		}
		return c.visitBinaryOp("-", call, args)
	case "_*_":
		return c.visitBinaryOp("*", call, args)
	case "_/_":
		return c.visitBinaryOp("/", call, args)
	case "!_":
		if len(call.Args()) != 1 {
			return "", fmt.Errorf("!_ operator expects 1 argument")
		}
		op, err := c.visit(call.Args()[0], args)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("NOT (%s)", op), nil
	case "@in":
		if len(call.Args()) != 2 {
			return "", fmt.Errorf("@in operator expects 2 arguments")
		}
		left, err := c.visit(call.Args()[0], args)
		if err != nil {
			return "", err
		}
		right, err := c.visit(call.Args()[1], args)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s = ANY(%s)", left, right), nil
	case "endsWith":
		if call.Target() == nil || len(call.Args()) != 1 {
			return "", fmt.Errorf("endsWith requires a target and 1 argument")
		}
		tgt, err := c.visit(call.Target(), args)
		if err != nil {
			return "", err
		}
		arg, err := c.visit(call.Args()[0], args)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s LIKE '%%' || %s", tgt, arg), nil
	case "startsWith":
		if call.Target() == nil || len(call.Args()) != 1 {
			return "", fmt.Errorf("startsWith requires a target and 1 argument")
		}
		tgt, err := c.visit(call.Target(), args)
		if err != nil {
			return "", err
		}
		arg, err := c.visit(call.Args()[0], args)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s LIKE %s || '%%'", tgt, arg), nil
	case "contains":
		if call.Target() == nil || len(call.Args()) != 1 {
			return "", fmt.Errorf("contains requires a target and 1 argument")
		}
		tgt, err := c.visit(call.Target(), args)
		if err != nil {
			return "", err
		}
		arg, err := c.visit(call.Args()[0], args)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s LIKE '%%' || %s || '%%'", tgt, arg), nil
	default:
		return "", fmt.Errorf("unsupported CEL function: %s", funcName)
	}
}

func (c *Compiler) visitBinaryOp(sqlOp string, call ast.CallExpr, args *[]any) (string, error) {
	if len(call.Args()) != 2 {
		return "", fmt.Errorf("operator %s expects 2 arguments", sqlOp)
	}

	left, err := c.visit(call.Args()[0], args)
	if err != nil {
		return "", err
	}

	right, err := c.visit(call.Args()[1], args)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("(%s %s %s)", left, sqlOp, right), nil
}

func extractPath(expr ast.Expr) (string, error) {
	switch expr.Kind() {
	case ast.IdentKind:
		return expr.AsIdent(), nil
	case ast.SelectKind:
		sel := expr.AsSelect()
		operand, err := extractPath(sel.Operand())
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s.%s", operand, sel.FieldName()), nil
	default:
		return "", fmt.Errorf("cannot extract path from kind %v", expr.Kind())
	}
}
