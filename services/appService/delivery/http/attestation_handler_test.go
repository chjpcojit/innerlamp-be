package http

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/services/appService/domain"
)

func newAttestationTestHandler(t *testing.T) (*AttestationHandler, *mockAttestationUsecase) {
	t.Helper()
	uc := &mockAttestationUsecase{}
	return NewAttestationHandler(uc, newTestValidator(t)), uc
}

func TestAttestationHandler_Challenge(t *testing.T) {
	t.Run("mints a challenge with purpose attest (AC1)", func(t *testing.T) {
		h, uc := newAttestationTestHandler(t)
		uc.On("RequestChallenge", mock.Anything, domain.PlatformAndroid, domain.ChallengePurposeAttest).
			Return(&domain.RequestChallengeResult{Challenge: "nonce", ExpiresInSeconds: 300}, nil)

		rec := doRequest(h.Challenge, `{"platform":"android"}`)

		require.Equal(t, http.StatusOK, rec.Code)
		var body challengeResponse
		decodeSuccessData(t, rec, &body)
		require.Equal(t, "nonce", body.Challenge)
		require.Equal(t, 300, body.ExpiresInSeconds)
		uc.AssertExpectations(t)
	})

	t.Run("rejects an invalid platform and does not call the usecase (AC2)", func(t *testing.T) {
		h, uc := newAttestationTestHandler(t)
		rec := doRequest(h.Challenge, `{"platform":"windows"}`)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "VALIDATION_ERROR", decodeErrorBody(t, rec).Code)
		uc.AssertNotCalled(t, "RequestChallenge", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("rejects malformed/unknown-field bodies (AC2)", func(t *testing.T) {
		for name, body := range map[string]string{
			"empty":         ``,
			"malformed":     `{"platform":`,
			"unknown field": `{"platform":"android","extra":1}`,
		} {
			t.Run(name, func(t *testing.T) {
				h, uc := newAttestationTestHandler(t)
				rec := doRequest(h.Challenge, body)
				require.Equal(t, http.StatusBadRequest, rec.Code)
				require.Equal(t, "VALIDATION_ERROR", decodeErrorBody(t, rec).Code)
				uc.AssertNotCalled(t, "RequestChallenge", mock.Anything, mock.Anything, mock.Anything)
			})
		}
	})
}

func TestAttestationHandler_AssertionChallenge(t *testing.T) {
	t.Run("mints an assertion challenge for ios (AC3)", func(t *testing.T) {
		h, uc := newAttestationTestHandler(t)
		uc.On("RequestChallenge", mock.Anything, domain.PlatformIOS, domain.ChallengePurposeAssert).
			Return(&domain.RequestChallengeResult{Challenge: "nonce", ExpiresInSeconds: 300}, nil)

		rec := doRequest(h.AssertionChallenge, `{"platform":"ios"}`)

		require.Equal(t, http.StatusOK, rec.Code)
		var body challengeResponse
		decodeSuccessData(t, rec, &body)
		require.Equal(t, "nonce", body.Challenge)
		uc.AssertExpectations(t)
	})

	t.Run("rejects a non-ios platform (AC4)", func(t *testing.T) {
		h, uc := newAttestationTestHandler(t)
		rec := doRequest(h.AssertionChallenge, `{"platform":"android"}`)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "VALIDATION_ERROR", decodeErrorBody(t, rec).Code)
		uc.AssertNotCalled(t, "RequestChallenge", mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestAttestationHandler_Verify(t *testing.T) {
	t.Run("android pass returns attested:true with signals (AC5)", func(t *testing.T) {
		h, uc := newAttestationTestHandler(t)
		uc.On("Verify", mock.Anything, domain.VerifyInput{
			Platform: domain.PlatformAndroid, Challenge: "nonce", Token: "token",
		}).Return(&domain.Verdict{
			Attested: true,
			Platform: domain.PlatformAndroid,
			Signals:  map[string]string{"app_recognition_verdict": "PLAY_RECOGNIZED"},
		}, nil)

		rec := doRequest(h.Verify, `{"platform":"android","challenge":"nonce","token":"token"}`)

		require.Equal(t, http.StatusOK, rec.Code)
		var body verifyResponse
		decodeSuccessData(t, rec, &body)
		require.True(t, body.Attested)
		require.Equal(t, "android", body.Platform)
		require.Equal(t, "PLAY_RECOGNIZED", body.Signals["app_recognition_verdict"])
		require.Empty(t, body.Reason)
		uc.AssertExpectations(t)
	})

	t.Run("android negative verdict is still 200 with reason (AC5)", func(t *testing.T) {
		h, uc := newAttestationTestHandler(t)
		uc.On("Verify", mock.Anything, mock.Anything).Return(&domain.Verdict{
			Attested: false,
			Platform: domain.PlatformAndroid,
			Reason:   domain.ReasonAppNotRecognized,
		}, nil)

		rec := doRequest(h.Verify, `{"platform":"android","challenge":"nonce","token":"token"}`)

		require.Equal(t, http.StatusOK, rec.Code)
		var body verifyResponse
		decodeSuccessData(t, rec, &body)
		require.False(t, body.Attested)
		require.Equal(t, domain.ReasonAppNotRecognized, body.Reason)
	})

	t.Run("ios attest decodes base64 and returns attested:true (AC8)", func(t *testing.T) {
		h, uc := newAttestationTestHandler(t)
		uc.On("Verify", mock.Anything, domain.VerifyInput{
			Platform: domain.PlatformIOS, Challenge: "nonce", KeyID: "key-1", Attestation: []byte("raw"),
		}).Return(&domain.Verdict{
			Attested: true,
			Platform: domain.PlatformIOS,
			Signals:  map[string]string{"environment": "production"},
		}, nil)

		rec := doRequest(h.Verify, `{"platform":"ios","challenge":"nonce","key_id":"key-1","attestation":"cmF3"}`)

		require.Equal(t, http.StatusOK, rec.Code)
		var body verifyResponse
		decodeSuccessData(t, rec, &body)
		require.True(t, body.Attested)
		require.Equal(t, "production", body.Signals["environment"])
		uc.AssertExpectations(t)
	})

	t.Run("missing token for android is 400 VALIDATION_ERROR (AC16)", func(t *testing.T) {
		h, uc := newAttestationTestHandler(t)
		rec := doRequest(h.Verify, `{"platform":"android","challenge":"nonce"}`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "VALIDATION_ERROR", decodeErrorBody(t, rec).Code)
		uc.AssertNotCalled(t, "Verify", mock.Anything, mock.Anything)
	})

	t.Run("missing key_id/attestation for ios is 400 (AC16)", func(t *testing.T) {
		for name, body := range map[string]string{
			"missing key_id":      `{"platform":"ios","challenge":"nonce","attestation":"cmF3"}`,
			"missing attestation": `{"platform":"ios","challenge":"nonce","key_id":"key-1"}`,
		} {
			t.Run(name, func(t *testing.T) {
				h, uc := newAttestationTestHandler(t)
				rec := doRequest(h.Verify, body)
				require.Equal(t, http.StatusBadRequest, rec.Code)
				require.Equal(t, "VALIDATION_ERROR", decodeErrorBody(t, rec).Code)
				uc.AssertNotCalled(t, "Verify", mock.Anything, mock.Anything)
			})
		}
	})

	t.Run("cross-platform fields are rejected (AC15)", func(t *testing.T) {
		for name, body := range map[string]string{
			"ios fields on android": `{"platform":"android","challenge":"nonce","token":"t","key_id":"k"}`,
			"token on ios":          `{"platform":"ios","challenge":"nonce","key_id":"k","attestation":"cmF3","token":"t"}`,
		} {
			t.Run(name, func(t *testing.T) {
				h, uc := newAttestationTestHandler(t)
				rec := doRequest(h.Verify, body)
				require.Equal(t, http.StatusBadRequest, rec.Code)
				require.Equal(t, "VALIDATION_ERROR", decodeErrorBody(t, rec).Code)
				uc.AssertNotCalled(t, "Verify", mock.Anything, mock.Anything)
			})
		}
	})

	t.Run("invalid base64 attestation is 400", func(t *testing.T) {
		h, uc := newAttestationTestHandler(t)
		rec := doRequest(h.Verify, `{"platform":"ios","challenge":"nonce","key_id":"k","attestation":"!!not-base64!!"}`)
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "VALIDATION_ERROR", decodeErrorBody(t, rec).Code)
		uc.AssertNotCalled(t, "Verify", mock.Anything, mock.Anything)
	})

	t.Run("challenge invalid maps to 400 ATTESTATION_CHALLENGE_INVALID (AC6)", func(t *testing.T) {
		h, uc := newAttestationTestHandler(t)
		uc.On("Verify", mock.Anything, mock.Anything).Return(nil, domain.ErrAttestationChallengeInvalid)

		rec := doRequest(h.Verify, `{"platform":"android","challenge":"nonce","token":"token"}`)

		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "ATTESTATION_CHALLENGE_INVALID", decodeErrorBody(t, rec).Code)
	})

	t.Run("upstream error maps to 502 ATTESTATION_UPSTREAM_ERROR (AC7)", func(t *testing.T) {
		h, uc := newAttestationTestHandler(t)
		uc.On("Verify", mock.Anything, mock.Anything).Return(nil, domain.ErrAttestationUpstreamError)

		rec := doRequest(h.Verify, `{"platform":"android","challenge":"nonce","token":"token"}`)

		require.Equal(t, http.StatusBadGateway, rec.Code)
		require.Equal(t, "ATTESTATION_UPSTREAM_ERROR", decodeErrorBody(t, rec).Code)
	})

	t.Run("unmapped error is 500 without leaking", func(t *testing.T) {
		h, uc := newAttestationTestHandler(t)
		uc.On("Verify", mock.Anything, mock.Anything).Return(nil, errUnmapped)

		rec := doRequest(h.Verify, `{"platform":"android","challenge":"nonce","token":"token"}`)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		require.NotContains(t, rec.Body.String(), errUnmapped.Error())
	})
}
