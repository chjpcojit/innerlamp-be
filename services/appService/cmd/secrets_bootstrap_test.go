package main

import (
	"context"
	"strings"
	"testing"

	appConfig "chjpcoj.com.innerlamp/services/appService/config"
)

const devConfigWithSecrets = `
dev_secrets:
  appservice-db-password: "postgres"
`

func TestNewSecretGetterDefaultsToDevResolver(t *testing.T) {
	t.Setenv(secretsProviderEnv, "")

	getter, release, err := newSecretGetter(context.Background(), []byte(devConfigWithSecrets))
	if err != nil {
		t.Fatalf("newSecretGetter: %v", err)
	}
	defer release()

	if _, ok := getter.(*appConfig.MockSecretGetter); !ok {
		t.Fatalf("getter = %T, want *config.MockSecretGetter", getter)
	}
}

func TestNewSecretGetterRequiresProjectForGCP(t *testing.T) {
	t.Setenv(secretsProviderEnv, "gcp")
	t.Setenv(secretsProjectEnv, "")

	if _, _, err := newSecretGetter(context.Background(), nil); err == nil {
		t.Fatal("expected an error when GOOGLE_CLOUD_PROJECT is empty")
	} else if !strings.Contains(err.Error(), secretsProjectEnv) {
		t.Fatalf("error = %v, want it to name %s", err, secretsProjectEnv)
	}
}
