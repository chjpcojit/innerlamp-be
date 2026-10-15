package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	jwtclaims "chjpcoj.com.innerlamp/pkg/jwt/claims"
	jwtcontext "chjpcoj.com.innerlamp/pkg/jwt/context"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

type fakeSessionChecker struct {
	err     error
	gotUser int64
	gotSID  string
	called  bool
}

func (f *fakeSessionChecker) EnsureSessionActive(_ context.Context, userID int64, sid string) error {
	f.called, f.gotUser, f.gotSID = true, userID, sid
	return f.err
}

func runActiveSession(t *testing.T, checker *fakeSessionChecker, claims *jwtclaims.CustomClaims) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	reached := false
	h := NewActiveSession(checker).WrapFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if claims != nil {
		req = req.WithContext(jwtcontext.WithClaims(req.Context(), claims))
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec, reached
}

func claimsWithSID(userID, sid string) *jwtclaims.CustomClaims {
	return &jwtclaims.CustomClaims{
		UserID: userID, TokenType: jwtclaims.TokenTypeAccess,
		Extra:            map[string]any{"sid": sid},
		RegisteredClaims: jwt.RegisteredClaims{},
	}
}

func TestActiveSession(t *testing.T) {
	t.Run("live session reaches the handler and is checked by user_id+sid", func(t *testing.T) {
		c := &fakeSessionChecker{}
		rec, reached := runActiveSession(t, c, claimsWithSID("42", "sid-1"))
		assert.True(t, reached)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, int64(42), c.gotUser)
		assert.Equal(t, "sid-1", c.gotSID)
	})

	t.Run("revoked session is 401 UNAUTHORIZED (AC21)", func(t *testing.T) {
		c := &fakeSessionChecker{err: domain.ErrSessionRevoked}
		rec, reached := runActiveSession(t, c, claimsWithSID("42", "sid-1"))
		assert.False(t, reached)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), "UNAUTHORIZED")
	})

	t.Run("database failure is 503, never let through (AC22)", func(t *testing.T) {
		c := &fakeSessionChecker{err: errors.New("connection refused")}
		rec, reached := runActiveSession(t, c, claimsWithSID("42", "sid-1"))
		assert.False(t, reached)
		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
		assert.NotContains(t, rec.Body.String(), "connection refused")
	})

	t.Run("missing claims, non-numeric user and missing sid are 401 without a lookup", func(t *testing.T) {
		for name, claims := range map[string]*jwtclaims.CustomClaims{
			"no claims":      nil,
			"non-numeric id": claimsWithSID("abc", "sid-1"),
		} {
			c := &fakeSessionChecker{}
			rec, reached := runActiveSession(t, c, claims)
			require.False(t, reached, name)
			assert.Equal(t, http.StatusUnauthorized, rec.Code, name)
			assert.False(t, c.called, name)
		}

		// A token with no sid is passed on as an empty sid; the use case
		// refuses it (covered in usecase tests), so here it must be 401 too.
		c := &fakeSessionChecker{err: domain.ErrSessionRevoked}
		rec, reached := runActiveSession(t, c, &jwtclaims.CustomClaims{UserID: "42", TokenType: jwtclaims.TokenTypeAccess})
		assert.False(t, reached)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
