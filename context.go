package idempotency

import "context"

type contextKey string

// idempotencyContextKey defines which key to use for context.Context.
var idempotencyContextKey contextKey = "idempotency-key"

// NewContext returns a new Context that carries value idempotencyKey.
func NewContext(ctx context.Context, idempotencyKey string) context.Context {
	return context.WithValue(ctx, idempotencyContextKey, idempotencyKey)
}

// FromContext returns the Idempotency Key value stored in ctx, if any.
func FromContext(ctx context.Context) (string, bool) {
	key, ok := ctx.Value(idempotencyContextKey).(string)
	return key, ok
}
