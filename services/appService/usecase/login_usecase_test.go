package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"chjpcoj.com.innerlamp/pkg/jwt"
	jwtclaims "chjpcoj.com.innerlamp/pkg/jwt/claims"
	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// Tests for US-002 (login, refresh, logout). The acceptance criteria each test
// proves are named in the test's own doc comment; the mapping is:
//
//	AC1  matching phone+password+device logs in             -> TestLogin_HappyPath
//	AC2  wrong password / unknown phone / pending-only      -> TestLogin_InvalidCredentials,
//	                                                            TestLogin_NoEnumeration
//	AC3  a new device gets its own session                  -> repo-level (design.md flow 3 step 6b)
//	AC4  a known device supersedes only its own session     -> repo-level (design.md flow 3 step 6c)
//	AC5  device without platform / device_model             -> TestLogin_Validation
//	AC6  bad phone / empty password                         -> TestLogin_Validation
//	AC7  five consecutive failures lock the number          -> TestLogin_FailurePolicy,
//	                                                            TestLogin_Locked,
//	                                                            TestLogin_LockAppliesToUnknownPhone
//	AC8  access 15m, refresh 7d, sid claim, hashed storage  -> TestLogin_TokenIssuerContract
//	AC9  refresh rotates the pair                           -> TestRefresh_HappyPath,
//	                                                            TestRefresh_TokenIssuerContract
//	AC10 every unusable refresh token is one 401            -> TestRefresh_Invalid
//	AC11 an already-rotated token revokes the session       -> TestRefresh_ReuseDetected
//	AC12 a device-less US-001 session refreshes too         -> TestRefresh_DeviceLessSession
//	AC13 logout revokes the session the sid names           -> TestLogout_Revokes
//	AC14 logout is idempotent and scoped to the user        -> TestLogout_Idempotent,
//	                                                            TestLogout_ScopedToUser
//
// AC3 and AC4 are decided in the repository's transaction (which session rows
// a login touches), so the use case's half of them is only that it hands the
// device through unchanged — asserted in TestLogin_HappyPath — and that a
// device-less session refreshes normally, in TestRefresh_DeviceLessSession.

const (
	// testUserRowID is the numeric primary key a users row carries. Distinct
	// from testUserID ("42", the string form a JWT claim holds) on purpose: the
	// repository deals in int64 and the claims in string, and a single constant
	// would let a type confusion between the two go unnoticed.
	testUserRowID = int64(42)

	// testSessionRowID is the surrogate key of a user_sessions row — the value
	// RotateSession addresses with SELECT ... FOR UPDATE. It is not
	// testSessionID, which is the opaque uuid the access token carries as "sid";
	// the row key is internal and never leaves the database.
	testSessionRowID = int64(7)

	testLockedUntilMinutes = 15
)

// testLoginPolicy mirrors config.dev.yaml's `login:` block. Pinned as literals
// (not read back out of config) so a mis-wired policy fails a test: if the
// composition root sent the wrong numbers through, the assertions on what
// RecordLoginFailure receives would still compare against these.
var testLoginPolicy = config.LoginConfig{
	MaxFailedAttempts:  5,
	LockDuration:       config.Duration(15 * time.Minute),
	FailureResetWindow: config.Duration(15 * time.Minute),
}

// testPasswordHash is the bcrypt hash of testPassword, computed once at package
// load.
//
// DefaultCost, not MinCost. The cost is what TestLogin_NoEnumeration compares:
// it asserts the unknown-phone branch costs about as much as the known-phone
// one, and the unknown branch always pays for dummyPasswordHash, which is built
// at DefaultCost. A MinCost hash here would make the known branch legitimately
// ~30x cheaper than the unknown one and the ratio assertion would fail for a
// reason that has nothing to do with the property under test. Matching the
// costs is what makes the two branches comparable at all.
var testPasswordHash = func() string {
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
	if err != nil {
		panic("hashing the test password failed: " + err.Error())
	}
	return string(hash)
}()

type loginFixture struct {
	usecase *LoginUsecase
	repo    *domain.MockLoginRepository
	jwt     *jwt.MockService
	logger  *logging.MockLogger
}

func newLoginFixture() *loginFixture {
	repo := &domain.MockLoginRepository{}
	jwtService := &jwt.MockService{}

	return &loginFixture{
		usecase: NewLoginUsecase(repo, jwtService, newLogger(), testSessionTTL, testLoginPolicy),
		repo:    repo,
		jwt:     jwtService,
		logger:  newLogger(),
	}
}

// testUser builds the account a successful login resolves to.
func testUser() *domain.User {
	return &domain.User{
		ID:           testUserRowID,
		Email:        testEmail,
		PhoneNumber:  testPhoneNumber,
		PasswordHash: testPasswordHash,
	}
}

// defaultLoginInput is a request that passes every check before the database.
func defaultLoginInput() domain.LoginInput {
	return domain.LoginInput{
		PhoneNumber: testPhoneNumberRaw,
		Password:    testPassword,
		Device: domain.DeviceInfo{
			Platform:    "ios",
			DeviceModel: "iPhone15,2",
			OSVersion:   "17.4",
			AppVersion:  "1.0.0",
		},
	}
}

// withLoginInput builds the default valid request and applies one mutation, so a
// validation case reads as "this one field is wrong".
func withLoginInput(mutate func(*domain.LoginInput)) domain.LoginInput {
	input := defaultLoginInput()
	mutate(&input)
	return input
}

// activeSession is a session row that refreshes and logs out normally.
func activeSession() *domain.Session {
	return &domain.Session{
		ID:           testSessionRowID,
		SessionID:    testSessionID,
		UserID:       testUserRowID,
		TokenVersion: 1,
		ExpiresAt:    time.Now().Add(testSessionTTL),
	}
}

// ---------------------------------------------------------------------------
// Login
// ---------------------------------------------------------------------------

// TestLogin_HappyPath covers AC1: a matching phone and password opens a session
// bound to the device, and the result carries the token pair plus the account.
//
// The phone number is asserted *normalized* on the calls that reach the
// repository, because "0901234567" and "+84901234567" must land on the same
// users row and the same login_attempts row — a raw value would give one human
// two counters and let them double their guess budget.
func TestLogin_HappyPath(t *testing.T) {
	f := newLoginFixture()

	user := testUser()
	device := defaultLoginInput().Device
	wantResult := &domain.LoginResult{
		User:         user,
		AccessToken:  testAccessToken,
		RefreshToken: testRefreshToken,
	}

	f.repo.On("FindLoginAttempt", mock.Anything, testPhoneNumber).Return(nil, nil)
	f.repo.On("FindUserByPhone", mock.Anything, testPhoneNumber).Return(user, nil)

	var capturedTTL time.Duration
	f.repo.On("CreateLoginSession", mock.Anything, user, mock.AnythingOfType("domain.DeviceInfo"), mock.AnythingOfType("time.Duration"), mock.AnythingOfType("domain.TokenIssuer")).
		Run(func(args mock.Arguments) {
			capturedTTL = args.Get(3).(time.Duration)
		}).
		Return(wantResult, nil)

	result, err := f.usecase.Login(context.Background(), defaultLoginInput())
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, testAccessToken, result.AccessToken)
	assert.Equal(t, testRefreshToken, result.RefreshToken)
	require.NotNil(t, result.User)
	assert.Equal(t, testUserRowID, result.User.ID)

	// The session lifetime comes from config, not from a constant in the use
	// case, so the stored expiry and the signed refresh token cannot drift.
	assert.Equal(t, testSessionTTL, capturedTTL)

	// AC3/AC4 are the repository's decision, but the device it decides on is
	// this layer's to hand over intact — a dropped field would collapse two
	// distinct devices into one fingerprint.
	capturedDevice := f.repo.Calls[len(f.repo.Calls)-1].Arguments.Get(2).(domain.DeviceInfo)
	assert.Equal(t, device, capturedDevice)

	// A correct password costs nothing: no attempt is recorded and no lock set.
	f.repo.AssertNotCalled(t, "RecordLoginFailure", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestLogin_TokenIssuerContract proves the issuer handed to the repository
// really signs both tokens, and that the access token carries the session ID as
// claim "sid" — logout finds the session through exactly that claim, so losing
// it would silently break AC13 rather than fail loudly.
func TestLogin_TokenIssuerContract(t *testing.T) {
	f := newLoginFixture()

	user := testUser()
	f.repo.On("FindLoginAttempt", mock.Anything, mock.Anything).Return(nil, nil)
	f.repo.On("FindUserByPhone", mock.Anything, testPhoneNumber).Return(user, nil)

	// Pinning the exact extra map (not mock.Anything) is what makes this test
	// fail if the claim is dropped or renamed.
	f.jwt.On("GenerateAccessToken", "42", testEmail, "user", map[string]any{"sid": testSessionID}).
		Return(testAccessToken, nil)
	f.jwt.On("GenerateRefreshToken", "42", mock.Anything).Return(testRefreshToken, nil)

	var issuer domain.TokenIssuer
	f.repo.On("CreateLoginSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.AnythingOfType("domain.TokenIssuer")).
		Run(func(args mock.Arguments) {
			issuer = args.Get(4).(domain.TokenIssuer)
		}).
		Return(&domain.LoginResult{User: user, AccessToken: testAccessToken, RefreshToken: testRefreshToken}, nil)

	_, err := f.usecase.Login(context.Background(), defaultLoginInput())
	require.NoError(t, err)
	require.NotNil(t, issuer)

	accessToken, refreshToken, err := issuer("42", testSessionID)
	require.NoError(t, err)
	assert.Equal(t, testAccessToken, accessToken)
	assert.Equal(t, testRefreshToken, refreshToken)
}

// TestLogin_TokenIssuerFailure pins that a signing failure surfaces as an error
// rather than an empty token pair, which the repository would otherwise store
// as a session nobody can use.
func TestLogin_TokenIssuerFailure(t *testing.T) {
	f := newLoginFixture()

	user := testUser()
	f.repo.On("FindLoginAttempt", mock.Anything, mock.Anything).Return(nil, nil)
	f.repo.On("FindUserByPhone", mock.Anything, testPhoneNumber).Return(user, nil)
	f.jwt.On("GenerateAccessToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return("", errors.New("signing key missing"))

	var issuer domain.TokenIssuer
	f.repo.On("CreateLoginSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.AnythingOfType("domain.TokenIssuer")).
		Run(func(args mock.Arguments) {
			issuer = args.Get(4).(domain.TokenIssuer)
		}).
		Return(&domain.LoginResult{User: user}, nil)

	_, err := f.usecase.Login(context.Background(), defaultLoginInput())
	require.NoError(t, err)
	require.NotNil(t, issuer)

	_, _, err = issuer("42", testSessionID)
	assert.ErrorContains(t, err, "signing key missing")
}

// TestLogin_InvalidCredentials covers AC2 from both directions.
//
// The two cases must be indistinguishable: same sentinel, same message, and —
// because the use case compares against a dummy hash when the account is
// missing — the same bcrypt cost. A test that only checked the sentinel would
// still pass if someone reintroduced an early return on the not-found branch.
func TestLogin_InvalidCredentials(t *testing.T) {
	tests := []struct {
		name     string
		user     *domain.User
		userErr  error
		password string
	}{
		{
			name:     "wrong password",
			user:     testUser(),
			password: "not-the-password",
		},
		{
			name:     "no account for the number",
			user:     nil,
			userErr:  domain.ErrUserNotFound,
			password: testPassword,
		},
		{
			// A number with a registration in flight has no users row yet, so
			// this is the not-found path — it must not answer "verify your OTP",
			// which would confirm the number is mid-registration.
			name:     "number exists only in pending_registration",
			user:     nil,
			userErr:  domain.ErrUserNotFound,
			password: testPassword,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newLoginFixture()

			f.repo.On("FindLoginAttempt", mock.Anything, testPhoneNumber).Return(nil, nil)
			f.repo.On("FindUserByPhone", mock.Anything, testPhoneNumber).Return(tt.user, tt.userErr)
			f.repo.On("RecordLoginFailure", mock.Anything, testPhoneNumber, testLoginPolicy.MaxFailedAttempts, testLoginPolicy.LockDuration.Duration(), testLoginPolicy.FailureResetWindow.Duration()).
				Return(&domain.LoginAttempt{PhoneNumber: testPhoneNumber, FailedCount: 1}, nil)

			input := defaultLoginInput()
			input.Password = tt.password

			result, err := f.usecase.Login(context.Background(), input)

			assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
			assert.Nil(t, result)

			// One sentinel and one message for every cause: the error must not
			// name which field was wrong, because that alone confirms the
			// account exists.
			assert.NotContains(t, err.Error(), "mật khẩu không tồn tại")
			assert.NotContains(t, err.Error(), "không tìm thấy")

			// The failure is counted, and no session is opened.
			f.repo.AssertCalled(t, "RecordLoginFailure", mock.Anything, testPhoneNumber, testLoginPolicy.MaxFailedAttempts, testLoginPolicy.LockDuration.Duration(), testLoginPolicy.FailureResetWindow.Duration())
			f.repo.AssertNotCalled(t, "CreateLoginSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

// TestLogin_Validation covers AC5 (device) and AC6 (phone, password): every one
// of these returns before the first query and, crucially, before any counter is
// touched. A malformed request is not a guess, and counting it would let anyone
// lock a stranger's number out by sending junk.
func TestLogin_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   domain.LoginInput
		wantErr error

		// reachesRepo marks the cases that are *not* rejected by validation and
		// therefore must be allowed through to the first query. The short
		// password is the one such case: it is refused as a bad credential
		// rather than as a malformed request, so it asserts the opposite of the
		// validation rows below — that the lookup DID happen.
		reachesRepo bool
	}{
		{
			name:    "phone is not a number",
			input:   withLoginInput(func(in *domain.LoginInput) { in.PhoneNumber = "not-a-phone" }),
			wantErr: domain.ErrInvalidPhone,
		},
		{
			name:    "phone is missing",
			input:   withLoginInput(func(in *domain.LoginInput) { in.PhoneNumber = "" }),
			wantErr: domain.ErrInvalidPhone,
		},
		{
			// "090123" normalizes to seven digits, below pkg/validation's floor
			// of eight, so NormalizePhone returns "" — see the same case in
			// register_usecase_test.go for why "0901234" would not do.
			name:    "phone has too few digits",
			input:   withLoginInput(func(in *domain.LoginInput) { in.PhoneNumber = "090123" }),
			wantErr: domain.ErrInvalidPhone,
		},
		{
			name:    "password is missing",
			input:   withLoginInput(func(in *domain.LoginInput) { in.Password = "" }),
			wantErr: domain.ErrInvalidPassword,
		},
		{
			// AC6 rejects an *empty* password, not a short one: a short password
			// that once met the rule must still be able to log in after the rule
			// tightens, so length is a registration-time concern only.
			name:        "password is too short but not empty",
			input:       withLoginInput(func(in *domain.LoginInput) { in.Password = "1234567" }),
			wantErr:     domain.ErrInvalidCredentials,
			reachesRepo: true,
		},
		{
			name:    "device has no platform",
			input:   withLoginInput(func(in *domain.LoginInput) { in.Device.Platform = "" }),
			wantErr: domain.ErrInvalidDevice,
		},
		{
			name:    "device has no model",
			input:   withLoginInput(func(in *domain.LoginInput) { in.Device.DeviceModel = "" }),
			wantErr: domain.ErrInvalidDevice,
		},
		{
			// Only platform and device_model are required, so blanking the
			// optional three must not produce ErrInvalidDevice. The row is
			// marked reachesRepo because that is what proves it: the request is
			// allowed all the way to the account lookup, where it fails for the
			// ordinary reason (no such user).
			name: "optional device fields may be blank",
			input: withLoginInput(func(in *domain.LoginInput) {
				in.Device.OSVersion = ""
				in.Device.AppVersion = ""
				in.Device.DeviceName = ""
			}),
			wantErr:     domain.ErrInvalidCredentials,
			reachesRepo: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newLoginFixture()

			if tt.reachesRepo {
				// These cases have to be allowed through to the first query,
				// which is the point: blank optional fields are not a failure,
				// and a short password is a wrong credential rather than a
				// malformed request.
				f.repo.On("FindLoginAttempt", mock.Anything, mock.Anything).Return(nil, nil)
				f.repo.On("FindUserByPhone", mock.Anything, mock.Anything).Return(nil, domain.ErrUserNotFound)
				f.repo.On("RecordLoginFailure", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(&domain.LoginAttempt{FailedCount: 1}, nil)

				result, err := f.usecase.Login(context.Background(), tt.input)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
				f.repo.AssertCalled(t, "FindLoginAttempt", mock.Anything, mock.Anything)
				return
			}

			result, err := f.usecase.Login(context.Background(), tt.input)

			assert.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, result)

			// Nothing is read and nothing is counted.
			f.repo.AssertNotCalled(t, "FindLoginAttempt")
			f.repo.AssertNotCalled(t, "FindUserByPhone")
			f.repo.AssertNotCalled(t, "RecordLoginFailure", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

// TestLogin_NoEnumeration pins AC2's timing property: an unknown phone number
// still pays for exactly one bcrypt comparison. The assertion is on the *cost*
// of the call, not on the answer — both branches return the same sentinel, so
// only the work done differs, and only the work done is what an attacker
// measures.
func TestLogin_NoEnumeration(t *testing.T) {
	measure := func(t *testing.T, user *domain.User, userErr error) time.Duration {
		t.Helper()
		f := newLoginFixture()

		f.repo.On("FindLoginAttempt", mock.Anything, mock.Anything).Return(nil, nil)
		f.repo.On("FindUserByPhone", mock.Anything, mock.Anything).Return(user, userErr)
		f.repo.On("RecordLoginFailure", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(&domain.LoginAttempt{FailedCount: 1}, nil)

		input := defaultLoginInput()
		input.Password = "definitely-not-the-password"

		start := time.Now()
		_, err := f.usecase.Login(context.Background(), input)
		elapsed := time.Since(start)

		require.ErrorIs(t, err, domain.ErrInvalidCredentials)
		return elapsed
	}

	known := measure(t, testUser(), nil)
	unknown := measure(t, nil, domain.ErrUserNotFound)

	// Both paths run one bcrypt comparison; the ratio is asserted loosely
	// because a shared CI machine cannot promise anything tighter. What it
	// catches is the regression that matters: an early return on the not-found
	// branch, which would make the unknown case ~100x faster, not 2x.
	ratio := float64(known) / float64(unknown)
	assert.Greater(t, ratio, 0.2, "unknown-phone branch returned too fast to have hashed anything")
	assert.Less(t, ratio, 5.0, "known-phone branch did far more work than the unknown one")
}

// TestLogin_FailurePolicy covers AC7's counting: the threshold, the lock
// duration and the reset window are config, and the use case must pass them
// through rather than deciding any of them itself — the atomic decision lives
// in the repository's UPSERT.
func TestLogin_FailurePolicy(t *testing.T) {
	f := newLoginFixture()

	f.repo.On("FindLoginAttempt", mock.Anything, testPhoneNumber).Return(nil, nil)
	f.repo.On("FindUserByPhone", mock.Anything, testPhoneNumber).Return(nil, domain.ErrUserNotFound)

	// The fifth failure: the counter trips and the lock is placed, but the
	// caller is still refused with a 401 — only the *next* request sees 429.
	lockedUntil := time.Now().Add(15 * time.Minute)
	f.repo.On("RecordLoginFailure",
		mock.Anything,
		testPhoneNumber,
		testLoginPolicy.MaxFailedAttempts,
		testLoginPolicy.LockDuration.Duration(),
		testLoginPolicy.FailureResetWindow.Duration(),
	).Return(&domain.LoginAttempt{
		PhoneNumber: testPhoneNumber,
		FailedCount: testLoginPolicy.MaxFailedAttempts,
		LockedUntil: &lockedUntil,
	}, nil)

	result, err := f.usecase.Login(context.Background(), defaultLoginInput())

	// The failure that trips the limit answers 401, not 429.
	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	assert.NotErrorIs(t, err, domain.ErrLoginLocked)
	assert.Nil(t, result)

	f.repo.AssertCalled(t, "RecordLoginFailure",
		mock.Anything,
		testPhoneNumber,
		testLoginPolicy.MaxFailedAttempts,
		testLoginPolicy.LockDuration.Duration(),
		testLoginPolicy.FailureResetWindow.Duration(),
	)
}

// TestLogin_Locked covers AC7 from the other side: while the lock holds, the
// password is not compared and the account is not looked up — otherwise an
// attacker could keep guessing for free, or the correct password would let a
// locked number straight back in.
func TestLogin_Locked(t *testing.T) {
	f := newLoginFixture()

	lockedUntil := time.Now().Add(10 * time.Minute)
	f.repo.On("FindLoginAttempt", mock.Anything, testPhoneNumber).
		Return(&domain.LoginAttempt{
			PhoneNumber: testPhoneNumber,
			FailedCount: testLoginPolicy.MaxFailedAttempts,
			LockedUntil: &lockedUntil,
		}, nil)

	// The correct password is supplied on purpose: the lock must refuse it too.
	result, err := f.usecase.Login(context.Background(), defaultLoginInput())

	assert.ErrorIs(t, err, domain.ErrLoginLocked)
	assert.Nil(t, result)

	var lockedErr *domain.LoginLockedError
	require.ErrorAs(t, err, &lockedErr)
	assert.WithinDuration(t, lockedUntil, lockedErr.LockedUntil, time.Second)

	f.repo.AssertNotCalled(t, "FindUserByPhone")
	f.repo.AssertNotCalled(t, "RecordLoginFailure", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestLogin_LockAppliesToUnknownPhone covers AC7's "the counter works for
// numbers with no account" rule, which is also AC2's: the lock is consulted
// *before* the account lookup, so a locked number answers 429 whether or not it
// is registered. Reversing the two would make the lock a registration oracle.
func TestLogin_LockAppliesToUnknownPhone(t *testing.T) {
	f := newLoginFixture()

	lockedUntil := time.Now().Add(10 * time.Minute)
	f.repo.On("FindLoginAttempt", mock.Anything, testPhoneNumber).
		Return(&domain.LoginAttempt{
			PhoneNumber: testPhoneNumber,
			FailedCount: testLoginPolicy.MaxFailedAttempts,
			LockedUntil: &lockedUntil,
		}, nil)

	_, err := f.usecase.Login(context.Background(), defaultLoginInput())

	assert.ErrorIs(t, err, domain.ErrLoginLocked)
	// The lock is answered without ever asking whether the account exists.
	f.repo.AssertNotCalled(t, "FindUserByPhone")
}

// TestLogin_ExpiredLockIsNotALock pins the boundary the repository reads rather
// than scrubs: a deadline already in the past is "not locked", so the login
// proceeds normally and the stale counter is corrected by the next failure.
func TestLogin_ExpiredLockIsNotALock(t *testing.T) {
	f := newLoginFixture()

	expired := time.Now().Add(-time.Minute)
	f.repo.On("FindLoginAttempt", mock.Anything, testPhoneNumber).
		Return(&domain.LoginAttempt{
			PhoneNumber: testPhoneNumber,
			FailedCount: testLoginPolicy.MaxFailedAttempts,
			LockedUntil: &expired,
		}, nil)
	f.repo.On("FindUserByPhone", mock.Anything, testPhoneNumber).Return(testUser(), nil)
	f.repo.On("CreateLoginSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.AnythingOfType("domain.TokenIssuer")).
		Return(&domain.LoginResult{User: testUser(), AccessToken: testAccessToken, RefreshToken: testRefreshToken}, nil)

	result, err := f.usecase.Login(context.Background(), defaultLoginInput())

	require.NoError(t, err)
	require.NotNil(t, result)
}

// TestLogin_InfrastructureFailure pins that errors from either port stay
// infrastructure errors: they must not be mistaken for the sentinels the
// handler maps to a 4xx, and a failure to *count* a guess must not be swallowed
// into a 401 as if the counter had worked.
func TestLogin_InfrastructureFailure(t *testing.T) {
	tests := []struct {
		name       string
		attemptErr error
		user       *domain.User
		userErr    error
		sessionErr error
		wantCause  string
	}{
		{
			name:       "counter read fails",
			attemptErr: errors.New("connection refused"),
			wantCause:  "find login attempt",
		},
		{
			name:      "account lookup fails",
			userErr:   errors.New("connection reset"),
			wantCause: "find user by phone",
		},
		{
			name:       "counter write fails",
			userErr:    domain.ErrUserNotFound,
			attemptErr: nil,
			wantCause:  "record login failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newLoginFixture()

			if tt.attemptErr != nil && tt.name != "counter write fails" {
				f.repo.On("FindLoginAttempt", mock.Anything, mock.Anything).Return(nil, tt.attemptErr)
			} else {
				f.repo.On("FindLoginAttempt", mock.Anything, mock.Anything).Return(nil, nil)
			}

			f.repo.On("FindUserByPhone", mock.Anything, mock.Anything).Return(tt.user, tt.userErr)

			if tt.name == "counter write fails" {
				f.repo.On("RecordLoginFailure", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("deadlock detected"))
			} else {
				f.repo.On("RecordLoginFailure", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(&domain.LoginAttempt{FailedCount: 1}, nil)
			}

			_, err := f.usecase.Login(context.Background(), defaultLoginInput())

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantCause)
			assert.NotErrorIs(t, err, domain.ErrInvalidCredentials)
			f.repo.AssertNotCalled(t, "CreateLoginSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

// TestLogin_SessionCreationFailure pins that a repository failure while opening
// the session is not reported as a login success with empty tokens.
func TestLogin_SessionCreationFailure(t *testing.T) {
	f := newLoginFixture()

	f.repo.On("FindLoginAttempt", mock.Anything, mock.Anything).Return(nil, nil)
	f.repo.On("FindUserByPhone", mock.Anything, testPhoneNumber).Return(testUser(), nil)
	f.repo.On("CreateLoginSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("unique violation"))

	result, err := f.usecase.Login(context.Background(), defaultLoginInput())

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorContains(t, err, "unique violation")
}

// ---------------------------------------------------------------------------
// Refresh
// ---------------------------------------------------------------------------

// TestRefresh_HappyPath covers AC9: a live refresh token buys a brand new pair.
//
// The assertion that matters is the hash the use case sends: the stored column
// is a digest, so a use case that passed the raw token would find no session at
// all. The other one is the row ID — rotation addresses the row it re-reads FOR
// UPDATE, not the session_id the tokens carry.
func TestRefresh_HappyPath(t *testing.T) {
	f := newLoginFixture()

	session := activeSession()
	f.jwt.On("ValidateToken", testRefreshToken).
		Return(&jwtclaims.CustomClaims{UserID: "42", TokenType: jwtclaims.TokenTypeRefresh}, nil)
	f.repo.On("FindSessionByRefreshHash", mock.Anything, domain.HashRefreshToken(testRefreshToken)).
		Return(session, false, nil)
	f.repo.On("FindUserByID", mock.Anything, testUserRowID).Return(testUser(), nil)

	var (
		capturedRowID int64
		capturedHash  string
		capturedTTL   time.Duration
	)
	f.repo.On("RotateSession", mock.Anything, mock.AnythingOfType("int64"), mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration"), mock.AnythingOfType("domain.TokenIssuer")).
		Run(func(args mock.Arguments) {
			capturedRowID = args.Get(1).(int64)
			capturedHash = args.String(2)
			capturedTTL = args.Get(3).(time.Duration)
		}).
		Return(&domain.RefreshResult{AccessToken: "new-access", RefreshToken: "new-refresh"}, nil)

	result, err := f.usecase.Refresh(context.Background(), domain.RefreshInput{RefreshToken: testRefreshToken})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "new-access", result.AccessToken)
	assert.Equal(t, "new-refresh", result.RefreshToken)

	// The presented token reaches the database only as its digest.
	assert.Equal(t, domain.HashRefreshToken(testRefreshToken), capturedHash)
	assert.NotEqual(t, testRefreshToken, capturedHash)

	// AC9's sliding window: the session is pushed out by the same TTL a new
	// login would give it.
	assert.Equal(t, testSessionRowID, capturedRowID)
	assert.Equal(t, testSessionTTL, capturedTTL)

	// A successful rotation is not a reuse, so nothing is revoked.
	f.repo.AssertNotCalled(t, "RevokeSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestRefresh_TokenIssuerContract proves the rotated pair is signed through the
// same issuer login uses, with the same "sid" claim: the session keeps its
// identity across refreshes, which is what AC9's "same sid" requires and what
// lets a logout after a refresh still find the row.
func TestRefresh_TokenIssuerContract(t *testing.T) {
	f := newLoginFixture()

	f.jwt.On("ValidateToken", mock.Anything).
		Return(&jwtclaims.CustomClaims{UserID: "42", TokenType: jwtclaims.TokenTypeRefresh}, nil)
	f.repo.On("FindSessionByRefreshHash", mock.Anything, mock.Anything).Return(activeSession(), false, nil)
	f.repo.On("FindUserByID", mock.Anything, mock.Anything).Return(testUser(), nil)

	f.jwt.On("GenerateAccessToken", "42", testEmail, "user", map[string]any{"sid": testSessionID}).
		Return(testAccessToken, nil)
	f.jwt.On("GenerateRefreshToken", "42", mock.Anything).Return(testRefreshToken, nil)

	var issuer domain.TokenIssuer
	f.repo.On("RotateSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.AnythingOfType("domain.TokenIssuer")).
		Run(func(args mock.Arguments) {
			issuer = args.Get(4).(domain.TokenIssuer)
		}).
		Return(&domain.RefreshResult{}, nil)

	_, err := f.usecase.Refresh(context.Background(), domain.RefreshInput{RefreshToken: testRefreshToken})
	require.NoError(t, err)
	require.NotNil(t, issuer)

	accessToken, refreshToken, err := issuer("42", testSessionID)
	require.NoError(t, err)
	assert.Equal(t, testAccessToken, accessToken)
	assert.Equal(t, testRefreshToken, refreshToken)
}

// TestRefresh_ReuseDetected covers AC11: a token from the previous generation
// is not merely refused, it takes the session down with it. The lesson is that
// whoever holds it is not the one who rotated it, so the device's live session
// dies and its owner has to log in again.
func TestRefresh_ReuseDetected(t *testing.T) {
	f := newLoginFixture()

	session := activeSession()
	f.jwt.On("ValidateToken", testRefreshToken).
		Return(&jwtclaims.CustomClaims{UserID: "42", TokenType: jwtclaims.TokenTypeRefresh}, nil)
	f.repo.On("FindSessionByRefreshHash", mock.Anything, domain.HashRefreshToken(testRefreshToken)).
		Return(session, true, nil)
	f.repo.On("RevokeSession", mock.Anything, testUserRowID, testSessionID, domain.RevokeReasonRefreshReuse).
		Return(nil)

	result, err := f.usecase.Refresh(context.Background(), domain.RefreshInput{RefreshToken: testRefreshToken})

	// Still one opaque 401 — the caller learns nothing about why.
	assert.ErrorIs(t, err, domain.ErrRefreshTokenInvalid)
	assert.Nil(t, result)

	// Scoped to the reused session, and to that user: another device's session
	// is untouched, so a stolen token cannot log its owner out everywhere.
	f.repo.AssertCalled(t, "RevokeSession", mock.Anything, testUserRowID, testSessionID, domain.RevokeReasonRefreshReuse)
	f.repo.AssertNotCalled(t, "RotateSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestRefresh_Invalid covers AC10: every reason a token cannot be used produces
// the same sentinel and, importantly, none of them reaches the database — a
// token that fails validation must cost no query, or the endpoint becomes a
// cheap way to probe the sessions table.
func TestRefresh_Invalid(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		claims     *jwtclaims.CustomClaims
		validateEr error
	}{
		{
			name:  "no token at all",
			token: "",
		},
		{
			name:       "malformed or badly signed",
			token:      "not-a-jwt",
			validateEr: errors.New("signature is invalid"),
		},
		{
			name:       "expired",
			token:      testRefreshToken,
			validateEr: errors.New("token is expired"),
		},
		{
			// AC10: an access token presented at /refresh must not be rolled
			// into a 7-day refresh token.
			name:   "an access token, not a refresh token",
			token:  testAccessToken,
			claims: &jwtclaims.CustomClaims{UserID: "42", TokenType: jwtclaims.TokenTypeAccess},
		},
		{
			// A session row that is revoked, expired, or simply absent comes
			// back as the same sentinel from the repository.
			name:   "no matching session",
			token:  testRefreshToken,
			claims: &jwtclaims.CustomClaims{UserID: "42", TokenType: jwtclaims.TokenTypeRefresh},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newLoginFixture()

			if tt.token != "" {
				f.jwt.On("ValidateToken", tt.token).Return(tt.claims, tt.validateEr)
			}
			// Only the "no matching session" case gets this far.
			f.repo.On("FindSessionByRefreshHash", mock.Anything, mock.Anything).
				Return(nil, false, domain.ErrRefreshTokenInvalid)

			result, err := f.usecase.Refresh(context.Background(), domain.RefreshInput{RefreshToken: tt.token})

			assert.ErrorIs(t, err, domain.ErrRefreshTokenInvalid)
			assert.Nil(t, result)
			f.repo.AssertNotCalled(t, "RotateSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			f.repo.AssertNotCalled(t, "RevokeSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

// TestRefresh_DeviceLessSession covers AC12: a session created by US-001's
// verify-otp has no device at all, and refresh must treat it like any other —
// the nil DeviceID is a legal value on the way through, not a special case.
func TestRefresh_DeviceLessSession(t *testing.T) {
	f := newLoginFixture()

	session := activeSession()
	session.DeviceID = nil

	f.jwt.On("ValidateToken", testRefreshToken).
		Return(&jwtclaims.CustomClaims{UserID: "42", TokenType: jwtclaims.TokenTypeRefresh}, nil)
	f.repo.On("FindSessionByRefreshHash", mock.Anything, mock.Anything).Return(session, false, nil)
	f.repo.On("FindUserByID", mock.Anything, testUserRowID).Return(testUser(), nil)
	f.repo.On("RotateSession", mock.Anything, testSessionRowID, mock.Anything, mock.Anything, mock.Anything).
		Return(&domain.RefreshResult{AccessToken: "new-access", RefreshToken: "new-refresh"}, nil)

	result, err := f.usecase.Refresh(context.Background(), domain.RefreshInput{RefreshToken: testRefreshToken})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "new-access", result.AccessToken)
}

// TestRefresh_UserGone covers a session that outlived its account. There is
// nothing to mint a token for, and the caller must not be able to tell this
// apart from any other unusable token.
func TestRefresh_UserGone(t *testing.T) {
	f := newLoginFixture()

	f.jwt.On("ValidateToken", testRefreshToken).
		Return(&jwtclaims.CustomClaims{UserID: "42", TokenType: jwtclaims.TokenTypeRefresh}, nil)
	f.repo.On("FindSessionByRefreshHash", mock.Anything, mock.Anything).Return(activeSession(), false, nil)
	f.repo.On("FindUserByID", mock.Anything, testUserRowID).Return(nil, domain.ErrUserNotFound)

	result, err := f.usecase.Refresh(context.Background(), domain.RefreshInput{RefreshToken: testRefreshToken})

	assert.ErrorIs(t, err, domain.ErrRefreshTokenInvalid)
	assert.Nil(t, result)
	f.repo.AssertNotCalled(t, "RotateSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestRefresh_LostRace covers the second half of AC9: two concurrent refreshes
// with the same token cannot both win. The loser is the repository's
// SELECT ... FOR UPDATE finding a hash that no longer matches, reported here as
// the ordinary 401 rather than as a system error — and crucially without
// revoking anything, because a genuine race is not evidence of theft.
func TestRefresh_LostRace(t *testing.T) {
	f := newLoginFixture()

	f.jwt.On("ValidateToken", testRefreshToken).
		Return(&jwtclaims.CustomClaims{UserID: "42", TokenType: jwtclaims.TokenTypeRefresh}, nil)
	f.repo.On("FindSessionByRefreshHash", mock.Anything, mock.Anything).Return(activeSession(), false, nil)
	f.repo.On("FindUserByID", mock.Anything, mock.Anything).Return(testUser(), nil)
	f.repo.On("RotateSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, domain.ErrRefreshTokenInvalid)

	result, err := f.usecase.Refresh(context.Background(), domain.RefreshInput{RefreshToken: testRefreshToken})

	assert.ErrorIs(t, err, domain.ErrRefreshTokenInvalid)
	assert.Nil(t, result)
	f.repo.AssertNotCalled(t, "RevokeSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestRefresh_InfrastructureFailure pins that a database failure on the lookup
// stays a 500 rather than being flattened into "your token is bad", which would
// tell a user with a perfectly good token to go and log in again.
func TestRefresh_InfrastructureFailure(t *testing.T) {
	f := newLoginFixture()

	f.jwt.On("ValidateToken", testRefreshToken).
		Return(&jwtclaims.CustomClaims{UserID: "42", TokenType: jwtclaims.TokenTypeRefresh}, nil)
	f.repo.On("FindSessionByRefreshHash", mock.Anything, mock.Anything).
		Return(nil, false, errors.New("connection refused"))

	_, err := f.usecase.Refresh(context.Background(), domain.RefreshInput{RefreshToken: testRefreshToken})

	require.Error(t, err)
	assert.ErrorContains(t, err, "find session")
	assert.NotErrorIs(t, err, domain.ErrRefreshTokenInvalid)
}

// ---------------------------------------------------------------------------
// Logout
// ---------------------------------------------------------------------------

// TestLogout_Revokes covers AC13: the session the access token names is revoked
// with the 'logout' reason, so a later refresh fails the way AC10 describes.
func TestLogout_Revokes(t *testing.T) {
	f := newLoginFixture()

	f.repo.On("RevokeSession", mock.Anything, testUserRowID, testSessionID, domain.RevokeReasonLogout).Return(nil)

	err := f.usecase.Logout(context.Background(), domain.LogoutInput{UserID: testUserRowID, SessionID: testSessionID})

	require.NoError(t, err)
	f.repo.AssertCalled(t, "RevokeSession", mock.Anything, testUserRowID, testSessionID, domain.RevokeReasonLogout)
}

// TestLogout_Idempotent covers AC14: a session that is already gone revokes
// nothing and is not an error. The repository reports "no rows changed" as
// success, which is what makes a retried logout safe — a client that never saw
// the first success must not get a failure on the retry.
func TestLogout_Idempotent(t *testing.T) {
	tests := []struct {
		name string
		// The repository answers nil for a session that is already revoked,
		// expired, or was never there.
		repoErr error
		wantErr bool
	}{
		{name: "already revoked", repoErr: nil, wantErr: false},
		{name: "no such session", repoErr: nil, wantErr: false},
		{name: "database unavailable", repoErr: errors.New("connection refused"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newLoginFixture()

			f.repo.On("RevokeSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tt.repoErr)

			err := f.usecase.Logout(context.Background(), domain.LogoutInput{UserID: testUserRowID, SessionID: testSessionID})

			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, "revoke session")
				return
			}
			assert.NoError(t, err)
		})
	}
}

// TestLogout_ScopedToUser covers AC14's lookup rule: the repository matches on
// BOTH ids, so naming a session that belongs to someone else revokes nothing.
// The use case's half is that it passes both through — it never looks the
// session up by ID alone.
func TestLogout_ScopedToUser(t *testing.T) {
	f := newLoginFixture()

	f.repo.On("RevokeSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err := f.usecase.Logout(context.Background(), domain.LogoutInput{
		UserID:    testUserRowID,
		SessionID: testSessionID,
	})
	require.NoError(t, err)

	args := f.repo.Calls[0].Arguments
	assert.Equal(t, testUserRowID, args.Get(1).(int64), "user_id must be passed, not just the session id")
	assert.Equal(t, testSessionID, args.String(2))
	assert.Equal(t, domain.RevokeReasonLogout, args.String(3))
}
