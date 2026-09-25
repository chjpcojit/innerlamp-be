package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// Tests for the shared OTP mechanism (otp-mechanism spec). The acceptance
// criteria each test proves are named in the test's own doc comment:
//
//	AC1  the mechanism carries no feature-specific logic        -> TestOTPService_Issue_HappyPath
//	AC2  operations are scoped by (phone_number, purpose)       -> TestOTPService_Issue_ScopesByPurpose
//	AC8  Generate keeps the dev constant 111111                 -> TestOTPService_Generate
//
// The register-specific halves of AC3–AC6 (same error codes, same response
// shapes, same persisted effects) are regression-pinned in
// register_usecase_test.go, which drives RegisterUsecase through the same
// OTPService.

type otpFixture struct {
	service *OTPService
	repo    *domain.MockOTPRepository
	sender  *domain.MockOTPSender
	logger  *logging.MockLogger
}

func newOTPFixture() *otpFixture {
	repo := &domain.MockOTPRepository{}
	sender := &domain.MockOTPSender{}
	logger := newLogger()
	return &otpFixture{
		service: NewOTPService(repo, sender, logger, testOTPPolicy),
		repo:    repo,
		sender:  sender,
		logger:  logger,
	}
}

// TestOTPService_Generate covers AC8: the dev constant is preserved, and it is
// hashed (bcrypt) rather than returned raw.
func TestOTPService_Generate(t *testing.T) {
	f := newOTPFixture()

	code, hash, err := f.service.Generate()

	require.NoError(t, err)
	assert.Equal(t, "111111", code, "AC8: Generate keeps the dev constant, not a random code")
	assert.NotEqual(t, code, hash)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(hash), []byte(code)))
}

// TestOTPService_Issue_HappyPath covers AC1: Issue performs generate → persist
// → send, touches nothing but its two ports, and reports the window the caller
// needs to build its response.
func TestOTPService_Issue_HappyPath(t *testing.T) {
	f := newOTPFixture()

	var (
		capturedHash   string
		capturedExpiry time.Time
		capturedLimits domain.SendLimits
		order          []string
	)

	f.repo.On("IssueOTP", mock.Anything, testPhoneNumber, domain.OTPPurposeRegister, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time"), mock.AnythingOfType("domain.SendLimits")).
		Run(func(args mock.Arguments) {
			order = append(order, "issue")
			capturedHash = args.String(3)
			capturedExpiry = args.Get(4).(time.Time)
			capturedLimits = args.Get(5).(domain.SendLimits)
		}).
		Return(&domain.SendState{SendCount: 1, Remaining: 2}, nil)
	f.sender.On("SendOTP", mock.Anything, testPhoneNumber, testOTPCode, domain.OTPPurposeRegister).
		Run(func(mock.Arguments) { order = append(order, "send") }).
		Return(nil)

	before := time.Now()
	result, err := f.service.Issue(context.Background(), testPhoneNumber, domain.OTPPurposeRegister)
	require.NoError(t, err)

	// The row is committed before the SMS is sent — a code the user holds while
	// its row rolled back is dead on arrival.
	assert.Equal(t, []string{"issue", "send"}, order)

	require.NotNil(t, result)
	assert.Equal(t, 90, result.ExpiresInSeconds)
	assert.Equal(t, 60, result.ResendAfterSeconds)
	assert.Equal(t, 2, result.ResendRemaining)

	// The persisted value is a bcrypt hash of the dev constant, never the code.
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(capturedHash), []byte(testOTPCode)))
	assert.WithinDuration(t, before.Add(90*time.Second), capturedExpiry, 5*time.Second)

	// The policy is projected onto the repository's SendLimits shape in one place.
	assert.Equal(t, domain.SendLimits{Cooldown: 60 * time.Second, LockDuration: 30 * time.Minute, MaxAttempts: 3}, capturedLimits)
}

// TestOTPService_Issue_ScopesByPurpose covers AC2: the purpose is threaded
// through to both the persistence and the send, never hardcoded to "register",
// so two active purposes on one phone number address independent rows and
// limits.
func TestOTPService_Issue_ScopesByPurpose(t *testing.T) {
	f := newOTPFixture()

	purposes := []domain.OTPPurpose{
		domain.OTPPurposeRegister,
		domain.OTPPurposeLogin,
		domain.OTPPurposeResetPassword,
	}

	for _, purpose := range purposes {
		f.repo.On("IssueOTP", mock.Anything, testPhoneNumber, purpose, mock.Anything, mock.Anything, mock.Anything).
			Return(&domain.SendState{Remaining: 3}, nil).Once()
		f.sender.On("SendOTP", mock.Anything, testPhoneNumber, mock.Anything, purpose).Return(nil).Once()
	}

	for _, purpose := range purposes {
		_, err := f.service.Issue(context.Background(), testPhoneNumber, purpose)
		require.NoError(t, err)
	}

	f.repo.AssertExpectations(t)
	f.sender.AssertExpectations(t)
}

// TestOTPService_Issue_SendLimitRefusals covers the send policy surfacing on
// the generic path: a cooldown or cap refusal is passed through unchanged (with
// its retry_after_seconds / locked_until), and nothing is sent.
func TestOTPService_Issue_SendLimitRefusals(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		check   func(t *testing.T, err error)
	}{
		{
			name:    "inside the cooldown",
			repoErr: &domain.OTPResendTooSoonError{RetryAfterSeconds: 40},
			check: func(t *testing.T, err error) {
				var tooSoon *domain.OTPResendTooSoonError
				require.ErrorAs(t, err, &tooSoon)
				assert.Equal(t, 40, tooSoon.RetryAfterSeconds)
			},
		},
		{
			name:    "send cap reached",
			repoErr: &domain.OTPSendLimitedError{LockedUntil: time.Now().Add(30 * time.Minute)},
			check: func(t *testing.T, err error) {
				var limited *domain.OTPSendLimitedError
				require.ErrorAs(t, err, &limited)
				assert.WithinDuration(t, time.Now().Add(30*time.Minute), limited.LockedUntil, time.Minute)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newOTPFixture()

			f.repo.On("IssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(nil, tt.repoErr)

			_, err := f.service.Issue(context.Background(), testPhoneNumber, domain.OTPPurposeRegister)

			assert.ErrorIs(t, err, tt.repoErr)
			tt.check(t, err)
			f.sender.AssertNotCalled(t, "SendOTP")
		})
	}
}

// TestOTPService_Issue_SMSFailure pins that a sender failure is reported as
// ErrSMSSendFailed even though the row is already committed — a code that never
// arrived is not a success.
func TestOTPService_Issue_SMSFailure(t *testing.T) {
	f := newOTPFixture()

	f.repo.On("IssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&domain.SendState{SendCount: 1, Remaining: 2}, nil)
	f.sender.On("SendOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("carrier said no"))

	_, err := f.service.Issue(context.Background(), testPhoneNumber, domain.OTPPurposeRegister)

	assert.ErrorIs(t, err, domain.ErrSMSSendFailed)
}

// TestOTPService_Verify_HappyPath covers the success path: a correct code
// returns the verified row (so the caller can consume it) and records nothing.
func TestOTPService_Verify_HappyPath(t *testing.T) {
	f := newOTPFixture()

	otp := activeOTP(time.Now())
	f.repo.On("FindActiveOTP", mock.Anything, testPhoneNumber, domain.OTPPurposeRegister).Return(otp, nil)

	got, err := f.service.Verify(context.Background(), testPhoneNumber, domain.OTPPurposeRegister, testOTPCode)

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, otp.ID, got.ID)

	f.repo.AssertNotCalled(t, "IncrementOTPAttempts", mock.Anything, mock.Anything)
	f.repo.AssertNotCalled(t, "LockOTP", mock.Anything, mock.Anything, mock.Anything)
}

// TestOTPService_Verify_WrongOTP covers the wrong-code path: every miss is
// recorded, and the lock is set only on the Nth consecutive miss. The error is
// always ErrOTPInvalid and never reveals how many tries remain.
func TestOTPService_Verify_WrongOTP(t *testing.T) {
	tests := []struct {
		name       string
		attempts   int
		wantLocked bool
	}{
		{name: "first wrong guess", attempts: 1, wantLocked: false},
		{name: "second wrong guess", attempts: 2, wantLocked: false},
		{name: "third wrong guess locks the number", attempts: 3, wantLocked: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newOTPFixture()

			otp := activeOTP(time.Now())
			f.repo.On("FindActiveOTP", mock.Anything, testPhoneNumber, domain.OTPPurposeRegister).Return(otp, nil)
			f.repo.On("IncrementOTPAttempts", mock.Anything, otp.ID).Return(tt.attempts, nil)

			if tt.wantLocked {
				f.repo.On("LockOTP", mock.Anything, otp.ID, mock.AnythingOfType("time.Time")).
					Run(func(args mock.Arguments) {
						lockedUntil := args.Get(2).(time.Time)
						// LockDuration from the moment of the third failure.
						assert.WithinDuration(t, time.Now().Add(30*time.Minute), lockedUntil, 5*time.Second)
					}).
					Return(nil)
			}

			got, err := f.service.Verify(context.Background(), testPhoneNumber, domain.OTPPurposeRegister, "000000")

			assert.ErrorIs(t, err, domain.ErrOTPInvalid)
			assert.Nil(t, got)
			assert.NotContains(t, err.Error(), "remaining")

			f.repo.AssertCalled(t, "IncrementOTPAttempts", mock.Anything, otp.ID)
			if !tt.wantLocked {
				f.repo.AssertNotCalled(t, "LockOTP", mock.Anything, mock.Anything, mock.Anything)
			}
		})
	}
}

// TestOTPService_Verify_Locked covers the lockout path: while the lock holds the
// code is not even compared and no attempt is recorded.
func TestOTPService_Verify_Locked(t *testing.T) {
	f := newOTPFixture()

	lockedUntil := time.Now().Add(9 * time.Minute)
	locked := activeOTP(time.Now().Add(-3 * time.Minute))
	locked.LockedUntil = &lockedUntil

	f.repo.On("FindActiveOTP", mock.Anything, testPhoneNumber, domain.OTPPurposeRegister).Return(locked, nil)

	got, err := f.service.Verify(context.Background(), testPhoneNumber, domain.OTPPurposeRegister, testOTPCode)

	assert.ErrorIs(t, err, domain.ErrOTPLocked)
	assert.Nil(t, got)

	var lockedErr *domain.OTPLockedError
	require.ErrorAs(t, err, &lockedErr)
	assert.WithinDuration(t, lockedUntil, lockedErr.LockedUntil, time.Second)

	f.repo.AssertNotCalled(t, "IncrementOTPAttempts")
}

// TestOTPService_Verify_NoActiveOTP covers the expired/used path: a missing
// active row is reported as ErrOTPExpired, not as a wrong guess.
func TestOTPService_Verify_NoActiveOTP(t *testing.T) {
	f := newOTPFixture()

	f.repo.On("FindActiveOTP", mock.Anything, testPhoneNumber, domain.OTPPurposeRegister).Return(nil, domain.ErrOTPInvalid)

	got, err := f.service.Verify(context.Background(), testPhoneNumber, domain.OTPPurposeRegister, testOTPCode)

	assert.ErrorIs(t, err, domain.ErrOTPExpired)
	assert.Nil(t, got)
}

// TestOTPService_EnsureNotLocked covers the request/resend pre-check: it is nil
// when there is no active OTP or the active one is not locked, and
// ErrOTPLocked (with the deadline) when it is.
func TestOTPService_EnsureNotLocked(t *testing.T) {
	t.Run("no active otp", func(t *testing.T) {
		f := newOTPFixture()
		f.repo.On("FindActiveOTP", mock.Anything, testPhoneNumber, domain.OTPPurposeRegister).Return(nil, domain.ErrOTPInvalid)

		require.NoError(t, f.service.EnsureNotLocked(context.Background(), testPhoneNumber, domain.OTPPurposeRegister))
	})

	t.Run("active but not locked", func(t *testing.T) {
		f := newOTPFixture()
		f.repo.On("FindActiveOTP", mock.Anything, testPhoneNumber, domain.OTPPurposeRegister).Return(activeOTP(time.Now()), nil)

		require.NoError(t, f.service.EnsureNotLocked(context.Background(), testPhoneNumber, domain.OTPPurposeRegister))
	})

	t.Run("locked", func(t *testing.T) {
		f := newOTPFixture()
		lockedUntil := time.Now().Add(9 * time.Minute)
		locked := activeOTP(time.Now().Add(-3 * time.Minute))
		locked.LockedUntil = &lockedUntil
		f.repo.On("FindActiveOTP", mock.Anything, testPhoneNumber, domain.OTPPurposeRegister).Return(locked, nil)

		err := f.service.EnsureNotLocked(context.Background(), testPhoneNumber, domain.OTPPurposeRegister)

		require.ErrorIs(t, err, domain.ErrOTPLocked)
		var lockedErr *domain.OTPLockedError
		require.ErrorAs(t, err, &lockedErr)
		assert.WithinDuration(t, lockedUntil, lockedErr.LockedUntil, time.Second)
	})
}
