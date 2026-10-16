package usecase

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// Tests for the shared OTP mechanism (otp-mechanism v2.0). Mapping:
//
//	AC1  scoped by (email, purpose)                       -> TestOTPService_Issue, TestOTPService_ScopesByPurpose
//	AC2  crypto/rand six digits, only the hash stored     -> TestOTPService_Generate_Random, TestOTPService_NewCode_Distribution
//	AC3  dev fixed code when set                          -> TestOTPService_Generate_DevFixedCode (the env guard: config.TestValidate*)
//	AC4  issue = policy + invalidate + insert, TTL expiry -> TestOTPService_Issue
//	AC5  refusal returns typed error, writes nothing      -> TestOTPService_Issue_Refused
//	AC6  ApplySendLimit without a code                    -> TestOTPService_ApplySendLimit
//	AC7  attempt reserved atomically before compare       -> TestOTPService_Verify (reserve-before-compare, no attempt left)
//	AC8  wrong / expired / locked outcomes                -> TestOTPService_Verify
//	AC9  right code returned, not marked used             -> TestOTPService_Verify ("right code")
//	AC10 ReleaseSendLimit                                 -> TestOTPService_ReleaseSendLimit
//	AC11 no account / pending / user logic                -> the strict mocks: OTPService only ever calls domain.OTPRepository
//
// The atomicity of AC7 itself (10 concurrent guesses keep 3) is proven against
// Postgres in repository.TestOTPRepository_ConsumeOTPAttempt.

var testNow = time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC)

type otpFixture struct {
	service *OTPService
	repo    *domain.MockOTPRepository
}

func newOTPFixture() *otpFixture {
	repo := &domain.MockOTPRepository{}
	svc := NewOTPService(repo, newLogger(), testOTPPolicy)
	svc.now = func() time.Time { return testNow }
	return &otpFixture{service: svc, repo: repo}
}

var sixDigits = regexp.MustCompile(`^[0-9]{6}$`)

func TestOTPService_Generate_Random(t *testing.T) {
	f := newOTPFixture()

	code, hash, err := f.service.Generate()

	require.NoError(t, err)
	assert.Regexp(t, sixDigits, code)
	assert.NotEqual(t, code, hash, "only the hash is for storage")
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(hash), []byte(code)))
}

// TestOTPService_NewCode_Distribution draws many codes (without bcrypt, which
// would make this slow) and checks every position takes every digit — leading
// zeros included — so the code is not, e.g., formatted without padding or
// drawn from a narrower range.
func TestOTPService_NewCode_Distribution(t *testing.T) {
	f := newOTPFixture()

	var seen [6][10]bool
	leadingZero := false
	for range 3000 {
		code, err := f.service.newCode()
		require.NoError(t, err)
		require.Regexp(t, sixDigits, code)
		if code[0] == '0' {
			leadingZero = true
		}
		for i, ch := range code {
			seen[i][ch-'0'] = true
		}
	}

	assert.True(t, leadingZero, "codes with a leading zero must occur")
	for i := range seen {
		for d := range seen[i] {
			assert.Truef(t, seen[i][d], "digit %d never drawn at position %d", d, i)
		}
	}
}

func TestOTPService_Generate_DevFixedCode(t *testing.T) {
	policy := testOTPPolicy
	policy.DevFixedCode = "123456"
	svc := NewOTPService(&domain.MockOTPRepository{}, newLogger(), policy)

	code, hash, err := svc.Generate()

	require.NoError(t, err)
	assert.Equal(t, "123456", code)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(hash), []byte(code)))
}

func TestOTPService_Issue(t *testing.T) {
	f := newOTPFixture()

	var storedHash string
	f.repo.On("IssueOTP", mock.Anything, testEmail, domain.OTPPurposeRegister, mock.AnythingOfType("string"),
		testNow.Add(testOTPPolicy.TTL.Duration()), f.service.SendLimits()).
		Run(func(args mock.Arguments) { storedHash = args.String(3) }).
		Return(&domain.SendState{SendCount: 1, Remaining: 2}, nil)

	res, err := f.service.Issue(context.Background(), testEmail, domain.OTPPurposeRegister)

	require.NoError(t, err)
	assert.Regexp(t, sixDigits, res.Code)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(res.Code)), "the stored hash is of the returned code")
	assert.Equal(t, 600, res.ExpiresInSeconds)
	assert.Equal(t, 60, res.ResendAfterSeconds)
	assert.Equal(t, 2, res.ResendRemaining)
	f.repo.AssertExpectations(t)
}

func TestOTPService_ScopesByPurpose(t *testing.T) {
	f := newOTPFixture()
	f.repo.On("IssueOTP", mock.Anything, testEmail, domain.OTPPurposeResetPassword, mock.Anything, mock.Anything, mock.Anything).
		Return(&domain.SendState{Remaining: 3}, nil)

	_, err := f.service.Issue(context.Background(), testEmail, domain.OTPPurposeResetPassword)

	require.NoError(t, err)
	f.repo.AssertExpectations(t)
}

func TestOTPService_Issue_Refused(t *testing.T) {
	lockedUntil := testNow.Add(30 * time.Minute)
	cases := map[string]error{
		"cooldown": &domain.OTPResendTooSoonError{RetryAfterSeconds: 42},
		"cap":      &domain.OTPSendLimitedError{LockedUntil: lockedUntil},
	}
	for name, refusal := range cases {
		t.Run(name, func(t *testing.T) {
			f := newOTPFixture()
			f.repo.On("IssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, refusal)

			res, err := f.service.Issue(context.Background(), testEmail, domain.OTPPurposeRegister)

			assert.Nil(t, res, "no code comes back from a refused send")
			assert.ErrorIs(t, err, refusal)
		})
	}
}

func TestOTPService_ApplySendLimit(t *testing.T) {
	t.Run("allowed", func(t *testing.T) {
		f := newOTPFixture()
		f.repo.On("ApplySendLimit", mock.Anything, testEmail, domain.OTPPurposeResetPassword, f.service.SendLimits()).
			Return(&domain.SendState{SendCount: 0, Remaining: 3}, nil)

		state, err := f.service.ApplySendLimit(context.Background(), testEmail, domain.OTPPurposeResetPassword)

		require.NoError(t, err)
		assert.Equal(t, 3, state.Remaining)
		f.repo.AssertNotCalled(t, "IssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("refused", func(t *testing.T) {
		f := newOTPFixture()
		refusal := &domain.OTPResendTooSoonError{RetryAfterSeconds: 10}
		f.repo.On("ApplySendLimit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, refusal)

		_, err := f.service.ApplySendLimit(context.Background(), testEmail, domain.OTPPurposeResetPassword)

		tooSoon, ok := errors.AsType[*domain.OTPResendTooSoonError](err)
		require.True(t, ok)
		assert.Equal(t, 10, tooSoon.RetryAfterSeconds)
	})
}

func TestOTPService_ReleaseSendLimit(t *testing.T) {
	f := newOTPFixture()
	f.repo.On("ReleaseSendLimit", mock.Anything, testEmail, domain.OTPPurposeRegister).Return(nil)

	require.NoError(t, f.service.ReleaseSendLimit(context.Background(), testEmail, domain.OTPPurposeRegister))
	f.repo.AssertExpectations(t)
}

func TestOTPService_InvalidateActive(t *testing.T) {
	f := newOTPFixture()
	f.repo.On("InvalidateActiveOTP", mock.Anything, testEmail, domain.OTPPurposeRegister).Return(nil)

	require.NoError(t, f.service.InvalidateActive(context.Background(), testEmail, domain.OTPPurposeRegister))
	f.repo.AssertExpectations(t)
}

func TestOTPService_EnsureNotLocked(t *testing.T) {
	lockedUntil := testNow.Add(10 * time.Minute)
	past := testNow.Add(-time.Minute)

	t.Run("no active code", func(t *testing.T) {
		f := newOTPFixture()
		f.repo.On("FindActiveOTP", mock.Anything, testEmail, domain.OTPPurposeRegister).Return(nil, domain.ErrOTPInvalid)
		assert.NoError(t, f.service.EnsureNotLocked(context.Background(), testEmail, domain.OTPPurposeRegister))
	})
	t.Run("lock lifted", func(t *testing.T) {
		f := newOTPFixture()
		f.repo.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(&domain.OTPCode{ID: 1, LockedUntil: &past}, nil)
		assert.NoError(t, f.service.EnsureNotLocked(context.Background(), testEmail, domain.OTPPurposeRegister))
	})
	t.Run("locked", func(t *testing.T) {
		f := newOTPFixture()
		f.repo.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(&domain.OTPCode{ID: 1, LockedUntil: &lockedUntil}, nil)
		err := f.service.EnsureNotLocked(context.Background(), testEmail, domain.OTPPurposeRegister)
		locked, ok := errors.AsType[*domain.OTPLockedError](err)
		require.True(t, ok)
		assert.Equal(t, lockedUntil, locked.LockedUntil)
	})
	t.Run("repository failure", func(t *testing.T) {
		f := newOTPFixture()
		f.repo.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db down"))
		err := f.service.EnsureNotLocked(context.Background(), testEmail, domain.OTPPurposeRegister)
		require.Error(t, err)
		assert.NotErrorIs(t, err, domain.ErrOTPLocked)
	})
}

// otpRow returns an active, unlocked code row whose hash matches code. MinCost
// keeps the many Verify cases fast; the cost is not what is under test.
func otpRow(t *testing.T, code string) *domain.OTPCode {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.MinCost)
	require.NoError(t, err)
	return &domain.OTPCode{
		ID:        7,
		Email:     testEmail,
		Purpose:   domain.OTPPurposeRegister,
		OTPHash:   string(h),
		ExpiresAt: testNow.Add(5 * time.Minute),
	}
}

func TestOTPService_Verify(t *testing.T) {
	const code = "042917"
	lockUntil := testNow.Add(testOTPPolicy.LockDuration.Duration())
	ctx := context.Background()

	t.Run("right code: attempt reserved first, row returned, not marked used", func(t *testing.T) {
		f := newOTPFixture()
		row := otpRow(t, code)
		f.repo.On("FindActiveOTP", mock.Anything, testEmail, domain.OTPPurposeRegister).Return(row, nil)
		f.repo.On("ConsumeOTPAttempt", mock.Anything, row.ID, testOTPPolicy.MaxAttempts, lockUntil).Return(1, true, nil)

		got, err := f.service.Verify(ctx, testEmail, domain.OTPPurposeRegister, code)

		require.NoError(t, err)
		assert.Same(t, row, got)
		f.repo.AssertExpectations(t) // and nothing else: marking used is the caller's
	})

	t.Run("wrong code: attempt still counted, no remaining count disclosed", func(t *testing.T) {
		f := newOTPFixture()
		row := otpRow(t, code)
		f.repo.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(row, nil)
		f.repo.On("ConsumeOTPAttempt", mock.Anything, row.ID, testOTPPolicy.MaxAttempts, lockUntil).Return(2, true, nil)

		_, err := f.service.Verify(ctx, testEmail, domain.OTPPurposeRegister, "000000")

		assert.Equal(t, domain.ErrOTPInvalid, err, "the bare sentinel, no attempt count attached")
		assert.NotContains(t, err.Error(), strconv.Itoa(testOTPPolicy.MaxAttempts-2))
		f.repo.AssertExpectations(t)
	})

	t.Run("no attempt left to reserve: locked, hash never compared", func(t *testing.T) {
		f := newOTPFixture()
		row := otpRow(t, code)
		f.repo.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(row, nil)
		f.repo.On("ConsumeOTPAttempt", mock.Anything, row.ID, mock.Anything, mock.Anything).Return(0, false, nil)

		_, err := f.service.Verify(ctx, testEmail, domain.OTPPurposeRegister, code) // even the right code

		locked, ok := errors.AsType[*domain.OTPLockedError](err)
		require.True(t, ok, "got %v", err)
		assert.Equal(t, lockUntil, locked.LockedUntil)
	})

	t.Run("locked row: no attempt reserved", func(t *testing.T) {
		f := newOTPFixture()
		row := otpRow(t, code)
		until := testNow.Add(20 * time.Minute)
		row.LockedUntil = &until
		f.repo.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(row, nil)

		_, err := f.service.Verify(ctx, testEmail, domain.OTPPurposeRegister, code)

		locked, ok := errors.AsType[*domain.OTPLockedError](err)
		require.True(t, ok)
		assert.Equal(t, until, locked.LockedUntil)
		f.repo.AssertNotCalled(t, "ConsumeOTPAttempt", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("no active code: expired", func(t *testing.T) {
		f := newOTPFixture()
		f.repo.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrOTPInvalid)

		_, err := f.service.Verify(ctx, testEmail, domain.OTPPurposeRegister, code)

		assert.ErrorIs(t, err, domain.ErrOTPExpired)
	})

	t.Run("past expires_at: expired, no attempt reserved", func(t *testing.T) {
		f := newOTPFixture()
		row := otpRow(t, code)
		row.ExpiresAt = testNow
		f.repo.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(row, nil)

		_, err := f.service.Verify(ctx, testEmail, domain.OTPPurposeRegister, code)

		assert.ErrorIs(t, err, domain.ErrOTPExpired)
		f.repo.AssertNotCalled(t, "ConsumeOTPAttempt", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("reservation failure is infrastructure, not a guess", func(t *testing.T) {
		f := newOTPFixture()
		row := otpRow(t, code)
		f.repo.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(row, nil)
		f.repo.On("ConsumeOTPAttempt", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(0, false, errors.New("db down"))

		_, err := f.service.Verify(ctx, testEmail, domain.OTPPurposeRegister, code)

		require.Error(t, err)
		assert.False(t, strings.Contains(err.Error(), domain.ErrOTPInvalid.Error()))
		assert.NotErrorIs(t, err, domain.ErrOTPLocked)
	})
}
