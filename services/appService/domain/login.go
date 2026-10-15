package domain

import (
	"context"
	"time"
)

// DeviceInfo describes the device a login came from.
//
// The first five fields are the client's own description of itself and arrive
// in the request body. IPAddress and UserAgent are filled in by the handler
// from the *http.Request — the client cannot supply them, and a value it could
// supply would be worthless for the fingerprint below.
//
// Plain data: no json/db tag, same reasoning as User.
type DeviceInfo struct {
	Platform    string
	DeviceModel string

	// Optional: the client may omit these and the login still succeeds. Only
	// Platform and DeviceModel are required (AC5), because they are what the
	// fingerprint is built from.
	OSVersion  string
	AppVersion string
	DeviceName string

	// Server-supplied, never from the request body.
	IPAddress string
	UserAgent string
}

// Revoke reasons written to user_sessions.revoked_reason. They are domain
// values rather than free-form strings because the column is read back by
// operators to answer "why did this session end" — a typo'd literal in one
// call site would silently become a fourth, meaningless reason.
const (
	// RevokeReasonLogout — the user asked to log out (AC13).
	RevokeReasonLogout = "logout"

	// RevokeReasonRefreshReuse — a refresh token that had already been rotated
	// was presented again (AC11). Either a client that lost a race with itself
	// or a stolen copy replaying; the two are indistinguishable, so the session
	// is killed.
	RevokeReasonRefreshReuse = "refresh_token_reuse"

	// RevokeReasonSupersededByLogin — the same device logged in again, so its
	// previous session is replaced (AC4).
	RevokeReasonSupersededByLogin = "superseded_by_login"
)

// LoginAttempt is the per-phone-number brute-force counter behind AC7.
//
// It is keyed by phone number and deliberately has no relation to users: the
// counter must work for numbers that have no account, or the lock itself would
// reveal which numbers are registered.
type LoginAttempt struct {
	PhoneNumber string

	// FailedCount is the number of consecutive failures recorded so far. It is
	// reset to zero by a successful login, by the lock expiring, and by the
	// reset window elapsing without a new failure.
	FailedCount int

	// LockedUntil is when the lock lifts, or nil when the number is not locked.
	// A past deadline is treated as "not locked" by the use case — the row is
	// not scrubbed, it is simply read as unlocked.
	LockedUntil *time.Time

	// LastFailedAt is when the most recent failure was recorded. It is what
	// makes "consecutive" mean failures each within FailureResetWindow of the
	// one before, rather than "ever".
	LastFailedAt time.Time
}

// Session is a row of user_sessions as the refresh flow needs to see it.
//
// DeviceID is a pointer because sessions created by US-001's verify-otp have
// no device at all — AC12 requires those to refresh and log out exactly like
// any other session, so nil must be a legal value here rather than a zero.
type Session struct {
	// ID is the row's surrogate key, used to address it in RotateSession's
	// SELECT ... FOR UPDATE. SessionID is the opaque identifier the tokens
	// carry as claim "sid".
	ID        int64
	SessionID string
	UserID    int64

	DeviceID     *int64
	TokenVersion int
	ExpiresAt    time.Time
	IsRevoked    bool
}

// LoginInput is the use-case input for POST /v1/auth/login.
type LoginInput struct {
	PhoneNumber string
	Password    string
	Device      DeviceInfo
}

// LoginResult is what a successful login returns. It is an alias rather than a
// distinct struct because the shape is genuinely identical to a verified
// registration's — same three fields, same meaning — and the handler reuses
// verifyOTPResponse's userResponse for it. A second struct would be a second
// definition to keep in sync for no gain.
type LoginResult = RegisterResult

// RefreshInput is the use-case input for POST /v1/auth/refresh.
type RefreshInput struct {
	RefreshToken string
}

// RefreshResult carries the rotated pair. The refresh token is new on every
// call: rotation is what makes a stolen token usable at most once, and the
// reuse of an already-rotated one is the theft signal AC11 acts on.
type RefreshResult struct {
	AccessToken  string
	RefreshToken string
}

// LogoutInput identifies the session to revoke.
//
// Both values come from the caller's access token, not from the request body:
// the middleware verified the signature, so the client cannot name a session
// that is not its own. UserID is numeric because that is what user_sessions
// stores and what the user_id claim carries; the handler converts the claim's
// string form and answers 401 if it is not a number.
type LogoutInput struct {
	UserID    int64
	SessionID string
}

// LoginRepository persists everything US-002 touches: the brute-force counter,
// the device registry, and the login sessions themselves.
//
// As in US-001, every write that must not be observed half-done is one method
// wrapping one transaction — CreateLoginSession and RotateSession each do
// several statements that are only correct together.
//
// The seven methods the login spec names are all here. One more,
// FindUserByID, is added: RotateSession has to mint an access token, which
// carries the user's email, and a refresh request arrives knowing only a
// session row's user_id — there is no phone number to look the account up by.
// The spec anticipates this ("cần thêm bước đọc"), so the read is a method
// rather than the session lookup smuggling the email along.
type LoginRepository interface {
	// FindUserByPhone looks up an account by its already-normalized phone
	// number. Returns ErrUserNotFound when there is no users row — including
	// when the number exists only in pending_registration, which the caller
	// must not be able to distinguish from "wrong password" (AC2).
	FindUserByPhone(ctx context.Context, phoneNumber string) (*User, error)

	// FindUserByID loads the account behind userID. Used by the refresh flow to
	// rebuild an access token's email claim. Returns ErrUserNotFound if the
	// account is gone, which the caller treats as an invalid refresh token.
	FindUserByID(ctx context.Context, userID int64) (*User, error)

	// FindLoginAttempt reads the counter row for phoneNumber. Returns (nil, nil)
	// when no row exists — a number nobody has ever failed against is not an
	// error, it is the ordinary first attempt.
	FindLoginAttempt(ctx context.Context, phoneNumber string) (*LoginAttempt, error)

	// RecordLoginFailure counts one wrong guess against phoneNumber and decides
	// whether that count locks the number, in a single atomic statement so two
	// concurrent failures cannot both read the same starting count (AC7).
	//
	// The row is reset to 1 when it is stale — the previous failure was more
	// than resetWindow ago, or a lock has since expired — so "consecutive"
	// means each failure within resetWindow of the one before. lockDuration is
	// measured from this failure, not from the first one.
	//
	// The returned attempt carries the new count and the lock deadline, so the
	// caller can log the lock without a second read. It does not change the
	// answer the caller gives the user: the failure that trips the limit is
	// still a 401, and only the next request is refused.
	RecordLoginFailure(ctx context.Context, phoneNumber string, maxAttempts int, lockDuration, resetWindow time.Duration) (*LoginAttempt, error)

	// CreateLoginSession opens a session for user on device and clears the
	// brute-force counter, in one transaction (AC1/AC3/AC4/AC7).
	//
	// It upserts the device, revokes any previous active session of that same
	// device, mints the token pair through issueTokens against a session ID
	// chosen here, writes the session row, and deletes the login_attempts row.
	// Sessions of the user's other devices, and US-001's device-less sessions,
	// are untouched.
	CreateLoginSession(ctx context.Context, user *User, device DeviceInfo, sessionTTL time.Duration, issueTokens TokenIssuer) (*LoginResult, error)

	// FindSessionByRefreshHash looks up the session a presented refresh token
	// belongs to, matching either its current hash or the one immediately
	// before it. The bool reports which: true means the token was already
	// rotated, which is the reuse signal AC11 acts on.
	//
	// A row that exists but is revoked or past ExpiresAt is reported as
	// ErrRefreshTokenInvalid, the same sentinel a missing row gets — the caller
	// distinguishes only the reuse case, because that one has a side effect.
	FindSessionByRefreshHash(ctx context.Context, refreshTokenHash string) (*Session, bool, error)

	// RotateSession replaces the session's refresh token, in one transaction
	// (AC9/AC10).
	//
	// It re-reads the row FOR UPDATE and refuses if the stored hash is no
	// longer oldHash — so of two concurrent refreshes with the same token
	// exactly one wins and the other gets ErrRefreshTokenInvalid rather than
	// minting a second valid pair. On success it mints through issueTokens,
	// marks the old hash as the previous one, bumps TokenVersion, and pushes
	// ExpiresAt out by sessionTTL (sliding).
	RotateSession(ctx context.Context, sessionRowID int64, oldHash string, sessionTTL time.Duration, issueTokens TokenIssuer) (*RefreshResult, error)

	// RevokeSession revokes one session of one user, identified by both IDs
	// (AC13/AC14). Matching user_id as well as session_id is what stops a
	// caller from revoking someone else's session by guessing a session_id.
	//
	// Idempotent: an already-revoked, expired or unknown session revokes
	// nothing and is not an error, which is what makes logout safe to retry.
	RevokeSession(ctx context.Context, userID int64, sessionID, reason string) error
}
