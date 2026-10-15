package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"chjpcoj.com.innerlamp/pkg/jwt"
	jwtclaims "chjpcoj.com.innerlamp/pkg/jwt/claims"
	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/pkg/validation"
	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// dummyPasswordHash is compared against whenever the phone number has no
// account, so that "unknown phone" and "wrong password" cost the same wall
// clock time. Without it the unknown-phone branch would return after a single
// indexed lookup while the wrong-password branch pays for a bcrypt
// comparison — a difference any caller can measure, which would turn the login
// endpoint into a way to enumerate registered numbers (AC2).
//
// Generated once at package load at the same cost as a real hash, so timing
// the two is indistinguishable. The plaintext is never a valid password for
// any account: Submit() rejects the attempt anyway when the user lookup missed,
// so even a caller who guessed this string gets nothing.
//
// Built with BcryptPasswordHasher, the scheme the composition root injects, so
// comparing against it costs what comparing against a real stored hash costs.
var dummyPasswordHash = func() string {
	hash, err := BcryptPasswordHasher{}.Hash("timing-equalizer-not-a-password")
	if err != nil {
		panic("hashing the dummy password failed: " + err.Error())
	}
	return hash
}()

// LoginUsecase implements US-002: the three endpoints that turn a phone number
// and password into a session (login), roll that session's tokens forward
// (refresh), and end it (logout).
//
// It is deliberately separate from RegisterUsecase even though the two share
// TokenIssuer, validation.NormalizePhone and the session table: registration's
// two-step OTP flow and login's single password check have no state in common
// beyond the user row, and merging them would put a brute-force counter, an OTP
// policy and a device registry in one struct.
type LoginUsecase struct {
	logins     domain.LoginRepository
	passwords  domain.PasswordHasher
	jwtService jwt.Service
	logger     logging.Logger

	// sessionTTL is the lifetime a session gets at login and again on every
	// refresh (sliding). It comes from JWTConfig.RefreshTokenExpiration so the
	// stored expiry and the refresh token's own expiry cannot drift apart.
	sessionTTL time.Duration

	// loginPolicy carries the brute-force thresholds. Like the OTP policy in
	// RegisterUsecase these are config, not constants: a dev environment
	// shortens them to exercise the lock, production lengthens them under load.
	loginPolicy config.LoginConfig
}

// NewLoginUsecase wires the use case. Every collaborator is a port, a pkg
// abstraction, or a value read from config — no concrete repository or signing
// implementation is visible here.
func NewLoginUsecase(
	logins domain.LoginRepository,
	passwords domain.PasswordHasher,
	jwtService jwt.Service,
	logger logging.Logger,
	sessionTTL time.Duration,
	loginPolicy config.LoginConfig,
) *LoginUsecase {
	return &LoginUsecase{
		logins:      logins,
		passwords:   passwords,
		jwtService:  jwtService,
		logger:      logger,
		sessionTTL:  sessionTTL,
		loginPolicy: loginPolicy,
	}
}

// Login runs POST /v1/auth/login.
//
// The order of the checks is the security property, not an implementation
// detail: the lockout is consulted before the account is looked up, so a locked
// number is refused without its password ever being compared, and the counter
// covers numbers with no account — otherwise the lock itself would report which
// numbers are registered (AC7).
//
// The failure modes this can return are the sentinels the handler maps:
// ErrInvalidPhone, ErrInvalidPassword, ErrInvalidDevice, ErrInvalidCredentials,
// and *LoginLockedError. Anything else is wrapped infrastructure failure.
func (u *LoginUsecase) Login(ctx context.Context, input domain.LoginInput) (*domain.LoginResult, error) {
	phoneNumber := validation.NormalizePhone(input.PhoneNumber)
	if phoneNumber == "" {
		return nil, domain.ErrInvalidPhone
	}
	if input.Password == "" {
		return nil, domain.ErrInvalidPassword
	}

	// A malformed request is not a password guess: it returns before the first
	// query and never touches login_attempts. Counting these would let anyone
	// lock a stranger's number out with junk requests (AC5/AC6).
	if err := validateDevice(input.Device); err != nil {
		return nil, err
	}

	now := time.Now()

	attempt, err := u.logins.FindLoginAttempt(ctx, phoneNumber)
	if err != nil {
		return nil, fmt.Errorf("find login attempt: %w", err)
	}
	if isLoginLocked(attempt, now) {
		// No password comparison, no user lookup: while the lock holds, even
		// the correct password is refused.
		return nil, &domain.LoginLockedError{LockedUntil: *attempt.LockedUntil}
	}

	user, err := u.logins.FindUserByPhone(ctx, phoneNumber)
	switch {
	case err == nil:
	case errors.Is(err, domain.ErrUserNotFound):
		// Not a branch of its own past this point: an absent account falls
		// through to the same bcrypt comparison and the same 401 as a wrong
		// password, so neither the body nor the response time tells them apart.
		user = nil
	default:
		return nil, fmt.Errorf("find user by phone: %w", err)
	}

	passwordHash := dummyPasswordHash
	if user != nil {
		passwordHash = user.PasswordHash
	}

	// Exactly one bcrypt comparison on every path, whatever the lookup
	// returned — that is what keeps the two failure modes the same size.
	// The hasher is the one registration stored the hash with
	// (bcrypt(base64(SHA-256(NFC(pw)))), onboarding AC19).
	passwordMatches := u.passwords.Compare(passwordHash, input.Password) == nil

	if user == nil || !passwordMatches {
		if err := u.recordFailure(ctx, phoneNumber); err != nil {
			return nil, err
		}
		return nil, domain.ErrInvalidCredentials
	}

	result, err := u.logins.CreateLoginSession(ctx, user, input.Device, u.sessionTTL, u.tokenIssuer(user))
	if err != nil {
		return nil, err
	}

	u.logger.Info("login succeeded", "user_id", user.ID, "phone_number", phoneNumber)

	return result, nil
}

// Refresh runs POST /v1/auth/refresh: exchange a live refresh token for a new
// access token and a new refresh token.
//
// Rotation is the point. The presented token stops working the moment it is
// used, so a copy of it is worth at most one exchange — and if the copy is used
// *after* the legitimate holder rotated, the session is revoked outright
// (AC9/AC11).
func (u *LoginUsecase) Refresh(ctx context.Context, input domain.RefreshInput) (*domain.RefreshResult, error) {
	if input.RefreshToken == "" {
		return nil, domain.ErrRefreshTokenInvalid
	}

	parsed, err := u.jwtService.ValidateToken(input.RefreshToken)
	if err != nil {
		// Bad signature, expired, malformed — all one answer (AC10). The
		// distinction is only useful to someone probing.
		return nil, domain.ErrRefreshTokenInvalid
	}
	if parsed.TokenType != jwtclaims.TokenTypeRefresh {
		// An access token presented here is not a refresh credential; refusing
		// it keeps a stolen 15-minute token from being rolled into a 7-day one.
		return nil, domain.ErrRefreshTokenInvalid
	}

	// The token goes to the database only as this digest: the stored column is
	// what a database dump yields, and a dump must not yield a working token.
	presentedHash := domain.HashRefreshToken(input.RefreshToken)

	session, matchedPrevious, err := u.logins.FindSessionByRefreshHash(ctx, presentedHash)
	if err != nil {
		if errors.Is(err, domain.ErrRefreshTokenInvalid) {
			return nil, domain.ErrRefreshTokenInvalid
		}
		return nil, fmt.Errorf("find session: %w", err)
	}

	if matchedPrevious {
		if session == nil {
			return nil, domain.ErrRefreshTokenInvalid
		}
		// The token immediately before the current one was used. Either a
		// client that lost a race with itself or a stolen copy replaying — the
		// two are indistinguishable, so the session of that device dies and the
		// caller learns nothing beyond a normal 401 (AC11).
		if err := u.logins.RevokeSession(ctx, session.UserID, session.SessionID, domain.RevokeReasonRefreshReuse); err != nil {
			return nil, fmt.Errorf("revoke reused session: %w", err)
		}
		u.logger.Warn("refresh token reuse detected", "user_id", session.UserID, "session_id", session.SessionID)
		return nil, domain.ErrRefreshTokenInvalid
	}

	if session == nil {
		return nil, domain.ErrRefreshTokenInvalid
	}

	// A refresh request knows only a user ID, but minting an access token needs
	// the account's email claim, so the account is read back here.
	user, err := u.logins.FindUserByID(ctx, session.UserID)
	if err != nil {
		// The session outlived its user — nothing to refresh into.
		return nil, domain.ErrRefreshTokenInvalid
	}

	return u.logins.RotateSession(ctx, session.ID, presentedHash, u.sessionTTL, u.tokenIssuer(user))
}

// Logout runs POST /v1/auth/logout by revoking the session the caller's access
// token names.
//
// The repository matches on user_id *and* session_id, so a caller cannot end
// someone else's session by guessing an ID (AC14). It is idempotent by
// construction: revoking an already-revoked or unknown session changes no rows
// and is not an error, which is what makes a retried logout safe.
func (u *LoginUsecase) Logout(ctx context.Context, input domain.LogoutInput) error {
	if err := u.logins.RevokeSession(ctx, input.UserID, input.SessionID, domain.RevokeReasonLogout); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}

	u.logger.Info("logout", "user_id", input.UserID, "session_id", input.SessionID)

	return nil
}

// recordFailure counts one wrong guess and reports the outcome of counting it.
//
// The count itself is the repository's job — one atomic UPSERT, so two
// concurrent failures cannot both read the same starting value (AC7). This
// method only decides what a failure to *record* means: the guess was wrong
// either way, so the caller is refused the same way, but a database error must
// not be silently swallowed into a 401 as if the counter had worked.
func (u *LoginUsecase) recordFailure(ctx context.Context, phoneNumber string) error {
	attempt, err := u.logins.RecordLoginFailure(
		ctx,
		phoneNumber,
		u.loginPolicy.MaxFailedAttempts,
		u.loginPolicy.LockDuration.Duration(),
		u.loginPolicy.FailureResetWindow.Duration(),
	)
	if err != nil {
		return fmt.Errorf("record login failure: %w", err)
	}

	// Logged so an operator can see a lock being placed; the caller is told
	// nothing. The failure that tripped the limit is still a 401 — only the
	// next request meets the lock (AC7).
	if attempt != nil && attempt.LockedUntil != nil {
		u.logger.Warn("login locked after repeated failures",
			"phone_number", phoneNumber,
			"locked_until", *attempt.LockedUntil,
		)
	}

	return nil
}

// tokenIssuer builds the closure the repository calls inside its transaction,
// after it has chosen the session ID and before the session row is written.
//
// The account is captured here rather than looked up inside the transaction:
// the use case already has it, and the access token's email claim is the only
// thing the issuer needs from it. "sid" ties the access token to the session
// row so logout can revoke exactly that session; the refresh token carries no
// such claim because it is found by hash.
func (u *LoginUsecase) tokenIssuer(user *domain.User) domain.TokenIssuer {
	return func(userID, sessionID string) (string, string, error) {
		accessToken, err := u.jwtService.GenerateAccessToken(userID, user.Email, userRole, map[string]any{"sid": sessionID})
		if err != nil {
			return "", "", fmt.Errorf("generate access token: %w", err)
		}

		refreshToken, err := u.jwtService.GenerateRefreshToken(userID, nil)
		if err != nil {
			return "", "", fmt.Errorf("generate refresh token: %w", err)
		}

		return accessToken, refreshToken, nil
	}
}

// validateDevice rejects a request with no usable device description.
//
// platform and device_model are required because they, plus the server-side
// os_version and user_agent, are what the fingerprint is built from; a login
// with neither could not be attributed to a device at all, and AC4's
// "one active session per device" would have nothing to key on.
func validateDevice(device domain.DeviceInfo) error {
	if device.Platform == "" || device.DeviceModel == "" {
		return domain.ErrInvalidDevice
	}
	return nil
}

// isLoginLocked reports whether the counter row's lock is still in force. A nil
// attempt (nobody has ever failed against this number) is never locked, and a
// deadline already in the past is treated as unlocked — the row is read as
// stale rather than scrubbed.
func isLoginLocked(attempt *domain.LoginAttempt, now time.Time) bool {
	return attempt != nil && attempt.LockedUntil != nil && attempt.LockedUntil.After(now)
}
