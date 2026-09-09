//go:build integration

package auth

import "context"

// WithSessionContextForIntegration attaches the same immutable session shape
// produced by RequireAuthenticated. It exists only in integration builds so
// production code cannot manufacture an authenticated request context.
func WithSessionContextForIntegration(ctx context.Context, session SessionContext) context.Context {
	return context.WithValue(ctx, requestSessionContextKey, session)
}
