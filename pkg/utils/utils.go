package utils

import (
	"net/http"
	"os"
	"strings"
)

// Security constants
const (
	// MaxRequestSize limits HTTP request body size to prevent DoS attacks
	MaxRequestSize = 1024 * 1024 // 1MB
	// MaxUserAgentLength limits user agent string length for logging safety
	MaxUserAgentLength = 200
)

// GetResourceType returns the appropriate GCP resource type
func GetResourceType() string {
	// Check if running on Cloud Run
	if os.Getenv("K_SERVICE") != "" {
		return "cloud_run_revision"
	}
	// Check if running on GKE
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return "k8s_container"
	}
	// Check if running on GCE
	if os.Getenv("GCE_INSTANCE_ID") != "" {
		return "gce_instance"
	}
	// Default to generic task
	return "generic_task"
}

// GetResourceLabels returns resource labels based on environment
func GetResourceLabels() map[string]string {
	labels := make(map[string]string)

	// Cloud Run
	if serviceName := os.Getenv("K_SERVICE"); serviceName != "" {
		labels["project_id"] = GetEnv("GCP_PROJECT_ID", GetEnv("GOOGLE_CLOUD_PROJECT", ""))
		labels["service_name"] = serviceName
		labels["revision_name"] = GetEnv("K_REVISION", "unknown")
		labels["location"] = GetEnv("CLOUD_RUN_REGION", "us-central1")
		labels["configuration_name"] = GetEnv("K_CONFIGURATION", serviceName)
		return labels
	}

	// GKE
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		labels["project_id"] = GetEnv("GCP_PROJECT_ID", GetEnv("GOOGLE_CLOUD_PROJECT", ""))
		labels["location"] = GetEnv("GKE_CLUSTER_LOCATION", "us-central1")
		labels["cluster_name"] = GetEnv("GKE_CLUSTER_NAME", "default-cluster")
		labels["namespace_name"] = GetEnv("KUBERNETES_NAMESPACE", "default")
		labels["pod_name"] = GetEnv("HOSTNAME", "unknown")
		labels["container_name"] = "users-service"
		return labels
	}

	// Generic task (default)
	labels["project_id"] = GetEnv("GCP_PROJECT_ID", GetEnv("GOOGLE_CLOUD_PROJECT", ""))
	labels["location"] = GetEnv("LOCATION", "global")
	labels["namespace"] = "users"
	labels["job"] = "users-service"
	labels["task_id"] = GetEnv("HOSTNAME", "default")

	return labels
}

// GetEnv gets environment variable with default value
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// SanitizeUserAgent sanitizes user agent strings for safe logging
// It truncates the string to prevent log injection and removes dangerous characters
func SanitizeUserAgent(userAgent string) string {
	// Truncate user agent to prevent log injection and limit size
	if len(userAgent) > MaxUserAgentLength {
		userAgent = userAgent[:MaxUserAgentLength] + "..."
	}
	// Remove potentially dangerous characters that could be used for log injection
	return strings.ReplaceAll(strings.ReplaceAll(userAgent, "\n", ""), "\r", "")
}

// LimitRequestBodySize applies size limits to HTTP request bodies for security
// Returns true if the request was limited, false otherwise
func LimitRequestBodySize(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestSize)
}

// IsRequestTooLarge checks if an error indicates the request body was too large
func IsRequestTooLarge(err error) bool {
	return strings.Contains(err.Error(), "request body too large")
}
