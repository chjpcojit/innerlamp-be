package usecase

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"chjpcoj.com.innerlamp/pkg/jwt"
	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/mock"
)

func validInput() domain.RegisterInput {
	return domain.RegisterInput{
		Email:       "New.User@Example.com",
		Password:    "s3cret!!",
		PhoneNumber: "+84912345678",
		Device: domain.DeviceInfo{
			Platform:  "iOS",
			OSVersion: "17.0",
			UserAgent: "innerlamp-ios/1.0",
		},
	}
}

func newLogger() *logging.MockLogger {
	l := &logging.MockLogger{}
	l.On("WithContext", mock.Anything).Return(l)
	l.On("WithFields", mock.Anything).Return(l)
	l.On("WithError", mock.Anything).Return(l)
	l.On("Info", mock.Anything, mock.Anything).Return()
	l.On("Error", mock.Anything, mock.Anything).Return()
	return l
}

// TestRegister_Success covers AC1 (immediate login), AC6 (password hashed,
// never stored plaintext) and AC5 (optional fields may be empty).
func TestRegister_Success(t *testing.T) {
	repo := &domain.MockUserRepository{}
	jwtSvc := &jwt.MockService{}
	logger := newLogger()

	input := validInput()

	repo.On("Register", mock.Anything, mock.AnythingOfType("*domain.User"), mock.AnythingOfType("*domain.Device"), mock.AnythingOfType("domain.TokenIssuer")).
		Run(func(args mock.Arguments) {
			user := args.Get(1).(*domain.User)

			// AC6: password must be hashed, never stored plaintext.
			if user.PasswordHash == input.Password {
				t.Fatalf("password stored in plaintext")
			}
			if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
				t.Fatalf("password hash does not match original password: %v", err)
			}

			user.ID = 42

			issueTokens := args.Get(3).(domain.TokenIssuer)
			accessToken, refreshToken, err := issueTokens("42")
			if err != nil {
				t.Fatalf("issueTokens returned error: %v", err)
			}
			if accessToken != "access-token" || refreshToken != "refresh-token" {
				t.Fatalf("unexpected tokens from issueTokens: %s / %s", accessToken, refreshToken)
			}
		}).
		Return(&domain.Session{
			UserID:       42,
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
		}, nil)

	jwtSvc.On("GenerateAccessToken", "42", "new.user@example.com", tokenRole, mock.Anything).Return("access-token", nil)
	jwtSvc.On("GenerateRefreshToken", "42", mock.Anything).Return("refresh-token", nil)

	uc := NewRegisterUsecase(repo, jwtSvc, logger)

	result, err := uc.Register(context.Background(), input)
	if err != nil {
		t.Fatalf("Register() unexpected error: %v", err)
	}
	if result.AccessToken != "access-token" || result.RefreshToken != "refresh-token" {
		t.Fatalf("Register() tokens = %s / %s, want access-token / refresh-token", result.AccessToken, result.RefreshToken)
	}
	// Email must be normalized (lowercased/trimmed) before persisting.
	if result.User.Email != "new.user@example.com" {
		t.Errorf("Register() email = %q, want normalized lowercase", result.User.Email)
	}

	repo.AssertExpectations(t)
	jwtSvc.AssertExpectations(t)
}

// TestRegister_NormalizesInput covers what the use-case still owns after
// format validation moved to the delivery layer's struct tags: canonicalizing
// the values the business persists — a lowercased/trimmed email so uniqueness
// is case-insensitive, and a trimmed phone.
func TestRegister_NormalizesInput(t *testing.T) {
	repo := &domain.MockUserRepository{}
	jwtSvc := &jwt.MockService{}
	logger := newLogger()

	repo.On("Register", mock.Anything, mock.AnythingOfType("*domain.User"), mock.AnythingOfType("*domain.Device"), mock.AnythingOfType("domain.TokenIssuer")).
		Run(func(args mock.Arguments) {
			user := args.Get(1).(*domain.User)
			if user.Email != "new.user@example.com" {
				t.Errorf("persisted email = %q, want normalized lowercase", user.Email)
			}
			if user.PhoneNumber != "+84912345678" {
				t.Errorf("persisted phone = %q, want trimmed %q", user.PhoneNumber, "+84912345678")
			}
		}).
		Return(&domain.Session{AccessToken: "at", RefreshToken: "rt"}, nil)
	jwtSvc.On("GenerateAccessToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("at", nil)
	jwtSvc.On("GenerateRefreshToken", mock.Anything, mock.Anything).Return("rt", nil)

	uc := NewRegisterUsecase(repo, jwtSvc, logger)

	input := validInput()
	input.PhoneNumber = "  +84912345678  "
	if _, err := uc.Register(context.Background(), input); err != nil {
		t.Fatalf("Register() unexpected error: %v", err)
	}

	repo.AssertExpectations(t)
}

// TestRegister_OptionalFieldsEmpty covers AC5: dharma name / ordination
// place may be left blank and registration still succeeds.
func TestRegister_OptionalFieldsEmpty(t *testing.T) {
	repo := &domain.MockUserRepository{}
	jwtSvc := &jwt.MockService{}
	logger := newLogger()

	input := validInput()
	input.DharmaName = ""
	input.OrdinationPlace = ""

	repo.On("Register", mock.Anything, mock.AnythingOfType("*domain.User"), mock.AnythingOfType("*domain.Device"), mock.AnythingOfType("domain.TokenIssuer")).
		Return(&domain.Session{AccessToken: "at", RefreshToken: "rt"}, nil)
	jwtSvc.On("GenerateAccessToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("at", nil)
	jwtSvc.On("GenerateRefreshToken", mock.Anything, mock.Anything).Return("rt", nil)

	uc := NewRegisterUsecase(repo, jwtSvc, logger)

	if _, err := uc.Register(context.Background(), input); err != nil {
		t.Fatalf("Register() unexpected error with empty optional fields: %v", err)
	}
}

// TestRegister_RepositoryErrorsPropagate covers AC2: email/phone already
// taken must surface as distinct, unchanged errors from the use-case.
func TestRegister_RepositoryErrorsPropagate(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
	}{
		{"email taken", domain.ErrEmailTaken},
		{"phone taken", domain.ErrPhoneTaken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &domain.MockUserRepository{}
			jwtSvc := &jwt.MockService{}
			logger := newLogger()

			repo.On("Register", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(nil, tt.repoErr)

			uc := NewRegisterUsecase(repo, jwtSvc, logger)

			_, err := uc.Register(context.Background(), validInput())
			if !errors.Is(err, tt.repoErr) {
				t.Fatalf("Register() error = %v, want %v", err, tt.repoErr)
			}
		})
	}
}

// TestRegister_TokenIssuanceFailure ensures a token-generation error inside
// the TokenIssuer closure is wrapped and propagated rather than swallowed.
func TestRegister_TokenIssuanceFailure(t *testing.T) {
	repo := &domain.MockUserRepository{}
	jwtSvc := &jwt.MockService{}
	logger := newLogger()

	wantErr := errors.New("signing key unavailable")
	jwtSvc.On("GenerateAccessToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", wantErr)

	repo.On("Register", mock.Anything, mock.Anything, mock.Anything, mock.AnythingOfType("domain.TokenIssuer")).
		Run(func(args mock.Arguments) {
			issueTokens := args.Get(3).(domain.TokenIssuer)
			if _, _, err := issueTokens("1"); err == nil {
				t.Fatalf("issueTokens() expected error, got nil")
			}
		}).
		Return(nil, errors.New("issue tokens: signing key unavailable"))

	uc := NewRegisterUsecase(repo, jwtSvc, logger)

	if _, err := uc.Register(context.Background(), validInput()); err == nil {
		t.Fatalf("Register() expected error when token issuance fails")
	}
}
