package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Common errors
var (
	ErrUnsupportedDriver     = errors.New("unsupported database driver")
	ErrNotConnected          = errors.New("database not connected")
	ErrInvalidConfig         = errors.New("invalid database configuration")
	ErrConnectionFailed      = errors.New("failed to connect to database")
	ErrOperationNotSupported = errors.New("operation not supported for this database type")
)

// DBType represents the type of database
type DBType string

const (
	MySQL      DBType = "mysql"
	PostgreSQL DBType = "postgresql"
	MongoDB    DBType = "mongodb"
)

// Config holds database configuration
type Config struct {
	Type            DBType
	Host            string
	Port            int
	Username        string
	Password        string
	Database        string
	SSLMode         string // For PostgreSQL
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	Timeout         time.Duration
	Options         map[string]string
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig(dbType DBType) *Config {
	return &Config{
		Type:            dbType,
		Host:            "localhost",
		Port:            getDefaultPort(dbType),
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
		Timeout:         10 * time.Second,
		SSLMode:         "disable",
		Options:         make(map[string]string),
	}
}

func getDefaultPort(dbType DBType) int {
	switch dbType {
	case MySQL:
		return 3306
	case PostgreSQL:
		return 5432
	case MongoDB:
		return 27017
	default:
		return 0
	}
}

// Database interface defines common operations for all database types
type Database interface {
	// Connection management
	Connect(ctx context.Context) error
	Close() error
	Ping(ctx context.Context) error
	IsConnected() bool

	// SQL database operations (for MySQL and PostgreSQL)
	Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) *sql.Row
	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)

	// Transaction support (for SQL databases)
	Begin(ctx context.Context) (*sql.Tx, error)

	// Get underlying connection
	GetSQLDB() (*sql.DB, error)
	GetMongoClient() (any, error)

	// Metadata
	GetType() DBType
	GetConfig() *Config
}

// FactoryFunc is the function type for creating database instances
type FactoryFunc func(*Config) (Database, error)

var (
	factories = make(map[DBType]FactoryFunc)
)

// Register registers a database factory function
func Register(dbType DBType, factory FactoryFunc) {
	factories[dbType] = factory
}

// New creates a new database instance based on configuration
func New(config *Config) (Database, error) {
	if config == nil {
		return nil, ErrInvalidConfig
	}

	// Validate required fields
	if config.Type == "" {
		return nil, fmt.Errorf("%w: database type is required", ErrInvalidConfig)
	}

	if config.Host == "" {
		return nil, fmt.Errorf("%w: host is required", ErrInvalidConfig)
	}

	// Set defaults if not provided
	if config.Port <= 0 {
		config.Port = getDefaultPort(config.Type)
	}
	if config.MaxOpenConns <= 0 {
		config.MaxOpenConns = 25
	}
	if config.MaxIdleConns <= 0 {
		config.MaxIdleConns = 5
	}
	if config.ConnMaxLifetime <= 0 {
		config.ConnMaxLifetime = 5 * time.Minute
	}
	if config.ConnMaxIdleTime <= 0 {
		config.ConnMaxIdleTime = 10 * time.Minute
	}
	if config.Timeout <= 0 {
		config.Timeout = 10 * time.Second
	}
	if config.Options == nil {
		config.Options = make(map[string]string)
	}

	// Get factory for the database type
	factory, ok := factories[config.Type]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedDriver, config.Type)
	}

	return factory(config)
}

// ConnectionPool manages multiple database connections
type ConnectionPool struct {
	connections map[string]Database
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool() *ConnectionPool {
	return &ConnectionPool{
		connections: make(map[string]Database),
	}
}

// Add adds a database connection to the pool
func (p *ConnectionPool) Add(name string, db Database) error {
	if db == nil {
		return errors.New("database cannot be nil")
	}
	p.connections[name] = db
	return nil
}

// Get retrieves a database connection from the pool
func (p *ConnectionPool) Get(name string) (Database, error) {
	db, ok := p.connections[name]
	if !ok {
		return nil, fmt.Errorf("database connection '%s' not found", name)
	}
	return db, nil
}

// Remove removes a database connection from the pool
func (p *ConnectionPool) Remove(name string) error {
	if db, ok := p.connections[name]; ok {
		if err := db.Close(); err != nil {
			return err
		}
		delete(p.connections, name)
	}
	return nil
}

// CloseAll closes all database connections in the pool
func (p *ConnectionPool) CloseAll() error {
	var errs []error
	for name, db := range p.connections {
		if err := db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close '%s': %w", name, err))
		}
	}
	p.connections = make(map[string]Database)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}
	return nil
}

// List returns the names of all connections in the pool
func (p *ConnectionPool) List() []string {
	names := make([]string, 0, len(p.connections))
	for name := range p.connections {
		names = append(names, name)
	}
	return names
}
