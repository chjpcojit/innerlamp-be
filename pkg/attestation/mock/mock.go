// Package mock is a ⚠️ DEV-ONLY attestation.Verifier that fakes verification
// results instead of calling Google or Apple. It is the only implementation
// today — wire in the real providers (pkg/attestation/playintegrity,
// pkg/attestation/appattest) before production by adding sibling subpackages
// that register their own attestation.ProviderType; pkg/attestation.Verifier
// does not change to support them.
package mock

import (
	"context"
	"fmt"
	"time"

	"chjpcoj.com.innerlamp/pkg/attestation"
)

// Option keys the mock reads from attestation.Config.Options. They are exported
// so the composition root can wire the mock's echo values (package name,
// verdict threshold) from the same config the use case compares against.
const (
	OptionMode                     = "mode"
	OptionAssertionMode            = "assertion_mode"
	OptionPackageName              = "package_name"
	OptionAppRecognitionVerdict    = "app_recognition_verdict"
	OptionDeviceRecognitionVerdict = "device_recognition_verdict"
	OptionAppLicensingVerdict      = "app_licensing_verdict"
	OptionIOSEnvironment           = "ios_environment"
)

// Mode values for OptionMode.
const (
	// ModePass — everything verifies (the default).
	ModePass = "pass"
	// ModeFail — android returns a non-recognized verdict, iOS attest/assert
	// return a crypto-failure error (so verify yields attested:false and the
	// assertion middleware yields 401).
	ModeFail = "fail"
	// ModeUpstreamError — android returns an error (so verify yields 502),
	// iOS still verifies.
	ModeUpstreamError = "upstream_error"
)

// Assertion mode values for OptionAssertionMode.
const (
	AssertionModePass    = "pass"
	AssertionModeInvalid = "invalid"
	AssertionModeReplay  = "replay"
)

func init() {
	attestation.Register(attestation.ProviderMock, New)
}

// mockVerifier fakes every verification. It echoes the configured package name
// and verdicts back so a dev E2E round-trip (challenge → verify) can produce
// attested:true without Google or Apple.
type mockVerifier struct {
	packageName              string
	appRecognitionVerdict    string
	deviceRecognitionVerdict string
	appLicensingVerdict      string
	iosEnvironment           string
	mode                     string
	assertionMode            string
}

// New builds the mock Verifier from cfg.Options. A nil config is a caller
// error; all options fall back to sensible pass-mode defaults.
func New(cfg *attestation.Config) (attestation.Verifier, error) {
	if cfg == nil {
		return nil, fmt.Errorf("%w: config is required for the mock attestation provider", attestation.ErrInvalidConfig)
	}
	return &mockVerifier{
		packageName:              opt(cfg, OptionPackageName, "com.innerlamp.app"),
		appRecognitionVerdict:    opt(cfg, OptionAppRecognitionVerdict, "PLAY_RECOGNIZED"),
		deviceRecognitionVerdict: opt(cfg, OptionDeviceRecognitionVerdict, "MEETS_DEVICE_INTEGRITY"),
		appLicensingVerdict:      opt(cfg, OptionAppLicensingVerdict, "LICENSED"),
		iosEnvironment:           opt(cfg, OptionIOSEnvironment, "production"),
		mode:                     opt(cfg, OptionMode, ModePass),
		assertionMode:            opt(cfg, OptionAssertionMode, AssertionModePass),
	}, nil
}

// VerifyAndroid echoes a decode result that matches the configured package and
// the presented challenge, so the use case's identity checks pass in "pass"
// mode. In "fail" mode the app recognition verdict is downgraded; in
// "upstream_error" mode it simulates Google being unreachable.
func (v *mockVerifier) VerifyAndroid(_ context.Context, _ string, challenge string) (*attestation.AndroidResult, error) {
	if v.mode == ModeUpstreamError {
		return nil, fmt.Errorf("mock: play integrity decode unavailable")
	}

	verdict := v.appRecognitionVerdict
	if v.mode == ModeFail {
		verdict = "UNRECOGNIZED_VERSION"
	}

	return &attestation.AndroidResult{
		PackageName:              v.packageName,
		Nonce:                    challenge,
		TimestampMillis:          time.Now().UnixMilli(),
		AppRecognitionVerdict:    verdict,
		DeviceRecognitionVerdict: v.deviceRecognitionVerdict,
		AppLicensingVerdict:      v.appLicensingVerdict,
	}, nil
}

// VerifyIOSAttestation returns a synthetic key in "pass" mode and a crypto
// failure in "fail" mode.
func (v *mockVerifier) VerifyIOSAttestation(_ context.Context, keyID string, _ []byte, _ string) (*attestation.IOSAttestationResult, error) {
	if v.mode == ModeFail {
		return nil, attestation.ErrAssertionInvalid
	}
	return &attestation.IOSAttestationResult{
		PublicKey:   []byte("mock-public-key-" + keyID),
		Receipt:     []byte("mock-receipt"),
		Environment: v.iosEnvironment,
	}, nil
}

// VerifyIOSAssertion returns storedCounter+1 in "pass" mode, a crypto failure in
// "invalid" mode, and a counter replay in "replay" mode.
func (v *mockVerifier) VerifyIOSAssertion(_ context.Context, _ []byte, _, _ []byte, storedCounter int64) (int64, error) {
	switch v.assertionMode {
	case AssertionModeInvalid:
		return 0, attestation.ErrAssertionInvalid
	case AssertionModeReplay:
		return storedCounter, attestation.ErrCounterNotIncreased
	}
	if v.mode == ModeFail {
		return 0, attestation.ErrAssertionInvalid
	}
	return storedCounter + 1, nil
}

// opt returns cfg.Options[key] if present, else fallback.
func opt(cfg *attestation.Config, key, fallback string) string {
	if cfg.Options == nil {
		return fallback
	}
	if v, ok := cfg.Options[key]; ok {
		return v
	}
	return fallback
}
