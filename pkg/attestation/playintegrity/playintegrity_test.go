package playintegrity

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"google.golang.org/api/playintegrity/v1"

	"chjpcoj.com.innerlamp/pkg/attestation"
)

func TestRegisteredWithProviderPlayIntegrity(t *testing.T) {
	v, err := attestation.New(&attestation.Config{
		Provider: attestation.ProviderPlayIntegrity,
		Options:  map[string]string{OptionPackageName: "com.example.app"},
	})
	require.NoError(t, err)
	require.NotNil(t, v)
}

func TestNew_RequiresPackageName(t *testing.T) {
	_, err := New(&attestation.Config{Provider: attestation.ProviderPlayIntegrity})
	require.ErrorIs(t, err, attestation.ErrInvalidConfig)

	_, err = New(nil)
	require.ErrorIs(t, err, attestation.ErrInvalidConfig)
}

func TestVerifyIOSMethods_NotSupported(t *testing.T) {
	v, err := New(&attestation.Config{
		Provider: attestation.ProviderPlayIntegrity,
		Options:  map[string]string{OptionPackageName: "com.example.app"},
	})
	require.NoError(t, err)

	_, err = v.VerifyIOSAttestation(context.Background(), "key", []byte("obj"), "nonce")
	require.ErrorIs(t, err, attestation.ErrPlatformNotSupported)

	_, err = v.VerifyIOSAssertion(context.Background(), []byte("pk"), []byte("a"), []byte("c"), 0)
	require.ErrorIs(t, err, attestation.ErrPlatformNotSupported)
}

func TestMapPayload(t *testing.T) {
	t.Run("flattens a full payload", func(t *testing.T) {
		result, err := mapPayload(&playintegrity.TokenPayloadExternal{
			RequestDetails: &playintegrity.RequestDetails{
				RequestPackageName: "com.example.app",
				Nonce:              "nonce",
				TimestampMillis:    1234,
			},
			AppIntegrity:    &playintegrity.AppIntegrity{AppRecognitionVerdict: "PLAY_RECOGNIZED"},
			DeviceIntegrity: &playintegrity.DeviceIntegrity{DeviceRecognitionVerdict: []string{"MEETS_BASIC_INTEGRITY", "MEETS_DEVICE_INTEGRITY"}},
			AccountDetails:  &playintegrity.AccountDetails{AppLicensingVerdict: "LICENSED"},
		})
		require.NoError(t, err)
		require.Equal(t, "com.example.app", result.PackageName)
		require.Equal(t, "nonce", result.Nonce)
		require.Equal(t, int64(1234), result.TimestampMillis)
		require.Equal(t, "PLAY_RECOGNIZED", result.AppRecognitionVerdict)
		require.Equal(t, "MEETS_BASIC_INTEGRITY,MEETS_DEVICE_INTEGRITY", result.DeviceRecognitionVerdict)
		require.Equal(t, "LICENSED", result.AppLicensingVerdict)
	})

	t.Run("nil sub-structs are tolerated", func(t *testing.T) {
		result, err := mapPayload(&playintegrity.TokenPayloadExternal{
			RequestDetails: &playintegrity.RequestDetails{RequestPackageName: "com.example.app"},
			AppIntegrity:   &playintegrity.AppIntegrity{AppRecognitionVerdict: "UNRECOGNIZED_VERSION"},
		})
		require.NoError(t, err)
		require.Equal(t, "UNRECOGNIZED_VERSION", result.AppRecognitionVerdict)
		require.Empty(t, result.DeviceRecognitionVerdict)
		require.Empty(t, result.AppLicensingVerdict)
	})

	t.Run("missing required fields is an error", func(t *testing.T) {
		_, err := mapPayload(nil)
		require.Error(t, err)

		_, err = mapPayload(&playintegrity.TokenPayloadExternal{})
		require.Error(t, err)
	})
}
