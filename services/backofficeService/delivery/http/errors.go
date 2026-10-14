package http

import (
	"errors"
	"fmt"
	"net/http"
	"slices"

	jwtcontext "chjpcoj.com.innerlamp/pkg/jwt/context"
	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/backofficeService/domain"
)

// writeDecodeError maps a utils.DecodeJSONBodyStrict failure to the shared
// envelope: 413 for an oversized body, 400 VALIDATION_ERROR for everything else.
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

// writeDomainError maps a use-case error to its HTTP status. Unrecognized
// errors become 500 INTERNAL_ERROR without leaking the cause.
func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		utils.WriteBusinessError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", err.Error(), nil)
	case errors.Is(err, domain.ErrInvalidAccountInput):
		utils.WriteValidationError(w, "VALIDATION_ERROR", err.Error(), nil)
	case errors.Is(err, domain.ErrEmailTaken):
		utils.WriteBusinessError(w, http.StatusConflict, "EMAIL_TAKEN", err.Error(), map[string]any{"field": "email"})
	case errors.Is(err, domain.ErrPasswordExpired):
		utils.WriteBusinessError(w, http.StatusUnauthorized, "PASSWORD_EXPIRED", err.Error(), nil)
	case errors.Is(err, domain.ErrCannotResetSelf):
		utils.WriteBusinessError(w, http.StatusConflict, "CANNOT_RESET_SELF", err.Error(), nil)
	case errors.Is(err, domain.ErrAdminNotFound):
		utils.WriteBusinessError(w, http.StatusNotFound, "ADMIN_NOT_FOUND", err.Error(), nil)
	case errors.Is(err, domain.ErrInviteEmailFailed):
		utils.WriteBusinessError(w, http.StatusBadGateway, "EMAIL_SEND_FAILED", err.Error(), nil)
	case errors.Is(err, domain.ErrWrongCurrentPassword):
		utils.WriteValidationError(w, "WRONG_CURRENT_PASSWORD", err.Error(), map[string]any{"field": "current_password"})
	case errors.Is(err, domain.ErrInvalidStatus):
		utils.WriteValidationError(w, "INVALID_STATUS", err.Error(), map[string]any{"field": "status"})
	case errors.Is(err, domain.ErrInvalidPage):
		utils.WriteValidationError(w, "INVALID_PAGE", err.Error(), map[string]any{"field": "page"})
	case errors.Is(err, domain.ErrTempleInputInvalid):
		details := map[string]any(nil)
		if ie, ok := errors.AsType[*domain.TempleInputError](err); ok {
			details = map[string]any{"field": ie.Field}
		}
		utils.WriteValidationError(w, "VALIDATION_ERROR", err.Error(), details)
	case errors.Is(err, domain.ErrSuggestionNotFound):
		utils.WriteBusinessError(w, http.StatusNotFound, "TEMPLE_SUGGESTION_NOT_FOUND", err.Error(), nil)
	case errors.Is(err, domain.ErrSuggestionAlreadyCreated):
		utils.WriteBusinessError(w, http.StatusConflict, "SUGGESTION_ALREADY_CREATED", err.Error(), nil)
	case errors.Is(err, domain.ErrReopenForbidden):
		utils.WriteBusinessError(w, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
	case errors.Is(err, domain.ErrInvalidAnnouncement):
		utils.WriteValidationError(w, "INVALID_ANNOUNCEMENT", err.Error(), inputField(err))
	case errors.Is(err, domain.ErrInvalidIssuingBodyName):
		utils.WriteValidationError(w, "VALIDATION_ERROR", err.Error(), inputField(err))
	case errors.Is(err, domain.ErrInvalidIssuingBody):
		utils.WriteValidationError(w, "INVALID_ISSUING_BODY", err.Error(), map[string]any{"field": "issuing_body_id"})
	case errors.Is(err, domain.ErrTempleNotFound):
		utils.WriteBusinessError(w, http.StatusNotFound, "TEMPLE_NOT_FOUND", err.Error(), nil)
	case errors.Is(err, domain.ErrAnnouncementNotFound):
		utils.WriteBusinessError(w, http.StatusNotFound, "ANNOUNCEMENT_NOT_FOUND", err.Error(), nil)
	case errors.Is(err, domain.ErrAnnouncementRevoked):
		utils.WriteBusinessError(w, http.StatusConflict, "ANNOUNCEMENT_REVOKED", err.Error(), nil)
	case errors.Is(err, domain.ErrIssuingBodyExists):
		utils.WriteBusinessError(w, http.StatusConflict, "ISSUING_BODY_EXISTS", err.Error(), map[string]any{"field": "name"})
	case errors.Is(err, domain.ErrWeakPassword):
		utils.WriteValidationError(w, "WEAK_PASSWORD",
			fmt.Sprintf("new password must be at least %d characters and differ from the current one", domain.MinPasswordLen),
			map[string]any{"field": "new_password"})
	default:
		utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)
	}
}

// inputField returns {"field": ...} for an AnnouncementInputError, else nil.
func inputField(err error) map[string]any {
	if ie, ok := errors.AsType[*domain.AnnouncementInputError](err); ok {
		return map[string]any{"field": ie.Field}
	}
	return nil
}

// WriteAuthError is the auth middleware's ErrorHandler: a 401 in the shared
// envelope. The cause is deliberately not echoed.
func WriteAuthError(w http.ResponseWriter, _ *http.Request, _ error) {
	utils.WriteBusinessError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
}

// RequireRole wraps next so only a token whose role claim equals role passes;
// anything else is 403 FORBIDDEN. An admin who still must change their password
// is refused with 403 PASSWORD_CHANGE_REQUIRED. It must run inside the auth
// middleware, which is what puts the claims in the context.
func RequireRole(role string, next http.HandlerFunc) http.HandlerFunc {
	return guard(func(r string) bool { return r == role }, false, next)
}

// RequireAnyRole is RequireRole for several roles: a token whose role claim is
// any one of roles passes. Same 403 and password-change rules as RequireRole.
func RequireAnyRole(next http.HandlerFunc, roles ...string) http.HandlerFunc {
	return guard(func(r string) bool { return slices.Contains(roles, r) }, false, next)
}

// RequireStaff is RequireRole for routes any backoffice account may use,
// whatever its role (a known role is still required).
func RequireStaff(next http.HandlerFunc) http.HandlerFunc {
	return guard(domain.ValidRole, false, next)
}

// RequireStaffAllowPasswordChange is RequireStaff without the must-change-
// password refusal; only the change-password route uses it.
func RequireStaffAllowPasswordChange(next http.HandlerFunc) http.HandlerFunc {
	return guard(domain.ValidRole, true, next)
}

func guard(roleAllowed func(string) bool, allowPasswordChange bool, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := jwtcontext.GetClaims(r.Context())
		if !ok {
			WriteAuthError(w, r, errors.New("no claims in context"))
			return
		}
		if !roleAllowed(claims.Role) {
			utils.WriteBusinessError(w, http.StatusForbidden, "FORBIDDEN", "forbidden", nil)
			return
		}
		if v, _ := claims.GetExtra(domain.ExtraMustChangePassword); v == true && !allowPasswordChange {
			utils.WriteBusinessError(w, http.StatusForbidden, "PASSWORD_CHANGE_REQUIRED", domain.ErrPasswordChangeRequired.Error(), nil)
			return
		}
		next(w, r)
	}
}
