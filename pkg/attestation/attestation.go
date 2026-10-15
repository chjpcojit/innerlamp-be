// Package attestation verifies platform attestation evidence through a
// provider-agnostic interface, following the same Register/New factory pattern
// as pkg/database and pkg/storage: this package declares the interface
// and the registry, a subpackage per provider self-registers via init(), and a
// consumer activates one with a blank import.
//
// The interface mirrors the two-platform asymmetry of attestation:
//   - VerifyAndroid is server-to-server: the provider decodes a Play Integrity
//     token on Google's servers.
//   - VerifyIOSAttestation / VerifyIOSAssertion are local crypto: the provider
//     verifies an App Attest attestation object or an ES256 assertion itself,
//     without calling Apple.
//
// The concrete crypto/CBOR work lives in the provider subpackages
// (playintegrity, appattest); this package has no dependency on any service's
// domain types, and turning the generic result shapes into a service's domain
// types is services/appService/repository's job.
package attestation

import (
	"context"
	"errors"
	"fmt"
)

// Common errors.
var (
	ErrUnsupportedProvider = errors.New("unsupported attestation provider")
	ErrInvalidConfig       = errors.New("invalid attestation configuration")

	// ErrCounterNotIncreased is returned by VerifyIOSAssertion when the
	// assertion's authenticator counter is not strictly greater than the stored
	// one (a replay or rollback).
	ErrCounterNotIncreased = errors.New("assertion counter did not increase")

	// ErrAssertionInvalid is returned by VerifyIOSAssertion when the assertion
	// signature or RP ID does not verify against the stored public key.
	ErrAssertionInvalid = errors.New("assertion is invalid")

	// ErrPlatformNotSupported is returned by a provider for a platform it does
	// not implement — e.g. the playintegrity provider asked to verify an iOS
	// assertion. A provider only answers the platform(s) it knows.
	ErrPlatformNotSupported = errors.New("platform not supported by this provider")
)

// ProviderType selects which registered Verifier implementation New builds.
type ProviderType string

const (
	// ProviderMock is the dev/test implementation in pkg/attestation/mock —
	// it answers both platforms and verifies nothing (it fakes results).
	ProviderMock ProviderType = "mock"

	// ProviderPlayIntegrity is the Android implementation in
	// pkg/attestation/playintegrity — it decodes Play Integrity tokens on
	// Google's servers. It does not answer iOS.
	ProviderPlayIntegrity ProviderType = "playintegrity"

	// ProviderAppAttest is the iOS implementation in pkg/attestation/appattest —
	// it verifies App Attest attestation objects and assertions locally. It
	// does not answer Android.
	ProviderAppAttest ProviderType = "appattest"
)

// Config configures a Verifier. Options carries provider-specific settings
// (e.g. the mock's expected package name / verdict threshold); a provider that
// needs more structure is free to ignore the map and define its own.
type Config struct {
	Provider ProviderType
	Options  map[string]string
}

// AndroidResult is the generic shape of a decoded Play Integrity token.
type AndroidResult struct {
	PackageName              string
	Nonce                    string
	TimestampMillis          int64
	AppRecognitionVerdict    string
	DeviceRecognitionVerdict string
	AppLicensingVerdict      string
}

// IOSAttestationResult is the generic shape of a verified App Attest
// attestation object.
type IOSAttestationResult struct {
	PublicKey   []byte
	Receipt     []byte
	Environment string
}

// Verifier verifies platform attestation evidence. The three methods map to the
// three operations the attestation feature needs; the caller (a service's
// repository adapter) converts these generic shapes into its own domain types.
type Verifier interface {
	// VerifyAndroid decodes the Play Integrity token for token against the
	// challenge nonce. An error means the decode could not be performed
	// (unreachable, timeout, HTTP != 2xx).
	VerifyAndroid(ctx context.Context, token, challenge string) (*AndroidResult, error)

	// VerifyIOSAttestation verifies an App Attest attestation object. An error
	// means the object failed crypto verification.
	VerifyIOSAttestation(ctx context.Context, keyID string, attestation []byte, challenge string) (*IOSAttestationResult, error)

	// VerifyIOSAssertion verifies an App Attest assertion over clientData
	// against publicKey (the stored COSE ES256 key) and returns the
	// authenticator counter it read. It returns ErrCounterNotIncreased when
	// counter <= storedCounter and ErrAssertionInvalid when the signature or RP
	// ID does not verify.
	VerifyIOSAssertion(ctx context.Context, publicKey []byte, assertion, clientData []byte, storedCounter int64) (int64, error)
}

// FactoryFunc builds a Verifier from Config. Registered by each provider
// subpackage's init().
type FactoryFunc func(*Config) (Verifier, error)

var factories = make(map[ProviderType]FactoryFunc)

// Register binds provider to factory. Called from a provider subpackage's
// init(), never from this file — adding a provider must not require editing
// pkg/attestation itself.
func Register(provider ProviderType, factory FactoryFunc) {
	factories[provider] = factory
}

// New builds the Verifier for config.Provider. The caller activates a provider
// with a blank import (e.g. `_ "chjpcoj.com.innerlamp/pkg/attestation/mock"`)
// before calling this — an unregistered provider is ErrUnsupportedProvider, not
// a panic.
func New(config *Config) (Verifier, error) {
	if config == nil {
		return nil, ErrInvalidConfig
	}
	if config.Provider == "" {
		return nil, fmt.Errorf("%w: provider is required", ErrInvalidConfig)
	}

	factory, ok := factories[config.Provider]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedProvider, config.Provider)
	}

	return factory(config)
}
