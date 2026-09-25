package usecase

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/logging"
	"chjpcoj.com.innerlamp/services/appService/config"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// testAttestationConfig mirrors the values in config.dev.yaml's `attestation:`
// block. The use case reads its policy from config at runtime, so the fixture
// supplies one explicitly — and pinning the literals here (rather than reading
// them back out of the config package) is deliberate, for the same reason as
// testOTPPolicy.
var testAttestationConfig = config.AttestationConfig{
	ChallengeTTL:    config.Duration(5 * time.Minute),
	FreshnessWindow: config.Duration(5 * time.Minute),
	Android: config.AttestationAndroidConfig{
		Provider:                      "mock",
		PackageName:                   "com.innerlamp.app",
		RequiredAppRecognitionVerdict: "PLAY_RECOGNIZED",
	},
	IOS: config.AttestationIOSConfig{
		Provider: "mock",
		AppID:    "TEAMID.com.innerlamp.app",
	},
}

type attestationFixture struct {
	usecase  *AttestationUsecase
	repo     *domain.MockAttestationRepository
	verifier *domain.MockAttestationVerifier
	logger   *logging.MockLogger
}

func newAttestationFixture() *attestationFixture {
	repo := &domain.MockAttestationRepository{}
	verifier := &domain.MockAttestationVerifier{}
	logger := newLogger()
	return &attestationFixture{
		usecase:  NewAttestationUsecase(repo, verifier, logger, testAttestationConfig),
		repo:     repo,
		verifier: verifier,
		logger:   logger,
	}
}

// validAndroidResult returns a Play Integrity decode that matches the test
// config and the given challenge, so only one check can be flipped per test.
func validAndroidResult(challenge string) *domain.AndroidVerdict {
	return &domain.AndroidVerdict{
		PackageName:              "com.innerlamp.app",
		Nonce:                    challenge,
		TimestampMillis:          time.Now().UnixMilli(),
		AppRecognitionVerdict:    "PLAY_RECOGNIZED",
		DeviceRecognitionVerdict: "MEETS_DEVICE_INTEGRITY",
		AppLicensingVerdict:      "LICENSED",
	}
}

// stubValidChallenge makes FindChallenge return a live, unused challenge for
// the given purpose, and MarkChallengeUsed report one row consumed.
func (fx *attestationFixture) stubValidChallenge(challenge string, purpose domain.ChallengePurpose) {
	fx.repo.On("FindChallenge", mock.Anything, challenge, purpose).Return(&domain.AttestationChallenge{
		Challenge: challenge,
		Purpose:   purpose,
	}, nil)
}

func TestAttestationUsecase_RequestChallenge(t *testing.T) {
	t.Run("mints a random nonce and persists with the right purpose (AC1)", func(t *testing.T) {
		fx := newAttestationFixture()
		var gotExpires time.Time
		fx.repo.On("SaveChallenge",
			mock.Anything, mock.Anything, domain.PlatformAndroid, domain.ChallengePurposeAttest, mock.AnythingOfType("time.Time"),
		).Run(func(args mock.Arguments) {
			gotExpires = args.Get(4).(time.Time)
		}).Return(nil)

		result, err := fx.usecase.RequestChallenge(context.Background(), domain.PlatformAndroid, domain.ChallengePurposeAttest)
		require.NoError(t, err)
		require.Equal(t, 300, result.ExpiresInSeconds)

		raw, err := base64.StdEncoding.DecodeString(result.Challenge)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(raw), 16, "nonce must carry at least 16 raw bytes")

		// expiresAt is now + ChallengeTTL, within a small skew of the clock.
		want := time.Now().Add(testAttestationConfig.ChallengeTTL.Duration())
		require.WithinDuration(t, want, gotExpires, 5*time.Second)

		fx.repo.AssertExpectations(t)
	})

	t.Run("assert purpose is persisted as assert (AC3)", func(t *testing.T) {
		fx := newAttestationFixture()
		fx.repo.On("SaveChallenge", mock.Anything, mock.Anything, domain.PlatformIOS, domain.ChallengePurposeAssert, mock.AnythingOfType("time.Time")).Return(nil)

		_, err := fx.usecase.RequestChallenge(context.Background(), domain.PlatformIOS, domain.ChallengePurposeAssert)
		require.NoError(t, err)
		fx.repo.AssertExpectations(t)
	})

	t.Run("a DB failure is wrapped (AC17)", func(t *testing.T) {
		fx := newAttestationFixture()
		fx.repo.On("SaveChallenge", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errUnmappedDB)

		_, err := fx.usecase.RequestChallenge(context.Background(), domain.PlatformAndroid, domain.ChallengePurposeAttest)
		require.Error(t, err)
		require.NotErrorIs(t, err, domain.ErrAttestationChallengeInvalid)
	})
}

var errUnmappedDB = errors.New("db is down")

func TestAttestationUsecase_VerifyAndroid(t *testing.T) {
	challenge := "nonce-abc"

	t.Run("all checks pass returns attested:true with signals (AC5)", func(t *testing.T) {
		fx := newAttestationFixture()
		fx.stubValidChallenge(challenge, domain.ChallengePurposeAttest)
		fx.verifier.On("VerifyAndroid", mock.Anything, "token", challenge).Return(validAndroidResult(challenge), nil)
		fx.repo.On("MarkChallengeUsed", mock.Anything, challenge).Return(true, nil)

		verdict, err := fx.usecase.Verify(context.Background(), domain.VerifyInput{
			Platform:  domain.PlatformAndroid,
			Challenge: challenge,
			Token:     "token",
		})
		require.NoError(t, err)
		require.True(t, verdict.Attested)
		require.Equal(t, domain.PlatformAndroid, verdict.Platform)
		require.Empty(t, verdict.Reason)
		require.Equal(t, "PLAY_RECOGNIZED", verdict.Signals["app_recognition_verdict"])
		require.Equal(t, "MEETS_DEVICE_INTEGRITY", verdict.Signals["device_recognition_verdict"])
		require.Equal(t, "LICENSED", verdict.Signals["app_licensing_verdict"])
		fx.repo.AssertExpectations(t)
	})

	t.Run("app not recognized returns attested:false with reason (AC5)", func(t *testing.T) {
		fx := newAttestationFixture()
		fx.stubValidChallenge(challenge, domain.ChallengePurposeAttest)
		res := validAndroidResult(challenge)
		res.AppRecognitionVerdict = "UNRECOGNIZED_VERSION"
		fx.verifier.On("VerifyAndroid", mock.Anything, "token", challenge).Return(res, nil)
		fx.repo.On("MarkChallengeUsed", mock.Anything, challenge).Return(true, nil)

		verdict, err := fx.usecase.Verify(context.Background(), domain.VerifyInput{
			Platform:  domain.PlatformAndroid,
			Challenge: challenge,
			Token:     "token",
		})
		require.NoError(t, err)
		require.False(t, verdict.Attested)
		require.Equal(t, domain.ReasonAppNotRecognized, verdict.Reason)
		require.Empty(t, verdict.Signals)
		fx.repo.AssertExpectations(t)
	})

	for name, mutate := range map[string]func(*domain.AndroidVerdict){
		"package name mismatch": func(v *domain.AndroidVerdict) { v.PackageName = "com.other.app" },
		"nonce mismatch":        func(v *domain.AndroidVerdict) { v.Nonce = "other-nonce" },
		"stale timestamp":       func(v *domain.AndroidVerdict) { v.TimestampMillis = time.Now().Add(-time.Hour).UnixMilli() },
		"future timestamp":      func(v *domain.AndroidVerdict) { v.TimestampMillis = time.Now().Add(time.Hour).UnixMilli() },
	} {
		t.Run(name+" returns attested:false reason attestation_invalid", func(t *testing.T) {
			fx := newAttestationFixture()
			fx.stubValidChallenge(challenge, domain.ChallengePurposeAttest)
			res := validAndroidResult(challenge)
			mutate(res)
			fx.verifier.On("VerifyAndroid", mock.Anything, "token", challenge).Return(res, nil)
			fx.repo.On("MarkChallengeUsed", mock.Anything, challenge).Return(true, nil)

			verdict, err := fx.usecase.Verify(context.Background(), domain.VerifyInput{
				Platform:  domain.PlatformAndroid,
				Challenge: challenge,
				Token:     "token",
			})
			require.NoError(t, err)
			require.False(t, verdict.Attested)
			require.Equal(t, domain.ReasonAttestationInvalid, verdict.Reason)
			fx.repo.AssertExpectations(t)
		})
	}

	t.Run("unknown/expired/used challenge returns ErrAttestationChallengeInvalid and does not call the verifier (AC6)", func(t *testing.T) {
		fx := newAttestationFixture()
		fx.repo.On("FindChallenge", mock.Anything, challenge, domain.ChallengePurposeAttest).Return(nil, domain.ErrAttestationChallengeInvalid)

		_, err := fx.usecase.Verify(context.Background(), domain.VerifyInput{
			Platform:  domain.PlatformAndroid,
			Challenge: challenge,
			Token:     "token",
		})
		require.ErrorIs(t, err, domain.ErrAttestationChallengeInvalid)
		fx.verifier.AssertNotCalled(t, "VerifyAndroid", mock.Anything, mock.Anything, mock.Anything)
		fx.repo.AssertNotCalled(t, "MarkChallengeUsed", mock.Anything, mock.Anything)
	})

	t.Run("upstream failure returns ErrAttestationUpstreamError and leaves the challenge unconsumed (AC7)", func(t *testing.T) {
		fx := newAttestationFixture()
		fx.stubValidChallenge(challenge, domain.ChallengePurposeAttest)
		fx.verifier.On("VerifyAndroid", mock.Anything, "token", challenge).Return(nil, errors.New("google down"))

		_, err := fx.usecase.Verify(context.Background(), domain.VerifyInput{
			Platform:  domain.PlatformAndroid,
			Challenge: challenge,
			Token:     "token",
		})
		require.ErrorIs(t, err, domain.ErrAttestationUpstreamError)
		fx.repo.AssertNotCalled(t, "MarkChallengeUsed", mock.Anything, mock.Anything)
	})

	t.Run("a race-losing MarkChallengeUsed is ErrAttestationChallengeInvalid", func(t *testing.T) {
		fx := newAttestationFixture()
		fx.stubValidChallenge(challenge, domain.ChallengePurposeAttest)
		fx.verifier.On("VerifyAndroid", mock.Anything, "token", challenge).Return(validAndroidResult(challenge), nil)
		fx.repo.On("MarkChallengeUsed", mock.Anything, challenge).Return(false, nil)

		_, err := fx.usecase.Verify(context.Background(), domain.VerifyInput{
			Platform:  domain.PlatformAndroid,
			Challenge: challenge,
			Token:     "token",
		})
		require.ErrorIs(t, err, domain.ErrAttestationChallengeInvalid)
	})
}

func TestAttestationUsecase_VerifyIOS(t *testing.T) {
	challenge := "nonce-ios"

	t.Run("attestation passes: persists the key and marks used atomically (AC8)", func(t *testing.T) {
		fx := newAttestationFixture()
		fx.stubValidChallenge(challenge, domain.ChallengePurposeAttest)
		fx.verifier.On("VerifyIOSAttestation", mock.Anything, "key-1", []byte("attestation"), challenge).
			Return(&domain.IOSAttestationResult{PublicKey: []byte("pk"), Receipt: []byte("rcpt"), Environment: "production"}, nil)

		var savedKey *domain.AttestationKey
		var savedChallenge string
		fx.repo.On("SaveKey", mock.Anything, mock.Anything, challenge).
			Run(func(args mock.Arguments) {
				savedKey = args.Get(1).(*domain.AttestationKey)
				savedChallenge = args.Get(2).(string)
			}).Return(nil)

		verdict, err := fx.usecase.Verify(context.Background(), domain.VerifyInput{
			Platform:    domain.PlatformIOS,
			Challenge:   challenge,
			KeyID:       "key-1",
			Attestation: []byte("attestation"),
		})
		require.NoError(t, err)
		require.True(t, verdict.Attested)
		require.Equal(t, "production", verdict.Signals["environment"])
		require.Equal(t, "key-1", savedKey.KeyID)
		require.Equal(t, []byte("pk"), savedKey.PublicKey)
		require.Equal(t, int64(0), savedKey.SignCounter)
		require.Equal(t, challenge, savedChallenge)
		fx.repo.AssertExpectations(t)
	})

	t.Run("attestation crypto failure returns attested:false and does not persist a key (AC9)", func(t *testing.T) {
		fx := newAttestationFixture()
		fx.stubValidChallenge(challenge, domain.ChallengePurposeAttest)
		fx.verifier.On("VerifyIOSAttestation", mock.Anything, "key-1", []byte("bad"), challenge).
			Return(nil, domain.ErrAttestationAssertionInvalid)
		fx.repo.On("MarkChallengeUsed", mock.Anything, challenge).Return(true, nil)

		verdict, err := fx.usecase.Verify(context.Background(), domain.VerifyInput{
			Platform:    domain.PlatformIOS,
			Challenge:   challenge,
			KeyID:       "key-1",
			Attestation: []byte("bad"),
		})
		require.NoError(t, err)
		require.False(t, verdict.Attested)
		require.Equal(t, domain.ReasonAttestationInvalid, verdict.Reason)
		fx.repo.AssertNotCalled(t, "SaveKey", mock.Anything, mock.Anything, mock.Anything)
		fx.repo.AssertExpectations(t)
	})

	t.Run("invalid challenge returns ErrAttestationChallengeInvalid without crypto (AC6)", func(t *testing.T) {
		fx := newAttestationFixture()
		fx.repo.On("FindChallenge", mock.Anything, challenge, domain.ChallengePurposeAttest).Return(nil, domain.ErrAttestationChallengeInvalid)

		_, err := fx.usecase.Verify(context.Background(), domain.VerifyInput{
			Platform:    domain.PlatformIOS,
			Challenge:   challenge,
			KeyID:       "key-1",
			Attestation: []byte("attestation"),
		})
		require.ErrorIs(t, err, domain.ErrAttestationChallengeInvalid)
		fx.verifier.AssertNotCalled(t, "VerifyIOSAttestation", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestAttestationUsecase_VerifyAssertion(t *testing.T) {
	challenge := "assert-nonce"
	keyID := "key-1"
	storedKey := &domain.AttestationKey{KeyID: keyID, PublicKey: []byte("pk"), SignCounter: 5}

	t.Run("success verifies, advances the counter and consumes the challenge (AC11)", func(t *testing.T) {
		fx := newAttestationFixture()
		fx.stubValidChallenge(challenge, domain.ChallengePurposeAssert)
		fx.repo.On("FindKey", mock.Anything, keyID).Return(storedKey, nil)
		fx.verifier.On("VerifyIOSAssertion", mock.Anything, []byte("pk"), []byte("assertion"), []byte("client-data"), int64(5)).Return(int64(6), nil)
		fx.repo.On("UpdateKeyCounter", mock.Anything, keyID, int64(6), challenge).Return(nil)

		err := fx.usecase.VerifyAssertion(context.Background(), domain.VerifyAssertionInput{
			KeyID:      keyID,
			Assertion:  []byte("assertion"),
			ClientData: []byte("client-data"),
			Challenge:  challenge,
		})
		require.NoError(t, err)
		fx.repo.AssertExpectations(t)
	})

	t.Run("counter not increased surfaces the sentinel and does not update (AC12)", func(t *testing.T) {
		fx := newAttestationFixture()
		fx.stubValidChallenge(challenge, domain.ChallengePurposeAssert)
		fx.repo.On("FindKey", mock.Anything, keyID).Return(storedKey, nil)
		fx.verifier.On("VerifyIOSAssertion", mock.Anything, []byte("pk"), []byte("assertion"), []byte("client-data"), int64(5)).
			Return(int64(0), domain.ErrAttestationCounterNotIncreased)

		err := fx.usecase.VerifyAssertion(context.Background(), domain.VerifyAssertionInput{
			KeyID: keyID, Assertion: []byte("assertion"), ClientData: []byte("client-data"), Challenge: challenge,
		})
		require.ErrorIs(t, err, domain.ErrAttestationCounterNotIncreased)
		fx.repo.AssertNotCalled(t, "UpdateKeyCounter", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("unknown key surfaces ErrAttestationKeyNotFound (AC13)", func(t *testing.T) {
		fx := newAttestationFixture()
		fx.stubValidChallenge(challenge, domain.ChallengePurposeAssert)
		fx.repo.On("FindKey", mock.Anything, keyID).Return(nil, domain.ErrAttestationKeyNotFound)

		err := fx.usecase.VerifyAssertion(context.Background(), domain.VerifyAssertionInput{
			KeyID: keyID, Assertion: []byte("a"), ClientData: []byte("c"), Challenge: challenge,
		})
		require.ErrorIs(t, err, domain.ErrAttestationKeyNotFound)
		fx.verifier.AssertNotCalled(t, "VerifyIOSAssertion", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("invalid assertion challenge surfaces ErrAttestationChallengeInvalid (AC14)", func(t *testing.T) {
		fx := newAttestationFixture()
		fx.repo.On("FindChallenge", mock.Anything, challenge, domain.ChallengePurposeAssert).Return(nil, domain.ErrAttestationChallengeInvalid)

		err := fx.usecase.VerifyAssertion(context.Background(), domain.VerifyAssertionInput{
			KeyID: keyID, Assertion: []byte("a"), ClientData: []byte("c"), Challenge: challenge,
		})
		require.ErrorIs(t, err, domain.ErrAttestationChallengeInvalid)
	})

	t.Run("signature failure surfaces ErrAttestationAssertionInvalid", func(t *testing.T) {
		fx := newAttestationFixture()
		fx.stubValidChallenge(challenge, domain.ChallengePurposeAssert)
		fx.repo.On("FindKey", mock.Anything, keyID).Return(storedKey, nil)
		fx.verifier.On("VerifyIOSAssertion", mock.Anything, []byte("pk"), []byte("assertion"), []byte("client-data"), int64(5)).
			Return(int64(0), domain.ErrAttestationAssertionInvalid)

		err := fx.usecase.VerifyAssertion(context.Background(), domain.VerifyAssertionInput{
			KeyID: keyID, Assertion: []byte("assertion"), ClientData: []byte("client-data"), Challenge: challenge,
		})
		require.ErrorIs(t, err, domain.ErrAttestationAssertionInvalid)
		fx.repo.AssertNotCalled(t, "UpdateKeyCounter", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}
