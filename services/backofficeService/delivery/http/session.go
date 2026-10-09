package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	jwtcontext "chjpcoj.com.innerlamp/pkg/jwt/context"
	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/backofficeService/domain"
)

// sessionValidator is the subset of *usecase.AdminAuthUsecase the session guard
// needs, declared here (the consumer) so it can be tested with a mock.
type sessionValidator interface {
	ValidateSession(ctx context.Context, adminID int64, issuedAt time.Time) error
}

// RequireActiveSession wraps next so a token that is validly signed but has been
// revoked (issued before the admin's tokens_valid_after, or the admin is gone or
// deactivated) is refused with 401. It must run inside the auth middleware and
// before RequireRole, so a revoked token never reaches a role decision.
func RequireActiveSession(v sessionValidator, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := jwtcontext.GetClaims(r.Context())
		if !ok || claims.IssuedAt == nil {
			WriteAuthError(w, r, errors.New("no claims or iat in context"))
			return
		}
		adminID, err := strconv.ParseInt(claims.UserID, 10, 64)
		if err != nil {
			WriteAuthError(w, r, err)
			return
		}
		if err := v.ValidateSession(r.Context(), adminID, claims.IssuedAt.Time); err != nil {
			if errors.Is(err, domain.ErrSessionRevoked) {
				WriteAuthError(w, r, err)
				return
			}
			utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)
			return
		}
		next(w, r)
	}
}
