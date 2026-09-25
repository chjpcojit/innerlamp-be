package usecase

import (
	"context"
	"errors"
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

// Tests for US-001 (register with SMS OTP). The acceptance criteria each test
// proves are named in the test's own doc comment; the mapping is:
//
//	AC1  RequestOTP sends an OTP and creates no account     -> TestRequestOTP_HappyPath
//	AC2  verify creates the account and logs in             -> TestVerifyOTP_HappyPath
//	AC3  the pending row is consumed with the account       -> TestVerifyOTP_HappyPath
//	AC4  wrong / expired / already-used OTP creates nothing -> TestVerifyOTP_WrongOTP,
//	                                                           TestVerifyOTP_NoActiveOTP,
//	                                                           TestVerifyOTP_PendingExpired
//	AC5  send cooldown, 4-send cap, old OTP invalidated     -> TestRequestOTP_Cooldown,
//	                                                           TestRequestOTP_SendLimited,
//	                                                           TestResendOTP_SendLimitRefusals,
//	                                                           TestResendOTP_HappyPath
//	AC6  phone / email already taken                        -> TestRequestOTP_Conflicts
//	AC7  password shorter than 8 rejected server-side        -> TestRequestOTP_Validation
//	AC8  malformed phone / email rejected                   -> TestRequestOTP_Validation
//	AC9  optional profile fields may be blank               -> TestRequestOTP_OptionalFields
//	AC10 the password is stored hashed                      -> TestRequestOTP_HappyPath
//	AC12 three wrong OTPs lock the number for 30 minutes    -> TestVerifyOTP_LocksAfterMaxAttempts,
//	                                                           TestVerifyOTP_Locked,
//	                                                           TestRequestOTP_LockedNumber
//	                                                           TestResendOTP_Blocked
//
// AC11 (staying logged in across a restart) is only half-satisfiable here: the
// refresh token is issued and stored, but POST /v1/auth/refresh does not exist
// in this scope, so nothing can exchange it for a new access token.
// AC13/AC14 concern login and email-sending flows outside US-001.

const (
	testPhoneNumber    = "+84901234567"
	testPhoneNumberRaw = "0901234567"
	testEmail          = "user@example.com"
	testEmailRaw       = "  User@Example.com  "
	testPassword       = "supersecret"
	testDharmaName     = "Thích Nhất Hạnh"
	testOrdination     = "Chùa Từ Hiếu"

	// testOTPCode mirrors the constant the use case currently generates
	// (generateOTP). Pinning the literal here — rather than reading the
	// function under test — is deliberate: when the OTP becomes random this
	// test must fail and be rewritten around the value actually sent.
	testOTPCode = "111111"

	testUserID       = "42"
	testSessionID    = "9b1f5c1e-6f0a-4a57-9d55-0f4a6f6a0001"
	testAccessToken  = "access-token"
	testRefreshToken = "refresh-token"

	testSessionTTL = 168 * time.Hour

	// testPendingTTL mirrors OTPConfig.PendingRegistrationTTL from
	// config.dev.yaml — the pending payload's own lifetime, a register-specific
	// value now threaded to RegisterUsecase separately from the OTP policy.
	testPendingTTL = 24 * time.Hour
)

// testOTPPolicy mirrors the values in config.dev.yaml's `otp:` block. The use
// case reads its policy from config at runtime, so the fixture has to supply
// one explicitly — and pinning the literals here (rather than reading them back
// out of the config package) is deliberate: a test that asserted against
// whatever the policy happened to contain would pass even if the wiring sent
// the wrong numbers through.
var testOTPPolicy = config.OTPConfig{
	TTL:                    config.Duration(90 * time.Second),
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
	sender  *domain.MockOTPSender
	jwt     *jwt.MockService
	logger  *logging.MockLogger
}

func newRegisterFixture() *registerFixture {
	repo := &domain.MockRegistrationRepository{}
	otpRepo := &domain.MockOTPRepository{}
	sender := &domain.MockOTPSender{}
	jwtService := &jwt.MockService{}
	logger := newLogger()
	otpService := NewOTPService(otpRepo, sender, logger, testOTPPolicy)

	return &registerFixture{
		usecase: NewRegisterUsecase(repo, otpService, jwtService, logger, testSessionTTL, testPendingTTL),
		repo:    repo,
		otpRepo: otpRepo,
		sender:  sender,
		jwt:     jwtService,
		logger:  logger,
	}
}

// newLogger stubs every method the use case may reach so a test only has to add
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

// noConflicts is the FindConflicts return for a phone number and email nobody
// has claimed yet.
func noConflicts() *domain.RegistrationConflicts {
	return &domain.RegistrationConflicts{}
}

// activeOTP builds an OTP row that is neither locked nor used, created at the
// given moment. Tests move createdAt around to get in or out of the cooldown
// window; the send counter itself now lives in otp_send_limits.
func activeOTP(createdAt time.Time) *domain.OTPCode {
	return &domain.OTPCode{
		ID:          7,
		PhoneNumber: testPhoneNumber,
		Purpose:     domain.OTPPurposeRegister,
		OTPHash:     testOTPHash,
		CreatedAt:   createdAt,
		ExpiresAt:   createdAt.Add(90 * time.Second),
	}
}

// testOTPHash is the bcrypt hash of testOTPCode, computed once at package load.
// Deriving it here rather than inside activeOTP keeps that helper free of
// *testing.T: several tests build OTP rows inside table literals and closures,
// where no t is in scope. bcrypt.MinCost because the cost is irrelevant to what
// these tests assert and DefaultCost would add ~60ms per row.
var testOTPHash = func() string {
	hash, err := bcrypt.GenerateFromPassword([]byte(testOTPCode), bcrypt.MinCost)
	if err != nil {
		panic("hashing the test OTP failed: " + err.Error())
	}
	return string(hash)
}()

func defaultRequestInput() domain.RequestRegisterOTPInput {
	return domain.RequestRegisterOTPInput{
		Email:           testEmailRaw,
		Password:        testPassword,
		PhoneNumber:     testPhoneNumberRaw,
		DharmaName:      testDharmaName,
		OrdinationPlace: testOrdination,
	}
}

// ---------------------------------------------------------------------------
// RequestOTP
// ---------------------------------------------------------------------------

// TestRequestOTP_HappyPath covers AC1 (OTP sent, no account created), AC7/AC8
// (inputs that pass validation reach the repository), AC9 and AC10 (the stored
// password is a bcrypt hash, never the plaintext).
func TestRequestOTP_HappyPath(t *testing.T) {
	f := newRegisterFixture()

	var (
		capturedPending   *domain.PendingRegistration
		capturedOTPHash   string
		capturedOTPExpiry time.Time
		order             []string
	)

	f.repo.On("FindConflicts", mock.Anything, testPhoneNumber, testEmail).
		Return(noConflicts(), nil)
	f.otpRepo.On("FindActiveOTP", mock.Anything, testPhoneNumber, domain.OTPPurposeRegister).
		Return(nil, domain.ErrOTPInvalid)
	f.repo.On("SavePendingAndIssueOTP", mock.Anything, mock.AnythingOfType("*domain.PendingRegistration"), mock.AnythingOfType("string"), mock.AnythingOfType("time.Time"), mock.AnythingOfType("domain.SendLimits")).
		Run(func(args mock.Arguments) {
			order = append(order, "save")
			capturedPending = args.Get(1).(*domain.PendingRegistration)
			capturedOTPHash = args.String(2)
			capturedOTPExpiry = args.Get(3).(time.Time)
		}).
		Return(&domain.SendState{SendCount: 0, Remaining: 3}, nil)
	f.sender.On("SendOTP", mock.Anything, testPhoneNumber, testOTPCode, domain.OTPPurposeRegister).
		Run(func(mock.Arguments) { order = append(order, "send") }).
		Return(nil)

	before := time.Now()
	result, err := f.usecase.RequestOTP(context.Background(), defaultRequestInput())
	require.NoError(t, err)

	// AC1: an OTP goes out and no users row is written at this step.
	require.NotNil(t, result)
	assert.Equal(t, 90, result.ExpiresInSeconds)
	assert.Equal(t, 60, result.ResendAfterSeconds)
	f.repo.AssertNotCalled(t, "CreateUserAndSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

	// The SMS is sent after the transaction commits, never before: an OTP the
	// user holds while its row rolled back would be dead on arrival.
	assert.Equal(t, []string{"save", "send"}, order)

	require.NotNil(t, capturedPending)

	// The phone number is normalized before it is stored, so "0901234567" and
	// "+84901234567" collide on the UNIQUE constraint instead of coexisting.
	assert.Equal(t, testPhoneNumber, capturedPending.PhoneNumber)

	// Email is normalized too, so the uniqueness check is case-insensitive.
	assert.Equal(t, testEmail, capturedPending.Email)

	// Optional profile fields travel through untouched when present.
	require.NotNil(t, capturedPending.DharmaName)
	assert.Equal(t, testDharmaName, *capturedPending.DharmaName)
	require.NotNil(t, capturedPending.OrdinationPlace)
	assert.Equal(t, testOrdination, *capturedPending.OrdinationPlace)

	// AC10: what is handed to the repository is a bcrypt hash, not the
	// password. Both halves matter — the plaintext must be unreadable in the
	// column, and the hash must actually verify against it.
	assert.NotEqual(t, testPassword, capturedPending.PasswordHash)
	assert.NotContains(t, capturedPending.PasswordHash, testPassword)
	require.NoError(t, bcrypt.CompareHashAndPassword(
		[]byte(capturedPending.PasswordHash), []byte(testPassword)))

	// The OTP is hashed with the same care as the password: six digits is a
	// 10^6 keyspace, so a fast hash would be reversible from a stolen row.
	assert.NotEqual(t, testOTPCode, capturedOTPHash)
	require.NoError(t, bcrypt.CompareHashAndPassword(
		[]byte(capturedOTPHash), []byte(testOTPCode)))

	// v5.1: the pending payload's deadline is decoupled from the code's — 24h
	// against 90 seconds — so an attempt outlives the 30-minute send lock and
	// can be resumed.
	assert.WithinDuration(t, before.Add(24*time.Hour), capturedPending.ExpiresAt, 5*time.Second)
	assert.WithinDuration(t, before.Add(90*time.Second), capturedOTPExpiry, 5*time.Second)
}

// TestRequestOTP_OptionalFields covers AC9: dharma name and ordination place
// are optional, and a blank one is stored as "no value" rather than as an empty
// string.
func TestRequestOTP_OptionalFields(t *testing.T) {
	tests := []struct {
		name            string
		dharmaName      string
		ordinationPlace string
		wantDharma      *string
		wantOrdination  *string
	}{
		{"both blank", "", "", nil, nil},
		{"whitespace only is blank", "   ", "\t\n", nil, nil},
		{"both set", testDharmaName, testOrdination, ptr(testDharmaName), ptr(testOrdination)},
		{"only dharma name", testDharmaName, "", ptr(testDharmaName), nil},
		{"padded values are trimmed", "  " + testDharmaName + "  ", " " + testOrdination, ptr(testDharmaName), ptr(testOrdination)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRegisterFixture()

			var captured *domain.PendingRegistration
			f.repo.On("FindConflicts", mock.Anything, mock.Anything, mock.Anything).Return(noConflicts(), nil)
			f.otpRepo.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrOTPInvalid)
			f.repo.On("SavePendingAndIssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) {
					captured = args.Get(1).(*domain.PendingRegistration)
				}).
				Return(&domain.SendState{Remaining: 3}, nil)
			f.sender.On("SendOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

			input := defaultRequestInput()
			input.DharmaName = tt.dharmaName
			input.OrdinationPlace = tt.ordinationPlace

			_, err := f.usecase.RequestOTP(context.Background(), input)
			require.NoError(t, err)
			require.NotNil(t, captured)

			assert.Equal(t, tt.wantDharma, captured.DharmaName)
			assert.Equal(t, tt.wantOrdination, captured.OrdinationPlace)
		})
	}
}

// TestRequestOTP_Validation covers AC7 (password floor) and AC8 (malformed
// phone or email): all three are rejected before the repository is touched, so
// a bad request costs no query.
func TestRequestOTP_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   domain.RequestRegisterOTPInput
		wantErr error
	}{
		{
			name:    "email is not an address",
			input:   withInput(func(in *domain.RequestRegisterOTPInput) { in.Email = "not-an-email" }),
			wantErr: domain.ErrInvalidEmail,
		},
		{
			name:    "email is missing",
			input:   withInput(func(in *domain.RequestRegisterOTPInput) { in.Email = "" }),
			wantErr: domain.ErrInvalidEmail,
		},
		{
			name:    "email has no domain",
			input:   withInput(func(in *domain.RequestRegisterOTPInput) { in.Email = "user@" }),
			wantErr: domain.ErrInvalidEmail,
		},
		{
			name:    "phone is not a number",
			input:   withInput(func(in *domain.RequestRegisterOTPInput) { in.PhoneNumber = "not-a-phone" }),
			wantErr: domain.ErrInvalidPhone,
		},
		{
			name:    "phone is missing",
			input:   withInput(func(in *domain.RequestRegisterOTPInput) { in.PhoneNumber = "" }),
			wantErr: domain.ErrInvalidPhone,
		},
		{
			// "090123" normalizes to "+8490123" — seven digits, below
			// IsValidPhone's floor of eight, so NormalizePhone returns "".
			// Note that "0901234" would NOT do: it normalizes to eight digits
			// and the shared phone rule accepts anything from eight up, so the
			// use case cannot reject it without contradicting pkg/validation.
			name:    "phone has too few digits",
			input:   withInput(func(in *domain.RequestRegisterOTPInput) { in.PhoneNumber = "090123" }),
			wantErr: domain.ErrInvalidPhone,
		},
		{
			name:    "phone has too many digits",
			input:   withInput(func(in *domain.RequestRegisterOTPInput) { in.PhoneNumber = "+84901234567890123" }),
			wantErr: domain.ErrInvalidPhone,
		},
		{
			name:    "password is seven characters",
			input:   withInput(func(in *domain.RequestRegisterOTPInput) { in.Password = "1234567" }),
			wantErr: domain.ErrInvalidPassword,
		},
		{
			name:    "password is missing",
			input:   withInput(func(in *domain.RequestRegisterOTPInput) { in.Password = "" }),
			wantErr: domain.ErrInvalidPassword,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRegisterFixture()

			result, err := f.usecase.RequestOTP(context.Background(), tt.input)

			assert.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, result)
			// No branch of validation may touch the database.
			f.repo.AssertNotCalled(t, "FindConflicts")
			f.repo.AssertNotCalled(t, "SavePendingAndIssueOTP")
			f.sender.AssertNotCalled(t, "SendOTP")
		})
	}
}

// TestRequestOTP_Conflicts covers AC6: a registered phone or email blocks the
// attempt, and when both are taken the phone number is reported first because it
// is the primary identity in the MVP. A pending row for the same phone is not in
// this table on purpose — v5.1 lets that caller overwrite their own payload.
func TestRequestOTP_Conflicts(t *testing.T) {
	tests := []struct {
		name      string
		conflicts domain.RegistrationConflicts
		wantErr   error
	}{
		{
			name:      "phone belongs to an account",
			conflicts: domain.RegistrationConflicts{PhoneInUsers: true},
			wantErr:   domain.ErrPhoneTaken,
		},
		{
			name:      "email belongs to an account",
			conflicts: domain.RegistrationConflicts{EmailInUsers: true},
			wantErr:   domain.ErrEmailTaken,
		},
		{
			// AC6's explicit rule: the unverified email of another phone's
			// attempt lives in pending_registration, not in users, so checking
			// users alone would let two people claim the same address.
			name:      "email belongs to another phone's in-flight registration",
			conflicts: domain.RegistrationConflicts{EmailInPending: true},
			wantErr:   domain.ErrEmailTaken,
		},
		{
			name:      "phone wins over email",
			conflicts: domain.RegistrationConflicts{PhoneInUsers: true, EmailInUsers: true},
			wantErr:   domain.ErrPhoneTaken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRegisterFixture()

			f.repo.On("FindConflicts", mock.Anything, testPhoneNumber, testEmail).
				Return(&tt.conflicts, nil)

			result, err := f.usecase.RequestOTP(context.Background(), defaultRequestInput())

			assert.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, result)
			f.repo.AssertNotCalled(t, "SavePendingAndIssueOTP")
			f.sender.AssertNotCalled(t, "SendOTP")
		})
	}
}

// TestRequestOTP_LockedNumber covers AC12's "no exception for request-otp"
// rule: while a number is locked out, asking for another code is refused rather
// than handing the attacker a fresh counter.
func TestRequestOTP_LockedNumber(t *testing.T) {
	f := newRegisterFixture()

	lockedUntil := time.Now().Add(10 * time.Minute)
	locked := activeOTP(time.Now().Add(-2 * time.Minute))
	locked.LockedUntil = &lockedUntil

	f.repo.On("FindConflicts", mock.Anything, mock.Anything, mock.Anything).Return(noConflicts(), nil)
	f.otpRepo.On("FindActiveOTP", mock.Anything, testPhoneNumber, domain.OTPPurposeRegister).Return(locked, nil)

	_, err := f.usecase.RequestOTP(context.Background(), defaultRequestInput())

	assert.ErrorIs(t, err, domain.ErrOTPLocked)

	var lockedErr *domain.OTPLockedError
	require.ErrorAs(t, err, &lockedErr)
	assert.WithinDuration(t, lockedUntil, lockedErr.LockedUntil, time.Second)

	f.repo.AssertNotCalled(t, "SavePendingAndIssueOTP")
	f.sender.AssertNotCalled(t, "SendOTP")
}

// TestRequestOTP_Cooldown covers AC5's 60-second floor on the request path.
// The decision is made by the repository (it owns the per-phone counter), so
// the use case must surface the typed error with its retry_after_seconds
// unchanged — the handler turns that into 429 + Retry-After.
func TestRequestOTP_Cooldown(t *testing.T) {
	f := newRegisterFixture()

	f.repo.On("FindConflicts", mock.Anything, mock.Anything, mock.Anything).Return(noConflicts(), nil)
	f.otpRepo.On("FindActiveOTP", mock.Anything, testPhoneNumber, domain.OTPPurposeRegister).
		Return(activeOTP(time.Now().Add(-30*time.Second)), nil)
	f.repo.On("SavePendingAndIssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, &domain.OTPResendTooSoonError{RetryAfterSeconds: 30})

	_, err := f.usecase.RequestOTP(context.Background(), defaultRequestInput())

	assert.ErrorIs(t, err, domain.ErrOTPResendTooSoon)

	var tooSoon *domain.OTPResendTooSoonError
	require.ErrorAs(t, err, &tooSoon)
	assert.Equal(t, 30, tooSoon.RetryAfterSeconds)

	f.sender.AssertNotCalled(t, "SendOTP")
}

// TestRequestOTP_SendLimited covers AC5's cap on the request path: request-otp
// and resend-otp share one per-phone counter, so an exhausted cycle refuses a
// fresh request too. Crucially the refusal is a *temporary* lock with a
// deadline (locked_until), not the permanent "start over" the old resend cap
// produced — that is the whole point of v5.0.
func TestRequestOTP_SendLimited(t *testing.T) {
	f := newRegisterFixture()

	lockedUntil := time.Now().Add(30 * time.Minute)
	f.repo.On("FindConflicts", mock.Anything, mock.Anything, mock.Anything).Return(noConflicts(), nil)
	f.otpRepo.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrOTPInvalid)
	f.repo.On("SavePendingAndIssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, &domain.OTPSendLimitedError{LockedUntil: lockedUntil})

	_, err := f.usecase.RequestOTP(context.Background(), defaultRequestInput())

	assert.ErrorIs(t, err, domain.ErrOTPSendLimited)

	var limited *domain.OTPSendLimitedError
	require.ErrorAs(t, err, &limited)
	assert.WithinDuration(t, lockedUntil, limited.LockedUntil, time.Second)

	f.sender.AssertNotCalled(t, "SendOTP")
}

// TestRequestOTP_AfterCooldownProceeds is the other half of AC5 on this path:
// once the cooldown has elapsed the request goes through. Under v5.0 it also
// consumes a send slot — request-otp is no longer a free restart — but that
// decision lives in the repository; this test pins that an allowed send still
// returns 200 with the configured expiry.
func TestRequestOTP_AfterCooldownProceeds(t *testing.T) {
	f := newRegisterFixture()

	f.repo.On("FindConflicts", mock.Anything, mock.Anything, mock.Anything).Return(noConflicts(), nil)
	f.otpRepo.On("FindActiveOTP", mock.Anything, testPhoneNumber, domain.OTPPurposeRegister).
		Return(activeOTP(time.Now().Add(-5*time.Minute)), nil)
	f.repo.On("SavePendingAndIssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&domain.SendState{Remaining: 3}, nil)
	f.sender.On("SendOTP", mock.Anything, mock.Anything, testOTPCode, domain.OTPPurposeRegister).Return(nil)

	result, err := f.usecase.RequestOTP(context.Background(), defaultRequestInput())

	require.NoError(t, err)
	assert.Equal(t, 90, result.ExpiresInSeconds)
}

// TestRequestOTP_NoActiveOTPProceeds pins the first-registration path: having
// no OTP row yet is not an error, it just means there is nothing to rate-limit.
func TestRequestOTP_NoActiveOTPProceeds(t *testing.T) {
	f := newRegisterFixture()

	f.repo.On("FindConflicts", mock.Anything, mock.Anything, mock.Anything).Return(noConflicts(), nil)
	f.otpRepo.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrOTPInvalid)
	f.repo.On("SavePendingAndIssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&domain.SendState{Remaining: 3}, nil)
	f.sender.On("SendOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := f.usecase.RequestOTP(context.Background(), defaultRequestInput())

	require.NoError(t, err)
}

// TestRequestOTP_SMSFailure pins the deliberate trade-off in design.md flow 1
// Step 5: the OTP row is already committed, but a code the user never receives
// is not a success, so the caller gets an error instead of a 200.
func TestRequestOTP_SMSFailure(t *testing.T) {
	f := newRegisterFixture()

	f.repo.On("FindConflicts", mock.Anything, mock.Anything, mock.Anything).Return(noConflicts(), nil)
	f.otpRepo.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrOTPInvalid)
	f.repo.On("SavePendingAndIssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&domain.SendState{Remaining: 3}, nil)
	f.sender.On("SendOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("carrier said no"))

	result, err := f.usecase.RequestOTP(context.Background(), defaultRequestInput())

	assert.ErrorIs(t, err, domain.ErrSMSSendFailed)
	assert.Nil(t, result)

	// The registration state stays committed — the retry is expected to hit
	// the cooldown rather than find nothing at all.
	f.repo.AssertCalled(t, "SavePendingAndIssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

	// Neither the OTP nor the password may reach the log. The assertion names
	// the forbidden shape — a record that carries one of them — rather than the
	// Error method as a whole: logging this failure is expected, logging the
	// secret in it is not.
	for _, method := range []string{"Error", "Warn", "Info", "Debug"} {
		f.logger.AssertNotCalled(t, method, mock.Anything, mock.MatchedBy(logsSecret))
	}
}

// logsSecret reports whether a mock logger call's key/value slice carries the
// OTP or the plaintext password. Matching on the value rather than the key means
// a secret logged under an unexpected key still fails the test.
func logsSecret(keysAndValues []interface{}) bool {
	for _, value := range keysAndValues {
		if s, ok := value.(string); ok && (s == testOTPCode || s == testPassword) {
			return true
		}
	}
	return false
}

// TestRequestOTP_RepositoryFailure pins that an infrastructure failure stays an
// infrastructure failure — it must not be mistaken for one of the sentinels the
// handler maps to a 4xx.
func TestRequestOTP_RepositoryFailure(t *testing.T) {
	f := newRegisterFixture()

	f.repo.On("FindConflicts", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("connection refused"))

	_, err := f.usecase.RequestOTP(context.Background(), defaultRequestInput())

	require.Error(t, err)
	assert.ErrorContains(t, err, "find registration conflicts")
	assert.NotErrorIs(t, err, domain.ErrPhoneTaken)
	assert.NotErrorIs(t, err, domain.ErrEmailTaken)
	f.repo.AssertNotCalled(t, "SavePendingAndIssueOTP")
}

// ---------------------------------------------------------------------------
// ResendOTP
// ---------------------------------------------------------------------------

// TestResendOTP_HappyPath covers AC5's normal case: a new OTP replaces the old
// one, the send counter (owned by the repository) advances, and the pending
// payload's deadline is pushed forward so the new code has something to verify
// against.
func TestResendOTP_HappyPath(t *testing.T) {
	f := newRegisterFixture()

	var (
		capturedExpiry time.Time
		capturedHash   string
		order          []string
	)

	f.repo.On("FindPendingRegistration", mock.Anything, testPhoneNumber).
		Return(&domain.PendingRegistration{ID: 1, PhoneNumber: testPhoneNumber, Email: testEmail}, nil)
	f.otpRepo.On("FindActiveOTP", mock.Anything, testPhoneNumber, domain.OTPPurposeRegister).
		Return(activeOTP(time.Now().Add(-5*time.Minute)), nil)
	f.otpRepo.On("IssueOTP", mock.Anything, testPhoneNumber, domain.OTPPurposeRegister, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time"), mock.AnythingOfType("domain.SendLimits")).
		Run(func(args mock.Arguments) {
			order = append(order, "issue")
			capturedHash = args.String(3)
			capturedExpiry = args.Get(4).(time.Time)
		}).
		Return(&domain.SendState{SendCount: 2, Remaining: 1}, nil)
	f.sender.On("SendOTP", mock.Anything, testPhoneNumber, testOTPCode, domain.OTPPurposeRegister).
		Run(func(mock.Arguments) { order = append(order, "send") }).
		Return(nil)

	before := time.Now()
	result, err := f.usecase.ResendOTP(context.Background(), domain.ResendOTPInput{
		PhoneNumber: testPhoneNumberRaw,
		Purpose:     domain.OTPPurposeRegister,
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"issue", "send"}, order)

	require.NotNil(t, result)
	assert.Equal(t, 90, result.ExpiresInSeconds)
	assert.Equal(t, 60, result.ResendAfterSeconds)

	// The repository reports 2 counted sends of the 3 allowed, so one send
	// remains in this cycle.
	assert.Equal(t, 1, result.ResendRemaining)

	// The replacement OTP is hashed like the first one, and lives the same 90s.
	assert.NotEqual(t, testOTPCode, capturedHash)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(capturedHash), []byte(testOTPCode)))
	assert.WithinDuration(t, before.Add(90*time.Second), capturedExpiry, 5*time.Second)

	// A resend must not rewrite the account payload: the email and password
	// were captured at step 1 and the client does not get to change them by
	// asking for another code.
	f.repo.AssertNotCalled(t, "SavePendingAndIssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestResendOTP_UnsupportedPurpose pins that this endpoint is shared across
// flows but only the register branch is implemented, and that the other three
// purposes cost no query.
func TestResendOTP_UnsupportedPurpose(t *testing.T) {
	purposes := []domain.OTPPurpose{
		domain.OTPPurposeLogin,
		domain.OTPPurposeResetPassword,
		domain.OTPPurposeVerifyEmail,
	}

	for _, purpose := range purposes {
		t.Run(string(purpose), func(t *testing.T) {
			f := newRegisterFixture()

			result, err := f.usecase.ResendOTP(context.Background(), domain.ResendOTPInput{
				PhoneNumber: testPhoneNumberRaw,
				Purpose:     purpose,
			})

			assert.ErrorIs(t, err, domain.ErrUnsupportedOTPPurpose)
			assert.Nil(t, result)
			f.repo.AssertNotCalled(t, "FindPendingRegistration")
			f.otpRepo.AssertNotCalled(t, "IssueOTP")
			f.sender.AssertNotCalled(t, "SendOTP")
		})
	}
}

// TestResendOTP_PendingFailures covers the two ways the temporary payload can be
// unusable. Both mean "start over from step 1", and neither may be reported as
// a 404 — a caller must not be able to probe which numbers are mid-registration.
func TestResendOTP_PendingFailures(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{
			name:    "no pending registration",
			repoErr: domain.ErrPendingRegistrationNotFound,
			wantErr: domain.ErrPendingRegistrationNotFound,
		},
		{
			name:    "pending registration expired",
			repoErr: domain.ErrPendingRegistrationExpired,
			wantErr: domain.ErrPendingRegistrationExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRegisterFixture()

			f.repo.On("FindPendingRegistration", mock.Anything, testPhoneNumber).
				Return(nil, tt.repoErr)

			_, err := f.usecase.ResendOTP(context.Background(), domain.ResendOTPInput{
				PhoneNumber: testPhoneNumberRaw,
				Purpose:     domain.OTPPurposeRegister,
			})

			assert.ErrorIs(t, err, tt.wantErr)
			f.otpRepo.AssertNotCalled(t, "IssueOTP")
			f.sender.AssertNotCalled(t, "SendOTP")
		})
	}
}

// TestResendOTP_AfterCodeExpired covers the v5.1 change: once the 90-second code
// window has closed but the pending payload is still live, a resend issues a
// fresh code instead of reporting OTP_EXPIRED. This is what lets a caller resume
// after the 30-minute send lock, whose expiry is far beyond the code's.
func TestResendOTP_AfterCodeExpired(t *testing.T) {
	f := newRegisterFixture()

	f.repo.On("FindPendingRegistration", mock.Anything, testPhoneNumber).
		Return(&domain.PendingRegistration{ID: 1, PhoneNumber: testPhoneNumber}, nil)
	f.otpRepo.On("FindActiveOTP", mock.Anything, testPhoneNumber, domain.OTPPurposeRegister).
		Return(nil, domain.ErrOTPInvalid)
	f.otpRepo.On("IssueOTP", mock.Anything, testPhoneNumber, domain.OTPPurposeRegister, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time"), mock.AnythingOfType("domain.SendLimits")).
		Return(&domain.SendState{SendCount: 1, Remaining: 2}, nil)
	f.sender.On("SendOTP", mock.Anything, testPhoneNumber, testOTPCode, domain.OTPPurposeRegister).Return(nil)

	result, err := f.usecase.ResendOTP(context.Background(), domain.ResendOTPInput{
		PhoneNumber: testPhoneNumberRaw,
		Purpose:     domain.OTPPurposeRegister,
	})

	require.NoError(t, err)
	assert.Equal(t, 90, result.ExpiresInSeconds)
	assert.Equal(t, 2, result.ResendRemaining)
	f.otpRepo.AssertCalled(t, "IssueOTP", mock.Anything, testPhoneNumber, domain.OTPPurposeRegister, mock.Anything, mock.Anything, mock.Anything)
}

// TestResendOTP_Blocked covers AC12's wrong-guess lockout on the resend path:
// while the number is locked out, resending is refused rather than handing the
// brute-forcer a way to clear the attempt counter. The AC5 send limits (cooldown,
// cap) are the repository's job and are covered by
// TestResendOTP_SendLimitRefusals.
func TestResendOTP_Blocked(t *testing.T) {
	f := newRegisterFixture()

	lockedUntil := time.Now().Add(12 * time.Minute)
	otp := activeOTP(time.Now().Add(-5 * time.Minute))
	otp.LockedUntil = &lockedUntil

	f.repo.On("FindPendingRegistration", mock.Anything, testPhoneNumber).
		Return(&domain.PendingRegistration{ID: 1, PhoneNumber: testPhoneNumber}, nil)
	f.otpRepo.On("FindActiveOTP", mock.Anything, testPhoneNumber, domain.OTPPurposeRegister).
		Return(otp, nil)

	_, err := f.usecase.ResendOTP(context.Background(), domain.ResendOTPInput{
		PhoneNumber: testPhoneNumberRaw,
		Purpose:     domain.OTPPurposeRegister,
	})

	assert.ErrorIs(t, err, domain.ErrOTPLocked)

	var lockedErr *domain.OTPLockedError
	require.ErrorAs(t, err, &lockedErr)
	assert.WithinDuration(t, lockedUntil, lockedErr.LockedUntil, time.Second)

	// Nothing is written on a blocked branch.
	f.otpRepo.AssertNotCalled(t, "IssueOTP")
	f.sender.AssertNotCalled(t, "SendOTP")
}

// TestResendOTP_SendLimitRefusals covers AC5's cooldown and cap on the resend
// path. Both are decided by the repository inside IssueOTP's transaction, so
// the use case must surface the typed errors unchanged — the handler needs
// retry_after_seconds / locked_until to answer 429, and wrapping either into a
// generic error would turn it into a 500.
func TestResendOTP_SendLimitRefusals(t *testing.T) {
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
			f := newRegisterFixture()

			f.repo.On("FindPendingRegistration", mock.Anything, testPhoneNumber).
				Return(&domain.PendingRegistration{ID: 1, PhoneNumber: testPhoneNumber}, nil)
			f.otpRepo.On("FindActiveOTP", mock.Anything, testPhoneNumber, domain.OTPPurposeRegister).
				Return(activeOTP(time.Now().Add(-5*time.Minute)), nil)
			f.otpRepo.On("IssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(nil, tt.repoErr)

			_, err := f.usecase.ResendOTP(context.Background(), domain.ResendOTPInput{
				PhoneNumber: testPhoneNumberRaw,
				Purpose:     domain.OTPPurposeRegister,
			})

			assert.ErrorIs(t, err, tt.repoErr)
			tt.check(t, err)
			f.sender.AssertNotCalled(t, "SendOTP")
		})
	}
}

// TestResendOTP_SMSFailure mirrors the request-otp contract: a code that never
// reached the user is reported as a failure even though the row exists.
func TestResendOTP_SMSFailure(t *testing.T) {
	f := newRegisterFixture()

	f.repo.On("FindPendingRegistration", mock.Anything, testPhoneNumber).
		Return(&domain.PendingRegistration{ID: 1, PhoneNumber: testPhoneNumber}, nil)
	f.otpRepo.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).
		Return(activeOTP(time.Now().Add(-5*time.Minute)), nil)
	f.otpRepo.On("IssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&domain.SendState{SendCount: 1, Remaining: 2}, nil)
	f.sender.On("SendOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(errors.New("carrier said no"))

	_, err := f.usecase.ResendOTP(context.Background(), domain.ResendOTPInput{
		PhoneNumber: testPhoneNumberRaw,
		Purpose:     domain.OTPPurposeRegister,
	})

	assert.ErrorIs(t, err, domain.ErrSMSSendFailed)
	f.otpRepo.AssertCalled(t, "IssueOTP", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// ---------------------------------------------------------------------------
// VerifyOTP
// ---------------------------------------------------------------------------

// TestVerifyOTP_HappyPath covers AC2 and AC3: a correct OTP turns the pending
// payload into a real account with a session, and consumes the pending row in
// the same transaction. It also pins the token contract — the access token is
// minted for the new user ID plus the email captured at step 1, and the refresh
// token carries the ID alone.
func TestVerifyOTP_HappyPath(t *testing.T) {
	f := newRegisterFixture()

	pending := &domain.PendingRegistration{
		ID:           1,
		PhoneNumber:  testPhoneNumber,
		Email:        testEmail,
		PasswordHash: "hash-from-step-one",
	}
	otp := activeOTP(time.Now().Add(-10 * time.Second))

	f.repo.On("FindPendingRegistration", mock.Anything, testPhoneNumber).Return(pending, nil)
	f.otpRepo.On("FindActiveOTP", mock.Anything, testPhoneNumber, domain.OTPPurposeRegister).Return(otp, nil)

	var capturedTTL time.Duration
	f.repo.On("CreateUserAndSession", mock.Anything, pending, otp.ID, mock.AnythingOfType("time.Duration"), mock.AnythingOfType("domain.TokenIssuer")).
		Run(func(args mock.Arguments) {
			capturedTTL = args.Get(3).(time.Duration)
		}).
		Return(&domain.RegisterResult{
			User:         &domain.User{ID: 42, Email: testEmail, PhoneNumber: testPhoneNumber},
			AccessToken:  testAccessToken,
			RefreshToken: testRefreshToken,
		}, nil)

	f.jwt.On("GenerateAccessToken", testUserID, testEmail, "user", mock.Anything).Return(testAccessToken, nil)
	f.jwt.On("GenerateRefreshToken", testUserID, mock.Anything).Return(testRefreshToken, nil)

	result, err := f.usecase.VerifyOTP(context.Background(), domain.VerifyRegisterOTPInput{
		PhoneNumber: testPhoneNumberRaw,
		OTPCode:     testOTPCode,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, testAccessToken, result.AccessToken)
	assert.Equal(t, testRefreshToken, result.RefreshToken)
	require.NotNil(t, result.User)
	assert.Equal(t, testEmail, result.User.Email)

	// The session lifetime comes from config, not from a constant in the use
	// case, so the stored expiry and the signed refresh token cannot drift.
	assert.Equal(t, testSessionTTL, capturedTTL)

	// A correct guess costs nothing — no attempt is recorded and no lock is set.
	f.otpRepo.AssertNotCalled(t, "IncrementOTPAttempts", mock.Anything, mock.Anything)
	f.otpRepo.AssertNotCalled(t, "LockOTP", mock.Anything, mock.Anything, mock.Anything)
}

// TestVerifyOTP_TokenIssuerContract proves the issuer handed to the repository
// really signs both tokens, so the repository can call it mid-transaction
// without knowing anything about JWT.
func TestVerifyOTP_TokenIssuerContract(t *testing.T) {
	f := newRegisterFixture()

	pending := &domain.PendingRegistration{ID: 1, PhoneNumber: testPhoneNumber, Email: testEmail}
	otp := activeOTP(time.Now())

	f.repo.On("FindPendingRegistration", mock.Anything, testPhoneNumber).Return(pending, nil)
	f.otpRepo.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(otp, nil)
	// The access token must carry the session id as claim "sid" — logout finds
	// the session through it. Pinning the exact extra map (not mock.Anything)
	// is what makes this test fail if the claim is dropped or renamed.
	f.jwt.On("GenerateAccessToken", testUserID, testEmail, "user", map[string]any{"sid": testSessionID}).Return(testAccessToken, nil)
	f.jwt.On("GenerateRefreshToken", testUserID, mock.Anything).Return(testRefreshToken, nil)

	var issuer domain.TokenIssuer
	f.repo.On("CreateUserAndSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.AnythingOfType("domain.TokenIssuer")).
		Run(func(args mock.Arguments) {
			issuer = args.Get(4).(domain.TokenIssuer)
		}).
		// The repository always returns the account it just inserted, so the
		// stub does too — a result with a nil User would be a port violation,
		// not a case the use case has to defend against.
		Return(&domain.RegisterResult{User: &domain.User{ID: 42, Email: testEmail}}, nil)

	_, err := f.usecase.VerifyOTP(context.Background(), domain.VerifyRegisterOTPInput{
		PhoneNumber: testPhoneNumberRaw,
		OTPCode:     testOTPCode,
	})
	require.NoError(t, err)
	require.NotNil(t, issuer)

	accessToken, refreshToken, err := issuer(testUserID, testSessionID)
	require.NoError(t, err)
	assert.Equal(t, testAccessToken, accessToken)
	assert.Equal(t, testRefreshToken, refreshToken)
}

// TestVerifyOTP_TokenIssuerFailure pins that a signing failure surfaces as an
// error rather than an empty token pair, which the repository would otherwise
// store as a session nobody can use.
func TestVerifyOTP_TokenIssuerFailure(t *testing.T) {
	f := newRegisterFixture()

	pending := &domain.PendingRegistration{ID: 1, PhoneNumber: testPhoneNumber, Email: testEmail}
	otp := activeOTP(time.Now())

	f.repo.On("FindPendingRegistration", mock.Anything, testPhoneNumber).Return(pending, nil)
	f.otpRepo.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(otp, nil)
	f.jwt.On("GenerateAccessToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("", errors.New("signing key missing"))

	var issuer domain.TokenIssuer
	f.repo.On("CreateUserAndSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.AnythingOfType("domain.TokenIssuer")).
		Run(func(args mock.Arguments) {
			issuer = args.Get(4).(domain.TokenIssuer)
		}).
		// The port always returns a populated result on success — the use case
		// reads result.User.ID to log, so an empty struct here would be the mock
		// violating the contract rather than a state the real repository reaches.
		Return(&domain.RegisterResult{User: &domain.User{ID: 42}}, nil)

	_, err := f.usecase.VerifyOTP(context.Background(), domain.VerifyRegisterOTPInput{
		PhoneNumber: testPhoneNumberRaw,
		OTPCode:     testOTPCode,
	})
	require.NoError(t, err)
	require.NotNil(t, issuer)

	_, _, err = issuer(testUserID, testSessionID)
	assert.ErrorContains(t, err, "signing key missing")
}

// TestVerifyOTP_WrongOTP covers AC4's wrong-code branch and AC12's counting:
// every wrong guess is recorded, and the number is only locked once the third
// one lands. The caller is told "invalid" and nothing more — revealing how many
// tries remain would help exactly the person brute-forcing.
func TestVerifyOTP_WrongOTP(t *testing.T) {
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
			f := newRegisterFixture()

			pending := &domain.PendingRegistration{ID: 1, PhoneNumber: testPhoneNumber, Email: testEmail}
			otp := activeOTP(time.Now())

			f.repo.On("FindPendingRegistration", mock.Anything, testPhoneNumber).Return(pending, nil)
			f.otpRepo.On("FindActiveOTP", mock.Anything, testPhoneNumber, domain.OTPPurposeRegister).Return(otp, nil)
			f.otpRepo.On("IncrementOTPAttempts", mock.Anything, otp.ID).Return(tt.attempts, nil)

			if tt.wantLocked {
				f.otpRepo.On("LockOTP", mock.Anything, otp.ID, mock.AnythingOfType("time.Time")).
					Run(func(args mock.Arguments) {
						lockedUntil := args.Get(2).(time.Time)
						// AC12: thirty minutes from the moment of the third
						// failure, not from the first.
						assert.WithinDuration(t, time.Now().Add(30*time.Minute), lockedUntil, 5*time.Second)
					}).
					Return(nil)
			}

			before := time.Now()
			result, err := f.usecase.VerifyOTP(context.Background(), domain.VerifyRegisterOTPInput{
				PhoneNumber: testPhoneNumberRaw,
				OTPCode:     "000000",
			})

			assert.ErrorIs(t, err, domain.ErrOTPInvalid)
			assert.Nil(t, result)

			// The error says nothing about the remaining attempts.
			assert.NotContains(t, err.Error(), "2")
			assert.NotContains(t, err.Error(), "remaining")
			assert.Less(t, time.Since(before), 30*time.Second)

			f.otpRepo.AssertCalled(t, "IncrementOTPAttempts", mock.Anything, otp.ID)
			// No account is created on a wrong code, no matter how close.
			f.repo.AssertNotCalled(t, "CreateUserAndSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

			if !tt.wantLocked {
				f.otpRepo.AssertNotCalled(t, "LockOTP", mock.Anything, mock.Anything, mock.Anything)
			}
		})
	}
}

// TestVerifyOTP_Locked covers AC12 from the other side: while the lock holds,
// the code is not even compared and no attempt is counted — otherwise an
// attacker could extend the lock forever, or keep guessing for free.
func TestVerifyOTP_Locked(t *testing.T) {
	f := newRegisterFixture()

	lockedUntil := time.Now().Add(9 * time.Minute)
	locked := activeOTP(time.Now().Add(-3 * time.Minute))
	locked.LockedUntil = &lockedUntil

	f.repo.On("FindPendingRegistration", mock.Anything, testPhoneNumber).
		Return(&domain.PendingRegistration{ID: 1, PhoneNumber: testPhoneNumber}, nil)
	f.otpRepo.On("FindActiveOTP", mock.Anything, testPhoneNumber, domain.OTPPurposeRegister).Return(locked, nil)

	_, err := f.usecase.VerifyOTP(context.Background(), domain.VerifyRegisterOTPInput{
		PhoneNumber: testPhoneNumberRaw,
		OTPCode:     testOTPCode,
	})

	assert.ErrorIs(t, err, domain.ErrOTPLocked)

	var lockedErr *domain.OTPLockedError
	require.ErrorAs(t, err, &lockedErr)
	assert.WithinDuration(t, lockedUntil, lockedErr.LockedUntil, time.Second)

	f.otpRepo.AssertNotCalled(t, "IncrementOTPAttempts")
	f.repo.AssertNotCalled(t, "CreateUserAndSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestVerifyOTP_PendingFailures covers AC4's unusable-attempt branches: no
// payload, or a payload past its deadline. The "not found" case must not be a
// 404 — that would confirm which numbers are mid-registration.
func TestVerifyOTP_PendingFailures(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{
			name:    "no pending registration",
			repoErr: domain.ErrPendingRegistrationNotFound,
			wantErr: domain.ErrPendingRegistrationNotFound,
		},
		{
			name:    "pending registration expired",
			repoErr: domain.ErrPendingRegistrationExpired,
			wantErr: domain.ErrPendingRegistrationExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRegisterFixture()

			f.repo.On("FindPendingRegistration", mock.Anything, testPhoneNumber).Return(nil, tt.repoErr)

			_, err := f.usecase.VerifyOTP(context.Background(), domain.VerifyRegisterOTPInput{
				PhoneNumber: testPhoneNumberRaw,
				OTPCode:     testOTPCode,
			})

			assert.ErrorIs(t, err, tt.wantErr)
			f.otpRepo.AssertNotCalled(t, "FindActiveOTP")
			f.repo.AssertNotCalled(t, "CreateUserAndSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

// TestVerifyOTP_NoActiveOTP covers AC4's expired and already-used branches: the
// row is gone, so the caller is told the code expired rather than that it was
// wrong — they did nothing wrong, they were just too slow.
func TestVerifyOTP_NoActiveOTP(t *testing.T) {
	f := newRegisterFixture()

	f.repo.On("FindPendingRegistration", mock.Anything, testPhoneNumber).
		Return(&domain.PendingRegistration{ID: 1, PhoneNumber: testPhoneNumber}, nil)
	f.otpRepo.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, domain.ErrOTPInvalid)

	_, err := f.usecase.VerifyOTP(context.Background(), domain.VerifyRegisterOTPInput{
		PhoneNumber: testPhoneNumberRaw,
		OTPCode:     testOTPCode,
	})

	assert.ErrorIs(t, err, domain.ErrOTPExpired)
	f.repo.AssertNotCalled(t, "CreateUserAndSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestVerifyOTP_InvalidPhone covers AC8 on the second step: a malformed number
// is rejected before any query, so the endpoint cannot be used to probe for
// pending registrations.
func TestVerifyOTP_InvalidPhone(t *testing.T) {
	tests := []struct {
		name  string
		phone string
	}{
		{name: "missing", phone: ""},
		{name: "not a number", phone: "not-a-phone"},
		{name: "too short", phone: "090123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRegisterFixture()

			_, err := f.usecase.VerifyOTP(context.Background(), domain.VerifyRegisterOTPInput{
				PhoneNumber: tt.phone,
				OTPCode:     testOTPCode,
			})

			assert.ErrorIs(t, err, domain.ErrInvalidPhone)
			f.repo.AssertNotCalled(t, "FindPendingRegistration")
		})
	}
}

// TestVerifyOTP_ConflictRace covers AC6's last line of defence: two people can
// both hold a pending registration for the same unverified email, so the second
// one to verify is stopped inside the account-creating transaction and sees the
// same 409 the earlier check would have given.
func TestVerifyOTP_ConflictRace(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "email claimed while verifying", repoErr: domain.ErrEmailTaken, wantErr: domain.ErrEmailTaken},
		{name: "phone claimed while verifying", repoErr: domain.ErrPhoneTaken, wantErr: domain.ErrPhoneTaken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRegisterFixture()

			otp := activeOTP(time.Now())
			f.repo.On("FindPendingRegistration", mock.Anything, testPhoneNumber).
				Return(&domain.PendingRegistration{ID: 1, PhoneNumber: testPhoneNumber, Email: testEmail}, nil)
			f.otpRepo.On("FindActiveOTP", mock.Anything, mock.Anything, mock.Anything).Return(otp, nil)
			f.repo.On("CreateUserAndSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(nil, tt.repoErr)

			_, err := f.usecase.VerifyOTP(context.Background(), domain.VerifyRegisterOTPInput{
				PhoneNumber: testPhoneNumberRaw,
				OTPCode:     testOTPCode,
			})

			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func ptr(s string) *string { return &s }

// withInput builds the default valid request and applies one mutation, so a
// validation case reads as "this one field is wrong" instead of restating the
// whole body.
func withInput(mutate func(*domain.RequestRegisterOTPInput)) domain.RequestRegisterOTPInput {
	input := defaultRequestInput()
	mutate(&input)
	return input
}
