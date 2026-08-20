package usecase

import (
	"context"
	"errors"
	"testing"

	"chjpcoj.com.innerlamp/pkg/mailer"
	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/mock"
)

func newPasswordResetUsecase(users domain.UserRepository, m mailer.Mailer, token string) *PasswordResetUsecase {
	uc := NewPasswordResetUsecase(users, m, newLogger())
	uc.newResetToken = func() (string, error) { return token, nil }
	return uc
}

// TestForgotPassword_Success covers AC1: an existing email gets a token
// persisted and a reset email sent with the raw (unhashed) token.
func TestForgotPassword_Success(t *testing.T) {
	repo := &domain.MockUserRepository{}
	mailerMock := &mailer.MockMailer{}

	storedUser := &domain.User{ID: 7, Email: "new.user@example.com"}
	repo.On("FindByEmail", mock.Anything, "new.user@example.com").Return(storedUser, nil)
	repo.On("CreatePasswordResetToken", mock.Anything, int64(7), hashResetToken("raw-token"), mock.AnythingOfType("time.Time")).
		Return(&domain.PasswordResetToken{ID: "token-id"}, nil)
	mailerMock.On("SendPasswordResetEmail", mock.Anything, "new.user@example.com", "raw-token").Return(nil)

	uc := newPasswordResetUsecase(repo, mailerMock, "raw-token")

	if err := uc.ForgotPassword(context.Background(), domain.ForgotPasswordInput{Email: "New.User@Example.com"}); err != nil {
		t.Fatalf("ForgotPassword() unexpected error: %v", err)
	}

	repo.AssertExpectations(t)
	mailerMock.AssertExpectations(t)
}

// TestForgotPassword_UnknownEmail covers AC2: an unregistered email must
// look identical to success — no token created, no email sent, no error.
func TestForgotPassword_UnknownEmail(t *testing.T) {
	repo := &domain.MockUserRepository{}
	mailerMock := &mailer.MockMailer{}

	repo.On("FindByEmail", mock.Anything, "ghost@example.com").Return(nil, domain.ErrUserNotFound)

	uc := newPasswordResetUsecase(repo, mailerMock, "raw-token")

	if err := uc.ForgotPassword(context.Background(), domain.ForgotPasswordInput{Email: "ghost@example.com"}); err != nil {
		t.Fatalf("ForgotPassword() unexpected error: %v", err)
	}

	repo.AssertNotCalled(t, "CreatePasswordResetToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	mailerMock.AssertNotCalled(t, "SendPasswordResetEmail", mock.Anything, mock.Anything, mock.Anything)
}

// TestForgotPassword_RepositoryLookupError ensures a genuine infra failure
// while looking up the account is surfaced, unlike the not-found case.
func TestForgotPassword_RepositoryLookupError(t *testing.T) {
	repo := &domain.MockUserRepository{}
	mailerMock := &mailer.MockMailer{}

	dbErr := errors.New("connection reset")
	repo.On("FindByEmail", mock.Anything, mock.Anything).Return(nil, dbErr)

	uc := newPasswordResetUsecase(repo, mailerMock, "raw-token")

	err := uc.ForgotPassword(context.Background(), domain.ForgotPasswordInput{Email: "new.user@example.com"})
	if !errors.Is(err, dbErr) {
		t.Fatalf("ForgotPassword() error = %v, want wrapping %v", err, dbErr)
	}
}

// TestForgotPassword_TokenPersistFailure ensures a DB failure while
// writing the token is surfaced as a real error.
func TestForgotPassword_TokenPersistFailure(t *testing.T) {
	repo := &domain.MockUserRepository{}
	mailerMock := &mailer.MockMailer{}

	storedUser := &domain.User{ID: 7, Email: "new.user@example.com"}
	repo.On("FindByEmail", mock.Anything, "new.user@example.com").Return(storedUser, nil)
	repo.On("CreatePasswordResetToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("insert failed"))

	uc := newPasswordResetUsecase(repo, mailerMock, "raw-token")

	if err := uc.ForgotPassword(context.Background(), domain.ForgotPasswordInput{Email: "new.user@example.com"}); err == nil {
		t.Fatal("ForgotPassword() expected error when token persistence fails")
	}
	mailerMock.AssertNotCalled(t, "SendPasswordResetEmail", mock.Anything, mock.Anything, mock.Anything)
}

// TestForgotPassword_MailerFailure ensures a mail-provider outage is
// swallowed (logged only) rather than surfacing as an error — it must not
// give an attacker a different response shape than the unknown-email path.
func TestForgotPassword_MailerFailure(t *testing.T) {
	repo := &domain.MockUserRepository{}
	mailerMock := &mailer.MockMailer{}

	storedUser := &domain.User{ID: 7, Email: "new.user@example.com"}
	repo.On("FindByEmail", mock.Anything, "new.user@example.com").Return(storedUser, nil)
	repo.On("CreatePasswordResetToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&domain.PasswordResetToken{ID: "token-id"}, nil)
	mailerMock.On("SendPasswordResetEmail", mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("smtp timeout"))

	uc := newPasswordResetUsecase(repo, mailerMock, "raw-token")

	if err := uc.ForgotPassword(context.Background(), domain.ForgotPasswordInput{Email: "new.user@example.com"}); err != nil {
		t.Fatalf("ForgotPassword() error = %v, want nil (mailer failures must be swallowed)", err)
	}
}

// TestReset_Success covers AC3: a valid token + new password rotates the
// password (via the repository's atomic ResetPassword, which also revokes
// sessions — tested at the repository layer).
func TestReset_Success(t *testing.T) {
	repo := &domain.MockUserRepository{}
	mailerMock := &mailer.MockMailer{}

	repo.On("ResetPassword", mock.Anything, hashResetToken("raw-token"), mock.AnythingOfType("string")).
		Return(nil)

	uc := newPasswordResetUsecase(repo, mailerMock, "raw-token")

	err := uc.Reset(context.Background(), domain.ResetPasswordInput{Token: "raw-token", NewPassword: "new-s3cret!"})
	if err != nil {
		t.Fatalf("Reset() unexpected error: %v", err)
	}
	repo.AssertExpectations(t)
}

// TestReset_PasswordTooShort covers AC3's "≥ 8 ký tự" requirement — must
// fail before ever touching the repository.
func TestReset_PasswordTooShort(t *testing.T) {
	repo := &domain.MockUserRepository{}
	mailerMock := &mailer.MockMailer{}

	uc := newPasswordResetUsecase(repo, mailerMock, "raw-token")

	err := uc.Reset(context.Background(), domain.ResetPasswordInput{Token: "raw-token", NewPassword: "short1"})
	if !errors.Is(err, domain.ErrInvalidPassword) {
		t.Fatalf("Reset() error = %v, want %v", err, domain.ErrInvalidPassword)
	}
	repo.AssertNotCalled(t, "ResetPassword", mock.Anything, mock.Anything, mock.Anything)
}

// TestReset_InvalidToken covers AC4: expired/used/unknown tokens must
// propagate domain.ErrResetTokenInvalid unchanged.
func TestReset_InvalidToken(t *testing.T) {
	repo := &domain.MockUserRepository{}
	mailerMock := &mailer.MockMailer{}

	repo.On("ResetPassword", mock.Anything, mock.Anything, mock.Anything).Return(domain.ErrResetTokenInvalid)

	uc := newPasswordResetUsecase(repo, mailerMock, "raw-token")

	err := uc.Reset(context.Background(), domain.ResetPasswordInput{Token: "raw-token", NewPassword: "new-s3cret!"})
	if !errors.Is(err, domain.ErrResetTokenInvalid) {
		t.Fatalf("Reset() error = %v, want %v", err, domain.ErrResetTokenInvalid)
	}
}

// TestReset_RepositoryError ensures a genuine infra failure isn't
// misreported as an invalid token.
func TestReset_RepositoryError(t *testing.T) {
	repo := &domain.MockUserRepository{}
	mailerMock := &mailer.MockMailer{}

	dbErr := errors.New("connection reset")
	repo.On("ResetPassword", mock.Anything, mock.Anything, mock.Anything).Return(dbErr)

	uc := newPasswordResetUsecase(repo, mailerMock, "raw-token")

	err := uc.Reset(context.Background(), domain.ResetPasswordInput{Token: "raw-token", NewPassword: "new-s3cret!"})
	if errors.Is(err, domain.ErrResetTokenInvalid) {
		t.Fatalf("Reset() error = %v, unexpected infra failure must not look like an invalid token", err)
	}
	if !errors.Is(err, dbErr) {
		t.Fatalf("Reset() error = %v, want wrapping %v", err, dbErr)
	}
}
