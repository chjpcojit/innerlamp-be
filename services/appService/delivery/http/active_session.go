package http

import (
	"chjpcoj.com.innerlamp/services/appService/domain"
	"context"
	"errors"
	"net/http"
	"strconv"

	jwtcontext "chjpcoj.com.innerlamp/pkg/jwt/context"
	"chjpcoj.com.innerlamp/pkg/utils"
)

// SessionChecker is the slice of LoginUsecase the middleware needs.
type SessionChecker interface {
	EnsureSessionActive(ctx context.Context, userID int64, sessionID string) error
}

// ActiveSession is login Step A-2: it sits INSIDE the access-token middleware
// and refuses a validly signed token whose session was revoked or expired
// (AC21), so logout and password reset take effect on the next request.
//
// Not applied to POST /v1/auth/logout: logout is idempotent (AC14) and must
// answer 200 for an already-ended session.
type ActiveSession struct {
	checker SessionChecker
}

func NewActiveSession(checker SessionChecker) *ActiveSession {
	return &ActiveSession{checker: checker}
}

// WrapFunc guards next: 401 UNAUTHORIZED when the session is not live, 503 when
// the check cannot reach the database (AC22) — never "let it through".
func (a *ActiveSession) WrapFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := jwtcontext.GetClaims(r.Context())
		if !ok {
			WriteAuthError(w, r, errors.New("no claims in context"))
			return
		}
		userID, err := strconv.ParseInt(claims.UserID, 10, 64)
		if err != nil {
			WriteAuthError(w, r, errors.New("user_id claim is not numeric"))
			return
		}
		sessionID, _ := claims.Extra["sid"].(string)

		switch err := a.checker.EnsureSessionActive(r.Context(), userID, sessionID); {
		case err == nil:
			next(w, r)
		case errors.Is(err, domain.ErrSessionRevoked):
			WriteAuthError(w, r, err)
		default:
			utils.WriteErrorResponse(w, http.StatusServiceUnavailable, utils.ErrorTypeSystem,
				"SERVICE_UNAVAILABLE", "service temporarily unavailable", nil)
		}
	}
}
