// Package sms sends text messages through a provider-agnostic interface,
// following the same Register/New factory pattern as pkg/database and
// pkg/storage: this package declares the interface and the registry, a
// subpackage per provider self-registers via init(), and a consumer
// activates one with a blank import.
//
// It was introduced for US-001 (docs/specs/features/onboarding), whose only
// requirement so far is "send an OTP by SMS" — the interface is deliberately
// generic (Send a message to a number) rather than OTP-specific, so a real
// provider added later (Twilio, AWS SNS, a GCP equivalent) is not coupled to
// any one feature's message shape. Turning an OTP code into a Send call is
// services/appService/repository's job (it already implements every other
// domain port with real infrastructure) — this package has no dependency on
// any service's domain types.
package sms

import (
	"context"
	"errors"
	"fmt"

	"chjpcoj.com.innerlamp/pkg/logging"
)

// Common errors.
var (
	ErrUnsupportedProvider = errors.New("unsupported sms provider")
	ErrInvalidConfig       = errors.New("invalid sms configuration")
)

// ProviderType selects which registered Sender implementation New builds.
type ProviderType string

const (
	// ProviderLog is the log-only implementation in pkg/sms/logsms — the
	// only provider that exists today. A real provider adds its own
	// ProviderType constant in its own subpackage; this file does not grow
	// a case per provider.
	ProviderLog ProviderType = "log"
)

// Config configures a Sender. Logger is here (rather than left for each
// provider to invent its own) because the first, and today only, provider's
// entire job is to log — a real provider is free to ignore it or use it for
// its own diagnostics.
type Config struct {
	Provider ProviderType
	Logger   logging.Logger
	Options  map[string]string
}

// Sender delivers a text message to a phone number. One method, not one per
// message kind: the caller composes the message text; this package only
// knows how to hand it to a provider.
type Sender interface {
	// Send delivers message to phoneNumber. A nil error means the provider
	// accepted the message for delivery, not that it was received.
	Send(ctx context.Context, phoneNumber, message string) error
}

// FactoryFunc builds a Sender from Config. Registered by each provider
// subpackage's init().
type FactoryFunc func(*Config) (Sender, error)

var factories = make(map[ProviderType]FactoryFunc)

// Register binds provider to factory. Called from a provider subpackage's
// init(), never from this file — adding a provider must not require editing
// pkg/sms itself.
func Register(provider ProviderType, factory FactoryFunc) {
	factories[provider] = factory
}

// New builds the Sender for config.Provider. The caller activates a
// provider with a blank import (e.g. `_ "chjpcoj.com.innerlamp/pkg/sms/logsms"`)
// before calling this — an unregistered provider is ErrUnsupportedProvider,
// not a panic.
func New(config *Config) (Sender, error) {
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
