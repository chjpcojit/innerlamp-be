package context

import (
	"context"

	"base_project/pkg/jwt/claims"
)

// contextKey is a private type for context keys to avoid collisions
type contextKey string

const (
	// claimsContextKey is the key for storing JWT claims in context
	claimsContextKey contextKey = "jwt_claims"

	// userIDContextKey is the key for storing user ID in context
	userIDContextKey contextKey = "jwt_user_id"
)

// WithClaims adds JWT claims to the context
func WithClaims(ctx context.Context, claims *claims.CustomClaims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

// GetClaims retrieves JWT claims from the context
func GetClaims(ctx context.Context) (*claims.CustomClaims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*claims.CustomClaims)
	return claims, ok
}

// WithUserID adds user ID to the context (convenience function)
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDContextKey, userID)
}

// GetUserID retrieves user ID from the context
func GetUserID(ctx context.Context) (string, bool) {
	// Try to get from userID key first
	if userID, ok := ctx.Value(userIDContextKey).(string); ok {
		return userID, true
	}

	// Fall back to getting from claims
	if claims, ok := GetClaims(ctx); ok {
		return claims.UserID, true
	}

	return "", false
}

// MustGetClaims retrieves JWT claims from context or panics
// Use this only when you're certain the claims exist (e.g., after middleware)
func MustGetClaims(ctx context.Context) *claims.CustomClaims {
	claims, ok := GetClaims(ctx)
	if !ok {
		panic("no JWT claims in context")
	}
	return claims
}

// MustGetUserID retrieves user ID from context or panics
// Use this only when you're certain the user ID exists (e.g., after middleware)
func MustGetUserID(ctx context.Context) string {
	userID, ok := GetUserID(ctx)
	if !ok {
		panic("no user ID in context")
	}
	return userID
}
