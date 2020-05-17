package logger

import "context"

type contextKey struct{}

// ToContext instruments context with logger
func ToContext(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}

// FromContext extracts logger from context or creates a default logger
// if context doesn't contain one
func FromContext(ctx context.Context) Logger {
	l := ctx.Value(contextKey{})
	if l == nil {
		return DefaultLogger
	}

	return l.(Logger)
}
