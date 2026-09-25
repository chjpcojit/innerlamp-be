package mock

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"chjpcoj.com.innerlamp/pkg/attestation"
)

func TestRegisteredWithProviderMock(t *testing.T) {
	v, err := attestation.New(&attestation.Config{Provider: attestation.ProviderMock})
	require.NoError(t, err)
	require.NotNil(t, v)
}

func TestNew_NilConfig(t *testing.T) {
	_, err := New(nil)
	require.ErrorIs(t, err, attestation.ErrInvalidConfig)
}

func TestVerifyAndroid_Pass(t *testing.T) {
	v, err := New(&attestation.Config{
		Provider: attestation.ProviderMock,
		Options:  map[string]string{OptionPackageName: "com.example.app"},
	})
	require.NoError(t, err)

	result, err := v.VerifyAndroid(context.Background(), "token", "nonce")
	require.NoError(t, err)
	require.Equal(t, "com.example.app", result.PackageName)
	require.Equal(t, "nonce", result.Nonce)
	require.Equal(t, "PLAY_RECOGNIZED", result.AppRecognitionVerdict)
	require.Positive(t, result.TimestampMillis)
}

func TestVerifyAndroid_Fail(t *testing.T) {
	v, err := New(&attestation.Config{Provider: attestation.ProviderMock, Options: map[string]string{OptionMode: ModeFail}})
	require.NoError(t, err)

	result, err := v.VerifyAndroid(context.Background(), "token", "nonce")
	require.NoError(t, err)
	require.Equal(t, "UNRECOGNIZED_VERSION", result.AppRecognitionVerdict)
}

func TestVerifyAndroid_UpstreamError(t *testing.T) {
	v, err := New(&attestation.Config{Provider: attestation.ProviderMock, Options: map[string]string{OptionMode: ModeUpstreamError}})
	require.NoError(t, err)

	_, err = v.VerifyAndroid(context.Background(), "token", "nonce")
	require.Error(t, err)
}

func TestVerifyIOSAttestation_PassAndFail(t *testing.T) {
	pass, err := New(&attestation.Config{Provider: attestation.ProviderMock, Options: map[string]string{OptionIOSEnvironment: "development"}})
	require.NoError(t, err)
	result, err := pass.VerifyIOSAttestation(context.Background(), "key-1", []byte("obj"), "nonce")
	require.NoError(t, err)
	require.Equal(t, "development", result.Environment)
	require.NotEmpty(t, result.PublicKey)
	require.NotEmpty(t, result.Receipt)

	fail, err := New(&attestation.Config{Provider: attestation.ProviderMock, Options: map[string]string{OptionMode: ModeFail}})
	require.NoError(t, err)
	_, err = fail.VerifyIOSAttestation(context.Background(), "key-1", []byte("obj"), "nonce")
	require.ErrorIs(t, err, attestation.ErrAssertionInvalid)
}

func TestVerifyIOSAssertion_Modes(t *testing.T) {
	pass, err := New(&attestation.Config{Provider: attestation.ProviderMock})
	require.NoError(t, err)
	counter, err := pass.VerifyIOSAssertion(context.Background(), []byte("pk"), []byte("a"), []byte("c"), 5)
	require.NoError(t, err)
	require.Equal(t, int64(6), counter)

	replay, err := New(&attestation.Config{Provider: attestation.ProviderMock, Options: map[string]string{OptionAssertionMode: AssertionModeReplay}})
	require.NoError(t, err)
	_, err = replay.VerifyIOSAssertion(context.Background(), []byte("pk"), []byte("a"), []byte("c"), 5)
	require.ErrorIs(t, err, attestation.ErrCounterNotIncreased)

	invalid, err := New(&attestation.Config{Provider: attestation.ProviderMock, Options: map[string]string{OptionAssertionMode: AssertionModeInvalid}})
	require.NoError(t, err)
	_, err = invalid.VerifyIOSAssertion(context.Background(), []byte("pk"), []byte("a"), []byte("c"), 5)
	require.ErrorIs(t, err, attestation.ErrAssertionInvalid)
}
