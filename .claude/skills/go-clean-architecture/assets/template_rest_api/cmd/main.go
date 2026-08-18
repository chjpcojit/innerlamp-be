package main

import (
	"log"
	"os"

	"yourapp/infrastructure"
)

func main() {
	// Load configuration from environment
	dbConfig := infrastructure.DatabaseConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     5432,
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "password"),
		Database: getEnv("DB_NAME", "myapp"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}

	// Initialize database
	db, err := infrastructure.NewPostgresDatabase(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Setup HTTP server
	server := infrastructure.NewHTTPServer(db)

	// Start server
	log.Printf("Server starting on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
