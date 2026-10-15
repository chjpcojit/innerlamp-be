// Package http holds US-001's HTTP delivery: request/response DTOs, the
// three route handlers, and the shared error-mapping helper other features
// under delivery/http reuse once they exist.
package http

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// writeDecodeError maps a utils.DecodeJSONBodyStrict failure to the shared
// utils.ApiResponse envelope every handler in the service replies with. None
// of these reach the use case — design.md's Step 1 for every route requires
// DecodeJSONBodyStrict to pass before anything else runs.
//
// Five of the six cases collapse to 400 VALIDATION_ERROR: an empty body, an
// unknown field, malformed JSON, a second JSON object, and the wrong
// Content-Type. Only an oversized body differs — it is the one decode
// failure the client can fix by sending less, so it gets
// 413 PAYLOAD_TOO_LARGE instead, through utils.WriteBusinessError.
// WriteValidationError cannot express it: it hardcodes 400.
func writeDecodeError(w http.ResponseWriter, err error) {
	if de, ok := errors.AsType[*utils.DecodeError](err); ok {
		if errors.Is(de.Err, utils.ErrBodyTooLarge) {
			utils.WriteBusinessError(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", de.Message, nil)
			return
		}

		var details map[string]any
		if de.Field != "" {
			details = map[string]any{"field": de.Field}
		}
		utils.WriteValidationError(w, "VALIDATION_ERROR", de.Message, details)
		return
	}
	utils.WriteValidationError(w, "VALIDATION_ERROR", "invalid request body", nil)
}

// writeValidationError maps a validator.ValidationErrors — the struct-tag
// failures caught before the use case ever runs — to 400 VALIDATION_ERROR,
// one field: failed-tag entry per broken field.
func writeValidationError(w http.ResponseWriter, verrs validator.ValidationErrors) {
	details := make(map[string]any, len(verrs))
	for _, fe := range verrs {
		details[fe.Field()] = fe.Tag()
	}
	utils.WriteValidationError(w, "VALIDATION_ERROR", "request validation failed", details)
}

// writeDomainError maps a domain sentinel or typed error returned by the use
// case to its HTTP status and code, per design.md's per-flow "Failure"
// branches and tasks.md's mapping table, through utils.WriteBusinessError /
// utils.WriteSystemError. Unrecognized errors fall through to
// utils.WriteSystemError's 500 INTERNAL_ERROR with no detail — that is the
// only branch that must not leak an internal cause to the client.
//
// The two typed errors (OTPLockedError, OTPResendTooSoonError) are checked
// with errors.As before the sentinel switch: the use case always returns
// them by pointer, never the bare sentinel, specifically so the deadline/
// retry-after they carry reaches the response.
func writeDomainError(w http.ResponseWriter, err error) {
	if locked, ok := errors.AsType[*domain.OTPLockedError](err); ok {
		utils.WriteBusinessError(w, http.StatusTooManyRequests, "OTP_LOCKED", domain.ErrOTPLocked.Error(),
			map[string]any{"locked_until": locked.LockedUntil})
		return
	}

	if locked, ok := errors.AsType[*domain.LoginLockedError](err); ok {
		utils.WriteBusinessError(w, http.StatusTooManyRequests, "LOGIN_LOCKED", domain.ErrLoginLocked.Error(),
			map[string]any{"locked_until": locked.LockedUntil})
		return
	}

	if tooSoon, ok := errors.AsType[*domain.OTPResendTooSoonError](err); ok {
		utils.WriteBusinessError(w, http.StatusTooManyRequests, "OTP_RESEND_TOO_SOON", domain.ErrOTPResendTooSoon.Error(),
			map[string]any{"retry_after_seconds": tooSoon.RetryAfterSeconds})
		return
	}

	if limited, ok := errors.AsType[*domain.OTPSendLimitedError](err); ok {
		utils.WriteBusinessError(w, http.StatusTooManyRequests, "OTP_SEND_LIMITED", domain.ErrOTPSendLimited.Error(),
			map[string]any{"locked_until": limited.LockedUntil})
		return
	}

	if busy, ok := errors.AsType[*domain.RegistrationInProgressError](err); ok {
		utils.WriteBusinessError(w, http.StatusConflict, "REGISTRATION_IN_PROGRESS", domain.ErrRegistrationInProgress.Error(),
			map[string]any{"retry_after_seconds": busy.RetryAfterSeconds})
		return
	}

	if bad, ok := errors.AsType[*domain.MemorialValidationError](err); ok {
		utils.WriteValidationError(w, "VALIDATION_ERROR", bad.Message, map[string]any{"field": bad.Field})
		return
	}

	if bad, ok := errors.AsType[*domain.GenealogyValidationError](err); ok {
		utils.WriteValidationError(w, "VALIDATION_ERROR", bad.Message, map[string]any{"field": bad.Field})
		return
	}

	if bad, ok := errors.AsType[*domain.LampValidationError](err); ok {
		utils.WriteValidationError(w, "VALIDATION_ERROR", bad.Message, map[string]any{"field": bad.Field})
		return
	}

	if bad, ok := errors.AsType[*domain.ProfileValidationError](err); ok {
		utils.WriteValidationError(w, "VALIDATION_ERROR", bad.Message, map[string]any{"field": bad.Field})
		return
	}

	if bad, ok := errors.AsType[*domain.TempleSuggestionValidationError](err); ok {
		utils.WriteValidationError(w, "VALIDATION_ERROR", bad.Message, map[string]any{"field": bad.Field})
		return
	}

	if errors.Is(err, domain.ErrInvalidAdministrativeUnitQuery) {
		utils.WriteBusinessError(w, http.StatusBadRequest, "INVALID_FILTER", domain.ErrInvalidAdministrativeUnitQuery.Error(),
			map[string]any{"field": "q"})
		return
	}

	if errors.Is(err, domain.ErrInvalidTempleQuery) {
		utils.WriteBusinessError(w, http.StatusBadRequest, "INVALID_FILTER", domain.ErrInvalidTempleQuery.Error(),
			map[string]any{"field": "q"})
		return
	}

	if bad, ok := errors.AsType[*domain.InvalidSutraFilterError](err); ok {
		utils.WriteBusinessError(w, http.StatusBadRequest, "INVALID_FILTER", domain.ErrInvalidSutraFilter.Error(),
			map[string]any{"field": bad.Field})
		return
	}

	switch {
	case errors.Is(err, domain.ErrPhoneTaken):
		utils.WriteBusinessError(w, http.StatusConflict, "PHONE_TAKEN", domain.ErrPhoneTaken.Error(), nil)
	case errors.Is(err, domain.ErrInvalidPhone):
		utils.WriteBusinessError(w, http.StatusBadRequest, "INVALID_PHONE", err.Error(), map[string]any{"field": "phone_number"})
	case errors.Is(err, domain.ErrInvalidEmail):
		utils.WriteBusinessError(w, http.StatusBadRequest, "INVALID_EMAIL", err.Error(), map[string]any{"field": "email"})
	case errors.Is(err, domain.ErrInvalidPassword):
		utils.WriteBusinessError(w, http.StatusBadRequest, "INVALID_PASSWORD", err.Error(), map[string]any{"field": "password"})
	case errors.Is(err, domain.ErrInvalidFullName):
		utils.WriteBusinessError(w, http.StatusBadRequest, "INVALID_FULL_NAME", err.Error(), map[string]any{"field": "full_name"})
	case errors.Is(err, domain.ErrInvalidDharmaName):
		utils.WriteBusinessError(w, http.StatusBadRequest, "INVALID_DHARMA_NAME", err.Error(), map[string]any{"field": "dharma_name"})
	case errors.Is(err, domain.ErrInvalidTemple):
		utils.WriteBusinessError(w, http.StatusBadRequest, "INVALID_TEMPLE", domain.ErrInvalidTemple.Error(), map[string]any{"field": "ordination_temple_id"})
	case errors.Is(err, domain.ErrInvalidAvatar):
		utils.WriteBusinessError(w, http.StatusBadRequest, "INVALID_AVATAR", domain.ErrInvalidAvatar.Error(), nil)
	case errors.Is(err, domain.ErrAvatarUploadNotFound):
		utils.WriteBusinessError(w, http.StatusNotFound, "AVATAR_UPLOAD_NOT_FOUND", domain.ErrAvatarUploadNotFound.Error(), nil)
	case errors.Is(err, domain.ErrInvalidCredentials):
		utils.WriteBusinessError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", err.Error(), nil)
	case errors.Is(err, domain.ErrInvalidDevice):
		utils.WriteBusinessError(w, http.StatusBadRequest, "INVALID_DEVICE", err.Error(), map[string]any{"field": "device"})
	case errors.Is(err, domain.ErrRefreshTokenInvalid):
		utils.WriteBusinessError(w, http.StatusUnauthorized, "REFRESH_TOKEN_INVALID", err.Error(), nil)
	case errors.Is(err, domain.ErrOTPInvalid):
		utils.WriteBusinessError(w, http.StatusBadRequest, "OTP_INVALID", domain.ErrOTPInvalid.Error(), nil)
	case errors.Is(err, domain.ErrOTPExpired):
		utils.WriteBusinessError(w, http.StatusBadRequest, "OTP_EXPIRED", domain.ErrOTPExpired.Error(), nil)
	case errors.Is(err, domain.ErrPendingRegistrationExpired), errors.Is(err, domain.ErrPendingRegistrationNotFound):
		// "Never started" and "too old" are one answer (onboarding AC12/AC17).
		utils.WriteBusinessError(w, http.StatusBadRequest, "PENDING_REGISTRATION_EXPIRED", domain.ErrPendingRegistrationExpired.Error(), nil)
	case errors.Is(err, domain.ErrEmailSendFailed):
		// The sentinel's message only: the wrapped mailer error is for the log.
		utils.WriteSystemError(w, "EMAIL_SEND_FAILED", domain.ErrEmailSendFailed.Error(), nil)
	case errors.Is(err, domain.ErrAttestationChallengeInvalid):
		utils.WriteBusinessError(w, http.StatusBadRequest, "ATTESTATION_CHALLENGE_INVALID", err.Error(), nil)
	case errors.Is(err, domain.ErrAttestationUpstreamError):
		utils.WriteBusinessError(w, http.StatusBadGateway, "ATTESTATION_UPSTREAM_ERROR", err.Error(), nil)
	case errors.Is(err, domain.ErrTempleNotFound):
		utils.WriteBusinessError(w, http.StatusNotFound, "TEMPLE_NOT_FOUND", domain.ErrTempleNotFound.Error(), nil)
	case errors.Is(err, domain.ErrAdministrativeUnitNotFound):
		utils.WriteBusinessError(w, http.StatusNotFound, "ADMINISTRATIVE_UNIT_NOT_FOUND", domain.ErrAdministrativeUnitNotFound.Error(), nil)
	case errors.Is(err, domain.ErrDeityNotFound):
		utils.WriteBusinessError(w, http.StatusNotFound, "DEITY_NOT_FOUND", domain.ErrDeityNotFound.Error(), nil)
	case errors.Is(err, domain.ErrGalleryImageNotFound):
		utils.WriteBusinessError(w, http.StatusNotFound, "GALLERY_IMAGE_NOT_FOUND", domain.ErrGalleryImageNotFound.Error(), nil)
	case errors.Is(err, domain.ErrMemorialNotFound):
		utils.WriteBusinessError(w, http.StatusNotFound, "MEMORIAL_NOT_FOUND", domain.ErrMemorialNotFound.Error(), nil)
	case errors.Is(err, domain.ErrMemorialLimitReached):
		utils.WriteBusinessError(w, http.StatusConflict, "MEMORIAL_LIMIT_REACHED", domain.ErrMemorialLimitReached.Error(), nil)
	case errors.Is(err, domain.ErrGenealogyNotFound):
		utils.WriteBusinessError(w, http.StatusNotFound, "GENEALOGY_NOT_FOUND", domain.ErrGenealogyNotFound.Error(), nil)
	case errors.Is(err, domain.ErrGenealogyForbidden):
		utils.WriteBusinessError(w, http.StatusForbidden, "GENEALOGY_FORBIDDEN", domain.ErrGenealogyForbidden.Error(), nil)
	case errors.Is(err, domain.ErrGenealogyLimitReached):
		utils.WriteBusinessError(w, http.StatusConflict, "GENEALOGY_LIMIT_REACHED", domain.ErrGenealogyLimitReached.Error(), nil)
	case errors.Is(err, domain.ErrMemberNotFound):
		utils.WriteBusinessError(w, http.StatusNotFound, "MEMBER_NOT_FOUND", domain.ErrMemberNotFound.Error(), nil)
	case errors.Is(err, domain.ErrMemberLimitReached):
		utils.WriteBusinessError(w, http.StatusConflict, "MEMBER_LIMIT_REACHED", domain.ErrMemberLimitReached.Error(), nil)
	case errors.Is(err, domain.ErrMemberHasDescendants):
		utils.WriteBusinessError(w, http.StatusConflict, "MEMBER_HAS_DESCENDANTS", domain.ErrMemberHasDescendants.Error(), nil)
	case errors.Is(err, domain.ErrFounderNotDeletable):
		utils.WriteBusinessError(w, http.StatusConflict, "FOUNDER_NOT_DELETABLE", domain.ErrFounderNotDeletable.Error(), nil)
	case errors.Is(err, domain.ErrFounderNotMovable):
		utils.WriteBusinessError(w, http.StatusConflict, "FOUNDER_NOT_MOVABLE", domain.ErrFounderNotMovable.Error(), nil)
	case errors.Is(err, domain.ErrMemberPhoneInUse):
		utils.WriteBusinessError(w, http.StatusConflict, "MEMBER_PHONE_ALREADY_USED", domain.ErrMemberPhoneInUse.Error(), map[string]any{"field": "phone_number"})
	case errors.Is(err, domain.ErrViewerExists):
		utils.WriteBusinessError(w, http.StatusConflict, "VIEWER_ALREADY_EXISTS", domain.ErrViewerExists.Error(), nil)
	case errors.Is(err, domain.ErrViewerLimitReached):
		utils.WriteBusinessError(w, http.StatusConflict, "VIEWER_LIMIT_REACHED", domain.ErrViewerLimitReached.Error(), nil)
	case errors.Is(err, domain.ErrViewerNotFound):
		utils.WriteBusinessError(w, http.StatusNotFound, "VIEWER_NOT_FOUND", domain.ErrViewerNotFound.Error(), nil)
	case errors.Is(err, domain.ErrInvitationNotFound):
		utils.WriteBusinessError(w, http.StatusNotFound, "INVITATION_NOT_FOUND", domain.ErrInvitationNotFound.Error(), nil)
	case errors.Is(err, domain.ErrInvitationNotPending):
		utils.WriteBusinessError(w, http.StatusConflict, "INVITATION_NOT_PENDING", domain.ErrInvitationNotPending.Error(), nil)
	case errors.Is(err, domain.ErrInvitationAlreadyPending):
		utils.WriteBusinessError(w, http.StatusConflict, "INVITATION_ALREADY_PENDING", domain.ErrInvitationAlreadyPending.Error(), nil)
	case errors.Is(err, domain.ErrLampLimitReached):
		utils.WriteBusinessError(w, http.StatusConflict, "LAMP_LIMIT_REACHED", domain.ErrLampLimitReached.Error(), nil)
	case errors.Is(err, domain.ErrInvalidLampFilter):
		utils.WriteBusinessError(w, http.StatusBadRequest, "INVALID_FILTER", domain.ErrInvalidLampFilter.Error(), map[string]any{"field": "lamp_type"})
	case errors.Is(err, domain.ErrInvalidLimit):
		utils.WriteBusinessError(w, http.StatusBadRequest, "INVALID_LIMIT", domain.ErrInvalidLimit.Error(), map[string]any{"field": "limit"})
	case errors.Is(err, domain.ErrInvalidCursor):
		utils.WriteBusinessError(w, http.StatusBadRequest, "INVALID_CURSOR", domain.ErrInvalidCursor.Error(), map[string]any{"field": "cursor"})
	case errors.Is(err, domain.ErrArticleNotFound):
		utils.WriteBusinessError(w, http.StatusNotFound, "ARTICLE_NOT_FOUND", domain.ErrArticleNotFound.Error(), nil)
	case errors.Is(err, domain.ErrDharmaTalkNotFound):
		utils.WriteBusinessError(w, http.StatusNotFound, "DHARMA_TALK_NOT_FOUND", domain.ErrDharmaTalkNotFound.Error(), nil)
	case errors.Is(err, domain.ErrSutraNotFound):
		utils.WriteBusinessError(w, http.StatusNotFound, "SUTRA_NOT_FOUND", domain.ErrSutraNotFound.Error(), nil)
	case errors.Is(err, domain.ErrSutraTranslationNotFound):
		utils.WriteBusinessError(w, http.StatusNotFound, "TRANSLATION_NOT_FOUND", domain.ErrSutraTranslationNotFound.Error(), nil)
	case errors.Is(err, domain.ErrInvalidPageSize):
		utils.WriteBusinessError(w, http.StatusBadRequest, "INVALID_PAGE_SIZE", domain.ErrInvalidPageSize.Error(), map[string]any{"field": "page_size"})
	case errors.Is(err, domain.ErrInvalidTranslation):
		utils.WriteBusinessError(w, http.StatusBadRequest, "INVALID_TRANSLATION", domain.ErrInvalidTranslation.Error(), map[string]any{"field": "translation"})
	case errors.Is(err, domain.ErrInvalidDeityType):
		utils.WriteBusinessError(w, http.StatusBadRequest, "INVALID_DEITY_TYPE", domain.ErrInvalidDeityType.Error(), map[string]any{"field": "type"})
	case errors.Is(err, domain.ErrInvalidGalleryType):
		utils.WriteBusinessError(w, http.StatusBadRequest, "INVALID_GALLERY_TYPE", domain.ErrInvalidGalleryType.Error(), map[string]any{"field": "type"})
	case errors.Is(err, domain.ErrInvalidPage):
		utils.WriteBusinessError(w, http.StatusBadRequest, "INVALID_PAGE", domain.ErrInvalidPage.Error(), map[string]any{"field": "page"})
	case errors.Is(err, domain.ErrInvalidDate):
		// from/to of the temple-event window name themselves; calendar's single "date" is the default.
		field := "date"
		if fe, ok := errors.AsType[*domain.InvalidDateFieldError](err); ok {
			field = fe.Field
		}
		utils.WriteBusinessError(w, http.StatusBadRequest, "INVALID_DATE", domain.ErrInvalidDate.Error(), map[string]any{"field": field})
	case errors.Is(err, domain.ErrInvalidDateRange):
		utils.WriteBusinessError(w, http.StatusBadRequest, "INVALID_DATE_RANGE", domain.ErrInvalidDateRange.Error(), nil)
	case errors.Is(err, domain.ErrDateRangeTooLarge):
		utils.WriteBusinessError(w, http.StatusBadRequest, "DATE_RANGE_TOO_LARGE", domain.ErrDateRangeTooLarge.Error(), nil)
	case errors.Is(err, domain.ErrInvalidYear):
		utils.WriteBusinessError(w, http.StatusBadRequest, "INVALID_YEAR", domain.ErrInvalidYear.Error(), map[string]any{"field": "year"})
	default:
		utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)
	}
}

// WriteAuthError (exported so cmd/main.go can hand it to the middleware) is the auth middleware's ErrorHandler and the handler's own
// answer to a token that lacks what it needs. The middleware's default is a flat
// {"error":"Unauthorized"}, which would make logout the one route whose 401 does
// not share the ApiResponse envelope every other route replies with.
//
// The cause is deliberately not echoed: a client that sent a bad token learns
// only that it is unauthorized, not whether it was expired, forged or the wrong
// type.
func WriteAuthError(w http.ResponseWriter, _ *http.Request, _ error) {
	utils.WriteBusinessError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
}
