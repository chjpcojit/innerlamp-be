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
// The OTP mechanism itself (generate, hash, send, verify, count wrong guesses,
// lock) lives in OTPService (otp_service.go), which knows nothing about the
// register flow. RegisterUsecase keeps only what is register-specific — the
// validation, the conflict check, the pending payload, and turning a verified
// phone into a user + session — and calls OTPService for the code parts.
//
// This package imports domain (its ports, inputs and sentinels) plus the
// reusable pkg/ abstractions it needs — jwt for signing, logging for
// observability, validation for shape checks. It never imports repository or
// delivery: the concrete Postgres repository is injected as a
// domain.RegistrationRepository, and the HTTP shape of an error is decided one
// layer further out.
package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"chjpcoj.com.innerlamp/pkg/jwt"
	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/pkg/validation"
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
	otpService    *OTPService
	jwtService    jwt.Service
	logger        logging.Logger

	// sessionTTL is the lifetime of the user_sessions row created at
	// registration. It comes from JWTConfig.RefreshTokenExpiration rather than
	// a constant here so the stored expiry and the signed refresh token's own
	// expiry are the same number by construction — two sources would eventually
	// disagree, and the row would outlive or underlive its token.
	sessionTTL time.Duration

	// pendingRegistrationTTL is the lifetime of the captured account payload
	// (PendingRegistration). It is a register-specific concern — the pending
	// payload, not the OTP code — so it stays here while the OTP policy (TTL,
	// cooldown, caps, lockout) moves to OTPService. It comes from
	// OTPConfig.PendingRegistrationTTL, the same way sessionTTL comes from
	// JWTConfig.RefreshTokenExpiration.
	pendingRegistrationTTL time.Duration
}

// NewRegisterUsecase wires the use case. Every collaborator is a port, a pkg
// abstraction, or a plain value read from config; no concrete repository, sender
// or signing implementation is visible here.
//
// The shared OTP mechanism arrives as a constructed *OTPService (built in the
// composition root from repository.NewOTPRepository + the SMS sender + the OTP
// policy). sessionTTL and pendingRegistrationTTL are plain values read from
// config and passed through rather than declared here — a use case that reaches
// for a package-level config singleton has broken dependency injection, and a
// TTL baked in as a const cannot be shortened for a dev environment.
func NewRegisterUsecase(
	registrations domain.RegistrationRepository,
	otpService *OTPService,
	jwtService jwt.Service,
	logger logging.Logger,
	sessionTTL time.Duration,
	pendingRegistrationTTL time.Duration,
) *RegisterUsecase {
	return &RegisterUsecase{
		registrations:          registrations,
		otpService:             otpService,
		jwtService:             jwtService,
		logger:                 logger,
		sessionTTL:             sessionTTL,
		pendingRegistrationTTL: pendingRegistrationTTL,
	}
}

// RequestOTP runs step 1: validate the payload, make sure neither unique key is
// claimed, apply the per-phone send policy, stash the account payload, and send
// a code.
//
// No users row is written here — that is the whole point of the two-step design.
// The failure modes this method can return are limited to the sentinels the
// handler knows how to map: ErrInvalidEmail, ErrInvalidPhone, ErrInvalidPassword,
// ErrPhoneTaken, ErrEmailTaken, ErrInvalidTemple, ErrOTPLocked (wrong-guess lock),
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

	// The temple is checked after the identity conflicts (AC18): phone and
	// email are what the caller most needs to be told about first. A blank id
	// means "no temple" and costs no query. The FK on users/pending_registration
	// is the final guard; this is only the friendly early error.
	templeID := optionalString(input.OrdinationTempleID)
	if templeID != nil {
		exists, err := u.registrations.TempleExists(ctx, *templeID)
		if err != nil {
			return nil, fmt.Errorf("check temple: %w", err)
		}
		if !exists {
			return nil, domain.ErrInvalidTemple
		}
	}

	// AC12 has no exception for this endpoint: while a number is locked out
	// after too many wrong codes, asking for a fresh code would hand the
	// attacker a new attempt counter. The send-limit policy (AC5) is a separate,
	// weaker lock and is applied inside SavePendingAndIssueOTP in the same
	// transaction as the OTP write — it must NOT block verifying the last code,
	// so it cannot live on the OTP row read here.
	if err := u.otpService.EnsureNotLocked(ctx, phoneNumber, domain.OTPPurposeRegister); err != nil {
		return nil, err
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
		PhoneNumber:        phoneNumber,
		Email:              email,
		PasswordHash:       string(passwordHash),
		DharmaName:         optionalString(input.DharmaName),
		OrdinationTempleID: templeID,
		ExpiresAt:          now.Add(u.pendingRegistrationTTL),
	}

	code, otpHash, err := u.otpService.Generate()
	if err != nil {
		return nil, err
	}

	if _, err := u.registrations.SavePendingAndIssueOTP(ctx, pending, otpHash, u.otpService.CodeExpiry(now), u.otpService.SendLimits()); err != nil {
		return nil, wrapSendError("save pending registration", err)
	}

	// Sent only after the transaction committed. An OTP the user is holding
	// while its row rolled back is a code nothing can verify.
	if err := u.otpService.Send(ctx, phoneNumber, code, domain.OTPPurposeRegister); err != nil {
		return nil, err
	}

	u.logger.Info("registration otp requested", "phone_number", phoneNumber)

	return &domain.RequestOTPResult{
		ExpiresInSeconds:   u.otpService.ExpiresInSeconds(),
		ResendAfterSeconds: u.otpService.ResendAfterSeconds(),
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

	// The wrong-guess lock (AC12) is checked here because it lives on the OTP
	// row. The send policy (AC5) — cooldown, cap, send lock — is applied by
	// IssueOTP in its own transaction, so a blocked send writes nothing and
	// the two locks stay independent: this one blocks verify too, the send one
	// must not.
	if err := u.otpService.EnsureNotLocked(ctx, phoneNumber, input.Purpose); err != nil {
		return nil, err
	}

	result, err := u.otpService.Issue(ctx, phoneNumber, input.Purpose)
	if err != nil {
		return nil, err
	}

	u.logger.Info("registration otp resent", "phone_number", phoneNumber)

	return &domain.ResendOTPResult{
		ExpiresInSeconds:   result.ExpiresInSeconds,
		ResendAfterSeconds: result.ResendAfterSeconds,
		ResendRemaining:    result.ResendRemaining,
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
//
// The code check runs inside the shared OTPService.Verify; on success it returns
// the verified OTP row, and register's own completion step — CreateUserAndSession
// — consumes it (marking the row used) while creating the account.
func (u *RegisterUsecase) VerifyOTP(ctx context.Context, input domain.VerifyRegisterOTPInput) (*domain.RegisterResult, error) {
	phoneNumber := validation.NormalizePhone(input.PhoneNumber)
	if phoneNumber == "" {
		return nil, domain.ErrInvalidPhone
	}

	pending, err := u.registrations.FindPendingRegistration(ctx, phoneNumber)
	if err != nil {
		return nil, err
	}

	otp, err := u.otpService.Verify(ctx, phoneNumber, domain.OTPPurposeRegister, input.OTPCode)
	if err != nil {
		return nil, err
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
