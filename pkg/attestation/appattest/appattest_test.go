package appattest

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/attestation"
)

const testAppID = "TEAMID.com.example.app"

// --- cert chain generation (test fixtures) ----------------------------------

func generateRootCA(t *testing.T) (der []byte, cert *x509.Certificate, key *ecdsa.PrivateKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test App Attest Root"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	der, err = x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err = x509.ParseCertificate(der)
	require.NoError(t, err)
	return der, cert, key
}

func generateIntermediate(t *testing.T, root *x509.Certificate, rootKey *ecdsa.PrivateKey) (der []byte, cert *x509.Certificate, key *ecdsa.PrivateKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(2),
		Subject:               pkix.Name{CommonName: "Test App Attest Intermediate"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	der, err = x509.CreateCertificate(rand.Reader, tmpl, root, &key.PublicKey, rootKey)
	require.NoError(t, err)
	cert, err = x509.ParseCertificate(der)
	require.NoError(t, err)
	return der, cert, key
}

func generateLeaf(t *testing.T, intermediate *x509.Certificate, intermediateKey, leafKey *ecdsa.PrivateKey, nonce []byte) (der []byte, cert *x509.Certificate) {
	t.Helper()
	nonceASN1, err := asn1.Marshal(nonce)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: "Test Cred Cert"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtraExtensions: []pkix.Extension{
			{Id: nonceExtensionOID, Critical: true, Value: nonceASN1},
		},
	}
	der, err = x509.CreateCertificate(rand.Reader, tmpl, intermediate, &leafKey.PublicKey, intermediateKey)
	require.NoError(t, err)
	cert, err = x509.ParseCertificate(der)
	require.NoError(t, err)
	return der, cert
}

func rootPEM(der []byte) string {
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

// --- authenticator data / COSE key fixtures -------------------------------

func buildCOSEKey(pub *ecdsa.PublicKey) []byte {
	point := elliptic.Marshal(elliptic.P256(), pub.X, pub.Y) // 0x04 || x || y
	k := coseKey{Kty: 2, Alg: -7, Crv: 1, X: point[1:33], Y: point[33:65]}
	b, err := cbor.Marshal(k)
	if err != nil {
		panic(err)
	}
	return b
}

// buildAuthData builds an attestation authData (AT flag set, attested credential
// data present) for the given leaf public key.
func buildAuthData(appID string, counter uint32, aaguid []byte, leafPub *ecdsa.PublicKey) (authDataBytes, credentialID, keyID []byte) {
	rpIDHash := sha256.Sum256([]byte(appID))
	point := elliptic.Marshal(elliptic.P256(), leafPub.X, leafPub.Y)
	credentialID = make([]byte, 32)
	sha := sha256.Sum256(point)
	copy(credentialID, sha[:])
	keyID = []byte(base64.StdEncoding.EncodeToString(credentialID))

	cose := buildCOSEKey(leafPub)

	var b bytes.Buffer
	b.Write(rpIDHash[:])
	b.WriteByte(0x41) // UP + AT
	b.Write([]byte{byte(counter >> 24), byte(counter >> 16), byte(counter >> 8), byte(counter)})
	b.Write(aaguid)
	b.Write([]byte{byte(len(credentialID) >> 8), byte(len(credentialID))})
	b.Write(credentialID)
	b.Write(cose)
	return b.Bytes(), credentialID, keyID
}

// buildAssertionAuthData builds a bare authenticator data (no AT flag) for an
// assertion.
func buildAssertionAuthData(appID string, counter uint32) []byte {
	rpIDHash := sha256.Sum256([]byte(appID))
	var b bytes.Buffer
	b.Write(rpIDHash[:])
	b.WriteByte(0x01) // UP only
	b.Write([]byte{byte(counter >> 24), byte(counter >> 16), byte(counter >> 8), byte(counter)})
	return b.Bytes()
}

// newTestVerifier builds a verifier rooted at a freshly generated test root.
func newTestVerifier(t *testing.T) (*verifier, *x509.Certificate, *ecdsa.PrivateKey) {
	t.Helper()
	rootDER, rootCert, rootKey := generateRootCA(t)
	v, err := New(&attestation.Config{
		Provider: attestation.ProviderAppAttest,
		Options: map[string]string{
			OptionAppID:       testAppID,
			OptionRootCertPEM: rootPEM(rootDER),
		},
	})
	require.NoError(t, err)
	return v.(*verifier), rootCert, rootKey
}

// --- attestation tests -------------------------------------------------------

func TestVerifyIOSAttestation_Success(t *testing.T) {
	v, root, rootKey := newTestVerifier(t)
	intDER, intCert, intKey := generateIntermediate(t, root, rootKey)

	challenge := "nonce-base64"
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	authDataBytes, _, keyID := buildAuthData(testAppID, 0, aaguidProduction[:], &leafKey.PublicKey)

	nonce := attestationNonce(authDataBytes, challenge)
	leafDER, _ := generateLeaf(t, intCert, intKey, leafKey, nonce)

	obj := attestationObject{
		Fmt:      "apple-appattest",
		AttStmt:  attestationStmt{X5C: [][]byte{leafDER, intDER}, Receipt: []byte("receipt")},
		AuthData: authDataBytes,
	}
	objBytes, err := cbor.Marshal(obj)
	require.NoError(t, err)

	result, err := v.VerifyIOSAttestation(context.Background(), string(keyID), objBytes, challenge)
	require.NoError(t, err)
	require.Equal(t, "production", result.Environment)
	require.Equal(t, []byte("receipt"), result.Receipt)
	require.Equal(t, elliptic.Marshal(elliptic.P256(), leafKey.X, leafKey.Y), result.PublicKey)
}

func TestVerifyIOSAttestation_DevelopmentEnvironment(t *testing.T) {
	v, root, rootKey := newTestVerifier(t)
	intDER, intCert, intKey := generateIntermediate(t, root, rootKey)

	challenge := "nonce"
	leafKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	authDataBytes, _, keyID := buildAuthData(testAppID, 0, aaguidDevelopment[:], &leafKey.PublicKey)

	nonce := attestationNonce(authDataBytes, challenge)
	leafDER, _ := generateLeaf(t, intCert, intKey, leafKey, nonce)

	obj := attestationObject{Fmt: "apple-appattest", AttStmt: attestationStmt{X5C: [][]byte{leafDER, intDER}}, AuthData: authDataBytes}
	objBytes, _ := cbor.Marshal(obj)

	result, err := v.VerifyIOSAttestation(context.Background(), string(keyID), objBytes, challenge)
	require.NoError(t, err)
	require.Equal(t, "development", result.Environment)
}

func TestVerifyIOSAttestation_Failures(t *testing.T) {
	setup := func(t *testing.T, challenge string, counter uint32, aaguid []byte) (*verifier, []byte, []byte, func(func(*attestationObject)) []byte) {
		v, root, rootKey := newTestVerifier(t)
		intDER, intCert, intKey := generateIntermediate(t, root, rootKey)
		leafKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		authDataBytes, _, keyID := buildAuthData(testAppID, counter, aaguid, &leafKey.PublicKey)
		nonce := attestationNonce(authDataBytes, challenge)
		leafDER, _ := generateLeaf(t, intCert, intKey, leafKey, nonce)
		build := func(override func(*attestationObject)) []byte {
			obj := &attestationObject{Fmt: "apple-appattest", AttStmt: attestationStmt{X5C: [][]byte{leafDER, intDER}}, AuthData: authDataBytes}
			override(obj)
			b, _ := cbor.Marshal(obj)
			return b
		}
		return v, keyID, authDataBytes, build
	}

	t.Run("wrong fmt", func(t *testing.T) {
		v, keyID, _, build := setup(t, "c", 0, aaguidProduction[:])
		_, err := v.VerifyIOSAttestation(context.Background(), string(keyID), build(func(o *attestationObject) { o.Fmt = "none" }), "c")
		require.ErrorIs(t, err, attestation.ErrAssertionInvalid)
	})

	t.Run("counter is not zero", func(t *testing.T) {
		v, keyID, _, build := setup(t, "c", 1, aaguidProduction[:])
		_, err := v.VerifyIOSAttestation(context.Background(), string(keyID), build(func(o *attestationObject) {}), "c")
		require.ErrorIs(t, err, attestation.ErrAssertionInvalid)
	})

	t.Run("key id mismatch", func(t *testing.T) {
		v, _, _, build := setup(t, "c", 0, aaguidProduction[:])
		_, err := v.VerifyIOSAttestation(context.Background(), base64.StdEncoding.EncodeToString(make([]byte, 32)), build(func(o *attestationObject) {}), "c")
		require.ErrorIs(t, err, attestation.ErrAssertionInvalid)
	})

	t.Run("wrong app id (rp id hash mismatch)", func(t *testing.T) {
		v, root, rootKey := newTestVerifier(t)
		intDER, intCert, intKey := generateIntermediate(t, root, rootKey)
		leafKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		// authData built for a DIFFERENT app id than the verifier's testAppID.
		authDataBytes, _, keyID := buildAuthData("OTHER.com.example.app", 0, aaguidProduction[:], &leafKey.PublicKey)
		nonce := attestationNonce(authDataBytes, "c")
		leafDER, _ := generateLeaf(t, intCert, intKey, leafKey, nonce)
		obj := attestationObject{Fmt: "apple-appattest", AttStmt: attestationStmt{X5C: [][]byte{leafDER, intDER}}, AuthData: authDataBytes}
		b, _ := cbor.Marshal(obj)
		_, err := v.VerifyIOSAttestation(context.Background(), string(keyID), b, "c")
		require.ErrorIs(t, err, attestation.ErrAssertionInvalid)
	})
}

func TestVerifyIOSAssertion(t *testing.T) {
	newVerifierWithAppID := func(t *testing.T, appID string) *verifier {
		rootDER, _, _ := generateRootCA(t)
		v, err := New(&attestation.Config{
			Provider: attestation.ProviderAppAttest,
			Options:  map[string]string{OptionAppID: appID, OptionRootCertPEM: rootPEM(rootDER)},
		})
		require.NoError(t, err)
		return v.(*verifier)
	}

	t.Run("success returns the counter", func(t *testing.T) {
		v := newVerifierWithAppID(t, testAppID)
		key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		pubBytes := elliptic.Marshal(elliptic.P256(), key.X, key.Y)

		clientData := []byte("challenge||POST||/path||bodyhash")
		authenticatorData := buildAssertionAuthData(testAppID, 7)
		clientDataHash := sha256.Sum256(clientData)
		message := append(append([]byte{}, authenticatorData...), clientDataHash[:]...)
		digest := sha256.Sum256(message)
		sig, err := ecdsa.SignASN1(rand.Reader, key, digest[:])
		require.NoError(t, err)

		a := assertion{Signature: sig, AuthenticatorData: authenticatorData}
		aBytes, _ := cbor.Marshal(a)

		counter, err := v.VerifyIOSAssertion(context.Background(), pubBytes, aBytes, clientData, 6)
		require.NoError(t, err)
		require.Equal(t, int64(7), counter)
	})

	t.Run("counter not increased", func(t *testing.T) {
		v := newVerifierWithAppID(t, testAppID)
		key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		pubBytes := elliptic.Marshal(elliptic.P256(), key.X, key.Y)

		clientData := []byte("cd")
		authenticatorData := buildAssertionAuthData(testAppID, 5)
		clientDataHash := sha256.Sum256(clientData)
		message := append(append([]byte{}, authenticatorData...), clientDataHash[:]...)
		digest := sha256.Sum256(message)
		sig, _ := ecdsa.SignASN1(rand.Reader, key, digest[:])

		a := assertion{Signature: sig, AuthenticatorData: authenticatorData}
		aBytes, _ := cbor.Marshal(a)

		_, err := v.VerifyIOSAssertion(context.Background(), pubBytes, aBytes, clientData, 5)
		require.ErrorIs(t, err, attestation.ErrCounterNotIncreased)
	})

	t.Run("tampered client data fails the signature", func(t *testing.T) {
		v := newVerifierWithAppID(t, testAppID)
		key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		pubBytes := elliptic.Marshal(elliptic.P256(), key.X, key.Y)

		clientData := []byte("cd")
		authenticatorData := buildAssertionAuthData(testAppID, 5)
		clientDataHash := sha256.Sum256(clientData)
		message := append(append([]byte{}, authenticatorData...), clientDataHash[:]...)
		digest := sha256.Sum256(message)
		sig, _ := ecdsa.SignASN1(rand.Reader, key, digest[:])

		a := assertion{Signature: sig, AuthenticatorData: authenticatorData}
		aBytes, _ := cbor.Marshal(a)

		_, err := v.VerifyIOSAssertion(context.Background(), pubBytes, aBytes, []byte("other"), 4)
		require.ErrorIs(t, err, attestation.ErrAssertionInvalid)
	})

	t.Run("wrong app id (rp id hash mismatch)", func(t *testing.T) {
		v := newVerifierWithAppID(t, testAppID)
		key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		pubBytes := elliptic.Marshal(elliptic.P256(), key.X, key.Y)

		clientData := []byte("cd")
		authenticatorData := buildAssertionAuthData("OTHER.com.example.app", 5) // different app id
		clientDataHash := sha256.Sum256(clientData)
		message := append(append([]byte{}, authenticatorData...), clientDataHash[:]...)
		digest := sha256.Sum256(message)
		sig, _ := ecdsa.SignASN1(rand.Reader, key, digest[:])

		a := assertion{Signature: sig, AuthenticatorData: authenticatorData}
		aBytes, _ := cbor.Marshal(a)

		_, err := v.VerifyIOSAssertion(context.Background(), pubBytes, aBytes, clientData, 4)
		require.ErrorIs(t, err, attestation.ErrAssertionInvalid)
	})
}

func TestNew_Validation(t *testing.T) {
	t.Run("missing app id", func(t *testing.T) {
		_, err := New(&attestation.Config{Provider: attestation.ProviderAppAttest})
		require.ErrorIs(t, err, attestation.ErrInvalidConfig)
	})
	t.Run("nil config", func(t *testing.T) {
		_, err := New(nil)
		require.ErrorIs(t, err, attestation.ErrInvalidConfig)
	})
	t.Run("invalid root cert", func(t *testing.T) {
		_, err := New(&attestation.Config{
			Provider: attestation.ProviderAppAttest,
			Options:  map[string]string{OptionAppID: testAppID, OptionRootCertPEM: "not a pem"},
		})
		require.ErrorIs(t, err, attestation.ErrInvalidConfig)
	})
	t.Run("verify android is not supported", func(t *testing.T) {
		v, _, _ := newTestVerifier(t)
		_, err := v.VerifyAndroid(context.Background(), "token", "nonce")
		require.ErrorIs(t, err, attestation.ErrPlatformNotSupported)
	})
}
