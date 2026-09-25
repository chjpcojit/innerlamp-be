// Package appattest is the iOS attestation.Verifier: it verifies App Attest
// attestation objects and per-request assertions locally, without calling
// Apple. It follows Apple's "Attestation Object Validation Guide".
//
// An attestation object (the `attest` phase, one-time key registration) is a
// CBOR structure carrying the x5c certificate chain, a receipt, and the
// authenticator data; the driver checks the chain against Apple's App Attest
// root CA, the nonce, the RP ID hash, the counter, the aaguid environment and
// the public-key hash. An assertion (the per-request phase) is a CBOR structure
// carrying an ES256 signature over authenticatorData || SHA256(clientData);
// the driver verifies it against the stored public key and the replay counter.
//
// The caller activates this provider by importing this package (its init()
// registers attestation.ProviderAppAttest) and building it via attestation.New.
package appattest

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/fxamacker/cbor/v2"

	"chjpcoj.com.innerlamp/pkg/attestation"
)

// Option keys the provider reads from attestation.Config.Options.
const (
	// OptionAppID is the {team_id}.{bundle_id} the RP ID hash is computed from —
	// SHA256(app_id) must equal the authenticator data's rpIdHash.
	OptionAppID = "app_id"

	// OptionRootCertPEM overrides the embedded Apple App Attestation Root CA,
	// for tests and for rotating the root without a code change.
	OptionRootCertPEM = "root_cert_pem"
)

// nonceExtensionOID is the OID of the credCert extension carrying the
// attestation nonce (Apple App Attest).
var nonceExtensionOID = asn1.ObjectIdentifier{1, 2, 840, 113635, 100, 8, 2}

// aaguidProduction is the aaguid Apple writes in the production environment:
// the ASCII "appattest" followed by seven 0x00 bytes.
var aaguidProduction = [16]byte{'a', 'p', 'p', 'a', 't', 't', 'e', 's', 't', 0, 0, 0, 0, 0, 0, 0}

// aaguidDevelopment is the aaguid Apple writes in the development (sandbox)
// environment: the ASCII "appattestdevelop".
var aaguidDevelopment = [16]byte{'a', 'p', 'p', 'a', 't', 't', 'e', 's', 't', 'd', 'e', 'v', 'e', 'l', 'o', 'p'}

// appleRootPEM is Apple's App Attestation Root CA certificate — the public
// trust anchor for the x5c chain. It is public, not a secret, and is embedded
// so production verification needs no out-of-band file. OptionRootCertPEM can
// override it (tests, rotation).
const appleRootPEM = `-----BEGIN CERTIFICATE-----
MIICITCCAaegAwIBAgIQC/O+DvHN0uD7jG5yH2IXmDAKBggqhkjOPQQDAzBSMSYw
JAYDVQQDDB1BcHBsZSBBcHAgQXR0ZXN0YXRpb24gUm9vdCBDQTETMBEGA1UECgwK
QXBwbGUgSW5jLjETMBEGA1UECAwKQ2FsaWZvcm5pYTAeFw0yMDAzMTgxODMyNTNa
Fw00NTAzMTUwMDAwMDBaMFIxJjAkBgNVBAMMHUFwcGxlIEFwcCBBdHRlc3RhdGlv
biBSb290IENBMRMwEQYDVQQKDApBcHBsZSBJbmMuMRMwEQYDVQQIDApDYWxpZm9y
bmlhMHYwEAYHKoZIzj0CAQYFK4EEACIDYgAERTHhmLW07ATaFQIEVwTtT4dyctdh
NbJhFs/Ii2FdCgAHGbpphY3+d8qjuDngIN3WVhQUBHAoMeQ/cLiP1sOUtgjqK9au
Yen1mMEvRq9Sk3Jm5X8U62H+xTD3FE9TgS41o0IwQDAPBgNVHRMBAf8EBTADAQH/
MB0GA1UdDgQWBBSskRBTM72+aEH/pwyp5frq5eWKoTAOBgNVHQ8BAf8EBAMCAQYw
CgYIKoZIzj0EAwMDaAAwZQIwQgFGnByvsiVbpTKwSga0kP0e8EeDS4+sQmTvb7vn
53O5+FRXgeLhpJ06ysC5PrOyAjEAp5U4xDgEgllF7En3VcE3iexZZtKeYnpqtijV
oyFraWVIyd/dganmrduC1bmTBGwD
-----END CERTIFICATE-----`

func init() {
	attestation.Register(attestation.ProviderAppAttest, New)
}

// verifier verifies App Attest evidence locally.
type verifier struct {
	appID string
	root  *x509.Certificate
}

// New builds the App Attest verifier from cfg.Options. app_id is required; the
// root CA defaults to the embedded Apple root and can be overridden. It parses
// the root eagerly (config validation) but does no network I/O.
func New(cfg *attestation.Config) (attestation.Verifier, error) {
	if cfg == nil {
		return nil, fmt.Errorf("%w: config is required for the appattest provider", attestation.ErrInvalidConfig)
	}
	appID := opt(cfg, OptionAppID, "")
	if appID == "" {
		return nil, fmt.Errorf("%w: app_id is required for the appattest provider", attestation.ErrInvalidConfig)
	}

	rootPEM := opt(cfg, OptionRootCertPEM, appleRootPEM)
	root, err := parseRootCert(rootPEM)
	if err != nil {
		return nil, fmt.Errorf("%w: parse root cert: %v", attestation.ErrInvalidConfig, err)
	}

	return &verifier{appID: appID, root: root}, nil
}

// VerifyAndroid is not implemented by this provider.
func (v *verifier) VerifyAndroid(context.Context, string, string) (*attestation.AndroidResult, error) {
	return nil, attestation.ErrPlatformNotSupported
}

// VerifyIOSAttestation verifies an App Attest attestation object (the `attest`
// phase). On success it returns the ES256 public key to store for later
// assertions (X9.62 uncompressed point), the receipt, and the environment.
func (v *verifier) VerifyIOSAttestation(_ context.Context, keyID string, attestationObj []byte, challenge string) (*attestation.IOSAttestationResult, error) {
	obj, err := parseAttestationObject(attestationObj)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", attestation.ErrAssertionInvalid, err)
	}
	if obj.Fmt != "apple-appattest" {
		return nil, fmt.Errorf("%w: unexpected fmt %q", attestation.ErrAssertionInvalid, obj.Fmt)
	}
	if len(obj.AttStmt.X5C) < 1 {
		return nil, fmt.Errorf("%w: x5c chain is empty", attestation.ErrAssertionInvalid)
	}

	// (b) verify the cert chain up to Apple's root.
	leaf, err := x509.ParseCertificate(obj.AttStmt.X5C[0])
	if err != nil {
		return nil, fmt.Errorf("%w: parse leaf cert: %v", attestation.ErrAssertionInvalid, err)
	}
	if err := v.verifyCertChain(obj.AttStmt.X5C); err != nil {
		return nil, err
	}

	// Parse the authenticator data.
	authData, err := parseAuthData(obj.AuthData)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", attestation.ErrAssertionInvalid, err)
	}

	// (c) nonce = SHA256(authData || SHA256(challenge)); compare to the cert.
	expectedNonce, err := certNonce(leaf)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", attestation.ErrAssertionInvalid, err)
	}
	if !bytes.Equal(attestationNonce(obj.AuthData, challenge), expectedNonce) {
		return nil, fmt.Errorf("%w: nonce mismatch", attestation.ErrAssertionInvalid)
	}

	// (d) SHA256(app_id) == rpIdHash.
	if !v.matchesRPID(authData.RPIDHash) {
		return nil, fmt.Errorf("%w: rp id hash mismatch", attestation.ErrAssertionInvalid)
	}

	// (e) counter must be zero at registration.
	if authData.Counter != 0 {
		return nil, fmt.Errorf("%w: counter is not zero", attestation.ErrAssertionInvalid)
	}

	// (f) aaguid -> environment.
	env, err := aaguidEnvironment(authData.AAGUID)
	if err != nil {
		return nil, err
	}

	// (g) the credential public key's hash must equal the key id.
	publicKey, err := parseCOSEKey(authData.CredentialPublicKey)
	if err != nil {
		return nil, fmt.Errorf("%w: parse cose key: %v", attestation.ErrAssertionInvalid, err)
	}
	if !verifyKeyID(publicKey, keyID) {
		return nil, fmt.Errorf("%w: public key hash does not match key id", attestation.ErrAssertionInvalid)
	}
	// The leaf cert must carry the same public key as the credential.
	if !leaf.PublicKey.(*ecdsa.PublicKey).Equal(publicKey) {
		return nil, fmt.Errorf("%w: cert and credential public key differ", attestation.ErrAssertionInvalid)
	}

	return &attestation.IOSAttestationResult{
		PublicKey:   x962(publicKey),
		Receipt:     obj.AttStmt.Receipt,
		Environment: env,
	}, nil
}

// VerifyIOSAssertion verifies an App Attest assertion over clientData against
// publicKey (the stored X9.62 point). It returns the authenticator counter it
// read, or ErrCounterNotIncreased / ErrAssertionInvalid.
func (v *verifier) VerifyIOSAssertion(_ context.Context, publicKey []byte, assertionObj, clientData []byte, storedCounter int64) (int64, error) {
	var a assertion
	if err := cbor.Unmarshal(assertionObj, &a); err != nil {
		return 0, fmt.Errorf("%w: decode assertion: %v", attestation.ErrAssertionInvalid, err)
	}

	pub, err := x962ToPublicKey(publicKey)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", attestation.ErrAssertionInvalid, err)
	}

	// (b)/(c) verify the ES256 signature over authenticatorData || SHA256(clientData).
	clientDataHash := sha256.Sum256(clientData)
	message := append(append([]byte{}, a.AuthenticatorData...), clientDataHash[:]...)
	digest := sha256.Sum256(message)
	if !ecdsa.VerifyASN1(pub, digest[:], a.Signature) {
		return 0, fmt.Errorf("%w: signature verification failed", attestation.ErrAssertionInvalid)
	}

	// (d)/(e) RP ID hash + counter.
	authData, err := parseAuthData(a.AuthenticatorData)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", attestation.ErrAssertionInvalid, err)
	}
	if !v.matchesRPID(authData.RPIDHash) {
		return 0, fmt.Errorf("%w: rp id hash mismatch", attestation.ErrAssertionInvalid)
	}
	counter := int64(authData.Counter)
	if counter <= storedCounter {
		return 0, attestation.ErrCounterNotIncreased
	}

	return counter, nil
}

// --- pure helpers ------------------------------------------------------------

// attestationObject is the CBOR structure returned by DCAppAttestService.attestKey.
type attestationObject struct {
	Fmt      string          `cbor:"fmt"`
	AttStmt  attestationStmt `cbor:"attStmt"`
	AuthData []byte          `cbor:"authData"`
}

type attestationStmt struct {
	X5C     [][]byte `cbor:"x5c"`
	Receipt []byte   `cbor:"receipt"`
}

// assertion is the CBOR structure returned by DCAppAttestService.generateAssertion.
type assertion struct {
	Signature         []byte `cbor:"signature"`
	AuthenticatorData []byte `cbor:"authenticatorData"`
}

// coseKey is the COSE_Key structure of the credential public key.
type coseKey struct {
	Kty int64  `cbor:"1,keyasint,omitempty"`
	Alg int64  `cbor:"3,keyasint,omitempty"`
	Crv int64  `cbor:"-1,keyasint,omitempty"`
	X   []byte `cbor:"-2,keyasint,omitempty"`
	Y   []byte `cbor:"-3,keyasint,omitempty"`
}

// authData is the parsed WebAuthn authenticator data.
type authData struct {
	RPIDHash            []byte
	Flags               byte
	Counter             uint32
	AAGUID              []byte
	CredentialID        []byte
	CredentialPublicKey []byte
}

func parseAttestationObject(b []byte) (*attestationObject, error) {
	var obj attestationObject
	if err := cbor.Unmarshal(b, &obj); err != nil {
		return nil, fmt.Errorf("decode attestation object: %w", err)
	}
	return &obj, nil
}

// parseAuthData parses the WebAuthn authenticator data. When the AT flag is set
// (attested credential data present), it also extracts the aaguid, credential
// id and COSE public key.
func parseAuthData(b []byte) (*authData, error) {
	if len(b) < 37 {
		return nil, errors.New("authenticator data too short")
	}
	ad := &authData{
		RPIDHash: b[0:32],
		Flags:    b[32],
		Counter:  uint32(b[33])<<24 | uint32(b[34])<<16 | uint32(b[35])<<8 | uint32(b[36]),
	}
	if ad.Flags&0x40 != 0 { // AT flag
		rest := b[37:]
		if len(rest) < 18 {
			return nil, errors.New("attested credential data too short")
		}
		ad.AAGUID = rest[0:16]
		credIDLen := int(rest[16])<<8 | int(rest[17])
		rest = rest[18:]
		if len(rest) < credIDLen {
			return nil, errors.New("credential id truncated")
		}
		ad.CredentialID = rest[:credIDLen]
		ad.CredentialPublicKey = rest[credIDLen:]
	}
	return ad, nil
}

// parseCOSEKey decodes a COSE_Key (EC2/ES256/P-256) into an ECDSA public key.
func parseCOSEKey(b []byte) (*ecdsa.PublicKey, error) {
	var k coseKey
	if err := cbor.Unmarshal(b, &k); err != nil {
		return nil, fmt.Errorf("decode cose key: %w", err)
	}
	if k.Kty != 2 || k.Crv != 1 {
		return nil, fmt.Errorf("unsupported cose key: kty=%d crv=%d", k.Kty, k.Crv)
	}
	if len(k.X) == 0 || len(k.Y) == 0 {
		return nil, errors.New("cose key missing coordinates")
	}
	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(k.X),
		Y:     new(big.Int).SetBytes(k.Y),
	}, nil
}

// verifyCertChain verifies the leaf credCert and intermediate cert in x5c
// against Apple's root. It verifies the chain by hand (leaf signed by the
// intermediate, intermediate signed by the root, validity window, CA flag)
// rather than with x509.Verify: the credCert carries Apple's nonce extension
// marked critical, which x509.Verify rejects as "unhandled critical extension",
// but App Attest verification must accept it.
func (v *verifier) verifyCertChain(x5c [][]byte) error {
	if len(x5c) < 2 {
		return fmt.Errorf("%w: x5c must contain leaf and intermediate", attestation.ErrAssertionInvalid)
	}
	leaf, err := x509.ParseCertificate(x5c[0])
	if err != nil {
		return fmt.Errorf("%w: parse leaf: %v", attestation.ErrAssertionInvalid, err)
	}
	intermediate, err := x509.ParseCertificate(x5c[1])
	if err != nil {
		return fmt.Errorf("%w: parse intermediate: %v", attestation.ErrAssertionInvalid, err)
	}

	now := time.Now()
	for _, c := range []*x509.Certificate{leaf, intermediate} {
		if now.Before(c.NotBefore) || now.After(c.NotAfter) {
			return fmt.Errorf("%w: certificate not valid at this time", attestation.ErrAssertionInvalid)
		}
	}
	if !intermediate.IsCA {
		return fmt.Errorf("%w: intermediate is not a CA", attestation.ErrAssertionInvalid)
	}
	if err := leaf.CheckSignatureFrom(intermediate); err != nil {
		return fmt.Errorf("%w: leaf not signed by intermediate: %v", attestation.ErrAssertionInvalid, err)
	}
	if err := intermediate.CheckSignatureFrom(v.root); err != nil {
		return fmt.Errorf("%w: intermediate not signed by root: %v", attestation.ErrAssertionInvalid, err)
	}
	return nil
}

// certNonce extracts the attestation nonce from the credCert's nonce extension.
func certNonce(cert *x509.Certificate) ([]byte, error) {
	for _, ext := range cert.Extensions {
		if ext.Id.Equal(nonceExtensionOID) {
			var nonce []byte
			rest, err := asn1.Unmarshal(ext.Value, &nonce)
			if err != nil {
				return nil, fmt.Errorf("decode nonce extension: %w", err)
			}
			if len(rest) != 0 {
				return nil, errors.New("trailing data after nonce extension")
			}
			return nonce, nil
		}
	}
	return nil, errors.New("nonce extension missing")
}

// attestationNonce computes SHA256(authData || SHA256(challenge)).
func attestationNonce(authData []byte, challenge string) []byte {
	clientDataHash := sha256.Sum256([]byte(challenge))
	nonce := sha256.Sum256(append(append([]byte{}, authData...), clientDataHash[:]...))
	return nonce[:]
}

// matchesRPID reports whether the authenticator data's rpIdHash equals
// SHA256(app_id).
func (v *verifier) matchesRPID(rpIDHash []byte) bool {
	expected := sha256.Sum256([]byte(v.appID))
	return bytes.Equal(expected[:], rpIDHash)
}

// aaguidEnvironment maps the aaguid to the App Attest environment.
func aaguidEnvironment(aaguid []byte) (string, error) {
	if len(aaguid) != 16 {
		return "", fmt.Errorf("%w: invalid aaguid length", attestation.ErrAssertionInvalid)
	}
	switch {
	case bytes.Equal(aaguid, aaguidProduction[:]):
		return "production", nil
	case bytes.Equal(aaguid, aaguidDevelopment[:]):
		return "development", nil
	default:
		return "", fmt.Errorf("%w: unknown aaguid", attestation.ErrAssertionInvalid)
	}
}

// x962 returns the uncompressed (X9.62) encoding of the public key: 0x04 || X || Y.
func x962(pub *ecdsa.PublicKey) []byte {
	return elliptic.Marshal(pub.Curve, pub.X, pub.Y)
}

// x962ToPublicKey reverses x962.
func x962ToPublicKey(b []byte) (*ecdsa.PublicKey, error) {
	x, y := elliptic.Unmarshal(elliptic.P256(), b)
	if x == nil {
		return nil, errors.New("invalid public key point")
	}
	return &ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}, nil
}

// verifyKeyID checks that SHA256 of the public key's X9.62 encoding equals the
// base64 key id (Apple's credential id == key id rule).
func verifyKeyID(pub *ecdsa.PublicKey, keyID string) bool {
	hash := sha256.Sum256(x962(pub))
	decoded, err := base64.StdEncoding.DecodeString(keyID)
	if err != nil {
		return false
	}
	return bytes.Equal(hash[:], decoded)
}

// parseRootCert parses a PEM root certificate.
func parseRootCert(rootPEM string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(rootPEM))
	if block == nil {
		return nil, errors.New("no PEM block in root certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse root certificate: %w", err)
	}
	return cert, nil
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
