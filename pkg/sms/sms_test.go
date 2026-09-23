package sms

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubSender struct{}

func (stubSender) Send(ctx context.Context, phoneNumber, message string) error { return nil }

func TestNew_NilConfig(t *testing.T) {
	_, err := New(nil)
	require.ErrorIs(t, err, ErrInvalidConfig)
}

func TestNew_MissingProvider(t *testing.T) {
	_, err := New(&Config{})
	require.ErrorIs(t, err, ErrInvalidConfig)
}

func TestNew_UnregisteredProvider(t *testing.T) {
	_, err := New(&Config{Provider: ProviderType("does-not-exist")})
	require.ErrorIs(t, err, ErrUnsupportedProvider)
}

func TestNew_UsesRegisteredFactory(t *testing.T) {
	const testProvider ProviderType = "test-stub"
	Register(testProvider, func(cfg *Config) (Sender, error) {
		return stubSender{}, nil
	})

	sender, err := New(&Config{Provider: testProvider})
	require.NoError(t, err)
	require.NoError(t, sender.Send(context.Background(), "+84900000000", "hello"))
}

func TestNew_FactoryErrorPropagates(t *testing.T) {
	const testProvider ProviderType = "test-stub-failing"
	wantErr := errors.New("boom")
	Register(testProvider, func(cfg *Config) (Sender, error) {
		return nil, wantErr
	})

	_, err := New(&Config{Provider: testProvider})
	require.ErrorIs(t, err, wantErr)
}
