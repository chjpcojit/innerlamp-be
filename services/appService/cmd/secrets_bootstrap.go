package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	_ "chjpcoj.com.innerlamp/pkg/secrets/gcp" // registers the gcp provider

	pkgsecrets "chjpcoj.com.innerlamp/pkg/secrets"
	appConfig "chjpcoj.com.innerlamp/services/appService/config"
)

const (
	// secretsProviderEnv selects the resolver: "gcp" reads Secret Manager.
	secretsProviderEnv = "SECRETS_PROVIDER"
	// secretsProjectEnv names the project holding the secrets (Cloud Run sets
	// it as an explicit env var; GOOGLE_CLOUD_PROJECT is also injected).
	secretsProjectEnv = "GOOGLE_CLOUD_PROJECT"
	// secretsTimeout bounds every Secret Manager call.
	secretsTimeout = 5 * time.Second
)

// newSecretGetter chooses where the config's *_secret names are resolved:
//
//   - SECRETS_PROVIDER=gcp: Google Cloud Secret Manager, authenticated as the
//     Cloud Run service account (ADC). This is the deployed path; the YAML
//     carries names only.
//   - anything else: the dev-only plaintext resolver in config.dev.yaml.
//
// The returned release func closes the client and must be deferred.
func newSecretGetter(ctx context.Context, rawConfig []byte) (appConfig.SecretGetter, func(), error) {
	if strings.EqualFold(strings.TrimSpace(os.Getenv(secretsProviderEnv)), string(pkgsecrets.ProviderGCP)) {
		projectID := strings.TrimSpace(os.Getenv(secretsProjectEnv))
		if projectID == "" {
			return nil, nil, fmt.Errorf("%s is required when %s=gcp", secretsProjectEnv, secretsProviderEnv)
		}
		client, err := pkgsecrets.New(&pkgsecrets.Config{
			Provider: pkgsecrets.ProviderGCP,
			Timeout:  secretsTimeout,
			GCP:      &pkgsecrets.GCPConfig{ProjectID: projectID, UseDefaultCredentials: true},
		})
		if err != nil {
			return nil, nil, fmt.Errorf("create GCP secret client: %w", err)
		}
		if err := client.Connect(ctx); err != nil {
			_ = client.Close()
			return nil, nil, fmt.Errorf("connect GCP Secret Manager: %w", err)
		}
		return client, func() { _ = client.Close() }, nil
	}

	mock, err := appConfig.NewMockSecretGetterFromYAML(rawConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("load dev secrets: %w", err)
	}
	return mock, func() {}, nil
}
