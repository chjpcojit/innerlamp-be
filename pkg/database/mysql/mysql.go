package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"base_project/pkg/database"

	_ "github.com/go-sql-driver/mysql"
)

func init() {
	// Register MySQL factory with the database package
	database.Register(database.MySQL, NewMySQL)
}

// MySQL represents a MySQL database connection
type MySQL struct {
	db        *sql.DB
	config    *database.Config
	connected bool
}

// NewMySQL creates a new MySQL database instance
func NewMySQL(config *database.Config) (database.Database, error) {
	if config == nil {
		return nil, database.ErrInvalidConfig
	}

	return &MySQL{
		config:    config,
		connected: false,
	}, nil
}

// buildDSN builds a MySQL DSN string from the config
func (m *MySQL) buildDSN() string {
	// Check if DSN is provided in options
	if dsn, ok := m.config.Options["dsn"]; ok {
		return dsn
	}

	// Build DSN: username:password@tcp(host:port)/database?params
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		m.config.Username,
		m.config.Password,
		m.config.Host,
		m.config.Port,
		m.config.Database,
	)

	// Add timeout and other parameters
	params := fmt.Sprintf("?timeout=%s&readTimeout=%s&writeTimeout=%s&parseTime=true",
		m.config.Timeout.String(),
		m.config.Timeout.String(),
		m.config.Timeout.String(),
	)

	// Add custom options
	for key, value := range m.config.Options {
		if key != "dsn" {
			params += fmt.Sprintf("&%s=%s", key, value)
		}
	}

	return dsn + params
}

// Connect establishes a connection to the MySQL database
func (m *MySQL) Connect(ctx context.Context) error {
	if m.connected {
		return nil
	}

	dsn := m.buildDSN()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("%w: %v", database.ErrConnectionFailed, err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(m.config.MaxOpenConns)
	db.SetMaxIdleConns(m.config.MaxIdleConns)
	db.SetConnMaxLifetime(m.config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(m.config.ConnMaxIdleTime)

	// Test the connection
	ctx, cancel := context.WithTimeout(ctx, m.config.Timeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("%w: %v", database.ErrConnectionFailed, err)
	}

	m.db = db
	m.connected = true
	return nil
}

// Close closes the database connection
func (m *MySQL) Close() error {
	if !m.connected || m.db == nil {
		return nil
	}

	err := m.db.Close()
	m.connected = false
	m.db = nil
	return err
}

// Ping checks if the database connection is alive
func (m *MySQL) Ping(ctx context.Context) error {
	if !m.connected || m.db == nil {
		return database.ErrNotConnected
	}

	ctx, cancel := context.WithTimeout(ctx, m.config.Timeout)
	defer cancel()

	return m.db.PingContext(ctx)
}

// IsConnected returns true if the database is connected
func (m *MySQL) IsConnected() bool {
	return m.connected && m.db != nil
}

// Query executes a query that returns rows
func (m *MySQL) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if !m.connected || m.db == nil {
		return nil, database.ErrNotConnected
	}

	return m.db.QueryContext(ctx, query, args...)
}

// QueryRow executes a query that returns at most one row
func (m *MySQL) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	if !m.connected || m.db == nil {
		// Return a row that will error on Scan
		return &sql.Row{}
	}

	return m.db.QueryRowContext(ctx, query, args...)
}

// Exec executes a query that doesn't return rows
func (m *MySQL) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if !m.connected || m.db == nil {
		return nil, database.ErrNotConnected
	}

	return m.db.ExecContext(ctx, query, args...)
}

// Begin starts a new transaction
func (m *MySQL) Begin(ctx context.Context) (*sql.Tx, error) {
	if !m.connected || m.db == nil {
		return nil, database.ErrNotConnected
	}

	return m.db.BeginTx(ctx, nil)
}

// GetSQLDB returns the underlying *sql.DB
func (m *MySQL) GetSQLDB() (*sql.DB, error) {
	if !m.connected || m.db == nil {
		return nil, database.ErrNotConnected
	}

	return m.db, nil
}

// GetMongoClient returns an error as MySQL doesn't support MongoDB operations
func (m *MySQL) GetMongoClient() (any, error) {
	return nil, database.ErrOperationNotSupported
}

// GetType returns the database type
func (m *MySQL) GetType() database.DBType {
	return database.MySQL
}

// GetConfig returns the database configuration
func (m *MySQL) GetConfig() *database.Config {
	return m.config
}

// GetStats returns database statistics
func (m *MySQL) GetStats() sql.DBStats {
	if m.db == nil {
		return sql.DBStats{}
	}
	return m.db.Stats()
}

// HealthCheck performs a health check on the database
func (m *MySQL) HealthCheck(ctx context.Context) error {
	if err := m.Ping(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	// Try a simple query
	var result int
	query := "SELECT 1"
	if err := m.db.QueryRowContext(ctx, query).Scan(&result); err != nil {
		return fmt.Errorf("health check query failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("unexpected health check result: %d", result)
	}

	return nil
}

// SetMaxOpenConns sets the maximum number of open connections
func (m *MySQL) SetMaxOpenConns(n int) {
	if m.db != nil {
		m.db.SetMaxOpenConns(n)
		m.config.MaxOpenConns = n
	}
}

// SetMaxIdleConns sets the maximum number of idle connections
func (m *MySQL) SetMaxIdleConns(n int) {
	if m.db != nil {
		m.db.SetMaxIdleConns(n)
		m.config.MaxIdleConns = n
	}
}

// SetConnMaxLifetime sets the maximum lifetime of connections
func (m *MySQL) SetConnMaxLifetime(d time.Duration) {
	if m.db != nil {
		m.db.SetConnMaxLifetime(d)
		m.config.ConnMaxLifetime = d
	}
}

// SetConnMaxIdleTime sets the maximum idle time of connections
func (m *MySQL) SetConnMaxIdleTime(d time.Duration) {
	if m.db != nil {
		m.db.SetConnMaxIdleTime(d)
		m.config.ConnMaxIdleTime = d
	}
}
