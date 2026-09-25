package http

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

// newAssertionMiddleware builds a RequireAttestation over a mock usecase and a
// next handler that records whether it was reached and the body it saw.
func newAssertionMiddleware() (*RequireAttestation, *mockAssertionUsecase) {
	uc := &mockAssertionUsecase{}
	return NewRequireAttestation(uc), uc
}

func assertionRequest(method, path, body string, keyID, assertion, challenge string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if keyID != "" {
		req.Header.Set(HeaderAttestationKeyID, keyID)
	}
	if assertion != "" {
		req.Header.Set(HeaderAttestationAssertion, assertion)
	}
	if challenge != "" {
		req.Header.Set(HeaderAttestationChallenge, challenge)
	}
	return req
}

func nextHandler(t *testing.T, reached *bool, body *string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*reached = true
		b, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		*body = string(b)
		w.WriteHeader(http.StatusOK)
	})
}

func TestRequireAttestation_PassesOnValidAssertion(t *testing.T) {
	mw, uc := newAssertionMiddleware()

	challenge := "nonce-assert"
	assertionRaw := []byte("assertion-bytes")
	reqBody := `{"sensitive":true}`

	// clientData = challenge || method || path || SHA256(body)
	bodyHash := sha256.Sum256([]byte(reqBody))
	expectedClientData := append([]byte(challenge), []byte(http.MethodPost)...)
	expectedClientData = append(expectedClientData, "/v1/sensitive"...)
	expectedClientData = append(expectedClientData, bodyHash[:]...)

	uc.On("VerifyAssertion", mock.Anything, domain.VerifyAssertionInput{
		KeyID:      "key-1",
		Assertion:  assertionRaw,
		ClientData: expectedClientData,
		Challenge:  challenge,
	}).Return(nil)

	reached := false
	var nextBody string
	handler := mw.Wrap(nextHandler(t, &reached, &nextBody))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, assertionRequest(
		http.MethodPost, "/v1/sensitive", reqBody,
		"key-1", base64.StdEncoding.EncodeToString(assertionRaw), challenge,
	))

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, reached, "next handler must be reached on success")
	require.Equal(t, reqBody, nextBody, "the body must be passed through intact")
	uc.AssertExpectations(t)
}

func TestRequireAttestation_MissingHeader(t *testing.T) {
	for name, build := range map[string]func() *http.Request{
		"missing key id":     func() *http.Request { return assertionRequest("POST", "/", "b", "", "YWJj", "nonce") },
		"missing assertion":  func() *http.Request { return assertionRequest("POST", "/", "b", "key-1", "", "nonce") },
		"missing challenge":  func() *http.Request { return assertionRequest("POST", "/", "b", "key-1", "YWJj", "") },
		"missing everything": func() *http.Request { return assertionRequest("POST", "/", "b", "", "", "") },
	} {
		t.Run(name, func(t *testing.T) {
			mw, uc := newAssertionMiddleware()
			reached := false
			var nextBody string
			handler := mw.Wrap(nextHandler(t, &reached, &nextBody))

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, build())

			require.Equal(t, http.StatusUnauthorized, rec.Code)
			got := decodeErrorBody(t, rec)
			require.Equal(t, "ATTESTATION_FAILED", got.Code)
			require.Equal(t, domain.ReasonMissingEvidence, got.Details["reason"])
			require.False(t, reached)
			uc.AssertNotCalled(t, "VerifyAssertion", mock.Anything, mock.Anything)
		})
	}
}

func TestRequireAttestation_InvalidBase64Assertion(t *testing.T) {
	mw, _ := newAssertionMiddleware()
	reached := false
	var nextBody string
	handler := mw.Wrap(nextHandler(t, &reached, &nextBody))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, assertionRequest("POST", "/", "b", "key-1", "!!bad!!", "nonce"))

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	got := decodeErrorBody(t, rec)
	require.Equal(t, "ATTESTATION_FAILED", got.Code)
	require.Equal(t, domain.ReasonAssertionInvalid, got.Details["reason"])
	require.False(t, reached)
}

func TestRequireAttestation_ErrorMapping(t *testing.T) {
	for name, tc := range map[string]struct {
		err    error
		code   string
		reason any
	}{
		"unknown key":           {domain.ErrAttestationKeyNotFound, "ATTESTATION_FAILED", domain.ReasonUnknownKey},
		"counter not increased": {domain.ErrAttestationCounterNotIncreased, "ATTESTATION_FAILED", domain.ReasonCounterNotIncreased},
		"assertion invalid":     {domain.ErrAttestationAssertionInvalid, "ATTESTATION_FAILED", domain.ReasonAssertionInvalid},
		"challenge invalid":     {domain.ErrAttestationChallengeInvalid, "ATTESTATION_CHALLENGE_INVALID", nil},
	} {
		t.Run(name, func(t *testing.T) {
			mw, uc := newAssertionMiddleware()
			uc.On("VerifyAssertion", mock.Anything, mock.Anything).Return(tc.err)

			reached := false
			var nextBody string
			handler := mw.Wrap(nextHandler(t, &reached, &nextBody))

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, assertionRequest("POST", "/", "b", "key-1", "YWJj", "nonce"))

			require.Equal(t, http.StatusUnauthorized, rec.Code)
			got := decodeErrorBody(t, rec)
			require.Equal(t, tc.code, got.Code)
			require.Equal(t, tc.reason, got.Details["reason"])
			require.False(t, reached)
		})
	}
}

func TestRequireAttestation_InfraErrorIs500(t *testing.T) {
	mw, uc := newAssertionMiddleware()
	uc.On("VerifyAssertion", mock.Anything, mock.Anything).Return(errUnmapped)

	reached := false
	var nextBody string
	handler := mw.Wrap(nextHandler(t, &reached, &nextBody))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, assertionRequest("POST", "/", "b", "key-1", "YWJj", "nonce"))

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.NotContains(t, rec.Body.String(), errUnmapped.Error())
	require.False(t, reached)
}

func TestRequireAttestation_OversizedBodyIs413(t *testing.T) {
	mw, uc := newAssertionMiddleware()
	reached := false
	var nextBody string
	handler := mw.Wrap(nextHandler(t, &reached, &nextBody))

	oversized := strings.Repeat("x", maxAttestationBody+1)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, assertionRequest("POST", "/", oversized, "key-1", "YWJj", "nonce"))

	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	require.Equal(t, "PAYLOAD_TOO_LARGE", decodeErrorBody(t, rec).Code)
	require.False(t, reached)
	uc.AssertNotCalled(t, "VerifyAssertion", mock.Anything, mock.Anything)
}
