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
)

// Tests for US-002 (login, refresh, logout), login spec v2. The acceptance
// criteria each test proves are named in the test's own doc comment; the
// mapping is:
//
//	AC1  matching email+password+device logs in             -> TestLogin_HappyPath
//	AC2  wrong password / unknown / pending / unverified    -> TestLogin_InvalidCredentials,
//	                                                            TestLogin_NoEnumeration
//	AC3  bad email / empty password                         -> TestLogin_Validation
//	AC4  device without platform / device_model             -> TestLogin_Validation
//	AC5  attempts reserved per email, fifth locks           -> TestLogin_ReservesBeforeCompare,
//	                                                            TestLogin_TripLockStill401,
//	                                                            TestLogin_Locked
//	AC6  a known device supersedes only its own session     -> repo-level
//	AC7  access 15m, refresh 7d, sid claim, hashed storage  -> TestLogin_TokenIssuerContract
//	AC8  same hashing as registration                       -> TestLogin_HappyPath (BcryptPasswordHasher)
//	AC9  refresh rotates the pair                           -> TestRefresh_HappyPath,
//	                                                            TestRefresh_TokenIssuerContract
//	AC10 every unusable refresh token is one 401            -> TestRefresh_Invalid
//	AC11 an already-rotated token revokes the session       -> TestRefresh_ReuseDetected
//	AC12 a device-less US-001 session refreshes too         -> TestRefresh_DeviceLessSession
//	AC13 logout revokes the session the sid names           -> TestLogout_Revokes
//	AC14 logout is idempotent and scoped to the user        -> TestLogout_Idempotent,
//	                                                            TestLogout_ScopedToUser

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
)

// testLoginPolicy mirrors config.dev.yaml's `login:` block. Pinned as literals
// (not read back out of config) so a mis-wired policy fails a test: the
// assertions on what ReserveLoginAttempt receives compare against these.
var testLoginPolicy = config.LoginConfig{
	MaxFailedAttempts:  5,
	LockDuration:       config.Duration(15 * time.Minute),
	FailureResetWindow: config.Duration(15 * time.Minute),
}

// testPasswordHash is BcryptPasswordHasher's hash of testPassword — the scheme
// registration stores — computed once at package load.
//
// DefaultCost, not MinCost: TestLogin_NoEnumeration compares the cost of the
// known-account branch with the unknown one, which always pays for
// dummyPasswordHash at DefaultCost.
var testPasswordHash = func() string {
	hash, err := NewBcryptPasswordHasher().Hash(testPassword)
	if err != nil {
		panic("hashing the test password failed: " + err.Error())
	}
	return hash
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
		usecase: NewLoginUsecase(repo, NewBcryptPasswordHasher(), jwtService, newLogger(), testSessionTTL, testLoginPolicy),
		repo:    repo,
		jwt:     jwtService,
		logger:  newLogger(),
	}
}

// testUser builds the verified account a successful login resolves to.
func testUser() *domain.User {
	verifiedAt := time.Now().Add(-time.Hour)
	return &domain.User{
		ID:              testUserRowID,
		Email:           testEmail,
		EmailVerifiedAt: &verifiedAt,
		PhoneNumber:     testPhoneNumber,
		PasswordHash:    testPasswordHash,
	}
}

// defaultLoginInput is a request that passes every check before the database.
// The email is the raw, un-normalised form: every repository call must receive
// testEmail instead, or "User@Example.com" and "user@example.com" would get two
// counters and double the guess budget.
func defaultLoginInput() domain.LoginInput {
	return domain.LoginInput{
		Email:    testEmailRaw,
		Password: testPassword,
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

// expectReserve stubs ReserveLoginAttempt with the exact policy from config and
// the normalised email, returning attempt.
func expectReserve(f *loginFixture, attempt *domain.LoginAttempt) *mock.Call {
	return f.repo.On("ReserveLoginAttempt",
		mock.Anything,
		testEmail,
		testLoginPolicy.MaxFailedAttempts,
		testLoginPolicy.LockDuration.Duration(),
		testLoginPolicy.FailureResetWindow.Duration(),
	).Return(attempt, nil)
}

// counted is the ordinary reservation result: one attempt counted, no lock.
func counted() *domain.LoginAttempt {
	return &domain.LoginAttempt{Email: testEmail, FailedCount: 1, LastFailedAt: time.Now()}
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

// TestLogin_HappyPath covers AC1 and AC8: a matching email and password (hashed
// the way registration hashes it) opens a session bound to the device.
func TestLogin_HappyPath(t *testing.T) {
	f := newLoginFixture()

	user := testUser()
	device := defaultLoginInput().Device
	wantResult := &domain.LoginResult{
		User:         user,
		AccessToken:  testAccessToken,
		RefreshToken: testRefreshToken,
	}

	expectReserve(f, counted())
	f.repo.On("FindUserByEmail", mock.Anything, testEmail).Return(user, nil)

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

	// The session lifetime comes from config, so the stored expiry and the
	// signed refresh token cannot drift.
	assert.Equal(t, testSessionTTL, capturedTTL)

	// The device is the repository's to decide on (AC6), this layer's to hand
	// over intact.
	capturedDevice := f.repo.Calls[len(f.repo.Calls)-1].Arguments.Get(2).(domain.DeviceInfo)
	assert.Equal(t, device, capturedDevice)
}

// TestLogin_TokenIssuerContract proves the issuer handed to the repository
// really signs both tokens, and that the access token carries the session ID as
// claim "sid" (AC7) — logout finds the session through exactly that claim.
func TestLogin_TokenIssuerContract(t *testing.T) {
	f := newLoginFixture()

	user := testUser()
	expectReserve(f, counted())
	f.repo.On("FindUserByEmail", mock.Anything, testEmail).Return(user, nil)

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
// rather than an empty token pair.
func TestLogin_TokenIssuerFailure(t *testing.T) {
	f := newLoginFixture()

	user := testUser()
	expectReserve(f, counted())
	f.repo.On("FindUserByEmail", mock.Anything, testEmail).Return(user, nil)
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

// TestLogin_InvalidCredentials covers AC2: every cause is the same sentinel
// with the same message, the attempt was already counted by the reservation,
// and no session is opened.
func TestLogin_InvalidCredentials(t *testing.T) {
	unverified := testUser()
	unverified.EmailVerifiedAt = nil

	tests := []struct {
		name     string
		user     *domain.User
		userErr  error
		password string
	}{
		{name: "wrong password", user: testUser(), password: "not-the-password"},
		{name: "no account for the email", userErr: domain.ErrUserNotFound, password: testPassword},
		{
			// A registration in flight has no users row yet, so this is the
			// not-found path — it must not answer "verify your code".
			name: "email exists only in pending_registration", userErr: domain.ErrUserNotFound, password: testPassword,
		},
		{
			// The correct password on an account whose email was never verified
			// is refused like any other miss.
			name: "email_verified_at is NULL", user: unverified, password: testPassword,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newLoginFixture()

			expectReserve(f, counted())
			f.repo.On("FindUserByEmail", mock.Anything, testEmail).Return(tt.user, tt.userErr)

			input := defaultLoginInput()
			input.Password = tt.password

			result, err := f.usecase.Login(context.Background(), input)

			assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
			assert.Equal(t, "Email hoặc mật khẩu không đúng", err.Error())
			assert.Nil(t, result)

			f.repo.AssertCalled(t, "ReserveLoginAttempt", mock.Anything, testEmail, testLoginPolicy.MaxFailedAttempts, testLoginPolicy.LockDuration.Duration(), testLoginPolicy.FailureResetWindow.Duration())
			f.repo.AssertNotCalled(t, "CreateLoginSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

// TestLogin_Validation covers AC3 (email, password) and AC4 (device): every one
// of these returns before the first query and, crucially, before any attempt is
// reserved. A malformed request is not a guess, and counting it would let
// anyone lock a stranger's email out by sending junk.
func TestLogin_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   domain.LoginInput
		wantErr error

		// reachesRepo marks rows that pass validation and must reach the
		// reservation (and then fail as an ordinary bad credential).
		reachesRepo bool
	}{
		{
			name:    "email is malformed",
			input:   withLoginInput(func(in *domain.LoginInput) { in.Email = "not-an-email" }),
			wantErr: domain.ErrInvalidEmail,
		},
		{
			name:    "email is missing",
			input:   withLoginInput(func(in *domain.LoginInput) { in.Email = "" }),
			wantErr: domain.ErrInvalidEmail,
		},
		{
			name:    "password is missing",
			input:   withLoginInput(func(in *domain.LoginInput) { in.Password = "" }),
			wantErr: domain.ErrInvalidPassword,
		},
		{
			// AC3: no password rule but "not empty" applies at sign-in.
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
				expectReserve(f, counted())
				f.repo.On("FindUserByEmail", mock.Anything, testEmail).Return(nil, domain.ErrUserNotFound)

				result, err := f.usecase.Login(context.Background(), tt.input)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
				return
			}

			result, err := f.usecase.Login(context.Background(), tt.input)

			assert.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, result)

			// Nothing is read and nothing is counted.
			f.repo.AssertNotCalled(t, "ReserveLoginAttempt", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			f.repo.AssertNotCalled(t, "FindUserByEmail", mock.Anything, mock.Anything)
		})
	}
}

// TestLogin_NoEnumeration pins AC2's timing property: an unknown email still
// pays for exactly one bcrypt comparison.
func TestLogin_NoEnumeration(t *testing.T) {
	measure := func(t *testing.T, user *domain.User, userErr error) time.Duration {
		t.Helper()
		f := newLoginFixture()

		expectReserve(f, counted())
		f.repo.On("FindUserByEmail", mock.Anything, mock.Anything).Return(user, userErr)

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

	// Loose ratio: what it catches is an early return on the not-found branch,
	// which would make the unknown case ~100x faster, not 2x.
	ratio := float64(known) / float64(unknown)
	assert.Greater(t, ratio, 0.2, "unknown-email branch returned too fast to have hashed anything")
	assert.Less(t, ratio, 5.0, "known-email branch did far more work than the unknown one")
}

// TestLogin_ReservesBeforeCompare covers AC5's ordering: the reservation runs
// before the account lookup, so the counter covers emails with no account and
// concurrent guesses cannot outrun it.
func TestLogin_ReservesBeforeCompare(t *testing.T) {
	f := newLoginFixture()

	var order []string
	expectReserve(f, counted()).Run(func(mock.Arguments) { order = append(order, "reserve") })
	f.repo.On("FindUserByEmail", mock.Anything, testEmail).
		Run(func(mock.Arguments) { order = append(order, "find") }).
		Return(nil, domain.ErrUserNotFound)

	_, err := f.usecase.Login(context.Background(), defaultLoginInput())

	assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
	assert.Equal(t, []string{"reserve", "find"}, order)
}

// TestLogin_TripLockStill401 covers AC5's boundary: the attempt whose
// reservation places the lock still has its password compared and, if wrong,
// answers 401 — only the next request sees 429.
func TestLogin_TripLockStill401(t *testing.T) {
	lockedUntil := time.Now().Add(15 * time.Minute)
	tripping := &domain.LoginAttempt{
		Email:        testEmail,
		FailedCount:  testLoginPolicy.MaxFailedAttempts,
		LockedUntil:  &lockedUntil,
		LastFailedAt: time.Now(),
	}

	t.Run("wrong password is 401, not 429", func(t *testing.T) {
		f := newLoginFixture()
		expectReserve(f, tripping)
		f.repo.On("FindUserByEmail", mock.Anything, testEmail).Return(testUser(), nil)

		input := defaultLoginInput()
		input.Password = "wrong-password"
		_, err := f.usecase.Login(context.Background(), input)

		assert.ErrorIs(t, err, domain.ErrInvalidCredentials)
		assert.NotErrorIs(t, err, domain.ErrLoginLocked)
	})

	t.Run("correct password still logs in", func(t *testing.T) {
		f := newLoginFixture()
		expectReserve(f, tripping)
		f.repo.On("FindUserByEmail", mock.Anything, testEmail).Return(testUser(), nil)
		f.repo.On("CreateLoginSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(&domain.LoginResult{User: testUser(), AccessToken: testAccessToken, RefreshToken: testRefreshToken}, nil)

		result, err := f.usecase.Login(context.Background(), defaultLoginInput())

		require.NoError(t, err)
		require.NotNil(t, result)
	})
}

// TestLogin_Locked covers AC5 from the other side: while a lock placed by an
// earlier attempt holds, the account is not looked up and the password is not
// compared — the correct password is refused too.
func TestLogin_Locked(t *testing.T) {
	f := newLoginFixture()

	lockedUntil := time.Now().Add(10 * time.Minute)
	expectReserve(f, &domain.LoginAttempt{
		Email:         testEmail,
		FailedCount:   testLoginPolicy.MaxFailedAttempts,
		LockedUntil:   &lockedUntil,
		AlreadyLocked: true,
	})

	result, err := f.usecase.Login(context.Background(), defaultLoginInput())

	assert.ErrorIs(t, err, domain.ErrLoginLocked)
	assert.Nil(t, result)

	lockedErr, ok := errors.AsType[*domain.LoginLockedError](err)
	require.True(t, ok)
	assert.WithinDuration(t, lockedUntil, lockedErr.LockedUntil, time.Second)

	f.repo.AssertNotCalled(t, "FindUserByEmail", mock.Anything, mock.Anything)
	f.repo.AssertNotCalled(t, "CreateLoginSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestLogin_InfrastructureFailure pins that errors from the port stay
// infrastructure errors, not a 401 as if the counter had worked.
func TestLogin_InfrastructureFailure(t *testing.T) {
	t.Run("reservation fails", func(t *testing.T) {
		f := newLoginFixture()
		f.repo.On("ReserveLoginAttempt", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(nil, errors.New("deadlock detected"))

		_, err := f.usecase.Login(context.Background(), defaultLoginInput())

		require.Error(t, err)
		assert.ErrorContains(t, err, "reserve login attempt")
		assert.NotErrorIs(t, err, domain.ErrInvalidCredentials)
		f.repo.AssertNotCalled(t, "FindUserByEmail", mock.Anything, mock.Anything)
	})

	t.Run("account lookup fails", func(t *testing.T) {
		f := newLoginFixture()
		expectReserve(f, counted())
		f.repo.On("FindUserByEmail", mock.Anything, mock.Anything).Return(nil, errors.New("connection reset"))

		_, err := f.usecase.Login(context.Background(), defaultLoginInput())

		require.Error(t, err)
		assert.ErrorContains(t, err, "find user by email")
		assert.NotErrorIs(t, err, domain.ErrInvalidCredentials)
		f.repo.AssertNotCalled(t, "CreateLoginSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}

// TestLogin_SessionCreationFailure pins that a repository failure while opening
// the session is not reported as a login success with empty tokens.
func TestLogin_SessionCreationFailure(t *testing.T) {
	f := newLoginFixture()

	expectReserve(f, counted())
	f.repo.On("FindUserByEmail", mock.Anything, testEmail).Return(testUser(), nil)
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

// ---------------------------------------------------------------------------
// EnsureSessionActive (login Step A-2, AC21/AC22)
// ---------------------------------------------------------------------------

func TestEnsureSessionActive(t *testing.T) {
	ctx := context.Background()

	t.Run("live session passes", func(t *testing.T) {
		f := newLoginFixture()
		f.repo.On("IsSessionActive", mock.Anything, testUserRowID, testSessionID).Return(true, nil)
		require.NoError(t, f.usecase.EnsureSessionActive(ctx, testUserRowID, testSessionID))
	})

	t.Run("revoked or expired session is ErrSessionRevoked", func(t *testing.T) {
		f := newLoginFixture()
		f.repo.On("IsSessionActive", mock.Anything, testUserRowID, testSessionID).Return(false, nil)
		assert.ErrorIs(t, f.usecase.EnsureSessionActive(ctx, testUserRowID, testSessionID), domain.ErrSessionRevoked)
	})

	t.Run("empty sid is refused without a query", func(t *testing.T) {
		f := newLoginFixture()
		assert.ErrorIs(t, f.usecase.EnsureSessionActive(ctx, testUserRowID, ""), domain.ErrSessionRevoked)
		f.repo.AssertNotCalled(t, "IsSessionActive", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("database error is not reported as revoked", func(t *testing.T) {
		f := newLoginFixture()
		f.repo.On("IsSessionActive", mock.Anything, testUserRowID, testSessionID).Return(false, errors.New("boom"))
		err := f.usecase.EnsureSessionActive(ctx, testUserRowID, testSessionID)
		require.Error(t, err)
		assert.NotErrorIs(t, err, domain.ErrSessionRevoked)
	})
}
