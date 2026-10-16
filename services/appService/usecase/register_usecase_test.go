package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"chjpcoj.com.innerlamp/pkg/jwt"
	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// Tests for US-001 (onboarding v8.0, email verification). Mapping of
// requirements.md AC → test:
//
//	AC1  new email: pending + code, 200 numbers, no user    -> TestRequestOTP_NewEmail
//	AC2  email with account: same work, same body, notice   -> TestRequestOTP_EmailHasAccount
//	AC3  invalid email                                      -> TestRequestOTP_Validation
//	AC4  phone for its country, E.164 stored                -> TestRequestOTP_Validation, TestRequestOTP_NewEmail
//	AC5  password 8–64 after NFC                            -> TestRequestOTP_Validation
//	AC6  full name / dharma name                            -> TestRequestOTP_Validation, TestRequestOTP_BlankOptionalFields
//	     (validation order email → phone → password → names) -> TestRequestOTP_ValidationOrder
//	AC7  phone on an account → 409, nothing sent             -> TestRequestOTP_PhoneTaken
//	     (another pending's phone does not block: repository.TestRegistrationRepository_PhoneTaken)
//	AC8  unknown temple / blank temple                      -> TestRequestOTP_Temple
//	AC9  send policy refusal, both branches                 -> TestRequestOTP_SendRefused
//	AC10 mailer failure releases the slot                   -> TestRequestOTP_MailerFailure, TestResendOTP_MailerFailure
//	AC11 resend: new code, notice for account_exists        -> TestResendOTP_HappyPath, TestResendOTP_AccountExists, TestResendOTP_SendRefused
//	AC12 resend without live pending                        -> TestResendOTP_PendingGone
//	AC13 send lock does not block verify                    -> TestVerifyOTP_HappyPath (strict mocks: no send-limit call)
//	                                                           + repository.TestOTPRepository_SendLockDoesNotBlockVerify
//	AC14 verify creates user + session                      -> TestVerifyOTP_HappyPath, TestVerifyOTP_TokenIssuerContract
//	AC15 wrong / expired / locked                           -> TestVerifyOTP_CodeOutcomes (+ otp_service_test.go)
//	AC16 account_exists: every code wrong                   -> TestVerifyOTP_AccountExists
//	AC17 pending expired / missing                          -> TestVerifyOTP_PendingGone
//	AC18 phone taken / temple gone at creation              -> TestVerifyOTP_CreationConflicts
//	AC19 password hashed, never logged or emailed           -> TestRequestOTP_NewEmail, TestRequestOTP_NoSecretsLogged
//	AC20 random code, only hash stored, only in the email   -> TestRequestOTP_NewEmail, otp_service_test.go
//
// AC21 is config.TestValidate*; AC22 (per-IP limits) is deferred to login
// Phase 5; AC23 is the unchanged temple-suggestions route.

const (
	// Shared with login_usecase_test.go (login is still phone-based).
	testPhoneNumber    = "+84901234567"
	testPhoneNumberRaw = "0901234567"
	testEmail          = "user@example.com"
	testPassword       = "supersecret"

	testEmailRaw      = "  User@Example.com  "
	testPhoneCountry  = "VN"
	testPhoneTyped    = "090 123 4567"
	testDharmaName    = "Thích Nhất Hạnh"
	testFullName      = "Nguyễn Văn An"
	testTempleID      = "temple_north_011"
	testOTPCode       = "042917"
	testUserID        = "42"
	testSessionID     = "9b1f5c1e-6f0a-4a57-9d55-0f4a6f6a0001"
	testAccessToken   = "access-token"
	testRefreshToken  = "refresh-token"
	testSessionTTL    = 168 * time.Hour
	testPendingTTL    = 24 * time.Hour
	testDharmaNameMax = 255
	testFullNameMax   = 300
)

// testOTPPolicy mirrors config.dev.yaml's `otp:` block, minus dev_fixed_code:
// the tests exercise the random path. Pinned literals, not read back from
// config, so a wiring mistake cannot make the test agree with itself.
var testOTPPolicy = config.OTPConfig{
	TTL:                    config.Duration(600 * time.Second),
	SendCooldown:           config.Duration(60 * time.Second),
	MaxSendAttempts:        3,
	SendLockDuration:       config.Duration(30 * time.Minute),
	MaxAttempts:            3,
	LockDuration:           config.Duration(30 * time.Minute),
	PendingRegistrationTTL: config.Duration(24 * time.Hour),
}

type registerFixture struct {
	usecase *RegisterUsecase
	repo    *domain.MockRegistrationRepository
	otpRepo *domain.MockOTPRepository
	mailer  *domain.MockAuthMailer
	jwt     *jwt.MockService
	logger  *logging.MockLogger
}

// newRegisterFixture wires the real OTPService and the real
// BcryptPasswordHasher over strict mocks, with both clocks fixed at testNow.
// An unexpected repository or mailer call panics, which is how "writes
// nothing" / "sends nothing" are proven.
func newRegisterFixture() *registerFixture {
	repo := &domain.MockRegistrationRepository{}
	otpRepo := &domain.MockOTPRepository{}
	mailer := &domain.MockAuthMailer{}
	jwtService := &jwt.MockService{}
	logger := newLogger()

	otpService := NewOTPService(otpRepo, logger, testOTPPolicy)
	otpService.now = func() time.Time { return testNow }
	uc := NewRegisterUsecase(repo, otpService, mailer, NewBcryptPasswordHasher(), jwtService, logger,
		testSessionTTL, testPendingTTL, testDharmaNameMax, testFullNameMax)
	uc.now = func() time.Time { return testNow }

	return &registerFixture{usecase: uc, repo: repo, otpRepo: otpRepo, mailer: mailer, jwt: jwtService, logger: logger}
}

// newLogger stubs every method a use case may reach so a test only has to add
// the expectations it actually asserts on.
func newLogger() *logging.MockLogger {
	l := &logging.MockLogger{}
	l.On("Debug", mock.Anything, mock.Anything).Return()
	l.On("Info", mock.Anything, mock.Anything).Return()
	l.On("Warn", mock.Anything, mock.Anything).Return()
	l.On("Error", mock.Anything, mock.Anything).Return()
	l.On("WithFields", mock.Anything).Return(l)
	l.On("WithContext", mock.Anything).Return(l)
	l.On("WithError", mock.Anything).Return(l)
	return l
}

func defaultRequestInput() domain.RequestRegisterOTPInput {
	return domain.RequestRegisterOTPInput{
		Email:              testEmailRaw,
		Password:           testPassword,
		PhoneCountry:       testPhoneCountry,
		PhoneNumber:        testPhoneTyped,
		FullName:           "  " + testFullName + "  ",
		DharmaName:         testDharmaName,
		OrdinationTempleID: testTempleID,
	}
}

// requestCapture records what step 1 handed the repository and the mailer.
type requestCapture struct {
	pending    *domain.PendingRegistration
	otpHash    string
	expiresAt  time.Time
	limits     domain.SendLimits
	canReplace func(*domain.PendingRegistration) bool
	mailed     string // the code emailed, "" for the notice
	notice     bool
}

// expectRequestHappy stubs every collaborator of a request-otp that gets
// through, for an email with (accountExists) or without an account.
func (f *registerFixture) expectRequestHappy(accountExists bool) *requestCapture {
	c := &requestCapture{}
	f.repo.On("PhoneTaken", mock.Anything, testPhoneNumber).Return(false, nil)
	f.repo.On("TempleExists", mock.Anything, testTempleID).Return(true, nil)
	f.repo.On("EmailHasAccount", mock.Anything, testEmail).Return(accountExists, nil)
	f.otpRepo.On("FindActiveOTP", mock.Anything, testEmail, domain.OTPPurposeRegister).Return(nil, domain.ErrOTPInvalid)
	f.repo.On("SavePendingAndIssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			c.pending = args.Get(1).(*domain.PendingRegistration)
			c.otpHash = args.String(2)
			c.expiresAt = args.Get(3).(time.Time)
			c.limits = args.Get(4).(domain.SendLimits)
			c.canReplace, _ = args.Get(5).(func(*domain.PendingRegistration) bool)
		}).
		Return(&domain.SendState{Remaining: 3}, nil)
	if accountExists {
		f.mailer.On("SendAccountExistsNotice", mock.Anything, testEmail).
			Run(func(mock.Arguments) { c.notice = true }).Return(nil)
	} else {
		f.mailer.On("SendRegisterCode", mock.Anything, testEmail, mock.AnythingOfType("string"), 600*time.Second).
			Run(func(args mock.Arguments) { c.mailed = args.String(2) }).Return(nil)
	}
	return c
}

func TestRequestOTP_NewEmail(t *testing.T) {
	f := newRegisterFixture()
	c := f.expectRequestHappy(false)

	res, err := f.usecase.RequestOTP(context.Background(), defaultRequestInput())

	require.NoError(t, err)
	assert.Equal(t, &domain.RequestOTPResult{ExpiresInSeconds: 600, ResendAfterSeconds: 60}, res)

	// AC1: payload keyed by the normalised email; AC4: E.164 phone.
	require.NotNil(t, c.pending)
	assert.Equal(t, testEmail, c.pending.Email)
	assert.Equal(t, testPhoneNumber, c.pending.PhoneNumber)
	assert.Equal(t, testFullName, *c.pending.FullName, "trimmed")
	assert.Equal(t, testDharmaName, *c.pending.DharmaName)
	assert.Equal(t, testTempleID, *c.pending.OrdinationTempleID)
	assert.False(t, c.pending.AccountExists)
	assert.Equal(t, testNow.Add(testPendingTTL), c.pending.ExpiresAt)
	assert.Equal(t, testNow.Add(600*time.Second), c.expiresAt, "code expires after otp.ttl")
	assert.Equal(t, domain.SendLimits{Cooldown: time.Minute, LockDuration: 30 * time.Minute, MaxAttempts: 3}, c.limits)

	// AC19: only the hash is stored, and it is the shared hasher's scheme.
	assert.NotContains(t, c.pending.PasswordHash, testPassword)
	require.NoError(t, NewBcryptPasswordHasher().Compare(c.pending.PasswordHash, testPassword))

	// AC20: the emailed code is six digits and only its hash is stored.
	assert.Regexp(t, sixDigits, c.mailed)
	assert.NotEqual(t, c.mailed, c.otpHash)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(c.otpHash), []byte(c.mailed)))

	f.repo.AssertNotCalled(t, "CreateUserAndSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	f.mailer.AssertNotCalled(t, "SendAccountExistsNotice", mock.Anything, mock.Anything)
}

func TestRequestOTP_EmailHasAccount(t *testing.T) {
	newF := newRegisterFixture()
	newF.expectRequestHappy(false)
	newRes, err := newF.usecase.RequestOTP(context.Background(), defaultRequestInput())
	require.NoError(t, err)

	f := newRegisterFixture()
	c := f.expectRequestHappy(true)
	res, err := f.usecase.RequestOTP(context.Background(), defaultRequestInput())

	require.NoError(t, err)
	assert.Equal(t, newRes, res, "AC2: same body as a new email")
	require.NotNil(t, c.pending)
	assert.True(t, c.pending.AccountExists)
	// Same work: password hashed, code generated and hashed, one write, one mail.
	require.NoError(t, NewBcryptPasswordHasher().Compare(c.pending.PasswordHash, testPassword))
	_, err = bcrypt.Cost([]byte(c.otpHash))
	require.NoError(t, err, "a real code hash is stored even though the code is never disclosed")
	assert.True(t, c.notice)
	f.mailer.AssertNotCalled(t, "SendRegisterCode", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	f.repo.AssertNumberOfCalls(t, "SavePendingAndIssueOTP", 1)
}

func TestRequestOTP_Validation(t *testing.T) {
	long := func(n int) string { return strings.Repeat("ạ", n) }
	cases := []struct {
		name    string
		mutate  func(*domain.RequestRegisterOTPInput)
		wantErr error
	}{
		{"AC3 empty email", func(in *domain.RequestRegisterOTPInput) { in.Email = "  " }, domain.ErrInvalidEmail},
		{"AC3 no domain dot", func(in *domain.RequestRegisterOTPInput) { in.Email = "a@localhost" }, domain.ErrInvalidEmail},
		{"AC3 no at", func(in *domain.RequestRegisterOTPInput) { in.Email = "user.example.com" }, domain.ErrInvalidEmail},
		{"AC4 unknown country", func(in *domain.RequestRegisterOTPInput) { in.PhoneCountry = "GB" }, domain.ErrInvalidPhone},
		{"AC4 empty country", func(in *domain.RequestRegisterOTPInput) { in.PhoneCountry = "" }, domain.ErrInvalidPhone},
		{"AC4 too few digits", func(in *domain.RequestRegisterOTPInput) { in.PhoneNumber = "09012345" }, domain.ErrInvalidPhone},
		{"AC4 too many digits", func(in *domain.RequestRegisterOTPInput) { in.PhoneNumber = "0901234567890" }, domain.ErrInvalidPhone},
		{"AC4 plus sign typed", func(in *domain.RequestRegisterOTPInput) { in.PhoneNumber = "+84901234567" }, domain.ErrInvalidPhone},
		{"AC5 7 characters", func(in *domain.RequestRegisterOTPInput) { in.Password = "1234567" }, domain.ErrInvalidPassword},
		{"AC5 65 characters", func(in *domain.RequestRegisterOTPInput) { in.Password = long(65) }, domain.ErrInvalidPassword},
		{"AC5 empty", func(in *domain.RequestRegisterOTPInput) { in.Password = "" }, domain.ErrInvalidPassword},
		{"AC6 blank full name", func(in *domain.RequestRegisterOTPInput) { in.FullName = "   " }, domain.ErrInvalidFullName},
		{"AC6 full name too long", func(in *domain.RequestRegisterOTPInput) { in.FullName = long(301) }, domain.ErrInvalidFullName},
		{"AC6 dharma name too long", func(in *domain.RequestRegisterOTPInput) { in.DharmaName = long(256) }, domain.ErrInvalidDharmaName},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newRegisterFixture() // strict mocks: any write or lookup panics
			in := defaultRequestInput()
			tc.mutate(&in)

			_, err := f.usecase.RequestOTP(context.Background(), in)

			assert.ErrorIs(t, err, tc.wantErr)
		})
	}

	t.Run("AC5 64 multi-byte characters accepted", func(t *testing.T) {
		f := newRegisterFixture()
		c := f.expectRequestHappy(false)
		in := defaultRequestInput()
		in.Password = long(64) // 128 bytes, beyond bcrypt's 72

		_, err := f.usecase.RequestOTP(context.Background(), in)

		require.NoError(t, err)
		require.NoError(t, NewBcryptPasswordHasher().Compare(c.pending.PasswordHash, in.Password))
		assert.Error(t, NewBcryptPasswordHasher().Compare(c.pending.PasswordHash, long(63)+"a"), "the 64th character counts")
	})
}

func TestRequestOTP_ValidationOrder(t *testing.T) {
	in := domain.RequestRegisterOTPInput{Email: "bad", PhoneCountry: "XX", Password: "1", FullName: ""}
	steps := []struct {
		fix     func(*domain.RequestRegisterOTPInput)
		wantErr error
	}{
		{func(*domain.RequestRegisterOTPInput) {}, domain.ErrInvalidEmail},
		{func(i *domain.RequestRegisterOTPInput) { i.Email = testEmail }, domain.ErrInvalidPhone},
		{func(i *domain.RequestRegisterOTPInput) {
			i.PhoneCountry, i.PhoneNumber = testPhoneCountry, testPhoneTyped
		}, domain.ErrInvalidPassword},
		{func(i *domain.RequestRegisterOTPInput) { i.Password = testPassword }, domain.ErrInvalidFullName},
	}
	for _, s := range steps {
		s.fix(&in)
		_, err := newRegisterFixture().usecase.RequestOTP(context.Background(), in)
		assert.ErrorIs(t, err, s.wantErr)
	}
}

func TestRequestOTP_BlankOptionalFields(t *testing.T) {
	f := newRegisterFixture()
	c := f.expectRequestHappy(false)
	in := defaultRequestInput()
	in.DharmaName = "   "
	in.OrdinationTempleID = ""

	_, err := f.usecase.RequestOTP(context.Background(), in)

	require.NoError(t, err)
	assert.Nil(t, c.pending.DharmaName, "AC6: blank dharma name stored as NULL")
	assert.Nil(t, c.pending.OrdinationTempleID, "AC8: blank temple stored as NULL")
	f.repo.AssertNotCalled(t, "TempleExists", mock.Anything, mock.Anything)
}

func TestRequestOTP_PhoneTaken(t *testing.T) {
	f := newRegisterFixture()
	f.repo.On("PhoneTaken", mock.Anything, testPhoneNumber).Return(true, nil)

	_, err := f.usecase.RequestOTP(context.Background(), defaultRequestInput())

	assert.ErrorIs(t, err, domain.ErrPhoneTaken)
	// Strict mocks: no email lookup (nothing learned about the email), no
	// write, no mail.
}

func TestRequestOTP_Temple(t *testing.T) {
	t.Run("unknown", func(t *testing.T) {
		f := newRegisterFixture()
		f.repo.On("PhoneTaken", mock.Anything, mock.Anything).Return(false, nil)
		f.repo.On("TempleExists", mock.Anything, testTempleID).Return(false, nil)

		_, err := f.usecase.RequestOTP(context.Background(), defaultRequestInput())

		assert.ErrorIs(t, err, domain.ErrInvalidTemple)
	})
	t.Run("lookup failure is infrastructure", func(t *testing.T) {
		f := newRegisterFixture()
		f.repo.On("PhoneTaken", mock.Anything, mock.Anything).Return(false, nil)
		f.repo.On("TempleExists", mock.Anything, mock.Anything).Return(false, errors.New("db down"))

		_, err := f.usecase.RequestOTP(context.Background(), defaultRequestInput())

		require.Error(t, err)
		assert.NotErrorIs(t, err, domain.ErrInvalidTemple)
	})
}

// stubUpToSave stubs request-otp up to (not including) the pending write.
func (f *registerFixture) stubUpToSave(accountExists bool) {
	f.repo.On("PhoneTaken", mock.Anything, mock.Anything).Return(false, nil)
	f.repo.On("TempleExists", mock.Anything, mock.Anything).Return(true, nil)
	f.repo.On("EmailHasAccount", mock.Anything, mock.Anything).Return(accountExists, nil)
	f.otpRepo.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrOTPInvalid)
}

func TestRequestOTP_SendRefused(t *testing.T) {
	refusals := map[string]error{
		"cooldown": &domain.OTPResendTooSoonError{RetryAfterSeconds: 37},
		"cap":      &domain.OTPSendLimitedError{LockedUntil: testNow.Add(30 * time.Minute)},
	}
	for name, refusal := range refusals {
		for _, accountExists := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/account_exists=%v", name, accountExists), func(t *testing.T) {
				f := newRegisterFixture()
				f.stubUpToSave(accountExists)
				f.repo.On("SavePendingAndIssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, refusal)

				_, err := f.usecase.RequestOTP(context.Background(), defaultRequestInput())

				assert.ErrorIs(t, err, refusal, "AC9: the typed refusal reaches delivery, identical for both branches")
				// strict mailer mock: nothing sent
			})
		}
	}
}

func TestRequestOTP_CodeLocked(t *testing.T) {
	f := newRegisterFixture()
	until := testNow.Add(20 * time.Minute)
	f.repo.On("PhoneTaken", mock.Anything, mock.Anything).Return(false, nil)
	f.repo.On("TempleExists", mock.Anything, mock.Anything).Return(true, nil)
	f.repo.On("EmailHasAccount", mock.Anything, mock.Anything).Return(false, nil)
	f.otpRepo.On("FindActiveOTP", mock.Anything, testEmail, domain.OTPPurposeRegister).Return(&domain.OTPCode{ID: 1, LockedUntil: &until}, nil)

	_, err := f.usecase.RequestOTP(context.Background(), defaultRequestInput())

	locked, ok := errors.AsType[*domain.OTPLockedError](err)
	require.True(t, ok, "a fresh code must not reset the wrong-guess lock; got %v", err)
	assert.Equal(t, until, locked.LockedUntil)
}

func TestRequestOTP_MailerFailure(t *testing.T) {
	for _, accountExists := range []bool{false, true} {
		t.Run(fmt.Sprintf("account_exists=%v", accountExists), func(t *testing.T) {
			f := newRegisterFixture()
			f.stubUpToSave(accountExists)
			f.repo.On("SavePendingAndIssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&domain.SendState{}, nil)
			mailErr := errors.New("smtp: 451 try later")
			if accountExists {
				f.mailer.On("SendAccountExistsNotice", mock.Anything, testEmail).Return(mailErr)
			} else {
				f.mailer.On("SendRegisterCode", mock.Anything, testEmail, mock.Anything, mock.Anything).Return(mailErr)
			}
			f.otpRepo.On("ReleaseSendLimit", mock.Anything, testEmail, domain.OTPPurposeRegister).Return(nil)
			f.otpRepo.On("InvalidateActiveOTP", mock.Anything, testEmail, domain.OTPPurposeRegister).Return(nil)

			_, err := f.usecase.RequestOTP(context.Background(), defaultRequestInput())

			assert.ErrorIs(t, err, domain.ErrEmailSendFailed)
			f.otpRepo.AssertCalled(t, "ReleaseSendLimit", mock.Anything, testEmail, domain.OTPPurposeRegister)
			// The code the user never received must not stay live: it would make a
			// corrected resubmission look like a clash (REGISTRATION_IN_PROGRESS).
			// Mã người dùng chưa nhận không được còn hiệu lực.
			f.otpRepo.AssertCalled(t, "InvalidateActiveOTP", mock.Anything, testEmail, domain.OTPPurposeRegister)
		})
	}
}

// A failure to invalidate the undelivered code is only logged: the caller still
// learns that the email failed, and the send slot is still released.
// Lỗi vô hiệu mã chỉ ghi log, người dùng vẫn nhận EMAIL_SEND_FAILED.
func TestRequestOTP_MailerFailure_InvalidateFails(t *testing.T) {
	f := newRegisterFixture()
	f.stubUpToSave(false)
	f.repo.On("SavePendingAndIssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&domain.SendState{}, nil)
	f.mailer.On("SendRegisterCode", mock.Anything, testEmail, mock.Anything, mock.Anything).Return(errors.New("smtp down"))
	f.otpRepo.On("ReleaseSendLimit", mock.Anything, testEmail, domain.OTPPurposeRegister).Return(nil)
	f.otpRepo.On("InvalidateActiveOTP", mock.Anything, testEmail, domain.OTPPurposeRegister).Return(errors.New("db down"))

	_, err := f.usecase.RequestOTP(context.Background(), defaultRequestInput())

	assert.ErrorIs(t, err, domain.ErrEmailSendFailed)
	f.otpRepo.AssertCalled(t, "ReleaseSendLimit", mock.Anything, testEmail, domain.OTPPurposeRegister)
	f.otpRepo.AssertCalled(t, "InvalidateActiveOTP", mock.Anything, testEmail, domain.OTPPurposeRegister)
}

// Login AC33: an exhausted daily mail budget is a 429 with its wait, not a 500
// EMAIL_SEND_FAILED, and the user's own send slot is released.
func TestRequestOTP_MailBudgetExhausted(t *testing.T) {
	f := newRegisterFixture()
	f.stubUpToSave(false)
	f.repo.On("SavePendingAndIssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&domain.SendState{}, nil)
	f.mailer.On("SendRegisterCode", mock.Anything, testEmail, mock.Anything, mock.Anything).
		Return(&domain.RateLimitedError{RetryAfter: time.Hour})
	f.otpRepo.On("ReleaseSendLimit", mock.Anything, testEmail, domain.OTPPurposeRegister).Return(nil)
	f.otpRepo.On("InvalidateActiveOTP", mock.Anything, testEmail, domain.OTPPurposeRegister).Return(nil)

	_, err := f.usecase.RequestOTP(context.Background(), defaultRequestInput())

	rl, ok := errors.AsType[*domain.RateLimitedError](err)
	require.True(t, ok, "got %v", err)
	assert.Equal(t, time.Hour, rl.RetryAfter)
	assert.NotErrorIs(t, err, domain.ErrEmailSendFailed)
	f.otpRepo.AssertCalled(t, "ReleaseSendLimit", mock.Anything, testEmail, domain.OTPPurposeRegister)
	f.otpRepo.AssertCalled(t, "InvalidateActiveOTP", mock.Anything, testEmail, domain.OTPPurposeRegister)
}

func TestRequestOTP_RepositoryFailure(t *testing.T) {
	f := newRegisterFixture()
	f.stubUpToSave(false)
	f.repo.On("SavePendingAndIssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("connection reset"))

	_, err := f.usecase.RequestOTP(context.Background(), defaultRequestInput())

	require.Error(t, err)
	for _, sentinel := range []error{domain.ErrPhoneTaken, domain.ErrOTPResendTooSoon, domain.ErrOTPSendLimited, domain.ErrEmailSendFailed} {
		assert.NotErrorIs(t, err, sentinel)
	}
}

// TestRequestOTP_NoSecretsLogged pins AC19/AC20: neither the password nor the
// code appears in any log call.
func TestRequestOTP_NoSecretsLogged(t *testing.T) {
	f := newRegisterFixture()
	c := f.expectRequestHappy(false)

	_, err := f.usecase.RequestOTP(context.Background(), defaultRequestInput())
	require.NoError(t, err)

	for _, call := range f.logger.Calls {
		for _, arg := range call.Arguments {
			s := fmt.Sprint(arg)
			assert.NotContains(t, s, testPassword, "password logged by %s", call.Method)
			assert.NotContains(t, s, c.mailed, "code logged by %s", call.Method)
		}
	}
}

func pendingRow(accountExists bool) *domain.PendingRegistration {
	name := testFullName
	return &domain.PendingRegistration{
		ID:            5,
		Email:         testEmail,
		PhoneNumber:   testPhoneNumber,
		PasswordHash:  "hash",
		FullName:      &name,
		AccountExists: accountExists,
		ExpiresAt:     testNow.Add(time.Hour),
	}
}

func TestResendOTP_HappyPath(t *testing.T) {
	f := newRegisterFixture()
	f.repo.On("FindPendingRegistration", mock.Anything, testEmail).Return(pendingRow(false), nil)
	f.otpRepo.On("FindActiveOTP", mock.Anything, testEmail, domain.OTPPurposeRegister).Return(nil, domain.ErrOTPInvalid)
	var storedHash, mailed string
	f.otpRepo.On("IssueOTP", mock.Anything, testEmail, domain.OTPPurposeRegister, mock.Anything, testNow.Add(600*time.Second), mock.Anything).
		Run(func(args mock.Arguments) { storedHash = args.String(3) }).
		Return(&domain.SendState{SendCount: 2, Remaining: 1}, nil)
	f.mailer.On("SendRegisterCode", mock.Anything, testEmail, mock.Anything, 600*time.Second).
		Run(func(args mock.Arguments) { mailed = args.String(2) }).Return(nil)

	res, err := f.usecase.ResendOTP(context.Background(), domain.ResendOTPInput{Email: testEmailRaw})

	require.NoError(t, err)
	assert.Equal(t, &domain.ResendOTPResult{ExpiresInSeconds: 600, ResendAfterSeconds: 60, ResendRemaining: 1}, res)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(mailed)), "the emailed code is the issued one")
}

func TestResendOTP_AccountExists(t *testing.T) {
	f := newRegisterFixture()
	f.repo.On("FindPendingRegistration", mock.Anything, testEmail).Return(pendingRow(true), nil)
	f.otpRepo.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrOTPInvalid)
	f.otpRepo.On("IssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&domain.SendState{Remaining: 2}, nil)
	f.mailer.On("SendAccountExistsNotice", mock.Anything, testEmail).Return(nil)

	res, err := f.usecase.ResendOTP(context.Background(), domain.ResendOTPInput{Email: testEmail})

	require.NoError(t, err)
	assert.Equal(t, 2, res.ResendRemaining)
	f.mailer.AssertNotCalled(t, "SendRegisterCode", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestResendOTP_InvalidEmail(t *testing.T) {
	_, err := newRegisterFixture().usecase.ResendOTP(context.Background(), domain.ResendOTPInput{Email: "nope"})
	assert.ErrorIs(t, err, domain.ErrInvalidEmail)
}

func TestResendOTP_PendingGone(t *testing.T) {
	for _, repoErr := range []error{domain.ErrPendingRegistrationNotFound, domain.ErrPendingRegistrationExpired} {
		t.Run(repoErr.Error(), func(t *testing.T) {
			f := newRegisterFixture()
			f.repo.On("FindPendingRegistration", mock.Anything, testEmail).Return(nil, repoErr)

			_, err := f.usecase.ResendOTP(context.Background(), domain.ResendOTPInput{Email: testEmail})

			assert.ErrorIs(t, err, domain.ErrPendingRegistrationExpired, "AC12: both read as expired")
		})
	}
}

func TestResendOTP_SendRefused(t *testing.T) {
	f := newRegisterFixture()
	refusal := &domain.OTPSendLimitedError{LockedUntil: testNow.Add(30 * time.Minute)}
	f.repo.On("FindPendingRegistration", mock.Anything, mock.Anything).Return(pendingRow(false), nil)
	f.otpRepo.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrOTPInvalid)
	f.otpRepo.On("IssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, refusal)

	_, err := f.usecase.ResendOTP(context.Background(), domain.ResendOTPInput{Email: testEmail})

	assert.ErrorIs(t, err, refusal)
}

func TestResendOTP_MailerFailure(t *testing.T) {
	f := newRegisterFixture()
	f.repo.On("FindPendingRegistration", mock.Anything, mock.Anything).Return(pendingRow(false), nil)
	f.otpRepo.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrOTPInvalid)
	f.otpRepo.On("IssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&domain.SendState{}, nil)
	f.mailer.On("SendRegisterCode", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("down"))
	f.otpRepo.On("ReleaseSendLimit", mock.Anything, testEmail, domain.OTPPurposeRegister).Return(nil)
	f.otpRepo.On("InvalidateActiveOTP", mock.Anything, testEmail, domain.OTPPurposeRegister).Return(nil)

	_, err := f.usecase.ResendOTP(context.Background(), domain.ResendOTPInput{Email: testEmail})

	assert.ErrorIs(t, err, domain.ErrEmailSendFailed)
	f.otpRepo.AssertExpectations(t)
}

// stubVerifyCode makes FindActiveOTP return a live row for testOTPCode and
// lets the attempt reservation succeed.
func (f *registerFixture) stubVerifyCode(t *testing.T) *domain.OTPCode {
	row := otpRow(t, testOTPCode)
	f.otpRepo.On("FindActiveOTP", mock.Anything, testEmail, domain.OTPPurposeRegister).Return(row, nil)
	f.otpRepo.On("ConsumeOTPAttempt", mock.Anything, row.ID, testOTPPolicy.MaxAttempts, mock.Anything).Return(1, true, nil)
	return row
}

func TestVerifyOTP_HappyPath(t *testing.T) {
	f := newRegisterFixture()
	pending := pendingRow(false)
	f.repo.On("FindPendingRegistration", mock.Anything, testEmail).Return(pending, nil)
	row := f.stubVerifyCode(t)
	want := &domain.RegisterResult{User: &domain.User{ID: 42, Email: testEmail}, AccessToken: testAccessToken, RefreshToken: testRefreshToken}
	f.repo.On("CreateUserAndSession", mock.Anything, pending, row.ID, testSessionTTL, mock.AnythingOfType("domain.TokenIssuer")).Return(want, nil)

	got, err := f.usecase.VerifyOTP(context.Background(), domain.VerifyRegisterOTPInput{Email: testEmailRaw, OTPCode: testOTPCode})

	require.NoError(t, err)
	assert.Same(t, want, got)
	f.repo.AssertExpectations(t)
	// AC13: strict mocks — verify never consulted or touched the send limit.
	f.otpRepo.AssertNotCalled(t, "IssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestVerifyOTP_TokenIssuerContract pins that the closure handed to the
// repository really signs both tokens, with the session id as claim "sid".
func TestVerifyOTP_TokenIssuerContract(t *testing.T) {
	f := newRegisterFixture()
	f.repo.On("FindPendingRegistration", mock.Anything, testEmail).Return(pendingRow(false), nil)
	f.stubVerifyCode(t)
	f.jwt.On("GenerateAccessToken", testUserID, testEmail, "user", map[string]any{"sid": testSessionID}).Return(testAccessToken, nil)
	f.jwt.On("GenerateRefreshToken", testUserID, mock.Anything).Return(testRefreshToken, nil)

	var issuer domain.TokenIssuer
	f.repo.On("CreateUserAndSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.AnythingOfType("domain.TokenIssuer")).
		Run(func(args mock.Arguments) { issuer = args.Get(4).(domain.TokenIssuer) }).
		Return(&domain.RegisterResult{User: &domain.User{ID: 42, Email: testEmail}}, nil)

	_, err := f.usecase.VerifyOTP(context.Background(), domain.VerifyRegisterOTPInput{Email: testEmail, OTPCode: testOTPCode})
	require.NoError(t, err)
	require.NotNil(t, issuer)

	access, refresh, err := issuer(testUserID, testSessionID)
	require.NoError(t, err)
	assert.Equal(t, testAccessToken, access)
	assert.Equal(t, testRefreshToken, refresh)

	f.jwt.ExpectedCalls = nil
	f.jwt.On("GenerateAccessToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", errors.New("no key"))
	_, _, err = issuer(testUserID, testSessionID)
	assert.Error(t, err, "a signing failure is an error, never an empty pair")
}

func TestVerifyOTP_CodeOutcomes(t *testing.T) {
	until := testNow.Add(30 * time.Minute)
	t.Run("wrong", func(t *testing.T) {
		f := newRegisterFixture()
		f.repo.On("FindPendingRegistration", mock.Anything, mock.Anything).Return(pendingRow(false), nil)
		f.stubVerifyCode(t)

		_, err := f.usecase.VerifyOTP(context.Background(), domain.VerifyRegisterOTPInput{Email: testEmail, OTPCode: "999999"})

		assert.ErrorIs(t, err, domain.ErrOTPInvalid)
	})
	t.Run("expired", func(t *testing.T) {
		f := newRegisterFixture()
		f.repo.On("FindPendingRegistration", mock.Anything, mock.Anything).Return(pendingRow(false), nil)
		f.otpRepo.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrOTPInvalid)

		_, err := f.usecase.VerifyOTP(context.Background(), domain.VerifyRegisterOTPInput{Email: testEmail, OTPCode: testOTPCode})

		assert.ErrorIs(t, err, domain.ErrOTPExpired)
	})
	t.Run("third wrong guess locks: next attempt is locked", func(t *testing.T) {
		f := newRegisterFixture()
		f.repo.On("FindPendingRegistration", mock.Anything, mock.Anything).Return(pendingRow(false), nil)
		row := otpRow(t, testOTPCode)
		row.LockedUntil = &until
		f.otpRepo.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(row, nil)

		_, err := f.usecase.VerifyOTP(context.Background(), domain.VerifyRegisterOTPInput{Email: testEmail, OTPCode: testOTPCode})

		locked, ok := errors.AsType[*domain.OTPLockedError](err)
		require.True(t, ok)
		assert.Equal(t, until, locked.LockedUntil)
	})
}

func TestVerifyOTP_AccountExists(t *testing.T) {
	f := newRegisterFixture()
	f.repo.On("FindPendingRegistration", mock.Anything, testEmail).Return(pendingRow(true), nil)
	f.stubVerifyCode(t)

	_, err := f.usecase.VerifyOTP(context.Background(), domain.VerifyRegisterOTPInput{Email: testEmail, OTPCode: testOTPCode})

	assert.ErrorIs(t, err, domain.ErrOTPInvalid, "AC16: even the right code is wrong")
	f.otpRepo.AssertCalled(t, "ConsumeOTPAttempt", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	f.repo.AssertNotCalled(t, "CreateUserAndSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestVerifyOTP_PendingGone(t *testing.T) {
	for _, repoErr := range []error{domain.ErrPendingRegistrationNotFound, domain.ErrPendingRegistrationExpired} {
		t.Run(repoErr.Error(), func(t *testing.T) {
			f := newRegisterFixture()
			f.repo.On("FindPendingRegistration", mock.Anything, testEmail).Return(nil, repoErr)

			_, err := f.usecase.VerifyOTP(context.Background(), domain.VerifyRegisterOTPInput{Email: testEmail, OTPCode: testOTPCode})

			assert.ErrorIs(t, err, domain.ErrPendingRegistrationExpired)
		})
	}
	t.Run("infrastructure failure", func(t *testing.T) {
		f := newRegisterFixture()
		f.repo.On("FindPendingRegistration", mock.Anything, mock.Anything).Return(nil, errors.New("db down"))

		_, err := f.usecase.VerifyOTP(context.Background(), domain.VerifyRegisterOTPInput{Email: testEmail, OTPCode: testOTPCode})

		require.Error(t, err)
		assert.NotErrorIs(t, err, domain.ErrPendingRegistrationExpired)
	})
}

func TestVerifyOTP_InvalidEmail(t *testing.T) {
	_, err := newRegisterFixture().usecase.VerifyOTP(context.Background(), domain.VerifyRegisterOTPInput{Email: "x", OTPCode: testOTPCode})
	assert.ErrorIs(t, err, domain.ErrInvalidEmail)
}

func TestVerifyOTP_CreationConflicts(t *testing.T) {
	for _, repoErr := range []error{domain.ErrPhoneTaken, domain.ErrInvalidTemple} {
		t.Run(repoErr.Error(), func(t *testing.T) {
			f := newRegisterFixture()
			f.repo.On("FindPendingRegistration", mock.Anything, mock.Anything).Return(pendingRow(false), nil)
			f.stubVerifyCode(t)
			f.repo.On("CreateUserAndSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, repoErr)

			_, err := f.usecase.VerifyOTP(context.Background(), domain.VerifyRegisterOTPInput{Email: testEmail, OTPCode: testOTPCode})

			assert.ErrorIs(t, err, repoErr)
		})
	}
}

// storedPending is the row a victim's earlier request-otp left behind, hashed
// with the real hasher so the callback's bcrypt Compare is exercised.
func storedPending(t *testing.T, password string) *domain.PendingRegistration {
	t.Helper()
	hash, err := NewBcryptPasswordHasher().Hash(password)
	require.NoError(t, err)
	name, dharma, temple := testFullName, testDharmaName, testTempleID
	return &domain.PendingRegistration{
		Email: testEmail, PhoneNumber: testPhoneNumber, PasswordHash: hash,
		FullName: &name, DharmaName: &dharma, OrdinationTempleID: &temple,
	}
}

// AC24: the callback handed to the repository replaces only the same payload.
func TestRequestOTP_CanReplaceDecision(t *testing.T) {
	for _, accountExists := range []bool{false, true} {
		f := newRegisterFixture()
		c := f.expectRequestHappy(accountExists)
		_, err := f.usecase.RequestOTP(context.Background(), defaultRequestInput())
		require.NoError(t, err)
		require.NotNil(t, c.canReplace)

		t.Run("same payload", func(t *testing.T) {
			assert.True(t, c.canReplace(storedPending(t, testPassword)))
		})
		t.Run("different password", func(t *testing.T) {
			assert.False(t, c.canReplace(storedPending(t, "attacker-password")))
		})
		t.Run("different phone", func(t *testing.T) {
			p := storedPending(t, testPassword)
			p.PhoneNumber = "+84999999999"
			assert.False(t, c.canReplace(p))
		})
		t.Run("different full name", func(t *testing.T) {
			p := storedPending(t, testPassword)
			other := "Someone Else"
			p.FullName = &other
			assert.False(t, c.canReplace(p))
		})
		t.Run("different dharma name", func(t *testing.T) {
			p := storedPending(t, testPassword)
			p.DharmaName = nil
			assert.False(t, c.canReplace(p))
		})
		t.Run("different temple", func(t *testing.T) {
			p := storedPending(t, testPassword)
			p.OrdinationTempleID = nil
			assert.False(t, c.canReplace(p))
		})
	}
}

// AC24: a live code plus a different payload surfaces ErrRegistrationInProgress
// and sends no mail, identically for an email with and without an account.
func TestRequestOTP_RegistrationInProgress(t *testing.T) {
	for _, accountExists := range []bool{false, true} {
		f := newRegisterFixture()
		f.repo.On("PhoneTaken", mock.Anything, testPhoneNumber).Return(false, nil)
		f.repo.On("TempleExists", mock.Anything, testTempleID).Return(true, nil)
		f.repo.On("EmailHasAccount", mock.Anything, testEmail).Return(accountExists, nil)
		f.otpRepo.On("FindActiveOTP", mock.Anything, testEmail, domain.OTPPurposeRegister).Return(nil, domain.ErrOTPInvalid)
		f.repo.On("SavePendingAndIssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, &domain.RegistrationInProgressError{RetryAfterSeconds: 42})

		res, err := f.usecase.RequestOTP(context.Background(), defaultRequestInput())

		assert.Nil(t, res)
		require.ErrorIs(t, err, domain.ErrRegistrationInProgress)
		busy, ok := errors.AsType[*domain.RegistrationInProgressError](err)
		require.True(t, ok)
		assert.Equal(t, 42, busy.RetryAfterSeconds)
		f.mailer.AssertNotCalled(t, "SendRegisterCode", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		f.mailer.AssertNotCalled(t, "SendAccountExistsNotice", mock.Anything, mock.Anything)
	}
}
