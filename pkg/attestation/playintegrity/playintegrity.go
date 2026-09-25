// Package playintegrity is the Android attestation.Verifier: it decodes Play
// Integrity tokens on Google's servers using a service account, and returns the
// decoded identity fields (package name, nonce, timestamp, verdicts) for the
// use case to compare against config.
//
// It is server-to-server: unlike the iOS App Attest driver, there is no local
// crypto — the token is a JWE of a JWS that only Google can decode. The caller
// activates this provider by importing this package (its init() registers
// attestation.ProviderPlayIntegrity) and building it through attestation.New.
package playintegrity

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/api/option"
	"google.golang.org/api/playintegrity/v1"

	"chjpcoj.com.innerlamp/pkg/attestation"
)

// playIntegrityScope is the OAuth scope the decode call requires.
const playIntegrityScope = "https://www.googleapis.com/auth/playintegrity"

// Option keys the provider reads from attestation.Config.Options.
const (
	// OptionPackageName is the Play package name the decode call is made for.
	OptionPackageName = "package_name"

	// OptionCredentialsJSON is a service-account JSON key (raw) used to
	// authenticate the decode call.
	OptionCredentialsJSON = "credentials_json"

	// OptionCredentialsFile is a path to a service-account JSON key file. Used
	// only when OptionCredentialsJSON is absent.
	OptionCredentialsFile = "credentials_file"
)

func init() {
	attestation.Register(attestation.ProviderPlayIntegrity, New)
}

// verifier decodes Play Integrity tokens on Google's servers.
type verifier struct {
	packageName     string
	credentialsJSON []byte
	credentialsFile string
}

// New builds the Play Integrity verifier from cfg.Options. It only validates
// config — no network I/O happens here (the service is built lazily per call,
// or eagerly when credentials must be parsed).
func New(cfg *attestation.Config) (attestation.Verifier, error) {
	if cfg == nil {
		return nil, fmt.Errorf("%w: config is required for the playintegrity provider", attestation.ErrInvalidConfig)
	}
	packageName := opt(cfg, OptionPackageName, "")
	if packageName == "" {
		return nil, fmt.Errorf("%w: package_name is required for the playintegrity provider", attestation.ErrInvalidConfig)
	}

	v := &verifier{packageName: packageName}
	if j := opt(cfg, OptionCredentialsJSON, ""); j != "" {
		v.credentialsJSON = []byte(j)
	} else {
		v.credentialsFile = opt(cfg, OptionCredentialsFile, "")
	}
	return v, nil
}

// VerifyAndroid decodes the integrity token and returns the decoded identity
// fields. An error means the decode could not be performed (bad credentials,
// unreachable, timeout, non-2xx) — the use case surfaces it as an upstream
// error and leaves the challenge unconsumed.
func (v *verifier) VerifyAndroid(ctx context.Context, token, _ string) (*attestation.AndroidResult, error) {
	svc, err := v.buildService(ctx)
	if err != nil {
		return nil, fmt.Errorf("create play integrity service: %w", err)
	}

	resp, err := svc.V1.DecodeIntegrityToken(v.packageName, &playintegrity.DecodeIntegrityTokenRequest{
		IntegrityToken: token,
	}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("decode integrity token: %w", err)
	}

	return mapPayload(resp.TokenPayloadExternal)
}

// VerifyIOSAttestation is not implemented by this provider.
func (v *verifier) VerifyIOSAttestation(context.Context, string, []byte, string) (*attestation.IOSAttestationResult, error) {
	return nil, attestation.ErrPlatformNotSupported
}

// VerifyIOSAssertion is not implemented by this provider.
func (v *verifier) VerifyIOSAssertion(context.Context, []byte, []byte, []byte, int64) (int64, error) {
	return 0, attestation.ErrPlatformNotSupported
}

// buildService constructs the Play Integrity client. The scope is always set;
// an explicit service account (JSON or file) is used when configured, otherwise
// Application Default Credentials apply.
func (v *verifier) buildService(ctx context.Context) (*playintegrity.Service, error) {
	opts := []option.ClientOption{option.WithScopes(playIntegrityScope)}
	switch {
	case len(v.credentialsJSON) > 0:
		opts = append(opts, option.WithCredentialsJSON(v.credentialsJSON))
	case v.credentialsFile != "":
		opts = append(opts, option.WithCredentialsFile(v.credentialsFile))
	}
	return playintegrity.NewService(ctx, opts...)
}

// mapPayload flattens a decoded token into the generic AndroidResult shape,
// guarding against the nil-able sub-structs of the API response.
func mapPayload(p *playintegrity.TokenPayloadExternal) (*attestation.AndroidResult, error) {
	if p == nil || p.RequestDetails == nil || p.AppIntegrity == nil {
		return nil, errors.New("integrity token payload missing required fields")
	}

	result := &attestation.AndroidResult{
		PackageName:           p.RequestDetails.RequestPackageName,
		Nonce:                 p.RequestDetails.Nonce,
		TimestampMillis:       p.RequestDetails.TimestampMillis,
		AppRecognitionVerdict: p.AppIntegrity.AppRecognitionVerdict,
	}
	if p.DeviceIntegrity != nil {
		result.DeviceRecognitionVerdict = strings.Join(p.DeviceIntegrity.DeviceRecognitionVerdict, ",")
	}
	if p.AccountDetails != nil {
		result.AppLicensingVerdict = p.AccountDetails.AppLicensingVerdict
	}
	return result, nil
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
