package ops

import "context"

type OperatorContext struct {
	Operator     string
	OperatorType string // 'HUMAN' or 'ROBOT'
	GroupName    string
	JitGrantID   string
}

type contextKey struct{}

func WithOperatorContext(ctx context.Context, op *OperatorContext) context.Context {
	return context.WithValue(ctx, contextKey{}, op)
}

func OperatorContextFromContext(ctx context.Context) *OperatorContext {
	if v, ok := ctx.Value(contextKey{}).(*OperatorContext); ok {
		return v
	}
	return nil
}
