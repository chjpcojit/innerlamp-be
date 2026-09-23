// Package usecase holds US-001's registration flow: the business rules that sit
// between HTTP and the database.
//
// The three steps of the story are three methods — RequestOTP, ResendOTP,
// VerifyOTP — and the invariant they exist to protect is that no users row is
// ever created before the phone number is verified. That is why step one writes
// only to pending_registration and otp_codes, and why the account insert lives
// inside step two's transaction: there is no window in which a half-registered
// account can be observed.
//
// This package imports domain (its ports, inputs and sentinels) plus the
// reusable pkg/ abstractions it needs — jwt for signing, logging for
// observability, validation for shape checks. It never imports repository or
// delivery: the concrete Postgres repository is injected as a
// domain.RegistrationRepository, and the HTTP shape of an error is decided one
// layer further out.
//
// It also names config.OTPConfig, for the OTP policy's shape only. Every value
// still arrives as a constructor parameter — the use case never reads a
// package-level config singleton, which is the property that matters.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"chjpcoj.com.innerlamp/pkg/jwt"
	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/pkg/validation"
	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"

	"golang.org/x/crypto/bcrypt"
)

const (
	// userRole is the role claim every self-registered account carries. Not
	// deployment-varying: it names a role the schema and the JWT contract both
	// fix, so changing it is a migration or an API version, not a config edit.
	userRole = "user"

	// minPasswordLength is AC7's floor. Counted in bytes, not runes: the rule
	// is about how much an attacker has to guess, and a multi-byte character
	// costs the guesser more than one byte, so counting runes would let a
	// shorter secret through.
	minPasswordLength = 8
)

// RegisterUsecase implements US-001.
type RegisterUsecase struct {
	registrations domain.RegistrationRepository
	otpSender     domain.OTPSender
	jwtService    jwt.Service
	logger        logging.Logger

	// sessionTTL is the lifetime of the user_sessions row created at
	// registration. It comes from JWTConfig.RefreshTokenExpiration rather than
	// a constant here so the stored expiry and the signed refresh token's own
	// expiry are the same number by construction — two sources would eventually
	// disagree, and the row would outlive or underlive its token.
	sessionTTL time.Duration

	// otpPolicy is the tunable policy behind every code this use case issues:
	// TTL, resend cooldown and cap, attempt limit, lock duration and the pending
	// payload's lifetime.
	//
	// The type is config's rather than a local mirror. These six numbers used to
	// be a const block here, justified as "business decisions, not
	// configuration"; they are not — each is a number a dev environment shortens
	// to exercise expiry and production lengthens under load. A local struct
	// would then have to be filled in by hand at the composition root, and a
	// field missed there decodes to a zero value: an OTP TTL of 0 is a code that
	// is expired the instant it is issued, which is a silent failure. Naming the
	// config type directly means there is no second definition and no mapping to
	// get wrong.
	//
	// Named otpPolicy rather than otp because every method here also has a local
	// *domain.OTPCode called otp — the two would shadow, and a reader would have
	// to check which one a bare `otp` meant.
	otpPolicy config.OTPConfig
}

// NewRegisterUsecase wires the use case. Every collaborator is a port, a pkg
// abstraction, or a plain value read from config; no concrete repository, sender
// or signing implementation is visible here.
//
// sessionTTL and otpPolicy both come from services/appService/config/ and are
// passed through rather than declared here — a use case that reaches for a
// package-level config singleton has broken dependency injection, and a TTL
// baked in as a const cannot be shortened for a dev environment.
func NewRegisterUsecase(
	registrations domain.RegistrationRepository,
	otpSender domain.OTPSender,
	jwtService jwt.Service,
	logger logging.Logger,
	sessionTTL time.Duration,
	otpPolicy config.OTPConfig,
) *RegisterUsecase {
	return &RegisterUsecase{
		registrations: registrations,
		otpSender:     otpSender,
		jwtService:    jwtService,
		logger:        logger,
		sessionTTL:    sessionTTL,
		otpPolicy:     otpPolicy,
	}
}

// generateOTP returns the code sent to the user.
//
// It is a constant today (US-001 technical notes: 111111 in dev) and that is a
// deliberate placeholder, not an oversight — isolated in this one function so
// that switching to a cryptographically random six-digit code touches nothing
// else. Every caller treats the return value as opaque.
func generateOTP() string {
	return "111111"
}

// RequestOTP runs step 1: validate the payload, make sure neither unique key is
// claimed, apply the per-phone send policy, stash the account payload, and send
// a code.
//
// No users row is written here — that is the whole point of the two-step design.
// The failure modes this method can return are limited to the sentinels the
// handler knows how to map: ErrInvalidEmail, ErrInvalidPhone, ErrInvalidPassword,
// ErrPhoneTaken, ErrEmailTaken, ErrOTPLocked (wrong-guess lock),
// ErrOTPResendTooSoon, ErrOTPSendLimited, ErrSMSSendFailed. Anything else is an
// infrastructure failure and is returned wrapped, so the handler's 500 path is
// reached rather than a wrong 4xx.
func (u *RegisterUsecase) RequestOTP(ctx context.Context, input domain.RequestRegisterOTPInput) (*domain.RequestOTPResult, error) {
	email := validation.NormalizeEmail(input.Email)
	phoneNumber := validation.NormalizePhone(input.PhoneNumber)

	// Validation runs on the normalized values, so what is checked is exactly
	// what would be stored, and it runs before the first query: a malformed
	// request costs no database round-trip.
	if !validation.IsValidEmail(email) {
		return nil, domain.ErrInvalidEmail
	}
	if phoneNumber == "" {
		return nil, domain.ErrInvalidPhone
	}
	if len(input.Password) < minPasswordLength {
		return nil, domain.ErrInvalidPassword
	}

	conflicts, err := u.registrations.FindConflicts(ctx, phoneNumber, email)
	if err != nil {
		return nil, fmt.Errorf("find registration conflicts: %w", err)
	}
	// Phone first: it is the account's primary identity, so when both are taken
	// the caller is told about the phone and can decide what to do about it.
	if conflicts.HasPhoneConflict() {
		return nil, domain.ErrPhoneTaken
	}
	if conflicts.HasEmailConflict() {
		return nil, domain.ErrEmailTaken
	}

	// AC12 has no exception for this endpoint: while a number is locked out
	// after too many wrong codes, asking for a fresh code would hand the
	// attacker a new attempt counter. The send-limit policy (AC5) is a separate,
	// weaker lock and is applied inside SavePendingAndIssueOTP in the same
	// transaction as the OTP write — it must NOT block verifying the last code,
	// so it cannot live on the OTP row read here.
	otp, err := u.registrations.FindActiveOTP(ctx, phoneNumber, domain.OTPPurposeRegister)
	if err != nil && !errors.Is(err, domain.ErrOTPInvalid) {
		return nil, fmt.Errorf("find active otp: %w", err)
	}
	if otp != nil && isLocked(otp, time.Now()) {
		return nil, &domain.OTPLockedError{LockedUntil: *otp.LockedUntil}
	}

	// The hash, not the password, is what leaves this function. bcrypt's cost
	// factor is what makes a stolen pending_registration row expensive to
	// reverse, and the plaintext now exists only in this stack frame.
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now()
	pending := &domain.PendingRegistration{
		PhoneNumber:     phoneNumber,
		Email:           email,
		PasswordHash:    string(passwordHash),
		DharmaName:      optionalString(input.DharmaName),
		OrdinationPlace: optionalString(input.OrdinationPlace),
		ExpiresAt:       now.Add(u.otpPolicy.PendingRegistrationTTL.Duration()),
	}

	otpHash, err := bcrypt.GenerateFromPassword([]byte(generateOTP()), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash otp: %w", err)
	}

	if _, err := u.registrations.SavePendingAndIssueOTP(ctx, pending, string(otpHash), now.Add(u.otpPolicy.TTL.Duration()), u.sendLimits()); err != nil {
		return nil, wrapSendError("save pending registration", err)
	}

	// Sent only after the transaction committed. An OTP the user is holding
	// while its row rolled back is a code nothing can verify.
	if err := u.sendOTP(ctx, phoneNumber); err != nil {
		return nil, err
	}

	u.logger.Info("registration otp requested", "phone_number", phoneNumber)

	return &domain.RequestOTPResult{
		ExpiresInSeconds:   int(u.otpPolicy.TTL.Duration().Seconds()),
		ResendAfterSeconds: int(u.otpPolicy.SendCooldown.Duration().Seconds()),
	}, nil
}

// ResendOTP reissues a code for an attempt already in flight.
//
// Only the register purpose is implemented. The endpoint is shared across flows
// (the request body names its purpose), so anything else is rejected up front
// rather than issuing a code no flow in this service can consume.
//
// Note what a resend may not do: change the account payload. The email and
// password were captured at step 1; a caller who wants different values starts
// over (request-otp overwrites the pending row). The payload itself is left
// untouched, so its own TTL — not the code's 90 seconds — bounds the attempt.
func (u *RegisterUsecase) ResendOTP(ctx context.Context, input domain.ResendOTPInput) (*domain.ResendOTPResult, error) {
	if input.Purpose != domain.OTPPurposeRegister {
		return nil, domain.ErrUnsupportedOTPPurpose
	}

	phoneNumber := validation.NormalizePhone(input.PhoneNumber)
	if phoneNumber == "" {
		return nil, domain.ErrInvalidPhone
	}

	// Both are surfaced as-is: "not found" and "expired" already describe a dead
	// attempt, and delivery maps them to the same non-revealing response as a
	// wrong code — a distinguishable error here would tell a stranger which
	// numbers are mid-registration.
	if _, err := u.registrations.FindPendingRegistration(ctx, phoneNumber); err != nil {
		return nil, err
	}

	// An expired or already-consumed code is not a reason to refuse a resend:
	// the pending payload is the precondition, and issuing a fresh code is
	// exactly what resend is for. This is what lets a caller resume after the
	// 30-minute send lock (their last code is long expired) instead of being
	// told to start over (v5.1). ErrOTPInvalid therefore means "no row to
	// lock-check", not "dead attempt".
	otp, err := u.registrations.FindActiveOTP(ctx, phoneNumber, input.Purpose)
	if err != nil && !errors.Is(err, domain.ErrOTPInvalid) {
		return nil, fmt.Errorf("find active otp: %w", err)
	}

	// The wrong-guess lock (AC12) is checked here because it lives on the OTP
	// row. The send policy (AC5) — cooldown, cap, send lock — is applied by
	// ReissueOTP in its own transaction, so a blocked send writes nothing and
	// the two locks stay independent: this one blocks verify too, the send one
	// must not.
	if otp != nil && isLocked(otp, time.Now()) {
		return nil, &domain.OTPLockedError{LockedUntil: *otp.LockedUntil}
	}

	otpHash, err := bcrypt.GenerateFromPassword([]byte(generateOTP()), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash otp: %w", err)
	}

	state, err := u.registrations.ReissueOTP(ctx, phoneNumber, input.Purpose, string(otpHash), time.Now().Add(u.otpPolicy.TTL.Duration()), u.sendLimits())
	if err != nil {
		return nil, wrapSendError("reissue otp", err)
	}

	if err := u.sendOTP(ctx, phoneNumber); err != nil {
		return nil, err
	}

	u.logger.Info("registration otp resent", "phone_number", phoneNumber)

	return &domain.ResendOTPResult{
		ExpiresInSeconds:   int(u.otpPolicy.TTL.Duration().Seconds()),
		ResendAfterSeconds: int(u.otpPolicy.SendCooldown.Duration().Seconds()),
		ResendRemaining:    state.Remaining,
	}, nil
}

// VerifyOTP runs step 2: check the code, and only then turn the pending payload
// into an account with a session.
//
// Every branch before the account insert is a dead end — wrong code, expired
// code, locked number, missing payload all leave users untouched. The insert
// itself re-checks email uniqueness inside its own transaction, because two
// people can hold pending registrations for the same unverified address and
// only a completed account is authoritative for that claim.
func (u *RegisterUsecase) VerifyOTP(ctx context.Context, input domain.VerifyRegisterOTPInput) (*domain.RegisterResult, error) {
	phoneNumber := validation.NormalizePhone(input.PhoneNumber)
	if phoneNumber == "" {
		return nil, domain.ErrInvalidPhone
	}

	pending, err := u.registrations.FindPendingRegistration(ctx, phoneNumber)
	if err != nil {
		return nil, err
	}

	otp, err := u.registrations.FindActiveOTP(ctx, phoneNumber, domain.OTPPurposeRegister)
	if err != nil {
		if errors.Is(err, domain.ErrOTPInvalid) {
			return nil, domain.ErrOTPExpired
		}
		return nil, fmt.Errorf("find active otp: %w", err)
	}

	// Checked before the comparison: while the lock holds, the code is not
	// examined and no attempt is recorded, so an attacker can neither extend
	// the lock by guessing nor keep trying for free.
	now := time.Now()
	if isLocked(otp, now) {
		return nil, &domain.OTPLockedError{LockedUntil: *otp.LockedUntil}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(otp.OTPHash), []byte(input.OTPCode)); err != nil {
		return nil, u.recordFailedAttempt(ctx, otp)
	}

	result, err := u.registrations.CreateUserAndSession(
		ctx,
		pending,
		otp.ID,
		u.sessionTTL,
		u.tokenIssuer(pending),
	)
	if err != nil {
		return nil, err
	}

	u.logger.Info("registration verified", "phone_number", phoneNumber, "user_id", result.User.ID)

	return result, nil
}

// recordFailedAttempt counts a wrong code and locks the number once the limit
// is reached, then reports the failure.
//
// The returned error is always ErrOTPInvalid and never says how many tries
// remain: that number is exactly what someone brute-forcing a 10^6 keyspace
// would use to pace themselves. Once the count crosses MaxAttempts the lock is
// set for LockDuration from this failure, not from the first one.
func (u *RegisterUsecase) recordFailedAttempt(ctx context.Context, otp *domain.OTPCode) error {
	attempts, err := u.registrations.IncrementOTPAttempts(ctx, otp.ID)
	if err != nil {
		// The guess was wrong either way, so the caller still gets the same
		// answer; the failure to record it is what an operator needs to see.
		u.logger.Error("failed to record otp attempt", "error", err, "otp_id", otp.ID)
		return domain.ErrOTPInvalid
	}

	if attempts >= u.otpPolicy.MaxAttempts {
		if err := u.registrations.LockOTP(ctx, otp.ID, time.Now().Add(u.otpPolicy.LockDuration.Duration())); err != nil {
			u.logger.Error("failed to lock otp", "error", err, "otp_id", otp.ID)
		}
	}

	return domain.ErrOTPInvalid
}

// tokenIssuer builds the closure the repository calls inside its transaction,
// after the account insert has produced a user ID and before the session row
// needs a token.
//
// This is a callback rather than two round-trips because signing after the
// transaction would leave a committed account with no usable session whenever
// the second step failed. The repository calls it without importing pkg/jwt —
// it sees only the domain.TokenIssuer function type.
func (u *RegisterUsecase) tokenIssuer(pending *domain.PendingRegistration) domain.TokenIssuer {
	return func(userID, sessionID string) (string, string, error) {
		// "sid" ties the access token to its user_sessions row so logout can
		// revoke exactly that session. The refresh token needs no such claim —
		// it is looked up by hash.
		accessToken, err := u.jwtService.GenerateAccessToken(userID, pending.Email, userRole, map[string]any{"sid": sessionID})
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

// sendOTP delivers a code, translating any sender failure into ErrSMSSendFailed.
//
// The failure is reported rather than swallowed even though the OTP row is
// already committed: a registration whose code never arrived is not a success,
// and answering 200 would leave the caller waiting for an SMS that is not
// coming. The wrapped sentinel is what lets the handler pick 500 SMS_SEND_FAILED
// without matching on message text; the cause is kept for the log only.
func (u *RegisterUsecase) sendOTP(ctx context.Context, phoneNumber string) error {
	if err := u.otpSender.SendOTP(ctx, phoneNumber, generateOTP(), domain.OTPPurposeRegister); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrSMSSendFailed, err)
	}
	return nil
}

// sendLimits projects the OTP policy onto the shape the repository applies.
// Kept as one method so the three send-policy numbers can never be threaded
// inconsistently (e.g. a cap paired with the wrong lock duration).
func (u *RegisterUsecase) sendLimits() domain.SendLimits {
	return domain.SendLimits{
		Cooldown:     u.otpPolicy.SendCooldown.Duration(),
		LockDuration: u.otpPolicy.SendLockDuration.Duration(),
		MaxAttempts:  u.otpPolicy.MaxSendAttempts,
	}
}

// wrapSendError passes the send policy's typed refusals through unchanged and
// wraps everything else as infrastructure. The typed errors are business
// outcomes the handler maps to 429 with locked_until / retry_after_seconds;
// folding them into a generic message would make the handler answer 500 for a
// request that was merely too soon.
func wrapSendError(op string, err error) error {
	if _, ok := errors.AsType[*domain.OTPResendTooSoonError](err); ok {
		return err
	}
	if _, ok := errors.AsType[*domain.OTPSendLimitedError](err); ok {
		return err
	}
	return fmt.Errorf("%s: %w", op, err)
}

// isLocked reports whether the row's lock is still in force. A nil deadline
// means the row was never locked.
func isLocked(otp *domain.OTPCode, now time.Time) bool {
	return otp.LockedUntil != nil && otp.LockedUntil.After(now)
}

// optionalString trims a free-text profile field and turns "nothing" into a nil
// pointer, so a blank dharma name is stored as SQL NULL rather than as an empty
// string that would read back as a value the user never entered.
func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
