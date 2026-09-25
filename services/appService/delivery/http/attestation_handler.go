package http

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/pkg/validation"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// attestationUsecase is the subset of *usecase.AttestationUsecase these
// handlers depend on, declared here (the consumer) so the handler can be unit
// tested with a mock instead of the real use case and its repository/verifier.
type attestationUsecase interface {
	RequestChallenge(ctx context.Context, platform domain.Platform, purpose domain.ChallengePurpose) (*domain.RequestChallengeResult, error)
	Verify(ctx context.Context, input domain.VerifyInput) (*domain.Verdict, error)
}

// AttestationHandler serves the three public attestation routes: minting a
// challenge (attestation or assertion) and verifying platform evidence. All
// three are public — the caller has no account yet, so there is nothing to
// authenticate.
type AttestationHandler struct {
	usecase   attestationUsecase
	validator *validation.Validator
}

// NewAttestationHandler wires the handler. validator must already have its
// field-name tag set to "json" (composition-root concern, per cmd/main.go).
func NewAttestationHandler(usecase attestationUsecase, validator *validation.Validator) *AttestationHandler {
	return &AttestationHandler{usecase: usecase, validator: validator}
}

// challengeRequest is the body for both POST /v1/attestation/challenge and
// POST /v1/assertion/challenge. The assertion endpoint narrows platform to ios
// via its own request type below.
type challengeRequest struct {
	Platform string `json:"platform" validate:"required,oneof=android ios"`
}

type challengeResponse struct {
	Challenge        string `json:"challenge"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

// assertionChallengeRequest is the body for POST /v1/assertion/challenge. Only
// iOS has per-request assertions, so the platform is narrowed to ios.
type assertionChallengeRequest struct {
	Platform string `json:"platform" validate:"required,oneof=ios"`
}

// verifyRequest is the body for POST /v1/attestation/verify. Token is android-
// only; KeyID + Attestation are ios-only. The cross-field rules (token required
// for android and forbidden for ios, and vice versa) are checked by
// validateVerifyFields after struct-tag validation.
type verifyRequest struct {
	Platform    string `json:"platform" validate:"required,oneof=android ios"`
	Challenge   string `json:"challenge" validate:"required"`
	Token       string `json:"token,omitempty"`
	KeyID       string `json:"key_id,omitempty"`
	Attestation string `json:"attestation,omitempty"`
}

// verifyResponse is the wire shape of a Verdict. Signals and Reason are omitted
// when empty so a negative verdict is just {attested:false, platform, reason}.
type verifyResponse struct {
	Attested bool              `json:"attested"`
	Platform string            `json:"platform"`
	Signals  map[string]string `json:"signals,omitempty"`
	Reason   string            `json:"reason,omitempty"`
}

// Challenge serves POST /v1/attestation/challenge — design.md challenge flow.
func (h *AttestationHandler) Challenge(w http.ResponseWriter, r *http.Request) {
	var req challengeRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if err := h.validateStruct(w, req); err != nil {
		return
	}

	result, err := h.usecase.RequestChallenge(r.Context(), domain.Platform(req.Platform), domain.ChallengePurposeAttest)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	utils.WriteSuccessResponse(w, challengeResponse{
		Challenge:        result.Challenge,
		ExpiresInSeconds: result.ExpiresInSeconds,
	})
}

// AssertionChallenge serves POST /v1/assertion/challenge — design.md assertion
// challenge flow.
func (h *AttestationHandler) AssertionChallenge(w http.ResponseWriter, r *http.Request) {
	var req assertionChallengeRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if err := h.validateStruct(w, req); err != nil {
		return
	}

	result, err := h.usecase.RequestChallenge(r.Context(), domain.Platform(req.Platform), domain.ChallengePurposeAssert)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	utils.WriteSuccessResponse(w, challengeResponse{
		Challenge:        result.Challenge,
		ExpiresInSeconds: result.ExpiresInSeconds,
	})
}

// Verify serves POST /v1/attestation/verify — design.md verify flows (Android
// and iOS attest). A negative verdict is still 200; only a challenge-state
// error (400) or upstream failure (502) is an error.
func (h *AttestationHandler) Verify(w http.ResponseWriter, r *http.Request) {
	var req verifyRequest
	if err := utils.DecodeJSONBodyStrict(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if err := h.validateStruct(w, req); err != nil {
		return
	}
	if err := h.validateVerifyFields(w, req); err != nil {
		return
	}

	// The wire carries attestation as base64; delivery owns the wire format, so
	// it decodes here and passes raw bytes to the use case. A malformed
	// attestation is a validation error, not a crypto failure.
	attestation, err := decodeAttestation(req.Attestation)
	if err != nil {
		utils.WriteValidationError(w, "VALIDATION_ERROR", "request validation failed", map[string]any{"attestation": "base64"})
		return
	}

	verdict, err := h.usecase.Verify(r.Context(), domain.VerifyInput{
		Platform:    domain.Platform(req.Platform),
		Challenge:   req.Challenge,
		Token:       req.Token,
		KeyID:       req.KeyID,
		Attestation: attestation,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	utils.WriteSuccessResponse(w, verifyResponse{
		Attested: verdict.Attested,
		Platform: string(verdict.Platform),
		Signals:  verdict.Signals,
		Reason:   verdict.Reason,
	})
}

// validateVerifyFields enforces the platform-specific field rules (AC15/AC16):
// android requires token and forbids key_id/attestation; ios requires
// key_id + attestation and forbids token. A violation is 400 VALIDATION_ERROR.
func (h *AttestationHandler) validateVerifyFields(w http.ResponseWriter, req verifyRequest) error {
	switch req.Platform {
	case "android":
		if req.Token == "" {
			return writeFieldValidationError(w, "token", "required")
		}
		if req.KeyID != "" || req.Attestation != "" {
			return writeFieldValidationError(w, "key_id", "excluded")
		}
	case "ios":
		if req.KeyID == "" {
			return writeFieldValidationError(w, "key_id", "required")
		}
		if req.Attestation == "" {
			return writeFieldValidationError(w, "attestation", "required")
		}
		if req.Token != "" {
			return writeFieldValidationError(w, "token", "excluded")
		}
	}
	return nil
}

// writeFieldValidationError writes a 400 VALIDATION_ERROR naming one field, and
// returns a non-nil error so the caller can bail with
// `if err := ...; err != nil { return }`.
func writeFieldValidationError(w http.ResponseWriter, field, tag string) error {
	utils.WriteValidationError(w, "VALIDATION_ERROR", "request validation failed", map[string]any{field: tag})
	return errors.New("request validation failed")
}

// decodeAttestation base64-decodes the ios attestation field. An empty string
// decodes to nil (used when the platform is android and the field is absent).
func decodeAttestation(encoded string) ([]byte, error) {
	if encoded == "" {
		return nil, nil
	}
	return base64.StdEncoding.DecodeString(encoded)
}

// validateStruct runs req through the shared validator and, on failure, writes
// the response itself and returns a non-nil error — mirroring RegisterHandler.
func (h *AttestationHandler) validateStruct(w http.ResponseWriter, req any) error {
	err := h.validator.Struct(req)
	if err == nil {
		return nil
	}

	if verrs, ok := errors.AsType[validator.ValidationErrors](err); ok {
		writeValidationError(w, verrs)
		return err
	}
	utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)
	return err
}
