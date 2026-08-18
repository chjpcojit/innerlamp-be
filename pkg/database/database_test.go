package database

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	tests := []struct {
		name         string
		dbType       DBType
		expectedPort int
	}{
		{
			name:         "MySQL default config",
			dbType:       MySQL,
			expectedPort: 3306,
		},
		{
			name:         "PostgreSQL default config",
			dbType:       PostgreSQL,
			expectedPort: 5432,
		},
		{
			name:         "MongoDB default config",
			dbType:       MongoDB,
			expectedPort: 27017,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig(tt.dbType)

			if config.Type != tt.dbType {
				t.Errorf("Expected type %s, got %s", tt.dbType, config.Type)
			}

			if config.Port != tt.expectedPort {
				t.Errorf("Expected port %d, got %d", tt.expectedPort, config.Port)
			}

			if config.Host != "localhost" {
				t.Errorf("Expected host 'localhost', got '%s'", config.Host)
			}

			if config.MaxOpenConns != 25 {
				t.Errorf("Expected MaxOpenConns 25, got %d", config.MaxOpenConns)
			}

			if config.MaxIdleConns != 5 {
				t.Errorf("Expected MaxIdleConns 5, got %d", config.MaxIdleConns)
			}

			if config.ConnMaxLifetime != 5*time.Minute {
				t.Errorf("Expected ConnMaxLifetime 5m, got %v", config.ConnMaxLifetime)
			}

			if config.Timeout != 10*time.Second {
				t.Errorf("Expected Timeout 10s, got %v", config.Timeout)
			}

			if config.Options == nil {
				t.Error("Expected Options map to be initialized")
			}
		})
	}
}

func TestGetDefaultPort(t *testing.T) {
	tests := []struct {
		name         string
		dbType       DBType
		expectedPort int
	}{
		{"MySQL port", MySQL, 3306},
		{"PostgreSQL port", PostgreSQL, 5432},
		{"MongoDB port", MongoDB, 27017},
		{"Unknown port", DBType("unknown"), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := getDefaultPort(tt.dbType)
			if port != tt.expectedPort {
				t.Errorf("Expected port %d, got %d", tt.expectedPort, port)
			}
		})
	}
}

func TestNew_InvalidConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectedErr error
	}{
		{
			name:        "nil config",
			config:      nil,
			expectedErr: ErrInvalidConfig,
		},
		{
			name: "missing type",
			config: &Config{
				Host: "localhost",
				Port: 3306,
			},
			expectedErr: ErrInvalidConfig,
		},
		{
			name: "missing host",
			config: &Config{
				Type: MySQL,
				Port: 3306,
			},
			expectedErr: ErrInvalidConfig,
		},
		{
			name: "unsupported driver",
			config: &Config{
				Type: DBType("unsupported"),
				Host: "localhost",
				Port: 3306,
			},
			expectedErr: ErrUnsupportedDriver,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.config)
			if err == nil {
				t.Error("Expected error, got nil")
				return
			}

			// Check if the error contains the expected error
			if tt.expectedErr != nil && err.Error() != tt.expectedErr.Error() {
				// For wrapped errors, check if it contains the expected error
				if !contains(err.Error(), tt.expectedErr.Error()) {
					t.Errorf("Expected error containing '%v', got '%v'", tt.expectedErr, err)
				}
			}
		})
	}
}

func TestNew_ConfigDefaults(t *testing.T) {
	// Register a mock factory for testing
	mockCalled := false
	Register(DBType("test"), func(config *Config) (Database, error) {
		mockCalled = true

		// Verify defaults were set
		if config.Port != 9999 {
			t.Errorf("Port should be 9999, got %d", config.Port)
		}
		if config.MaxOpenConns <= 0 {
			t.Error("MaxOpenConns should be set to default")
		}
		if config.MaxIdleConns <= 0 {
			t.Error("MaxIdleConns should be set to default")
		}
		if config.ConnMaxLifetime <= 0 {
			t.Error("ConnMaxLifetime should be set to default")
		}
		if config.ConnMaxIdleTime <= 0 {
			t.Error("ConnMaxIdleTime should be set to default")
		}
		if config.Timeout <= 0 {
			t.Error("Timeout should be set to default")
		}
		if config.Options == nil {
			t.Error("Options should be initialized")
		}

		return nil, nil
	})

	config := &Config{
		Type: DBType("test"),
		Host: "localhost",
		Port: 9999,
	}

	_, _ = New(config)

	if !mockCalled {
		t.Error("Mock factory was not called")
	}
}

func TestConnectionPool(t *testing.T) {
	pool := NewConnectionPool()

	if pool == nil {
		t.Fatal("NewConnectionPool returned nil")
	}

	if len(pool.connections) != 0 {
		t.Error("New pool should be empty")
	}
}

func TestConnectionPool_Add(t *testing.T) {
	pool := NewConnectionPool()

	// Create a mock database
	mockDB := &mockDatabase{}

	err := pool.Add("test", mockDB)
	if err != nil {
		t.Errorf("Failed to add database: %v", err)
	}

	if len(pool.connections) != 1 {
		t.Errorf("Expected 1 connection, got %d", len(pool.connections))
	}
}

func TestConnectionPool_AddNil(t *testing.T) {
	pool := NewConnectionPool()

	err := pool.Add("test", nil)
	if err == nil {
		t.Error("Expected error when adding nil database")
	}
}

func TestConnectionPool_Get(t *testing.T) {
	pool := NewConnectionPool()
	mockDB := &mockDatabase{}

	pool.Add("test", mockDB)

	db, err := pool.Get("test")
	if err != nil {
		t.Errorf("Failed to get database: %v", err)
	}

	if db != mockDB {
		t.Error("Got different database instance")
	}
}

func TestConnectionPool_GetNotFound(t *testing.T) {
	pool := NewConnectionPool()

	_, err := pool.Get("nonexistent")
	if err == nil {
		t.Error("Expected error when getting nonexistent connection")
	}
}

func TestConnectionPool_Remove(t *testing.T) {
	pool := NewConnectionPool()
	mockDB := &mockDatabase{}

	pool.Add("test", mockDB)

	err := pool.Remove("test")
	if err != nil {
		t.Errorf("Failed to remove database: %v", err)
	}

	if !mockDB.closeCalled {
		t.Error("Close was not called on removed database")
	}

	if len(pool.connections) != 0 {
		t.Errorf("Expected 0 connections, got %d", len(pool.connections))
	}
}

func TestConnectionPool_RemoveNonexistent(t *testing.T) {
	pool := NewConnectionPool()

	err := pool.Remove("nonexistent")
	if err != nil {
		t.Errorf("Remove should not error for nonexistent connection: %v", err)
	}
}

func TestConnectionPool_List(t *testing.T) {
	pool := NewConnectionPool()

	pool.Add("db1", &mockDatabase{})
	pool.Add("db2", &mockDatabase{})
	pool.Add("db3", &mockDatabase{})

	names := pool.List()

	if len(names) != 3 {
		t.Errorf("Expected 3 connections, got %d", len(names))
	}

	// Check that all names are present
	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[name] = true
	}

	if !nameMap["db1"] || !nameMap["db2"] || !nameMap["db3"] {
		t.Error("Not all connection names are present in list")
	}
}

func TestConnectionPool_CloseAll(t *testing.T) {
	pool := NewConnectionPool()

	mock1 := &mockDatabase{}
	mock2 := &mockDatabase{}
	mock3 := &mockDatabase{}

	pool.Add("db1", mock1)
	pool.Add("db2", mock2)
	pool.Add("db3", mock3)

	err := pool.CloseAll()
	if err != nil {
		t.Errorf("CloseAll failed: %v", err)
	}

	if !mock1.closeCalled || !mock2.closeCalled || !mock3.closeCalled {
		t.Error("Not all databases were closed")
	}

	if len(pool.connections) != 0 {
		t.Errorf("Expected 0 connections after CloseAll, got %d", len(pool.connections))
	}
}

func TestRegister(t *testing.T) {
	originalLen := len(factories)

	testType := DBType("test_register")
	Register(testType, func(config *Config) (Database, error) {
		return nil, nil
	})

	if len(factories) != originalLen+1 {
		t.Error("Factory was not registered")
	}

	if _, ok := factories[testType]; !ok {
		t.Error("Factory not found in registry")
	}
}

// Mock database implementation for testing
type mockDatabase struct {
	connected   bool
	closeCalled bool
	closeError  error
}

func (m *mockDatabase) Connect(ctx context.Context) error {
	m.connected = true
	return nil
}

func (m *mockDatabase) Close() error {
	m.closeCalled = true
	m.connected = false
	return m.closeError
}

func (m *mockDatabase) Ping(ctx context.Context) error {
	if !m.connected {
		return ErrNotConnected
	}
	return nil
}

func (m *mockDatabase) IsConnected() bool {
	return m.connected
}

func (m *mockDatabase) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return nil, ErrOperationNotSupported
}

func (m *mockDatabase) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return nil
}

func (m *mockDatabase) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, ErrOperationNotSupported
}

func (m *mockDatabase) Begin(ctx context.Context) (*sql.Tx, error) {
	return nil, ErrOperationNotSupported
}

func (m *mockDatabase) GetSQLDB() (*sql.DB, error) {
	return nil, ErrOperationNotSupported
}

func (m *mockDatabase) GetMongoClient() (any, error) {
	return nil, ErrOperationNotSupported
}

func (m *mockDatabase) GetType() DBType {
	return DBType("mock")
}

func (m *mockDatabase) GetConfig() *Config {
	return nil
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
