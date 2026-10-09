package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"chjpcoj.com.innerlamp/pkg/jwt"
	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/services/backofficeService/domain"
)

func newTestLogger() *logging.MockLogger {
	l := &logging.MockLogger{}
	l.On("Info", mock.Anything, mock.Anything).Return()
	l.On("Warn", mock.Anything, mock.Anything).Return()
	return l
}

func newTestJWT(t *testing.T) jwt.Service {
	t.Helper()
	svc, err := jwt.New(&jwt.Config{
		Algorithm: jwt.AlgorithmHS256, SecretKey: "test-secret-key-at-least-32-characters!",
		AccessTokenExpiration: time.Minute, RefreshTokenExpiration: time.Hour,
		Issuer: "test", Audience: "test_backoffice",
	})
	require.NoError(t, err)
	return svc
}

func adminWith(t *testing.T, password string, active bool) *domain.Admin {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	return &domain.Admin{ID: 7, Email: "a@x.com", PasswordHash: string(hash), Name: "A", Role: domain.RoleAdmin, IsActive: active}
}

func TestAdminAuthUsecase_Login(t *testing.T) {
	ctx := context.Background()

	t.Run("issues an access token carrying the admin role", func(t *testing.T) {
		repo := new(domain.MockAdminRepository)
		repo.On("FindByEmail", mock.Anything, "a@x.com").Return(adminWith(t, "correct-horse-1", true), nil)
		repo.On("TouchLastLogin", mock.Anything, int64(7), mock.Anything).Return(nil)
		jwtSvc := newTestJWT(t)
		uc := NewAdminAuthUsecase(repo, jwtSvc, newTestLogger())

		res, err := uc.Login(ctx, domain.AdminLoginInput{Email: "  A@X.com ", Password: "correct-horse-1"})

		require.NoError(t, err)
		claims, err := jwtSvc.ValidateToken(res.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, "7", claims.UserID)
		assert.Equal(t, domain.RoleAdmin, claims.Role)
		repo.AssertExpectations(t)
	})

	t.Run("still succeeds when the last-login write fails", func(t *testing.T) {
		repo := new(domain.MockAdminRepository)
		repo.On("FindByEmail", mock.Anything, "a@x.com").Return(adminWith(t, "correct-horse-1", true), nil)
		repo.On("TouchLastLogin", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("db down"))

		_, err := NewAdminAuthUsecase(repo, newTestJWT(t), newTestLogger()).
			Login(ctx, domain.AdminLoginInput{Email: "a@x.com", Password: "correct-horse-1"})

		assert.NoError(t, err)
	})

	t.Run("rejects every bad credential the same way", func(t *testing.T) {
		tests := []struct {
			name  string
			setup func(*domain.MockAdminRepository)
			in    domain.AdminLoginInput
		}{
			{"wrong password", func(r *domain.MockAdminRepository) {
				r.On("FindByEmail", mock.Anything, "a@x.com").Return(adminWith(t, "correct-horse-1", true), nil)
			}, domain.AdminLoginInput{Email: "a@x.com", Password: "wrong"}},
			{"unknown email", func(r *domain.MockAdminRepository) {
				r.On("FindByEmail", mock.Anything, "a@x.com").Return(nil, domain.ErrAdminNotFound)
			}, domain.AdminLoginInput{Email: "a@x.com", Password: "whatever"}},
			{"inactive account", func(r *domain.MockAdminRepository) {
				r.On("FindByEmail", mock.Anything, "a@x.com").Return(adminWith(t, "correct-horse-1", false), nil)
			}, domain.AdminLoginInput{Email: "a@x.com", Password: "correct-horse-1"}},
			{"empty password", func(*domain.MockAdminRepository) {}, domain.AdminLoginInput{Email: "a@x.com"}},
			{"empty email", func(*domain.MockAdminRepository) {}, domain.AdminLoginInput{Password: "x"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repo := new(domain.MockAdminRepository)
				tt.setup(repo)

				_, err := NewAdminAuthUsecase(repo, newTestJWT(t), newTestLogger()).Login(ctx, tt.in)

				assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
				repo.AssertNotCalled(t, "TouchLastLogin", mock.Anything, mock.Anything, mock.Anything)
			})
		}
	})

	t.Run("propagates repository failures as non-credential errors", func(t *testing.T) {
		repo := new(domain.MockAdminRepository)
		boom := errors.New("db down")
		repo.On("FindByEmail", mock.Anything, "a@x.com").Return(nil, boom)

		_, err := NewAdminAuthUsecase(repo, newTestJWT(t), newTestLogger()).
			Login(ctx, domain.AdminLoginInput{Email: "a@x.com", Password: "x"})

		assert.ErrorIs(t, err, boom)
		assert.NotErrorIs(t, err, domain.ErrInvalidCredentials)
	})
}

func TestAdminAuthUsecase_LoginMustChangePassword(t *testing.T) {
	repo := new(domain.MockAdminRepository)
	a := adminWith(t, "admin", true)
	a.MustChangePassword = true
	repo.On("FindByEmail", mock.Anything, "a@x.com").Return(a, nil)
	repo.On("TouchLastLogin", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	jwtSvc := newTestJWT(t)

	res, err := NewAdminAuthUsecase(repo, jwtSvc, newTestLogger()).
		Login(context.Background(), domain.AdminLoginInput{Email: "a@x.com", Password: "admin"})

	require.NoError(t, err)
	assert.True(t, res.Admin.MustChangePassword)
	claims, err := jwtSvc.ValidateToken(res.AccessToken)
	require.NoError(t, err)
	v, ok := claims.GetExtra(domain.ExtraMustChangePassword)
	assert.True(t, ok)
	assert.Equal(t, true, v)
}

func TestAdminAuthUsecase_ChangePassword(t *testing.T) {
	ctx := context.Background()
	const newPw = "a-much-better-password"

	t.Run("stores the new hash and returns an unrestricted token", func(t *testing.T) {
		repo := new(domain.MockAdminRepository)
		a := adminWith(t, "admin", true)
		a.MustChangePassword = true
		repo.On("FindByID", mock.Anything, int64(7)).Return(a, nil)
		var stored string
		repo.On("UpdatePassword", mock.Anything, int64(7), mock.Anything).
			Run(func(args mock.Arguments) { stored = args.String(2) }).Return(nil)
		jwtSvc := newTestJWT(t)

		res, err := NewAdminAuthUsecase(repo, jwtSvc, newTestLogger()).
			ChangePassword(ctx, domain.ChangePasswordInput{AdminID: 7, CurrentPassword: "admin", NewPassword: newPw})

		require.NoError(t, err)
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(stored), []byte(newPw)))
		claims, err := jwtSvc.ValidateToken(res.AccessToken)
		require.NoError(t, err)
		_, restricted := claims.GetExtra(domain.ExtraMustChangePassword)
		assert.False(t, restricted)
	})

	t.Run("rejects", func(t *testing.T) {
		tests := []struct {
			name string
			in   domain.ChangePasswordInput
			want error
		}{
			{"wrong current password", domain.ChangePasswordInput{AdminID: 7, CurrentPassword: "nope", NewPassword: newPw}, domain.ErrWrongCurrentPassword},
			{"short new password", domain.ChangePasswordInput{AdminID: 7, CurrentPassword: "admin", NewPassword: "short"}, domain.ErrWeakPassword},
			{"new equals current", domain.ChangePasswordInput{AdminID: 7, CurrentPassword: "admin", NewPassword: "admin"}, domain.ErrWeakPassword},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repo := new(domain.MockAdminRepository)
				repo.On("FindByID", mock.Anything, int64(7)).Return(adminWith(t, "admin", true), nil)

				_, err := NewAdminAuthUsecase(repo, newTestJWT(t), newTestLogger()).ChangePassword(ctx, tt.in)

				assert.ErrorIs(t, err, tt.want)
				repo.AssertNotCalled(t, "UpdatePassword", mock.Anything, mock.Anything, mock.Anything)
			})
		}
	})

	t.Run("unknown or inactive admin is invalid credentials", func(t *testing.T) {
		repo := new(domain.MockAdminRepository)
		repo.On("FindByID", mock.Anything, int64(7)).Return(nil, domain.ErrAdminNotFound)
		_, err := NewAdminAuthUsecase(repo, newTestJWT(t), newTestLogger()).
			ChangePassword(ctx, domain.ChangePasswordInput{AdminID: 7, CurrentPassword: "admin", NewPassword: newPw})
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)

		repo = new(domain.MockAdminRepository)
		repo.On("FindByID", mock.Anything, int64(7)).Return(adminWith(t, "admin", false), nil)
		_, err = NewAdminAuthUsecase(repo, newTestJWT(t), newTestLogger()).
			ChangePassword(ctx, domain.ChangePasswordInput{AdminID: 7, CurrentPassword: "admin", NewPassword: newPw})
		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	})
}

func TestAdminAuthUsecase_EnsureBootstrapAdmin(t *testing.T) {
	ctx := context.Background()
	in := domain.BootstrapAdminInput{Email: " Admin ", Password: "admin", Name: "Administrator"}

	t.Run("creates with a hashed password under the normalized email", func(t *testing.T) {
		repo := new(domain.MockAdminRepository)
		var hash string
		repo.On("CreateBootstrapIfEmpty", mock.Anything, "admin", mock.Anything, "Administrator").
			Run(func(args mock.Arguments) { hash = args.String(2) }).Return(true, nil)

		require.NoError(t, NewAdminAuthUsecase(repo, newTestJWT(t), newTestLogger()).EnsureBootstrapAdmin(ctx, in))

		assert.NotEqual(t, "admin", hash)
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(hash), []byte("admin")))
	})

	t.Run("no-op when admins already exist", func(t *testing.T) {
		repo := new(domain.MockAdminRepository)
		repo.On("CreateBootstrapIfEmpty", mock.Anything, "admin", mock.Anything, "Administrator").Return(false, nil)
		assert.NoError(t, NewAdminAuthUsecase(repo, newTestJWT(t), newTestLogger()).EnsureBootstrapAdmin(ctx, in))
	})

	t.Run("propagates repository failure", func(t *testing.T) {
		repo := new(domain.MockAdminRepository)
		boom := errors.New("db down")
		repo.On("CreateBootstrapIfEmpty", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false, boom)
		assert.ErrorIs(t, NewAdminAuthUsecase(repo, newTestJWT(t), newTestLogger()).EnsureBootstrapAdmin(ctx, in), boom)
	})

	t.Run("requires email and password", func(t *testing.T) {
		repo := new(domain.MockAdminRepository)
		uc := NewAdminAuthUsecase(repo, newTestJWT(t), newTestLogger())
		assert.Error(t, uc.EnsureBootstrapAdmin(ctx, domain.BootstrapAdminInput{Password: "x"}))
		assert.Error(t, uc.EnsureBootstrapAdmin(ctx, domain.BootstrapAdminInput{Email: "admin"}))
	})
}

func TestAdminAuthUsecase_ValidateSession(t *testing.T) {
	ctx := context.Background()
	cutoff := time.Date(2026, 9, 29, 12, 0, 0, 700_000_000, time.UTC)
	withCutoff := func(active bool, at *time.Time) *domain.Admin {
		a := adminWith(t, "x", active)
		a.TokensValidAfter = at
		return a
	}

	tests := []struct {
		name     string
		admin    *domain.Admin
		findErr  error
		issuedAt time.Time
		want     error
	}{
		{"no cut-off", withCutoff(true, nil), nil, cutoff.Add(-time.Hour), nil},
		{"issued before cut-off is revoked", withCutoff(true, &cutoff), nil, cutoff.Add(-time.Second).Truncate(time.Second), domain.ErrSessionRevoked},
		{"issued in the cut-off second survives", withCutoff(true, &cutoff), nil, cutoff.Truncate(time.Second), nil},
		{"issued after cut-off", withCutoff(true, &cutoff), nil, cutoff.Add(time.Minute), nil},
		{"inactive admin", withCutoff(false, nil), nil, cutoff, domain.ErrSessionRevoked},
		{"admin gone", nil, domain.ErrAdminNotFound, cutoff, domain.ErrSessionRevoked},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(domain.MockAdminRepository)
			repo.On("FindByID", mock.Anything, int64(7)).Return(tt.admin, tt.findErr)

			err := NewAdminAuthUsecase(repo, newTestJWT(t), newTestLogger()).ValidateSession(ctx, 7, tt.issuedAt)

			if tt.want == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.want)
			}
		})
	}

	t.Run("repository failure is not a revocation", func(t *testing.T) {
		repo := new(domain.MockAdminRepository)
		boom := errors.New("db down")
		repo.On("FindByID", mock.Anything, int64(7)).Return(nil, boom)
		err := NewAdminAuthUsecase(repo, newTestJWT(t), newTestLogger()).ValidateSession(ctx, 7, cutoff)
		assert.ErrorIs(t, err, boom)
		assert.NotErrorIs(t, err, domain.ErrSessionRevoked)
	})
}

func TestAdminAuthUsecase_LoginTempPasswordExpiry(t *testing.T) {
	ctx := context.Background()
	past, future := time.Now().Add(-time.Minute), time.Now().Add(time.Hour)
	login := func(t *testing.T, must bool, exp *time.Time, password string) error {
		repo := new(domain.MockAdminRepository)
		a := adminWith(t, "111111", true)
		a.MustChangePassword, a.TempPasswordExpiresAt = must, exp
		repo.On("FindByEmail", mock.Anything, "a@x.com").Return(a, nil)
		repo.On("TouchLastLogin", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		_, err := NewAdminAuthUsecase(repo, newTestJWT(t), newTestLogger()).
			Login(ctx, domain.AdminLoginInput{Email: "a@x.com", Password: password})
		return err
	}

	assert.NoError(t, login(t, true, &future, "111111"), "not yet expired")
	assert.NoError(t, login(t, true, nil, "111111"), "no expiry (bootstrap admin)")
	assert.NoError(t, login(t, false, &past, "111111"), "expiry ignored once the admin chose their own password")
	assert.ErrorIs(t, login(t, true, &past, "111111"), domain.ErrPasswordExpired)
	err := login(t, true, &past, "wrong-guess")
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials, "a wrong guess must not learn the password had expired")
	assert.NotErrorIs(t, err, domain.ErrPasswordExpired)
}
