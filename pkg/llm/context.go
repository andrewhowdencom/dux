package llm

import (
	"context"
	"errors"
)

type contextKey string

const sessionIDKey contextKey = "session_id"

// ErrMissingSessionID is returned when a session ID is not found in the context.
var ErrMissingSessionID = errors.New("missing session ID in context")

// WithSessionID bounds the current execution context to a specific conversation lifecycle.
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey, sessionID)
}

// SessionIDFromContext unpacks the session ID bound to the active context execution flow.
func SessionIDFromContext(ctx context.Context) (string, error) {
	val, ok := ctx.Value(sessionIDKey).(string)
	if !ok || val == "" {
		return "", ErrMissingSessionID
	}
	return val, nil
}
