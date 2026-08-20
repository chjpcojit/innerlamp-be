package usecase

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"chjpcoj.com.innerlamp/pkg/jwt"
	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/mock"
)

func validLoginInput() domain.LoginInput {
	return domain.LoginInput{
		Email:    "New.User@Example.com",
		Password: "s3cret!!",
		Device: domain.DeviceInfo{
			Platform:  "iOS",
			OSVersion: "17.0",
			UserAgent: "innerlamp-ios/1.0",
		},
	}
}

func mustHash(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt.GenerateFromPassword() error = %v", err)
	}
	return string(hash)
}

// TestLogin_Success covers AC1: correct email+password returns the token
// pair, and AC4: the session is created for the device that logged in.
func TestLogin_Success(t *testing.T) {
	repo := &domain.MockUserRepository{}
	jwtSvc := &jwt.MockService{}
	logger := newLogger()

	storedUser := &domain.User{
		ID:           7,
		Email:        "new.user@example.com",
		PasswordHash: mustHash(t, "s3cret!!"),
	}

	repo.On("FindByEmail", mock.Anything, "new.user@example.com").Return(storedUser, nil)
	repo.On("CreateSession", mock.Anything, int64(7), mock.AnythingOfType("*domain.Device"), mock.AnythingOfType("domain.TokenIssuer")).
		Run(func(args mock.Arguments) {
			device := args.Get(2).(*domain.Device)
			if device.Platform != "iOS" {
				t.Errorf("device.Platform = %q, want iOS", device.Platform)
			}
			issueTokens := args.Get(3).(domain.TokenIssuer)
			if _, _, err := issueTokens("7"); err != nil {
				t.Fatalf("issueTokens() error = %v", err)
			}
		}).
		Return(&domain.Session{UserID: 7, AccessToken: "access-token", RefreshToken: "refresh-token"}, nil)

	jwtSvc.On("GenerateAccessToken", "7", "new.user@example.com", tokenRole, mock.Anything).Return("access-token", nil)
	jwtSvc.On("GenerateRefreshToken", "7", mock.Anything).Return("refresh-token", nil)

	uc := NewLoginUsecase(repo, jwtSvc, logger)

	result, err := uc.Login(context.Background(), validLoginInput())
	if err != nil {
		t.Fatalf("Login() unexpected error: %v", err)
	}
	if result.AccessToken != "access-token" || result.RefreshToken != "refresh-token" {
		t.Errorf("Login() tokens = %s/%s, want access-token/refresh-token", result.AccessToken, result.RefreshToken)
	}
	if result.User.ID != 7 {
		t.Errorf("Login() user.ID = %d, want 7", result.User.ID)
	}

	repo.AssertExpectations(t)
	jwtSvc.AssertExpectations(t)
}

// TestLogin_WrongPassword and TestLogin_UnknownEmail cover AC2/AC3: both
// must return the exact same domain.ErrInvalidCredentials, so delivery
// can't distinguish (and therefore can't leak) which one happened.
func TestLogin_WrongPassword(t *testing.T) {
	repo := &domain.MockUserRepository{}
	jwtSvc := &jwt.MockService{}
	logger := newLogger()

	storedUser := &domain.User{ID: 7, Email: "new.user@example.com", PasswordHash: mustHash(t, "correct-password")}
	repo.On("FindByEmail", mock.Anything, "new.user@example.com").Return(storedUser, nil)

	uc := NewLoginUsecase(repo, jwtSvc, logger)

	input := validLoginInput()
	input.Password = "wrong-password"

	_, err := uc.Login(context.Background(), input)
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want %v", err, domain.ErrInvalidCredentials)
	}
	repo.AssertNotCalled(t, "CreateSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestLogin_UnknownEmail(t *testing.T) {
	repo := &domain.MockUserRepository{}
	jwtSvc := &jwt.MockService{}
	logger := newLogger()

	repo.On("FindByEmail", mock.Anything, "ghost@example.com").Return(nil, domain.ErrUserNotFound)

	uc := NewLoginUsecase(repo, jwtSvc, logger)

	input := validLoginInput()
	input.Email = "ghost@example.com"

	_, err := uc.Login(context.Background(), input)
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want %v", err, domain.ErrInvalidCredentials)
	}
	repo.AssertNotCalled(t, "CreateSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestLogin_RepositoryLookupError ensures an unexpected (non-not-found)
// repository error is wrapped and propagated rather than being turned
// into a generic ErrInvalidCredentials or swallowed.
func TestLogin_RepositoryLookupError(t *testing.T) {
	repo := &domain.MockUserRepository{}
	jwtSvc := &jwt.MockService{}
	logger := newLogger()

	dbErr := errors.New("connection reset")
	repo.On("FindByEmail", mock.Anything, mock.Anything).Return(nil, dbErr)

	uc := NewLoginUsecase(repo, jwtSvc, logger)

	_, err := uc.Login(context.Background(), validLoginInput())
	if errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, unexpected infra failure must not look like bad credentials", err)
	}
	if !errors.Is(err, dbErr) {
		t.Fatalf("Login() error = %v, want wrapping %v", err, dbErr)
	}
}

// TestLogin_SessionCreationFailure ensures a repository failure while
// creating the session is propagated.
func TestLogin_SessionCreationFailure(t *testing.T) {
	repo := &domain.MockUserRepository{}
	jwtSvc := &jwt.MockService{}
	logger := newLogger()

	storedUser := &domain.User{ID: 7, Email: "new.user@example.com", PasswordHash: mustHash(t, "s3cret!!")}
	repo.On("FindByEmail", mock.Anything, "new.user@example.com").Return(storedUser, nil)
	repo.On("CreateSession", mock.Anything, int64(7), mock.Anything, mock.Anything).
		Return(nil, errors.New("insert failed"))

	uc := NewLoginUsecase(repo, jwtSvc, logger)

	if _, err := uc.Login(context.Background(), validLoginInput()); err == nil {
		t.Fatal("Login() expected error when session creation fails")
	}
}
