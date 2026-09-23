package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"regexp"
	"testing"
	"time"

	"chjpcoj.com.innerlamp/pkg/jwt/claims"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for docs/specs/features/jwt-token-uniqueness: every token carries a
// random jti so two tokens minted for the same user in the same second differ.

var jtiPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

const jtiTestSecret = "this-is-a-very-secure-secret-key-32chars"

func newJTITestService(t *testing.T, alg Algorithm) Service {
	t.Helper()

	cfg := &Config{
		Algorithm:              alg,
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test",
		Audience:               "test-users",
	}

	switch alg {
	case AlgorithmHS256:
		cfg.SecretKey = jtiTestSecret
	case AlgorithmRS256:
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		cfg.PrivateKey = key
		cfg.PublicKey = &key.PublicKey
	}

	svc, err := New(cfg)
	require.NoError(t, err)
	return svc
}

// waitForFreshSecond sleeps until shortly after a second boundary so that the
// calls that follow land in the same wall-clock second — the condition under
// which the pre-fix code produced byte-identical tokens.
func waitForFreshSecond() {
	time.Sleep(time.Until(time.Now().Truncate(time.Second).Add(time.Second + 50*time.Millisecond)))
}

func forEachAlgorithm(t *testing.T, fn func(t *testing.T, svc Service)) {
	t.Helper()
	for _, alg := range []Algorithm{AlgorithmHS256, AlgorithmRS256} {
		t.Run(string(alg), func(t *testing.T) {
			fn(t, newJTITestService(t, alg))
		})
	}
}

// AC1: same arguments, same second → different tokens.
func TestGenerate_SameSecondDistinct(t *testing.T) {
	forEachAlgorithm(t, func(t *testing.T, svc Service) {
		waitForFreshSecond()

		r1, err := svc.GenerateRefreshToken("42", nil)
		require.NoError(t, err)
		r2, err := svc.GenerateRefreshToken("42", nil)
		require.NoError(t, err)
		assert.NotEqual(t, r1, r2, "refresh tokens minted in the same second must differ")

		a1, err := svc.GenerateAccessToken("42", "e@x.com", "user", nil)
		require.NoError(t, err)
		a2, err := svc.GenerateAccessToken("42", "e@x.com", "user", nil)
		require.NoError(t, err)
		assert.NotEqual(t, a1, a2, "access tokens minted in the same second must differ")
	})
}

// AC2: jti is present, 32 lowercase hex chars, on both token kinds and both algorithms.
func TestGenerate_HasJTI(t *testing.T) {
	forEachAlgorithm(t, func(t *testing.T, svc Service) {
		access, err := svc.GenerateAccessToken("42", "e@x.com", "user", nil)
		require.NoError(t, err)
		refresh, err := svc.GenerateRefreshToken("42", nil)
		require.NoError(t, err)

		for name, token := range map[string]string{"access": access, "refresh": refresh} {
			parsed, err := svc.ParseToken(token)
			require.NoError(t, err, name)
			assert.Regexp(t, jtiPattern, parsed.ID, "%s token jti", name)
		}
	})
}

// AC3: a failing entropy source yields an error and no token.
func TestGenerate_JTIEntropyFailure(t *testing.T) {
	svc := newJTITestService(t, AlgorithmHS256)

	entropyErr := errors.New("entropy exhausted")
	orig := newTokenID
	newTokenID = func() (string, error) { return "", entropyErr }
	t.Cleanup(func() { newTokenID = orig })

	access, err := svc.GenerateAccessToken("42", "e@x.com", "user", nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, entropyErr)
	assert.Empty(t, access)

	refresh, err := svc.GenerateRefreshToken("42", nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, entropyErr)
	assert.Empty(t, refresh)
}

// AC4: a token minted before this change (no jti) must still validate.
func TestValidateToken_LegacyTokenWithoutJTI(t *testing.T) {
	svc := newJTITestService(t, AlgorithmHS256)

	now := time.Now()
	legacy := &claims.CustomClaims{
		UserID:    "42",
		TokenType: claims.TokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "test",
			Subject:   "42",
			Audience:  jwt.ClaimStrings{"test-users"},
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, legacy).SignedString([]byte(jtiTestSecret))
	require.NoError(t, err)

	got, err := svc.ValidateToken(signed)
	require.NoError(t, err)
	assert.Equal(t, "42", got.UserID)
	assert.Empty(t, got.ID, "a legacy token has no jti")
}

// AC5: ValidateToken and ParseToken expose the jti as claims.ID.
func TestValidateToken_ExposesJTI(t *testing.T) {
	forEachAlgorithm(t, func(t *testing.T, svc Service) {
		token, err := svc.GenerateRefreshToken("42", nil)
		require.NoError(t, err)

		validated, err := svc.ValidateToken(token)
		require.NoError(t, err)
		parsed, err := svc.ParseToken(token)
		require.NoError(t, err)

		assert.Regexp(t, jtiPattern, validated.ID)
		assert.Equal(t, validated.ID, parsed.ID)
	})
}

// AC6: the access token issued by RefreshAccessToken has its own jti.
func TestRefreshAccessToken_NewJTI(t *testing.T) {
	forEachAlgorithm(t, func(t *testing.T, svc Service) {
		refresh, err := svc.GenerateRefreshToken("42", nil)
		require.NoError(t, err)
		refreshClaims, err := svc.ValidateToken(refresh)
		require.NoError(t, err)

		access, err := svc.RefreshAccessToken(refresh, "e@x.com", "user")
		require.NoError(t, err)
		accessClaims, err := svc.ValidateToken(access)
		require.NoError(t, err)

		assert.Regexp(t, jtiPattern, accessClaims.ID)
		assert.NotEqual(t, refreshClaims.ID, accessClaims.ID)
	})
}

// AC7: caller-supplied extra can never override the generated jti.
func TestGenerate_ExtraCannotOverrideJTI(t *testing.T) {
	svc := newJTITestService(t, AlgorithmHS256)
	extra := map[string]any{"jti": "x", "id": "y", "ID": "z"}

	access, err := svc.GenerateAccessToken("42", "e@x.com", "user", extra)
	require.NoError(t, err)
	refresh, err := svc.GenerateRefreshToken("42", extra)
	require.NoError(t, err)

	for name, token := range map[string]string{"access": access, "refresh": refresh} {
		parsed, err := svc.ValidateToken(token)
		require.NoError(t, err, name)
		assert.Regexp(t, jtiPattern, parsed.ID, "%s token jti", name)
		assert.Equal(t, "x", parsed.Extra["jti"], "extra is preserved untouched under the extra claim")
	}
}
