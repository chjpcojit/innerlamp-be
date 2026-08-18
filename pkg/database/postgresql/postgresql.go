package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"base_project/pkg/database"

	_ "github.com/lib/pq"
)

func init() {
	// Register PostgreSQL factory with the database package
	database.Register(database.PostgreSQL, NewPostgreSQL)
}

// PostgreSQL represents a PostgreSQL database connection
type PostgreSQL struct {
	db        *sql.DB
	config    *database.Config
	connected bool
}

// NewPostgreSQL creates a new PostgreSQL database instance
func NewPostgreSQL(config *database.Config) (database.Database, error) {
	if config == nil {
		return nil, database.ErrInvalidConfig
	}

	return &PostgreSQL{
		config:    config,
		connected: false,
	}, nil
}

// buildDSN builds a PostgreSQL connection string from the config
func (p *PostgreSQL) buildDSN() string {
	// Check if DSN is provided in options
	if dsn, ok := p.config.Options["dsn"]; ok {
		return dsn
	}

	// Build connection string
	// Format: host=localhost port=5432 user=username password=password dbname=database sslmode=disable
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		p.config.Host,
		p.config.Port,
		p.config.Username,
		p.config.Password,
		p.config.Database,
		p.config.SSLMode,
	)

	// Add timeout
	if p.config.Timeout > 0 {
		dsn += fmt.Sprintf(" connect_timeout=%d", int(p.config.Timeout.Seconds()))
	}

	// Add custom options
	for key, value := range p.config.Options {
		if key != "dsn" {
			dsn += fmt.Sprintf(" %s=%s", key, value)
		}
	}

	return dsn
}

// Connect establishes a connection to the PostgreSQL database
func (p *PostgreSQL) Connect(ctx context.Context) error {
	if p.connected {
		return nil
	}

	dsn := p.buildDSN()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("%w: %v", database.ErrConnectionFailed, err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(p.config.MaxOpenConns)
	db.SetMaxIdleConns(p.config.MaxIdleConns)
	db.SetConnMaxLifetime(p.config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(p.config.ConnMaxIdleTime)

	// Test the connection
	ctx, cancel := context.WithTimeout(ctx, p.config.Timeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("%w: %v", database.ErrConnectionFailed, err)
	}

	p.db = db
	p.connected = true
	return nil
}

// Close closes the database connection
func (p *PostgreSQL) Close() error {
	if !p.connected || p.db == nil {
		return nil
	}

	err := p.db.Close()
	p.connected = false
	p.db = nil
	return err
}

// Ping checks if the database connection is alive
func (p *PostgreSQL) Ping(ctx context.Context) error {
	if !p.connected || p.db == nil {
		return database.ErrNotConnected
	}

	ctx, cancel := context.WithTimeout(ctx, p.config.Timeout)
	defer cancel()

	return p.db.PingContext(ctx)
}

// IsConnected returns true if the database is connected
func (p *PostgreSQL) IsConnected() bool {
	return p.connected && p.db != nil
}

// Query executes a query that returns rows
func (p *PostgreSQL) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if !p.connected || p.db == nil {
		return nil, database.ErrNotConnected
	}

	return p.db.QueryContext(ctx, query, args...)
}

// QueryRow executes a query that returns at most one row
func (p *PostgreSQL) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	if !p.connected || p.db == nil {
		// Return a row that will error on Scan
		return &sql.Row{}
	}

	return p.db.QueryRowContext(ctx, query, args...)
}

// Exec executes a query that doesn't return rows
func (p *PostgreSQL) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if !p.connected || p.db == nil {
		return nil, database.ErrNotConnected
	}

	return p.db.ExecContext(ctx, query, args...)
}

// Begin starts a new transaction
func (p *PostgreSQL) Begin(ctx context.Context) (*sql.Tx, error) {
	if !p.connected || p.db == nil {
		return nil, database.ErrNotConnected
	}

	return p.db.BeginTx(ctx, nil)
}

// GetSQLDB returns the underlying *sql.DB
func (p *PostgreSQL) GetSQLDB() (*sql.DB, error) {
	if !p.connected || p.db == nil {
		return nil, database.ErrNotConnected
	}

	return p.db, nil
}

// GetMongoClient returns an error as PostgreSQL doesn't support MongoDB operations
func (p *PostgreSQL) GetMongoClient() (any, error) {
	return nil, database.ErrOperationNotSupported
}

// GetType returns the database type
func (p *PostgreSQL) GetType() database.DBType {
	return database.PostgreSQL
}

// GetConfig returns the database configuration
func (p *PostgreSQL) GetConfig() *database.Config {
	return p.config
}

// GetStats returns database statistics
func (p *PostgreSQL) GetStats() sql.DBStats {
	if p.db == nil {
		return sql.DBStats{}
	}
	return p.db.Stats()
}

// HealthCheck performs a health check on the database
func (p *PostgreSQL) HealthCheck(ctx context.Context) error {
	if err := p.Ping(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	// Try a simple query
	var result int
	query := "SELECT 1"
	if err := p.db.QueryRowContext(ctx, query).Scan(&result); err != nil {
		return fmt.Errorf("health check query failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("unexpected health check result: %d", result)
	}

	return nil
}

// SetMaxOpenConns sets the maximum number of open connections
func (p *PostgreSQL) SetMaxOpenConns(n int) {
	if p.db != nil {
		p.db.SetMaxOpenConns(n)
		p.config.MaxOpenConns = n
	}
}

// SetMaxIdleConns sets the maximum number of idle connections
func (p *PostgreSQL) SetMaxIdleConns(n int) {
	if p.db != nil {
		p.db.SetMaxIdleConns(n)
		p.config.MaxIdleConns = n
	}
}

// SetConnMaxLifetime sets the maximum lifetime of connections
func (p *PostgreSQL) SetConnMaxLifetime(d time.Duration) {
	if p.db != nil {
		p.db.SetConnMaxLifetime(d)
		p.config.ConnMaxLifetime = d
	}
}

// SetConnMaxIdleTime sets the maximum idle time of connections
func (p *PostgreSQL) SetConnMaxIdleTime(d time.Duration) {
	if p.db != nil {
		p.db.SetConnMaxIdleTime(d)
		p.config.ConnMaxIdleTime = d
	}
}
