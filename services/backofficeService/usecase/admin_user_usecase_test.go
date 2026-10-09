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

	"chjpcoj.com.innerlamp/pkg/mailer"
	"chjpcoj.com.innerlamp/services/backofficeService/domain"
)

func newUserUC(repo *domain.MockAdminRepository, m *mailer.MockMailer, fixed string, t *testing.T) *AdminUserUsecase {
	return newUserUCWithTTL(repo, m, fixed, 0, t)
}

func newUserUCWithTTL(repo *domain.MockAdminRepository, m *mailer.MockMailer, fixed string, ttl time.Duration, t *testing.T) *AdminUserUsecase {
	l := newTestLogger()
	l.On("Error", mock.Anything, mock.Anything).Return()
	return NewAdminUserUsecase(repo, m, fixed, ttl, l)
}

func TestAdminUserUsecase_CreateAccount(t *testing.T) {
	ctx := context.Background()
	in := domain.CreateAccountInput{Email: "  New@X.com ", Name: " New Person ", Role: domain.RoleViewer}
	created := &domain.Admin{ID: 9, Email: "new@x.com", Name: "New Person", Role: domain.RoleViewer, MustChangePassword: true}

	t.Run("emails a random password whose hash is what is stored", func(t *testing.T) {
		repo, m := new(domain.MockAdminRepository), new(mailer.MockMailer)
		var hash, emailed string
		repo.On("CreateAccount", mock.Anything, "new@x.com", mock.Anything, "New Person", domain.RoleViewer, mock.Anything).
			Run(func(args mock.Arguments) { hash = args.String(2) }).Return(created, nil)
		m.On("SendAccountInviteEmail", mock.Anything, "new@x.com", "New Person", mock.Anything).
			Run(func(args mock.Arguments) { emailed = args.String(3) }).Return(nil)

		res, err := newUserUC(repo, m, "", t).CreateAccount(ctx, in)

		require.NoError(t, err)
		assert.Equal(t, int64(9), res.Admin.ID)
		assert.GreaterOrEqual(t, len(emailed), 16)
		assert.NotEqual(t, emailed, hash)
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(hash), []byte(emailed)))
	})

	t.Run("random passwords differ per account", func(t *testing.T) {
		repo, m := new(domain.MockAdminRepository), new(mailer.MockMailer)
		repo.On("CreateAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(created, nil)
		var sent []string
		m.On("SendAccountInviteEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) { sent = append(sent, args.String(3)) }).Return(nil)
		uc := newUserUC(repo, m, "", t)
		_, _ = uc.CreateAccount(ctx, in)
		_, _ = uc.CreateAccount(ctx, in)
		require.Len(t, sent, 2)
		assert.NotEqual(t, sent[0], sent[1])
	})

	t.Run("dev fixed password is what gets hashed and emailed", func(t *testing.T) {
		repo, m := new(domain.MockAdminRepository), new(mailer.MockMailer)
		var hash string
		repo.On("CreateAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) { hash = args.String(2) }).Return(created, nil)
		m.On("SendAccountInviteEmail", mock.Anything, "new@x.com", "New Person", "111111").Return(nil)

		_, err := newUserUC(repo, m, "111111", t).CreateAccount(ctx, in)

		require.NoError(t, err)
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(hash), []byte("111111")))
		m.AssertExpectations(t)
	})

	t.Run("rolls the account back when the email cannot be sent", func(t *testing.T) {
		repo, m := new(domain.MockAdminRepository), new(mailer.MockMailer)
		repo.On("CreateAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(created, nil)
		repo.On("DeleteByID", mock.Anything, int64(9)).Return(nil)
		m.On("SendAccountInviteEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("smtp down"))

		_, err := newUserUC(repo, m, "", t).CreateAccount(ctx, in)

		assert.ErrorIs(t, err, domain.ErrInviteEmailFailed)
		repo.AssertCalled(t, "DeleteByID", mock.Anything, int64(9))
	})

	t.Run("rejects invalid input before creating or emailing anything", func(t *testing.T) {
		tests := map[string]domain.CreateAccountInput{
			"bad email":    {Email: "not-an-email", Name: "A", Role: domain.RoleViewer},
			"empty email":  {Name: "A", Role: domain.RoleViewer},
			"empty name":   {Email: "a@x.com", Name: "  ", Role: domain.RoleViewer},
			"unknown role": {Email: "a@x.com", Name: "A", Role: "superuser"},
			"empty role":   {Email: "a@x.com", Name: "A"},
		}
		for name, bad := range tests {
			t.Run(name, func(t *testing.T) {
				repo, m := new(domain.MockAdminRepository), new(mailer.MockMailer)
				_, err := newUserUC(repo, m, "", t).CreateAccount(ctx, bad)
				assert.ErrorIs(t, err, domain.ErrInvalidAccountInput)
				repo.AssertNotCalled(t, "CreateAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				m.AssertNotCalled(t, "SendAccountInviteEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			})
		}
	})

	t.Run("a duplicate email sends nothing", func(t *testing.T) {
		repo, m := new(domain.MockAdminRepository), new(mailer.MockMailer)
		repo.On("CreateAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrEmailTaken)
		_, err := newUserUC(repo, m, "", t).CreateAccount(ctx, in)
		assert.ErrorIs(t, err, domain.ErrEmailTaken)
		m.AssertNotCalled(t, "SendAccountInviteEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("propagates repository failure", func(t *testing.T) {
		repo, m := new(domain.MockAdminRepository), new(mailer.MockMailer)
		boom := errors.New("db down")
		repo.On("CreateAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, boom)
		_, err := newUserUC(repo, m, "", t).CreateAccount(ctx, in)
		assert.ErrorIs(t, err, boom)
	})
}

func TestAdminUserUsecase_TempPasswordExpiry(t *testing.T) {
	ctx := context.Background()
	in := domain.CreateAccountInput{Email: "a@x.com", Name: "A", Role: domain.RoleViewer}

	t.Run("create stamps now+ttl", func(t *testing.T) {
		repo, m := new(domain.MockAdminRepository), new(mailer.MockMailer)
		var got *time.Time
		repo.On("CreateAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) { got = args.Get(5).(*time.Time) }).Return(&domain.Admin{ID: 1}, nil)
		m.On("SendAccountInviteEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		before := time.Now()
		_, err := newUserUCWithTTL(repo, m, "", 24*time.Hour, t).CreateAccount(ctx, in)

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.WithinDuration(t, before.Add(24*time.Hour), *got, 5*time.Second)
	})

	t.Run("ttl 0 means no expiry", func(t *testing.T) {
		repo, m := new(domain.MockAdminRepository), new(mailer.MockMailer)
		repo.On("CreateAccount", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, (*time.Time)(nil)).Return(&domain.Admin{ID: 1}, nil)
		m.On("SendAccountInviteEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		_, err := newUserUC(repo, m, "", t).CreateAccount(ctx, in)
		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

func TestAdminUserUsecase_ResetPassword(t *testing.T) {
	ctx := context.Background()

	t.Run("stores a new hash with a fresh expiry and emails the password", func(t *testing.T) {
		repo, m := new(domain.MockAdminRepository), new(mailer.MockMailer)
		var hash string
		var exp *time.Time
		repo.On("ResetTempPassword", mock.Anything, int64(9), mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) { hash = args.String(2); exp = args.Get(3).(*time.Time) }).
			Return(&domain.Admin{ID: 9, Email: "v@x.com", Name: "V"}, nil)
		m.On("SendAccountInviteEmail", mock.Anything, "v@x.com", "V", "111111").Return(nil)

		res, err := newUserUCWithTTL(repo, m, "111111", 24*time.Hour, t).ResetPassword(ctx, 1, 9)

		require.NoError(t, err)
		assert.Equal(t, int64(9), res.Admin.ID)
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(hash), []byte("111111")))
		require.NotNil(t, exp)
		assert.WithinDuration(t, time.Now().Add(24*time.Hour), *exp, 5*time.Second)
		m.AssertExpectations(t)
	})

	t.Run("refuses to reset yourself", func(t *testing.T) {
		repo, m := new(domain.MockAdminRepository), new(mailer.MockMailer)
		_, err := newUserUC(repo, m, "", t).ResetPassword(ctx, 5, 5)
		assert.ErrorIs(t, err, domain.ErrCannotResetSelf)
		repo.AssertNotCalled(t, "ResetTempPassword", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("unknown account sends nothing", func(t *testing.T) {
		repo, m := new(domain.MockAdminRepository), new(mailer.MockMailer)
		repo.On("ResetTempPassword", mock.Anything, int64(9), mock.Anything, mock.Anything).Return(nil, domain.ErrAdminNotFound)
		_, err := newUserUC(repo, m, "", t).ResetPassword(ctx, 1, 9)
		assert.ErrorIs(t, err, domain.ErrAdminNotFound)
		m.AssertNotCalled(t, "SendAccountInviteEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("email failure is reported", func(t *testing.T) {
		repo, m := new(domain.MockAdminRepository), new(mailer.MockMailer)
		repo.On("ResetTempPassword", mock.Anything, int64(9), mock.Anything, mock.Anything).Return(&domain.Admin{ID: 9, Email: "v@x.com"}, nil)
		m.On("SendAccountInviteEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("smtp down"))
		_, err := newUserUC(repo, m, "", t).ResetPassword(ctx, 1, 9)
		assert.ErrorIs(t, err, domain.ErrInviteEmailFailed)
	})
}
