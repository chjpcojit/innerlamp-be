package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// Tests for login US-003 (password reset). Mapping:
//
//	AC23 account -> code issued + mailed, same body        -> TestPasswordReset_Request_Account
//	AC24 no account -> same body, send policy, no mail      -> TestPasswordReset_Request_NoAccount
//	AC25 minimum response time (both branches, errors too)  -> TestPasswordReset_Request_MinResponseTime
//	AC26 send policy / lock refusals                        -> TestPasswordReset_Request_Refusals
//	AC27 malformed email before any database access         -> TestPasswordReset_Request_InvalidEmail
//	AC28 reset in one transaction call                      -> TestPasswordReset_Confirm_Success
//	AC29 wrong / expired / locked code                      -> TestPasswordReset_Confirm_CodeErrors
//	AC30 bad email / password before any database access    -> TestPasswordReset_Confirm_Validation

const minResponse = 1500 * time.Millisecond

type resetFixture struct {
	uc     *PasswordResetUsecase
	resets *domain.MockPasswordResetRepository
	otp    *domain.MockOTPRepository
	mailer *domain.MockAuthMailer

	clock time.Time
	slept []time.Duration
}

func newResetFixture() *resetFixture {
	f := &resetFixture{
		resets: &domain.MockPasswordResetRepository{},
		otp:    &domain.MockOTPRepository{},
		mailer: &domain.MockAuthMailer{},
		clock:  testNow,
	}
	svc := NewOTPService(f.otp, newLogger(), testOTPPolicy)
	svc.now = func() time.Time { return testNow }
	f.uc = NewPasswordResetUsecase(f.resets, svc, f.mailer, NewBcryptPasswordHasher(), newLogger(),
		config.PasswordResetConfig{MinResponseTime: config.Duration(minResponse)})
	f.uc.now = func() time.Time { return f.clock }
	f.uc.sleep = func(_ context.Context, d time.Duration) {
		f.slept = append(f.slept, d)
		f.clock = f.clock.Add(d)
	}
	return f
}

func verifiedUser() *domain.User {
	t := testNow.Add(-time.Hour)
	return &domain.User{ID: testUserRowID, Email: testEmail, EmailVerifiedAt: &t}
}

func TestPasswordReset_Request_Account(t *testing.T) {
	f := newResetFixture()
	ctx := context.Background()
	f.resets.On("FindVerifiedUserByEmail", mock.Anything, testEmail).Return(verifiedUser(), nil)
	f.otp.On("FindActiveOTP", mock.Anything, testEmail, domain.OTPPurposeResetPassword).Return(nil, domain.ErrOTPInvalid)
	f.otp.On("IssueOTP", mock.Anything, testEmail, domain.OTPPurposeResetPassword, mock.AnythingOfType("string"),
		mock.Anything, mock.Anything).Return(&domain.SendState{Remaining: 2}, nil)
	var sentCode string
	f.mailer.On("SendPasswordResetCode", mock.Anything, testEmail, mock.AnythingOfType("string"), testOTPPolicy.TTL.Duration()).
		Run(func(a mock.Arguments) { sentCode = a.String(2) }).Return(nil)

	res, err := f.uc.RequestOTP(ctx, domain.RequestPasswordResetInput{Email: testEmailRaw})

	require.NoError(t, err)
	assert.Equal(t, int(testOTPPolicy.TTL.Duration().Seconds()), res.ExpiresInSeconds)
	assert.Equal(t, int(testOTPPolicy.SendCooldown.Duration().Seconds()), res.ResendAfterSeconds)
	assert.Regexp(t, sixDigits, sentCode)
}

func TestPasswordReset_Request_NoAccount(t *testing.T) {
	f := newResetFixture()
	f.resets.On("FindVerifiedUserByEmail", mock.Anything, testEmail).Return(nil, domain.ErrUserNotFound)
	f.otp.On("ApplySendLimit", mock.Anything, testEmail, domain.OTPPurposeResetPassword, mock.Anything).
		Return(&domain.SendState{}, nil)

	res, err := f.uc.RequestOTP(context.Background(), domain.RequestPasswordResetInput{Email: testEmail})

	require.NoError(t, err)
	assert.Equal(t, int(testOTPPolicy.TTL.Duration().Seconds()), res.ExpiresInSeconds, "same body as the account branch")
	f.mailer.AssertNotCalled(t, "SendPasswordResetCode", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	f.otp.AssertNotCalled(t, "IssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestPasswordReset_Request_MailerFailureIsSwallowed(t *testing.T) {
	f := newResetFixture()
	f.resets.On("FindVerifiedUserByEmail", mock.Anything, testEmail).Return(verifiedUser(), nil)
	f.otp.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrOTPInvalid)
	f.otp.On("IssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&domain.SendState{}, nil)
	f.mailer.On("SendPasswordResetCode", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("smtp down"))
	f.otp.On("ReleaseSendLimit", mock.Anything, testEmail, domain.OTPPurposeResetPassword).Return(nil)

	res, err := f.uc.RequestOTP(context.Background(), domain.RequestPasswordResetInput{Email: testEmail})

	require.NoError(t, err, "a mail failure must not be distinguishable from success")
	require.NotNil(t, res)
	f.otp.AssertCalled(t, "ReleaseSendLimit", mock.Anything, testEmail, domain.OTPPurposeResetPassword)
}

func TestPasswordReset_Request_MinResponseTime(t *testing.T) {
	cases := map[string]func(f *resetFixture) error{
		"account": func(f *resetFixture) error {
			f.resets.On("FindVerifiedUserByEmail", mock.Anything, mock.Anything).Return(verifiedUser(), nil)
			f.otp.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrOTPInvalid)
			f.otp.On("IssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&domain.SendState{}, nil)
			f.mailer.On("SendPasswordResetCode", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			return nil
		},
		"no account": func(f *resetFixture) error {
			f.resets.On("FindVerifiedUserByEmail", mock.Anything, mock.Anything).Return(nil, domain.ErrUserNotFound)
			f.otp.On("ApplySendLimit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&domain.SendState{}, nil)
			return nil
		},
		"refused with 429": func(f *resetFixture) error {
			f.resets.On("FindVerifiedUserByEmail", mock.Anything, mock.Anything).Return(nil, domain.ErrUserNotFound)
			f.otp.On("ApplySendLimit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(nil, &domain.OTPResendTooSoonError{RetryAfterSeconds: 30})
			return domain.ErrOTPResendTooSoon
		},
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			f := newResetFixture()
			wantErr := setup(f)

			_, err := f.uc.RequestOTP(context.Background(), domain.RequestPasswordResetInput{Email: testEmail})

			if wantErr != nil {
				assert.ErrorIs(t, err, wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Len(t, f.slept, 1)
			assert.Equal(t, minResponse, f.slept[0], "no time elapsed on the fake clock, so the whole floor is waited")
		})
	}

	t.Run("no extra wait once the floor has passed", func(t *testing.T) {
		f := newResetFixture()
		f.resets.On("FindVerifiedUserByEmail", mock.Anything, mock.Anything).
			Run(func(mock.Arguments) { f.clock = f.clock.Add(2 * time.Second) }).Return(nil, domain.ErrUserNotFound)
		f.otp.On("ApplySendLimit", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&domain.SendState{}, nil)

		_, err := f.uc.RequestOTP(context.Background(), domain.RequestPasswordResetInput{Email: testEmail})

		require.NoError(t, err)
		assert.Empty(t, f.slept)
	})
}

func TestPasswordReset_Request_Refusals(t *testing.T) {
	t.Run("send policy refuses an account", func(t *testing.T) {
		f := newResetFixture()
		f.resets.On("FindVerifiedUserByEmail", mock.Anything, mock.Anything).Return(verifiedUser(), nil)
		f.otp.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrOTPInvalid)
		f.otp.On("IssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, &domain.OTPSendLimitedError{LockedUntil: testNow.Add(time.Hour)})

		_, err := f.uc.RequestOTP(context.Background(), domain.RequestPasswordResetInput{Email: testEmail})

		assert.ErrorIs(t, err, domain.ErrOTPSendLimited)
		f.mailer.AssertNotCalled(t, "SendPasswordResetCode", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("locked code is not replaced", func(t *testing.T) {
		f := newResetFixture()
		until := testNow.Add(10 * time.Minute)
		f.resets.On("FindVerifiedUserByEmail", mock.Anything, mock.Anything).Return(verifiedUser(), nil)
		f.otp.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).
			Return(&domain.OTPCode{ID: 1, LockedUntil: &until, ExpiresAt: testNow.Add(time.Minute)}, nil)

		_, err := f.uc.RequestOTP(context.Background(), domain.RequestPasswordResetInput{Email: testEmail})

		assert.ErrorIs(t, err, domain.ErrOTPLocked)
		f.otp.AssertNotCalled(t, "IssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestPasswordReset_Request_InvalidEmail(t *testing.T) {
	f := newResetFixture()

	_, err := f.uc.RequestOTP(context.Background(), domain.RequestPasswordResetInput{Email: "nope"})

	assert.ErrorIs(t, err, domain.ErrInvalidEmail)
	assert.Empty(t, f.slept, "answered at once (AC27)")
	f.resets.AssertNotCalled(t, "FindVerifiedUserByEmail", mock.Anything, mock.Anything)
}

func TestPasswordReset_Confirm_Success(t *testing.T) {
	const code = "042917"
	f := newResetFixture()
	row := otpRow(t, code)
	row.Purpose = domain.OTPPurposeResetPassword
	lockUntil := testNow.Add(testOTPPolicy.LockDuration.Duration())
	f.otp.On("FindActiveOTP", mock.Anything, testEmail, domain.OTPPurposeResetPassword).Return(row, nil)
	f.otp.On("ConsumeOTPAttempt", mock.Anything, row.ID, testOTPPolicy.MaxAttempts, lockUntil).Return(1, true, nil)
	f.resets.On("FindVerifiedUserByEmail", mock.Anything, testEmail).Return(verifiedUser(), nil)
	var gotHash string
	f.resets.On("ResetPassword", mock.Anything, testUserRowID, mock.AnythingOfType("string"), row.ID).
		Run(func(a mock.Arguments) { gotHash = a.String(2) }).Return(nil)

	err := f.uc.Confirm(context.Background(), domain.ConfirmPasswordResetInput{
		Email: testEmailRaw, OTPCode: code, NewPassword: "a-new-password",
	})

	require.NoError(t, err)
	// Stored with the registration scheme, never the clear password.
	assert.NoError(t, NewBcryptPasswordHasher().Compare(gotHash, "a-new-password"))
	assert.NotContains(t, gotHash, "a-new-password")
}

func TestPasswordReset_Confirm_CodeErrors(t *testing.T) {
	const code = "042917"
	ctx := context.Background()
	in := domain.ConfirmPasswordResetInput{Email: testEmail, OTPCode: "000000", NewPassword: "a-new-password"}

	t.Run("wrong code is OTP_INVALID and nothing changes", func(t *testing.T) {
		f := newResetFixture()
		row := otpRow(t, code)
		f.otp.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(row, nil)
		f.otp.On("ConsumeOTPAttempt", mock.Anything, row.ID, mock.Anything, mock.Anything).Return(1, true, nil)

		assert.ErrorIs(t, f.uc.Confirm(ctx, in), domain.ErrOTPInvalid)
		f.resets.AssertNotCalled(t, "ResetPassword", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("no code (unknown email) is OTP_EXPIRED", func(t *testing.T) {
		f := newResetFixture()
		f.otp.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrOTPInvalid)

		assert.ErrorIs(t, f.uc.Confirm(ctx, in), domain.ErrOTPExpired)
	})

	t.Run("locked code is OTP_LOCKED", func(t *testing.T) {
		f := newResetFixture()
		until := testNow.Add(10 * time.Minute)
		row := otpRow(t, code)
		row.LockedUntil = &until
		f.otp.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(row, nil)

		assert.ErrorIs(t, f.uc.Confirm(ctx, in), domain.ErrOTPLocked)
	})

	t.Run("code consumed by a concurrent confirm is OTP_EXPIRED", func(t *testing.T) {
		f := newResetFixture()
		row := otpRow(t, code)
		f.otp.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(row, nil)
		f.otp.On("ConsumeOTPAttempt", mock.Anything, row.ID, mock.Anything, mock.Anything).Return(1, true, nil)
		f.resets.On("FindVerifiedUserByEmail", mock.Anything, mock.Anything).Return(verifiedUser(), nil)
		f.resets.On("ResetPassword", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(domain.ErrOTPExpired)

		in := in
		in.OTPCode = code
		assert.ErrorIs(t, f.uc.Confirm(ctx, in), domain.ErrOTPExpired)
	})
}

func TestPasswordReset_Confirm_Validation(t *testing.T) {
	cases := map[string]struct {
		in   domain.ConfirmPasswordResetInput
		want error
	}{
		"bad email":      {domain.ConfirmPasswordResetInput{Email: "x", OTPCode: "042917", NewPassword: "a-new-password"}, domain.ErrInvalidEmail},
		"short password": {domain.ConfirmPasswordResetInput{Email: testEmail, OTPCode: "042917", NewPassword: "short"}, domain.ErrInvalidPassword},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := newResetFixture()

			assert.ErrorIs(t, f.uc.Confirm(context.Background(), tc.in), tc.want)
			f.otp.AssertNotCalled(t, "FindActiveOTP", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestPasswordReset_Request_BudgetExhaustedIs429NotSwallowed(t *testing.T) {
	f := newResetFixture()
	f.resets.On("FindVerifiedUserByEmail", mock.Anything, mock.Anything).Return(verifiedUser(), nil)
	f.otp.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrOTPInvalid)
	f.otp.On("IssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&domain.SendState{}, nil)
	f.mailer.On("SendPasswordResetCode", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&domain.RateLimitedError{RetryAfter: time.Hour})
	f.otp.On("ReleaseSendLimit", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := f.uc.RequestOTP(context.Background(), domain.RequestPasswordResetInput{Email: testEmail})

	assert.ErrorIs(t, err, domain.ErrRateLimited, "AC33: the exhausted budget is answered 429")
	f.otp.AssertCalled(t, "ReleaseSendLimit", mock.Anything, testEmail, domain.OTPPurposeResetPassword)
}
