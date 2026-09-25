package http

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"net/http"

	"chjpcoj.com.innerlamp/pkg/utils"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// Header names the middleware reads. They carry the App Attest assertion bound
// to the actual request (key id, assertion, and the single-use challenge).
const (
	HeaderAttestationKeyID     = "X-Attestation-Key-Id"
	HeaderAttestationAssertion = "X-Attestation-Assertion"
	HeaderAttestationChallenge = "X-Attestation-Challenge"
)

// maxAttestationBody is the largest request body the middleware will hash. It
// matches DecodeJSONBodyStrict's 1 MB cap so a protected route's body cannot be
// amplified past what the handler will already accept.
const maxAttestationBody = 1 << 20

// assertionUsecase is the subset of *usecase.AttestationUsecase the middleware
// depends on, declared here (the consumer) so RequireAttestation can be unit
// tested with a mock instead of the real use case.
type assertionUsecase interface {
	VerifyAssertion(ctx context.Context, input domain.VerifyAssertionInput) error
}

// RequireAttestation is the middleware that gates sensitive routes behind a
// per-request App Attest assertion (design.md assertion flow). It reads the
// three attestation headers, rebuilds clientData from the real request, and
// verifies the assertion — then passes the request on or rejects with 401.
//
// It is the real "gate": unlike the verify service, which returns a verdict,
// this binds an assertion to the exact request (method, path, body hash) so a
// replay for another request fails the signature check.
type RequireAttestation struct {
	usecase assertionUsecase
}

// NewRequireAttestation wires the middleware.
func NewRequireAttestation(usecase assertionUsecase) *RequireAttestation {
	return &RequireAttestation{usecase: usecase}
}

// Wrap returns a handler that verifies the assertion before delegating to next.
func (m *RequireAttestation) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keyID := r.Header.Get(HeaderAttestationKeyID)
		assertionHeader := r.Header.Get(HeaderAttestationAssertion)
		challenge := r.Header.Get(HeaderAttestationChallenge)

		// Step 1: all three headers are required. A missing one is a missing
		// piece of evidence, not a verification failure.
		if keyID == "" || assertionHeader == "" || challenge == "" {
			writeAttestationFailed(w, domain.ReasonMissingEvidence)
			return
		}

		// Step 2: tee the body so it can be hashed for the request binding and
		// still passed intact to the next handler.
		r.Body = http.MaxBytesReader(w, r.Body, maxAttestationBody)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				utils.WriteBusinessError(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", utils.ErrBodyTooLarge.Error(), nil)
				return
			}
			utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))

		// The assertion header is base64-encoded CBOR; a malformed one is an
		// invalid assertion rather than a server error.
		assertion, err := base64.StdEncoding.DecodeString(assertionHeader)
		if err != nil {
			writeAttestationFailed(w, domain.ReasonAssertionInvalid)
			return
		}

		// Step 3: rebuild clientData from the real request, then verify.
		clientData := buildClientData(challenge, r.Method, r.URL.Path, body)
		if err := m.usecase.VerifyAssertion(r.Context(), domain.VerifyAssertionInput{
			KeyID:      keyID,
			Assertion:  assertion,
			ClientData: clientData,
			Challenge:  challenge,
		}); err != nil {
			writeAttestationError(w, err)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// buildClientData reconstructs the exact bytes the client signed:
//
//	clientData = challenge || method || path || SHA256(body)
//
// The client hashes this (SHA256(clientData)) and passes it to
// generateAssertion; the server rebuilds it from the real request rather than
// trusting anything the client sends, so an assertion minted for a different
// method/path/body will not verify (design.md assertion flow Step 3).
func buildClientData(challenge, method, path string, body []byte) []byte {
	bodyHash := sha256.Sum256(body)
	buf := make([]byte, 0, len(challenge)+len(method)+len(path)+len(bodyHash))
	buf = append(buf, challenge...)
	buf = append(buf, method...)
	buf = append(buf, path...)
	buf = append(buf, bodyHash[:]...)
	return buf
}

// writeAttestationError maps the assertion sentinels to 401 (design.md
// assertion flow Step 3/4). The reason is carried in the details so the client
// can tell a replay from an unknown key without the status code differing.
func writeAttestationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrAttestationChallengeInvalid):
		utils.WriteBusinessError(w, http.StatusUnauthorized, "ATTESTATION_CHALLENGE_INVALID", err.Error(), nil)
	case errors.Is(err, domain.ErrAttestationKeyNotFound):
		writeAttestationFailed(w, domain.ReasonUnknownKey)
	case errors.Is(err, domain.ErrAttestationCounterNotIncreased):
		writeAttestationFailed(w, domain.ReasonCounterNotIncreased)
	case errors.Is(err, domain.ErrAttestationAssertionInvalid):
		writeAttestationFailed(w, domain.ReasonAssertionInvalid)
	default:
		utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)
	}
}

// writeAttestationFailed writes a 401 ATTESTATION_FAILED with the given reason.
func writeAttestationFailed(w http.ResponseWriter, reason string) {
	utils.WriteBusinessError(w, http.StatusUnauthorized, "ATTESTATION_FAILED", "attestation failed", map[string]any{"reason": reason})
}
