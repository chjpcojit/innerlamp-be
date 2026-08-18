// Package main demonstrates usage of the secrets management package.
// This file contains example code that should not be executed directly,
// but serves as documentation for how to use the secrets package.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"base_project/pkg/secrets"
	_ "base_project/pkg/secrets/aws" // Import to register AWS provider
	_ "base_project/pkg/secrets/gcp" // Import to register GCP provider
)

// DatabaseCredentials represents database connection credentials
type DatabaseCredentials struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
}

// ExampleGCPBasicUsage demonstrates basic usage with GCP Secret Manager
func ExampleGCPBasicUsage() {
	// Create configuration - all values must be provided explicitly
	config := &secrets.Config{
		Provider: secrets.ProviderGCP,
		Timeout:  30 * time.Second,
		GCP: &secrets.GCPConfig{
			ProjectID:             "your-project-id",
			UseDefaultCredentials: true, // Use ADC in Cloud Run/GCE
		},
	}

	// Create client
	client, err := secrets.New(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Connect to Secret Manager
	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Create a new secret
	secretValue := []byte("my-super-secret-password")
	secret, err := client.Create(ctx, "database-password", secretValue, &secrets.CreateOptions{
		Description: "Database password for production",
		Labels: map[string]string{
			"environment": "production",
			"team":        "backend",
		},
	})
	if err != nil {
		log.Printf("Failed to create secret: %v", err)
		return
	}
	fmt.Printf("Created secret: %s (version: %s)\n", secret.Name, secret.CurrentVersion)

	// Get secret value
	value, err := client.Get(ctx, "database-password", nil)
	if err != nil {
		log.Printf("Failed to get secret: %v", err)
		return
	}
	fmt.Printf("Secret value: %s\n", value.String())

	// Update secret (creates a new version)
	newValue := []byte("my-new-secret-password")
	version, err := client.Update(ctx, "database-password", newValue)
	if err != nil {
		log.Printf("Failed to update secret: %v", err)
		return
	}
	fmt.Printf("Updated secret, new version: %s\n", version.Version)

	// Delete secret
	err = client.Delete(ctx, "database-password", nil)
	if err != nil {
		log.Printf("Failed to delete secret: %v", err)
		return
	}
	fmt.Println("Secret deleted successfully")
}

// ExampleAWSBasicUsage demonstrates basic usage with AWS Secrets Manager
func ExampleAWSBasicUsage() {
	config := &secrets.Config{
		Provider: secrets.ProviderAWS,
		Timeout:  30 * time.Second,
		AWS: &secrets.AWSConfig{
			Region:                "us-east-1",
			UseDefaultCredentials: true, // Use default credential chain
		},
	}

	client, err := secrets.New(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Create a secret
	secret, err := client.Create(ctx, "api-key", []byte("sk-1234567890abcdef"), &secrets.CreateOptions{
		Description: "API key for external service",
		Labels: map[string]string{
			"service": "payment-gateway",
		},
	})
	if err != nil {
		log.Printf("Failed to create secret: %v", err)
		return
	}
	fmt.Printf("Created secret: %s\n", secret.Name)

	// Get secret
	value, err := client.Get(ctx, "api-key", nil)
	if err != nil {
		log.Printf("Failed to get secret: %v", err)
		return
	}
	fmt.Printf("API Key: %s\n", value.String())
}

// ExampleGCPWithCredentialsFile demonstrates using a credentials file
func ExampleGCPWithCredentialsFile() {
	config := &secrets.Config{
		Provider: secrets.ProviderGCP,
		Timeout:  30 * time.Second,
		GCP: &secrets.GCPConfig{
			ProjectID:       "your-project-id",
			CredentialsFile: "/path/to/service-account.json",
		},
	}

	client, err := secrets.New(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	fmt.Println("Connected with credentials file")
}

// ExampleAWSWithAccessKeys demonstrates using AWS access keys
func ExampleAWSWithAccessKeys() {
	config := &secrets.Config{
		Provider: secrets.ProviderAWS,
		Timeout:  30 * time.Second,
		AWS: &secrets.AWSConfig{
			Region:          "us-west-2",
			AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",                     // #nosec
			SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", // #nosec
			SessionToken:    "",                                         // Optional, for temporary credentials
		},
	}

	client, err := secrets.New(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	fmt.Println("Connected with access keys")
}

// ExampleListSecrets demonstrates listing all secrets
func ExampleListSecrets() {
	config := &secrets.Config{
		Provider: secrets.ProviderGCP,
		Timeout:  30 * time.Second,
		GCP: &secrets.GCPConfig{
			ProjectID:             "your-project-id",
			UseDefaultCredentials: true,
		},
	}

	client, err := secrets.New(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// List all secrets
	result, err := client.List(ctx, nil)
	if err != nil {
		log.Printf("Failed to list secrets: %v", err)
		return
	}

	fmt.Printf("Found %d secrets:\n", len(result.Secrets))
	for _, secret := range result.Secrets {
		fmt.Printf("  - %s (created: %v)\n", secret.Name, secret.CreatedAt)
	}

	// List with prefix filter
	opts := &secrets.ListOptions{
		Prefix:     "prod/",
		MaxResults: 100,
	}
	result, err = client.List(ctx, opts)
	if err != nil {
		log.Printf("Failed to list secrets with prefix: %v", err)
		return
	}

	fmt.Printf("\nSecrets with 'prod/' prefix: %d\n", len(result.Secrets))
}

// ExampleListSecretsWithPagination demonstrates paginated listing
func ExampleListSecretsWithPagination() {
	config := &secrets.Config{
		Provider: secrets.ProviderGCP,
		Timeout:  30 * time.Second,
		GCP: &secrets.GCPConfig{
			ProjectID:             "your-project-id",
			UseDefaultCredentials: true,
		},
	}

	client, err := secrets.New(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	opts := &secrets.ListOptions{
		MaxResults: 10, // Page size
	}

	totalSecrets := 0
	page := 1

	for {
		result, err := client.List(ctx, opts)
		if err != nil {
			log.Printf("Failed to list secrets: %v", err)
			return
		}

		fmt.Printf("Page %d: %d secrets\n", page, len(result.Secrets))
		totalSecrets += len(result.Secrets)

		if result.NextPageToken == "" {
			break
		}

		opts.PageToken = result.NextPageToken
		page++
	}

	fmt.Printf("Total secrets: %d\n", totalSecrets)
}

// ExampleVersionOperations demonstrates version management
func ExampleVersionOperations() {
	config := &secrets.Config{
		Provider: secrets.ProviderGCP,
		Timeout:  30 * time.Second,
		GCP: &secrets.GCPConfig{
			ProjectID:             "your-project-id",
			UseDefaultCredentials: true,
		},
	}

	client, err := secrets.New(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	secretName := "my-versioned-secret"

	// Create secret with initial value
	_, err = client.Create(ctx, secretName, []byte("version-1-value"), nil)
	if err != nil {
		log.Printf("Failed to create secret: %v", err)
		return
	}

	// Add more versions
	for i := 2; i <= 5; i++ {
		value := fmt.Sprintf("version-%d-value", i)
		version, err := client.Update(ctx, secretName, []byte(value))
		if err != nil {
			log.Printf("Failed to add version %d: %v", i, err)
			return
		}
		fmt.Printf("Added version: %s\n", version.Version)
	}

	// List all versions
	result, err := client.ListVersions(ctx, secretName, &secrets.ListVersionsOptions{
		IncludeDisabled:  true,
		IncludeDestroyed: false,
	})
	if err != nil {
		log.Printf("Failed to list versions: %v", err)
		return
	}

	fmt.Printf("\nVersions of %s:\n", secretName)
	for _, v := range result.Versions {
		fmt.Printf("  - Version %s: %s (created: %v)\n", v.Version, v.State, v.CreatedAt)
	}

	// Get specific version
	value, err := client.GetVersion(ctx, secretName, "1")
	if err != nil {
		log.Printf("Failed to get version 1: %v", err)
		return
	}
	fmt.Printf("\nVersion 1 value: %s\n", value.String())

	// Get latest version
	latestValue, err := client.Get(ctx, secretName, nil)
	if err != nil {
		log.Printf("Failed to get latest version: %v", err)
		return
	}
	fmt.Printf("Latest value: %s\n", latestValue.String())

	// Disable a version (GCP only)
	err = client.DisableVersion(ctx, secretName, "2")
	if err != nil {
		log.Printf("Failed to disable version 2: %v", err)
	} else {
		fmt.Println("\nDisabled version 2")
	}

	// Enable a version (GCP only)
	err = client.EnableVersion(ctx, secretName, "2")
	if err != nil {
		log.Printf("Failed to enable version 2: %v", err)
	} else {
		fmt.Println("Re-enabled version 2")
	}

	// Destroy a version (GCP only - permanent!)
	err = client.DestroyVersion(ctx, secretName, "1")
	if err != nil {
		log.Printf("Failed to destroy version 1: %v", err)
	} else {
		fmt.Println("Destroyed version 1 (permanent)")
	}

	// Clean up
	err = client.Delete(ctx, secretName, nil)
	if err != nil {
		log.Printf("Failed to delete secret: %v", err)
	}
}

// ExampleUpdateMetadata demonstrates updating secret metadata
func ExampleUpdateMetadata() {
	config := &secrets.Config{
		Provider: secrets.ProviderGCP,
		Timeout:  30 * time.Second,
		GCP: &secrets.GCPConfig{
			ProjectID:             "your-project-id",
			UseDefaultCredentials: true,
		},
	}

	client, err := secrets.New(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Create a secret
	_, err = client.Create(ctx, "metadata-example", []byte("secret-value"), &secrets.CreateOptions{
		Description: "Original description",
		Labels: map[string]string{
			"environment": "staging",
		},
	})
	if err != nil {
		log.Printf("Failed to create secret: %v", err)
		return
	}

	// Update metadata
	newDescription := "Updated description"
	opts := &secrets.UpdateMetadataOptions{
		Description: &newDescription,
		Labels: map[string]string{
			"environment": "production",
			"updated":     "true",
		},
	}

	secret, err := client.UpdateMetadata(ctx, "metadata-example", opts)
	if err != nil {
		log.Printf("Failed to update metadata: %v", err)
		return
	}

	fmt.Printf("Updated secret: %s\n", secret.Name)
	fmt.Printf("Labels: %v\n", secret.Labels)

	// Clean up
	_ = client.Delete(ctx, "metadata-example", nil)
}

// ExampleCheckExistence demonstrates checking if a secret exists
func ExampleCheckExistence() {
	config := &secrets.Config{
		Provider: secrets.ProviderGCP,
		Timeout:  30 * time.Second,
		GCP: &secrets.GCPConfig{
			ProjectID:             "your-project-id",
			UseDefaultCredentials: true,
		},
	}

	client, err := secrets.New(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Check if secret exists
	exists, err := client.Exists(ctx, "my-secret")
	if err != nil {
		log.Printf("Failed to check existence: %v", err)
		return
	}

	if exists {
		fmt.Println("Secret exists, fetching value...")
		value, err := client.Get(ctx, "my-secret", nil)
		if err != nil {
			log.Printf("Failed to get secret: %v", err)
			return
		}
		fmt.Printf("Value: %s\n", value.String())
	} else {
		fmt.Println("Secret does not exist, creating...")
		_, err := client.Create(ctx, "my-secret", []byte("new-value"), nil)
		if err != nil {
			log.Printf("Failed to create secret: %v", err)
			return
		}
		fmt.Println("Secret created")
	}
}

// ExampleErrorHandling demonstrates proper error handling
func ExampleErrorHandling() {
	config := &secrets.Config{
		Provider: secrets.ProviderGCP,
		Timeout:  30 * time.Second,
		GCP: &secrets.GCPConfig{
			ProjectID:             "your-project-id",
			UseDefaultCredentials: true,
		},
	}

	client, err := secrets.New(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Handle various error types
	value, err := client.Get(ctx, "nonexistent-secret", nil)
	if err != nil {
		switch {
		case errors.Is(err, secrets.ErrSecretNotFound):
			fmt.Println("Secret not found - creating new secret...")
			_, createErr := client.Create(ctx, "nonexistent-secret", []byte("new-value"), nil)
			if createErr != nil {
				if errors.Is(createErr, secrets.ErrSecretAlreadyExists) {
					fmt.Println("Secret was created by another process")
				} else {
					log.Printf("Failed to create: %v", createErr)
				}
			}

		case errors.Is(err, secrets.ErrPermissionDenied):
			fmt.Println("Permission denied - check IAM roles")

		case errors.Is(err, secrets.ErrVersionDisabled):
			fmt.Println("Secret version is disabled")

		case errors.Is(err, secrets.ErrVersionDestroyed):
			fmt.Println("Secret version has been destroyed")

		case errors.Is(err, secrets.ErrOperationTimeout):
			fmt.Println("Operation timed out - retry with backoff")

		case errors.Is(err, secrets.ErrNotConnected):
			fmt.Println("Client not connected - call Connect() first")

		default:
			log.Printf("Unexpected error: %v", err)
		}
		return
	}

	fmt.Printf("Value: %s\n", value.String())
}

// ExampleCloudRunIntegration demonstrates best practices for Cloud Run
func ExampleCloudRunIntegration() {
	// In Cloud Run, create the client once at startup and reuse it
	config := &secrets.Config{
		Provider: secrets.ProviderGCP,
		Timeout:  30 * time.Second,
		GCP: &secrets.GCPConfig{
			ProjectID:             "your-project-id",
			UseDefaultCredentials: true, // Cloud Run provides ADC automatically
		},
	}

	client, err := secrets.New(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		_ = client.Close()
		os.Exit(0)
	}()

	// Inject client into your HTTP handlers
	http.HandleFunc("/api/config", makeConfigHandler(client))
	http.HandleFunc("/api/database", makeDatabaseHandler(client))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port) // #nosec
	log.Fatal(http.ListenAndServe(":"+port, nil))  // #nosec
}

// makeConfigHandler creates a handler that uses the secrets client
func makeConfigHandler(client secrets.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		value, err := client.Get(ctx, "app-config", nil)
		if err != nil {
			if errors.Is(err, secrets.ErrSecretNotFound) {
				http.Error(w, "Configuration not found", http.StatusNotFound)
				return
			}
			log.Printf("Failed to get config: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(value.Value) // #nosec
	}
}

// makeDatabaseHandler creates a handler that fetches database credentials
func makeDatabaseHandler(client secrets.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		value, err := client.Get(ctx, "database-credentials", nil)
		if err != nil {
			log.Printf("Failed to get database credentials: %v", err)
			http.Error(w, "Failed to get credentials", http.StatusInternalServerError)
			return
		}

		// In real code, you would parse the JSON and use the credentials
		// to connect to the database
		_ = value.Value

		_, _ = w.Write([]byte("Database connection successful")) // #nosec
	}
}

// ExampleDeleteWithRecoveryWindow demonstrates AWS deletion with recovery window
func ExampleDeleteWithRecoveryWindow() {
	config := &secrets.Config{
		Provider: secrets.ProviderAWS,
		Timeout:  30 * time.Second,
		AWS: &secrets.AWSConfig{
			Region:                "us-east-1",
			UseDefaultCredentials: true,
		},
	}

	client, err := secrets.New(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Delete with 7-day recovery window
	opts := &secrets.DeleteOptions{
		RecoveryWindowDays: 7,
	}
	err = client.Delete(ctx, "my-secret", opts)
	if err != nil {
		log.Printf("Failed to delete: %v", err)
		return
	}
	fmt.Println("Secret scheduled for deletion in 7 days")

	// Force delete without recovery (immediate)
	forceOpts := &secrets.DeleteOptions{
		Force: true,
	}
	err = client.Delete(ctx, "another-secret", forceOpts)
	if err != nil {
		log.Printf("Failed to force delete: %v", err)
		return
	}
	fmt.Println("Secret deleted immediately")
}

// ExampleReplicationPolicy demonstrates GCP replication settings
func ExampleReplicationPolicy() {
	config := &secrets.Config{
		Provider: secrets.ProviderGCP,
		Timeout:  30 * time.Second,
		GCP: &secrets.GCPConfig{
			ProjectID:             "your-project-id",
			UseDefaultCredentials: true,
		},
	}

	client, err := secrets.New(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Create with automatic replication (default)
	_, err = client.Create(ctx, "auto-replicated-secret", []byte("value"), &secrets.CreateOptions{
		Replication: &secrets.ReplicationPolicy{
			Automatic: true,
		},
	})
	if err != nil {
		log.Printf("Failed to create: %v", err)
	}

	// Create with user-managed replication
	_, err = client.Create(ctx, "regional-secret", []byte("value"), &secrets.CreateOptions{
		Description: "Secret replicated to specific regions",
		Replication: &secrets.ReplicationPolicy{
			Automatic: false,
			Locations: []string{"us-east1", "us-west1", "europe-west1"},
		},
	})
	if err != nil {
		log.Printf("Failed to create: %v", err)
	}

	// Clean up
	_ = client.Delete(ctx, "auto-replicated-secret", nil)
	_ = client.Delete(ctx, "regional-secret", nil)
}

// ExampleSwitchingProviders demonstrates provider abstraction
func ExampleSwitchingProviders() {
	// The provider can be switched by changing configuration
	// All operations remain the same

	var config *secrets.Config

	// Choose provider based on deployment environment
	cloudProvider := os.Getenv("CLOUD_PROVIDER") // "gcp" or "aws"

	switch cloudProvider {
	case "aws":
		config = &secrets.Config{
			Provider: secrets.ProviderAWS,
			Timeout:  30 * time.Second,
			AWS: &secrets.AWSConfig{
				Region:                os.Getenv("AWS_REGION"),
				UseDefaultCredentials: true,
			},
		}
	default: // GCP as default
		config = &secrets.Config{
			Provider: secrets.ProviderGCP,
			Timeout:  30 * time.Second,
			GCP: &secrets.GCPConfig{
				ProjectID:             os.Getenv("GCP_PROJECT_ID"),
				UseDefaultCredentials: true,
			},
		}
	}

	client, err := secrets.New(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Same operations work for both providers
	fmt.Printf("Using provider: %s\n", client.GetProvider())

	value, err := client.Get(ctx, "my-secret", nil)
	if err != nil {
		log.Printf("Failed to get secret: %v", err)
		return
	}
	fmt.Printf("Secret value: %s\n", value.String())
}

func main() {
	// Run one of the examples
	fmt.Println("Secrets Management Examples")
	fmt.Println("===========================")
	fmt.Println("This file contains example code for documentation purposes.")
	fmt.Println("Uncomment and modify the example you want to run.")

	// Uncomment to run:
	// ExampleGCPBasicUsage()
	// ExampleAWSBasicUsage()
	// ExampleListSecrets()
	// ExampleVersionOperations()
	// ExampleUpdateMetadata()
	// ExampleCheckExistence()
	// ExampleErrorHandling()
	// ExampleCloudRunIntegration()
	// ExampleSwitchingProviders()
}
