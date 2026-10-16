// Package usecase holds US-001's registration flow: the business rules that sit
// between HTTP and the database.
//
// The three steps of the story are three methods — RequestOTP, ResendOTP,
// VerifyOTP — and the invariant they exist to protect is that no users row is
// ever created before the email is verified. That is why step one writes only
// to pending_registration and otp_codes, and why the account insert lives
// inside step two's transaction.
//
// The OTP mechanism itself (generate, hash, send policy, verify, attempt
// reservation, lock) lives in OTPService (otp_service.go), which knows nothing
// about registration. RegisterUsecase keeps what is register-specific — the
// validation, the phone/temple checks, the account-exists branch, the pending
// payload, the email wording, and turning a verified email into a user +
// session.
//
// This package imports domain (its ports, inputs and sentinels), config, and
// the reusable pkg/ abstractions it needs. It never imports repository or
// delivery.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"chjpcoj.com.innerlamp/pkg/jwt"
	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/pkg/validation"
	"chjpcoj.com.innerlamp/services/appService/domain"

	"golang.org/x/text/unicode/norm"
)

// userRole is the role claim every self-registered account carries. Not
// deployment-varying: it names a role the schema and the JWT contract both fix.
const userRole = "user"

// RegisterUsecase implements US-001 (onboarding v8.0, email verification).
type RegisterUsecase struct {
	registrations domain.RegistrationRepository
	otpService    *OTPService
	mailer        domain.AuthMailer
	passwords     domain.PasswordHasher
	jwtService    jwt.Service
	logger        logging.Logger

	// sessionTTL is the lifetime of the user_sessions row created at
	// registration. It comes from JWTConfig.RefreshTokenExpiration so the
	// stored expiry and the signed refresh token's own expiry agree.
	sessionTTL time.Duration

	// pendingRegistrationTTL is the lifetime of the captured account payload,
	// from OTPConfig.PendingRegistrationTTL.
	pendingRegistrationTTL time.Duration

	// dharmaNameMaxChars / fullNameMaxChars come from ProfileConfig so
	// registration and the profile edit share one rule.
	dharmaNameMaxChars int
	fullNameMaxChars   int

	// now is the clock; tests in this package replace it.
	now func() time.Time
}

// NewRegisterUsecase wires the use case. Every collaborator is a port, a pkg
// abstraction, or a plain value read from config.
func NewRegisterUsecase(
	registrations domain.RegistrationRepository,
	otpService *OTPService,
	mailer domain.AuthMailer,
	passwords domain.PasswordHasher,
	jwtService jwt.Service,
	logger logging.Logger,
	sessionTTL time.Duration,
	pendingRegistrationTTL time.Duration,
	dharmaNameMaxChars int,
	fullNameMaxChars int,
) *RegisterUsecase {
	return &RegisterUsecase{
		registrations:          registrations,
		otpService:             otpService,
		mailer:                 mailer,
		passwords:              passwords,
		jwtService:             jwtService,
		logger:                 logger,
		sessionTTL:             sessionTTL,
		pendingRegistrationTTL: pendingRegistrationTTL,
		dharmaNameMaxChars:     dharmaNameMaxChars,
		fullNameMaxChars:       fullNameMaxChars,
		now:                    time.Now,
	}
}

// RequestOTP runs step 1 (onboarding design Steps R-2…R-8).
//
// Validation runs in the spec's order — email, phone for its country,
// password, names — before the first query, so a malformed request costs no
// database round-trip and the caller learns about the first bad field (AC3–AC6).
//
// An email that already has an account goes through exactly the same work as
// a new one — password hash, code hash, pending + code write, send policy, one
// mailer call — and gets the same body (AC2); only the email that goes out
// differs. Returned errors are the sentinels the handler maps; anything else is
// wrapped infrastructure failure.
func (u *RegisterUsecase) RequestOTP(ctx context.Context, input domain.RequestRegisterOTPInput) (*domain.RequestOTPResult, error) {
	// Step R-2: validate, no DB.
	email, ok := validation.NormalizeEmail(input.Email)
	if !ok {
		return nil, domain.ErrInvalidEmail
	}
	phoneNumber, ok := validation.NormalizePhoneForCountry(input.PhoneCountry, input.PhoneNumber)
	if !ok {
		return nil, domain.ErrInvalidPhone
	}
	if err := ValidatePassword(input.Password); err != nil {
		return nil, err
	}
	fullName, err := normalizeFullName(input.FullName, u.fullNameMaxChars)
	if err != nil {
		return nil, err
	}
	dharmaName, err := normalizeDharmaName(input.DharmaName, u.dharmaNameMaxChars)
	if err != nil {
		return nil, err
	}

	// Step R-3: phone already on an account (AC7). Runs before the email
	// lookup so this 409 — the one accepted disclosure — reveals nothing about
	// the email.
	taken, err := u.registrations.PhoneTaken(ctx, phoneNumber)
	if err != nil {
		return nil, fmt.Errorf("check phone: %w", err)
	}
	if taken {
		return nil, domain.ErrPhoneTaken
	}

	// Step R-4: temple (AC8). Blank means "Cư sĩ tu tại gia" and costs no query.
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

	// Step R-5: picks the email of Step R-8, nothing else.
	accountExists, err := u.registrations.EmailHasAccount(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}

	// While the email's current code is locked after wrong guesses, a fresh
	// code would hand the guesser a new attempt counter (AC15).
	if err := u.otpService.EnsureNotLocked(ctx, email, domain.OTPPurposeRegister); err != nil {
		return nil, err
	}

	// Step R-6: the same two hashes on both branches.
	passwordHash, err := u.passwords.Hash(input.Password)
	if err != nil {
		return nil, err
	}
	code, otpHash, err := u.otpService.Generate()
	if err != nil {
		return nil, err
	}

	// Step R-7: pending + code, one transaction, send policy first (AC9).
	now := u.now()
	pending := &domain.PendingRegistration{
		Email:              email,
		PhoneNumber:        phoneNumber,
		PasswordHash:       passwordHash,
		FullName:           &fullName,
		DharmaName:         dharmaName,
		OrdinationTempleID: templeID,
		AccountExists:      accountExists,
		ExpiresAt:          now.Add(u.pendingRegistrationTTL),
	}
	if _, err := u.registrations.SavePendingAndIssueOTP(ctx, pending, otpHash, u.otpService.CodeExpiry(now), u.otpService.SendLimits()); err != nil {
		return nil, fmt.Errorf("save pending registration: %w", err)
	}

	// Step R-8: email after commit (AC10 on failure).
	if err := u.sendRegisterEmail(ctx, email, code, accountExists); err != nil {
		return nil, err
	}

	u.logger.Info("registration code requested")

	return &domain.RequestOTPResult{
		ExpiresInSeconds:   u.otpService.ExpiresInSeconds(),
		ResendAfterSeconds: u.otpService.ResendAfterSeconds(),
	}, nil
}

// ResendOTP runs Steps S-2…S-5: a new code for a live pending registration,
// invalidating the previous one, under the same send policy (AC11). The payload
// is not touched; a caller who wants different values starts over with
// request-otp. An account_exists row gets the notice again, so resend reveals
// no more than request-otp did.
func (u *RegisterUsecase) ResendOTP(ctx context.Context, input domain.ResendOTPInput) (*domain.ResendOTPResult, error) {
	email, ok := validation.NormalizeEmail(input.Email)
	if !ok {
		return nil, domain.ErrInvalidEmail
	}

	pending, err := u.findPending(ctx, email)
	if err != nil {
		return nil, err
	}

	if err := u.otpService.EnsureNotLocked(ctx, email, domain.OTPPurposeRegister); err != nil {
		return nil, err
	}

	issued, err := u.otpService.Issue(ctx, email, domain.OTPPurposeRegister)
	if err != nil {
		return nil, err
	}

	if err := u.sendRegisterEmail(ctx, email, issued.Code, pending.AccountExists); err != nil {
		return nil, err
	}

	u.logger.Info("registration code resent")

	return &domain.ResendOTPResult{
		ExpiresInSeconds:   issued.ExpiresInSeconds,
		ResendAfterSeconds: issued.ResendAfterSeconds,
		ResendRemaining:    issued.ResendRemaining,
	}, nil
}

// VerifyOTP runs Steps V-2…V-5: check the code, and only then turn the pending
// payload into an account with a session.
//
// An account_exists row goes through the same verification (the attempt is
// reserved and counted) and is then reported as a wrong code whatever was
// typed (AC16), so no account or session is ever created for that email.
func (u *RegisterUsecase) VerifyOTP(ctx context.Context, input domain.VerifyRegisterOTPInput) (*domain.RegisterResult, error) {
	email, ok := validation.NormalizeEmail(input.Email)
	if !ok {
		return nil, domain.ErrInvalidEmail
	}

	pending, err := u.findPending(ctx, email)
	if err != nil {
		return nil, err
	}

	otp, err := u.otpService.Verify(ctx, email, domain.OTPPurposeRegister, input.OTPCode)
	if err != nil {
		return nil, err
	}
	if pending.AccountExists {
		return nil, domain.ErrOTPInvalid
	}

	result, err := u.registrations.CreateUserAndSession(ctx, pending, otp.ID, u.sessionTTL, u.tokenIssuer(pending))
	if err != nil {
		return nil, err
	}

	u.logger.Info("registration verified", "user_id", result.User.ID)

	return result, nil
}

// findPending loads the pending row and folds "never existed" into "expired"
// (AC12, AC17): both mean "start over from step 1".
func (u *RegisterUsecase) findPending(ctx context.Context, email string) (*domain.PendingRegistration, error) {
	pending, err := u.registrations.FindPendingRegistration(ctx, email)
	switch {
	case err == nil:
		return pending, nil
	case errors.Is(err, domain.ErrPendingRegistrationNotFound), errors.Is(err, domain.ErrPendingRegistrationExpired):
		return nil, domain.ErrPendingRegistrationExpired
	default:
		return nil, fmt.Errorf("find pending registration: %w", err)
	}
}

// sendRegisterEmail sends the code, or the account-exists notice instead
// (AC2). On failure it releases the send slot so the failed send does not
// count, then reports ErrEmailSendFailed (AC10). The mailer's own error is
// kept for the log only.
func (u *RegisterUsecase) sendRegisterEmail(ctx context.Context, email, code string, accountExists bool) error {
	var err error
	if accountExists {
		err = u.mailer.SendAccountExistsNotice(ctx, email)
	} else {
		err = u.mailer.SendRegisterCode(ctx, email, code, u.otpService.TTL())
	}
	if err == nil {
		return nil
	}

	u.logger.Error("failed to send registration email", "error", err)
	if relErr := u.otpService.ReleaseSendLimit(ctx, email, domain.OTPPurposeRegister); relErr != nil {
		u.logger.Error("failed to release otp send limit", "error", relErr)
	}
	if errors.Is(err, domain.ErrRateLimited) {
		// The daily mail budget ran out (login AC33): a 429 with the wait, not
		// a 500 — and the slot released above, so the user's own send counter
		// is not spent on a mail that never left.
		return err
	}
	return fmt.Errorf("%w: %v", domain.ErrEmailSendFailed, err)
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

// normalizeDharmaName trims, normalises to NFC and length-checks an optional
// dharma name. Blank means "no name" and is returned as nil, so it is stored as
// SQL NULL. The length is counted in characters after normalisation, not bytes,
// so a Vietnamese letter with diacritics is one character whether it was typed
// composed or decomposed. Over the limit is ErrInvalidDharmaName, a 400, rather
// than the 500 the VARCHAR(255) column would raise on write.
func normalizeDharmaName(raw string, maxChars int) (*string, error) {
	name := norm.NFC.String(strings.TrimSpace(raw))
	if name == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(name) > maxChars {
		return nil, domain.ErrInvalidDharmaName
	}
	return &name, nil
}

// normalizeFullName trims, normalises to NFC and length-checks the required full
// name. Unlike the dharma name, blank is an error: registration cannot proceed
// without it. Length is counted in characters after normalisation, so a
// Vietnamese name is the same length whether typed composed or decomposed.
func normalizeFullName(raw string, maxChars int) (string, error) {
	name := norm.NFC.String(strings.TrimSpace(raw))
	if name == "" || utf8.RuneCountInString(name) > maxChars {
		return "", domain.ErrInvalidFullName
	}
	return name, nil
}
