package config

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gopkg.in/yaml.v3"
)

// LoadSource defines where config is loaded from.
// Supported formats:
// - Local file path: "/path/to/config.yaml" or "config/configs.yaml"
// - S3 URI: "s3://bucket-name/path/to/config.yaml"
// - Google Cloud Storage URI: "gs://bucket-name/path/to/config.yaml"
type LoadSource string

// LoadYAML loads raw YAML content from the specified source.
// Supports local filesystem, S3, and Google Cloud Storage.
func LoadYAML(ctx context.Context, source LoadSource) ([]byte, error) {
	sourceStr := string(source)

	// Local file
	if !strings.HasPrefix(sourceStr, "s3://") && !strings.HasPrefix(sourceStr, "gs://") {
		return loadLocalFile(sourceStr)
	}

	// S3
	if strings.HasPrefix(sourceStr, "s3://") {
		return loadFromS3(ctx, sourceStr)
	}

	// Google Cloud Storage
	if strings.HasPrefix(sourceStr, "gs://") {
		return loadFromGCS(ctx, sourceStr)
	}

	return nil, fmt.Errorf("unsupported config source: %s", sourceStr)
}

// UnmarshalYAML unmarshals YAML bytes into the destination.
func UnmarshalYAML(data []byte, dest interface{}) error {
	return yaml.Unmarshal(data, dest)
}

// LoadAndUnmarshal loads YAML from source and unmarshals into dest.
func LoadAndUnmarshal(ctx context.Context, source LoadSource, dest interface{}) error {
	data, err := LoadYAML(ctx, source)
	if err != nil {
		return err
	}
	return UnmarshalYAML(data, dest)
}

// loadLocalFile reads a configuration file from the local filesystem.
func loadLocalFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %s: %w", path, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	return data, nil
}

// loadFromS3 loads configuration from AWS S3.
// Source format: s3://bucket-name/path/to/config.yaml
// Requires AWS credentials configured via environment, shared config file, or IAM role.
func loadFromS3(ctx context.Context, source string) ([]byte, error) {
	bucket, key, err := parseCloudURI(source, "s3://", "S3")
	if err != nil {
		return nil, err
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg)
	result, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load S3 config object %s: %w", source, err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read S3 config object %s: %w", source, err)
	}

	return data, nil
}

// loadFromGCS loads configuration from Google Cloud Storage.
// Source format: gs://bucket-name/path/to/config.yaml
// Requires Google Cloud credentials configured via application default credentials or GOOGLE_APPLICATION_CREDENTIALS.
func loadFromGCS(ctx context.Context, source string) ([]byte, error) {
	bucket, object, err := parseCloudURI(source, "gs://", "GCS")
	if err != nil {
		return nil, err
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}
	defer client.Close()

	reader, err := client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load GCS config object %s: %w", source, err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read GCS config object %s: %w", source, err)
	}

	return data, nil
}

// parseCloudURI parses S3 and GCS URIs into bucket and object/key components.
func parseCloudURI(uri, prefix, name string) (bucket, object string, err error) {
	if !strings.HasPrefix(uri, prefix) {
		return "", "", fmt.Errorf("invalid %s URI %q", name, uri)
	}

	trimmed := strings.TrimPrefix(uri, prefix)
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid %s URI %q: expected %sbucket/object", name, uri, prefix)
	}

	return parts[0], parts[1], nil
}

// MergeYAML merges a YAML override map into a destination struct.
// Useful for overriding config values from environment variables.
// This is a simple implementation; for complex merging, consider using
// a library like github.com/imdario/mergo
func MergeYAML(base []byte, overrides map[string]interface{}) ([]byte, error) {
	var baseData map[string]interface{}
	if err := yaml.Unmarshal(base, &baseData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal base YAML: %w", err)
	}

	// Simple merge: overrides replace keys in base
	for k, v := range overrides {
		baseData[k] = v
	}

	return yaml.Marshal(baseData)
}
