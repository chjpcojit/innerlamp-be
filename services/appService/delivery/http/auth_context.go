package http

import (
	"net/http"
	"strconv"

	jwtcontext "chjpcoj.com.innerlamp/pkg/jwt/context"
)

// authenticatedUserID extracts the numeric user ID the auth middleware
// attaches to the request context. US-004 AC6: a request with no (or an
// invalid) authenticated user ID must be rejected as unauthenticated.
func authenticatedUserID(r *http.Request) (int64, bool) {
	idStr, ok := jwtcontext.GetUserID(r.Context())
	if !ok {
		return 0, false
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, false
	}

	return id, true
}
