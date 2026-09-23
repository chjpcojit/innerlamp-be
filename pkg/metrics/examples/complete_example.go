// Package main provides a complete example of using the metrics package with GCP Cloud Monitoring
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand" // #nosec
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chjpcoj.com.innerlamp/pkg/metrics"
	httpMetrics "chjpcoj.com.innerlamp/pkg/metrics/middleware/http"
	"chjpcoj.com.innerlamp/pkg/utils"

	_ "chjpcoj.com.innerlamp/pkg/metrics/gcp"
)

func main() {
	fmt.Println("Starting metrics example with GCP Cloud Monitoring backend")

	// Create GCP configuration
	config := createGCPConfig()

	// Create collector
	collector, err := metrics.NewCollector(config)
	if err != nil {
		log.Fatalf("Failed to create collector: %v", err)
	}

	// Start collector (required for GCP)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := collector.Start(ctx); err != nil {
		log.Fatalf("Failed to start collector: %v", err)
	}

	// Ensure graceful shutdown
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		fmt.Println("Flushing metrics...")
		if err := collector.Flush(shutdownCtx); err != nil {
			log.Printf("Error flushing metrics: %v", err)
		}

		fmt.Println("Stopping collector...")
		if err := collector.Stop(shutdownCtx); err != nil {
			log.Printf("Error stopping collector: %v", err)
		}
	}()

	// Register custom metrics
	if err := registerCustomMetrics(collector); err != nil {
		log.Fatalf("Failed to register metrics: %v", err)
	}

	// Start HTTP server with metrics
	go startHTTPServer(collector)

	// Simulate application activity
	go simulateActivity(collector)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	fmt.Println("Application started. Press Ctrl+C to stop.")
	<-sigChan
	fmt.Println("\nShutting down...")
}

func createGCPConfig() *metrics.Config {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
	if projectID == "" {
		log.Fatal("GCP_PROJECT_ID environment variable is required")
	}

	config := &metrics.Config{
		Backend:   metrics.BackendGCP,
		Namespace: "example_app",
		Subsystem: "demo",
		Enabled:   true,
		ConstLabels: metrics.Labels{
			"environment": getEnv("ENVIRONMENT", "development"),
			"version":     "1.0.0",
		},
		FlushInterval: 30 * time.Second, // Shorter for demo purposes
		MaxBatchSize:  100,
		GCP: &metrics.GCPConfig{
			ProjectID:             projectID,
			UseDefaultCredentials: true,
			MetricPrefix:          "custom.googleapis.com",
			Timeout:               30 * time.Second,
			BatchSize:             200,
			ResourceType:          detectResourceType(),
			ResourceLabels:        getResourceLabels(projectID),
		},
	}

	// Override with credentials file if provided
	if credFile := os.Getenv("GCP_CREDENTIALS_FILE"); credFile != "" {
		config.GCP.CredentialsFile = credFile
		config.GCP.UseDefaultCredentials = false
	}

	return config
}

func detectResourceType() string {
	// Cloud Run
	if os.Getenv("K_SERVICE") != "" {
		return "cloud_run_revision"
	}
	// GKE
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return "k8s_container"
	}
	// GCE
	if os.Getenv("GCE_INSTANCE_ID") != "" {
		return "gce_instance"
	}
	// Default
	return "generic_task"
}

func getResourceLabels(projectID string) map[string]string {
	labels := make(map[string]string)

	// Cloud Run
	if serviceName := os.Getenv("K_SERVICE"); serviceName != "" {
		labels["project_id"] = projectID
		labels["service_name"] = serviceName
		labels["revision_name"] = getEnv("K_REVISION", "unknown")
		labels["location"] = getEnv("CLOUD_RUN_REGION", "us-central1")
		labels["configuration_name"] = getEnv("K_CONFIGURATION", serviceName)
		return labels
	}

	// GKE
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		labels["project_id"] = projectID
		labels["location"] = getEnv("GKE_CLUSTER_LOCATION", "us-central1")
		labels["cluster_name"] = getEnv("GKE_CLUSTER_NAME", "default-cluster")
		labels["namespace_name"] = getEnv("KUBERNETES_NAMESPACE", "default")
		labels["pod_name"] = getEnv("HOSTNAME", "unknown")
		labels["container_name"] = "example-app"
		return labels
	}

	// Generic task
	labels["project_id"] = projectID
	labels["location"] = getEnv("LOCATION", "global")
	labels["namespace"] = "examples"
	labels["job"] = "metrics-example"
	labels["task_id"] = getEnv("HOSTNAME", "local")

	return labels
}

func registerCustomMetrics(collector metrics.Collector) error {
	metrics := []metrics.MetricDefinition{
		// Counter metrics
		metrics.NewCounterMetric(
			"orders_total",
			"Total number of orders",
			[]string{"status", "country"},
		),
		metrics.NewCounterMetric(
			"errors_total",
			"Total number of errors",
			[]string{"error_type"},
		),

		// Gauge metrics
		metrics.NewGaugeMetric(
			"active_users",
			"Number of active users",
			[]string{},
		),
		metrics.NewGaugeMetric(
			"queue_size",
			"Size of processing queue",
			[]string{"queue_name"},
		),

		// Histogram metrics
		metrics.NewHistogramMetricWithBuckets(
			"order_value_dollars",
			"Order value in dollars",
			[]string{"country"},
			[]float64{10, 50, 100, 500, 1000, 5000},
		),
		metrics.NewHistogramMetric(
			"processing_duration_seconds",
			"Processing duration in seconds",
			[]string{"operation"},
		),
	}

	for _, metric := range metrics {
		if err := collector.RegisterMetric(metric); err != nil {
			return fmt.Errorf("failed to register %s: %w", metric.Name, err)
		}
	}

	fmt.Printf("Registered %d custom metrics\n", len(metrics))
	return nil
}

func startHTTPServer(collector metrics.Collector) {
	// Create HTTP metrics
	httpMet, err := httpMetrics.NewMetrics(collector, "example_app", "http")
	if err != nil {
		log.Fatalf("Failed to create HTTP metrics: %v", err)
	}

	// Setup routes
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		utils.WriteSuccessResponse(w, map[string]any{"message": "Hello from metrics example", "backend": "gcp"})
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		utils.WriteSuccessResponse(w, map[string]any{"status": "healthy"})
	})

	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		// Simulate some processing
		time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond) // #nosec

		utils.WriteSuccessResponse(w, map[string]any{"users": []string{"alice", "bob", "charlie"}})
	})

	mux.HandleFunc("/api/orders", func(w http.ResponseWriter, r *http.Request) {
		// Simulate some processing
		time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond) // #nosec
		if rand.Float32() < 0.1 {                                    // #nosec
			utils.WriteSystemError(w, "INTERNAL_ERROR", "internal server error", nil)
			return
		}

		utils.WriteSuccessResponse(w, map[string]any{"orders": []map[string]any{{"id": 1, "total": 99.99}}})
	})

	// Wrap with metrics middleware
	handler := httpMet.Middleware(mux)

	// Start server
	addr := ":8080"
	fmt.Printf("HTTP server listening on %s\n", addr)
	fmt.Println("Endpoints:")
	fmt.Println("  GET  /              - Home")
	fmt.Println("  GET  /health        - Health check")
	fmt.Println("  GET  /api/users     - List users")
	fmt.Println("  GET  /api/orders    - List orders")

	if err := http.ListenAndServe(addr, handler); err != nil { // #nosec
		log.Fatalf("HTTP server error: %v", err)
	}
}

func simulateActivity(collector metrics.Collector) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	countries := []string{"US", "UK", "DE", "FR", "JP"}
	statuses := []string{"completed", "pending", "failed"}
	operations := []string{"payment", "validation", "shipping"}
	errorTypes := []string{"network", "validation", "timeout"}

	fmt.Println("Simulating application activity...")

	for {
		select {
		case <-ticker.C:
			// Simulate orders
			country := countries[rand.Intn(len(countries))] // #nosec
			status := statuses[rand.Intn(len(statuses))]    // #nosec

			_ = collector.IncrementCounter("orders_total", metrics.Labels{ // #nosec
				"status":  status,
				"country": country,
			})

			// Record order value for completed orders
			if status == "completed" {
				value := float64(rand.Intn(4990) + 10)                                       // #nosec
				_ = collector.ObserveHistogram("order_value_dollars", value, metrics.Labels{ // #nosec
					"country": country,
				})
			}

			// Simulate errors occasionally
			if rand.Float32() < 0.1 { // #nosec
				errorType := errorTypes[rand.Intn(len(errorTypes))]            // #nosec
				_ = collector.IncrementCounter("errors_total", metrics.Labels{ // #nosec
					"error_type": errorType,
				})
			}

			// Update active users (random fluctuation)
			activeUsers := float64(rand.Intn(50) + 100)                           // #nosec
			_ = collector.SetGauge("active_users", activeUsers, metrics.Labels{}) // #nosec
			for _, queueName := range []string{"email", "sms", "webhook"} {
				queueSize := float64(rand.Intn(20))                             // #nosec
				_ = collector.SetGauge("queue_size", queueSize, metrics.Labels{ // #nosec
					"queue_name": queueName,
				})
			}

			// Simulate processing duration
			operation := operations[rand.Intn(len(operations))]                                     // #nosec
			duration := float64(rand.Intn(500)+50) / 1000.0                                         // #nosec G404 -- 50-550ms simulation, not security-sensitive
			_ = collector.ObserveHistogram("processing_duration_seconds", duration, metrics.Labels{ // #nosec
				"operation": operation,
			})
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
