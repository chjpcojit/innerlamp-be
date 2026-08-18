package mongodb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"base_project/pkg/database"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func init() {
	// Register MongoDB factory with the database package
	database.Register(database.MongoDB, NewMongoDB)
}

// MongoDB represents a MongoDB database connection
type MongoDB struct {
	client    *mongo.Client
	database  *mongo.Database
	config    *database.Config
	connected bool
}

// NewMongoDB creates a new MongoDB database instance
func NewMongoDB(config *database.Config) (database.Database, error) {
	if config == nil {
		return nil, database.ErrInvalidConfig
	}

	return &MongoDB{
		config:    config,
		connected: false,
	}, nil
}

// buildURI builds a MongoDB connection URI from the config
func (m *MongoDB) buildURI() string {
	// Check if URI is provided in options
	if uri, ok := m.config.Options["uri"]; ok {
		return uri
	}

	// Build URI
	// Format: mongodb://[username:password@]host[:port][/[database][?options]]
	var uri string
	if m.config.Username != "" && m.config.Password != "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%d",
			m.config.Username,
			m.config.Password,
			m.config.Host,
			m.config.Port,
		)
	} else {
		uri = fmt.Sprintf("mongodb://%s:%d", m.config.Host, m.config.Port)
	}

	// Add database name if specified
	if m.config.Database != "" {
		uri += "/" + m.config.Database
	}

	// Add options
	if len(m.config.Options) > 0 {
		uri += "?"
		first := true
		for key, value := range m.config.Options {
			if key != "uri" {
				if !first {
					uri += "&"
				}
				uri += fmt.Sprintf("%s=%s", key, value)
				first = false
			}
		}
	}

	return uri
}

// Connect establishes a connection to the MongoDB database
func (m *MongoDB) Connect(ctx context.Context) error {
	if m.connected {
		return nil
	}

	uri := m.buildURI()

	// Set client options
	clientOpts := options.Client().ApplyURI(uri)

	// Set timeout
	if m.config.Timeout > 0 {
		clientOpts.SetConnectTimeout(m.config.Timeout)
		clientOpts.SetServerSelectionTimeout(m.config.Timeout)
	}

	// Set connection pool settings
	if m.config.MaxOpenConns > 0 {
		maxPoolSize := uint64(m.config.MaxOpenConns)
		clientOpts.SetMaxPoolSize(maxPoolSize)
	}

	if m.config.MaxIdleConns > 0 {
		minPoolSize := uint64(m.config.MaxIdleConns)
		clientOpts.SetMinPoolSize(minPoolSize)
	}

	if m.config.ConnMaxLifetime > 0 {
		clientOpts.SetMaxConnIdleTime(m.config.ConnMaxLifetime)
	}

	// Create MongoDB client
	ctx, cancel := context.WithTimeout(ctx, m.config.Timeout)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return fmt.Errorf("%w: %v", database.ErrConnectionFailed, err)
	}

	// Test the connection
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(ctx)
		return fmt.Errorf("%w: %v", database.ErrConnectionFailed, err)
	}

	m.client = client
	if m.config.Database != "" {
		m.database = client.Database(m.config.Database)
	}
	m.connected = true

	return nil
}

// Close closes the database connection
func (m *MongoDB) Close() error {
	if !m.connected || m.client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := m.client.Disconnect(ctx)
	m.connected = false
	m.client = nil
	m.database = nil
	return err
}

// Ping checks if the database connection is alive
func (m *MongoDB) Ping(ctx context.Context) error {
	if !m.connected || m.client == nil {
		return database.ErrNotConnected
	}

	ctx, cancel := context.WithTimeout(ctx, m.config.Timeout)
	defer cancel()

	return m.client.Ping(ctx, readpref.Primary())
}

// IsConnected returns true if the database is connected
func (m *MongoDB) IsConnected() bool {
	return m.connected && m.client != nil
}

// Query is not supported for MongoDB (SQL operation)
func (m *MongoDB) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return nil, database.ErrOperationNotSupported
}

// QueryRow is not supported for MongoDB (SQL operation)
func (m *MongoDB) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return &sql.Row{}
}

// Exec is not supported for MongoDB (SQL operation)
func (m *MongoDB) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, database.ErrOperationNotSupported
}

// Begin is not supported for MongoDB (use StartSession instead)
func (m *MongoDB) Begin(ctx context.Context) (*sql.Tx, error) {
	return nil, database.ErrOperationNotSupported
}

// GetSQLDB returns an error as MongoDB doesn't support SQL operations
func (m *MongoDB) GetSQLDB() (*sql.DB, error) {
	return nil, database.ErrOperationNotSupported
}

// GetMongoClient returns the underlying MongoDB client
func (m *MongoDB) GetMongoClient() (any, error) {
	if !m.connected || m.client == nil {
		return nil, database.ErrNotConnected
	}

	return m.client, nil
}

// GetType returns the database type
func (m *MongoDB) GetType() database.DBType {
	return database.MongoDB
}

// GetConfig returns the database configuration
func (m *MongoDB) GetConfig() *database.Config {
	return m.config
}

// HealthCheck performs a health check on the database
func (m *MongoDB) HealthCheck(ctx context.Context) error {
	if err := m.Ping(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	// Try to list databases to verify connection
	ctx, cancel := context.WithTimeout(ctx, m.config.Timeout)
	defer cancel()

	_, err := m.client.ListDatabaseNames(ctx, map[string]any{})
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// GetDatabase returns the MongoDB database instance
func (m *MongoDB) GetDatabase() (*mongo.Database, error) {
	if !m.connected || m.database == nil {
		return nil, database.ErrNotConnected
	}
	return m.database, nil
}

// GetClient returns the MongoDB client instance
func (m *MongoDB) GetClient() (*mongo.Client, error) {
	if !m.connected || m.client == nil {
		return nil, database.ErrNotConnected
	}
	return m.client, nil
}

// Collection returns a handle to a MongoDB collection
func (m *MongoDB) Collection(name string) (*mongo.Collection, error) {
	if !m.connected || m.database == nil {
		return nil, database.ErrNotConnected
	}
	return m.database.Collection(name), nil
}

// StartSession starts a new MongoDB session (for transactions)
func (m *MongoDB) StartSession(opts ...*options.SessionOptions) (mongo.Session, error) {
	if !m.connected || m.client == nil {
		return nil, database.ErrNotConnected
	}
	return m.client.StartSession(opts...)
}

// RunCommand runs a database command
func (m *MongoDB) RunCommand(ctx context.Context, command any) (*mongo.SingleResult, error) {
	if !m.connected || m.database == nil {
		return nil, database.ErrNotConnected
	}
	result := m.database.RunCommand(ctx, command)
	return result, nil
}

// ListCollections returns a list of collection names in the database
func (m *MongoDB) ListCollections(ctx context.Context) ([]string, error) {
	if !m.connected || m.database == nil {
		return nil, database.ErrNotConnected
	}

	return m.database.ListCollectionNames(ctx, map[string]any{})
}

// CreateCollection creates a new collection in the database
func (m *MongoDB) CreateCollection(ctx context.Context, name string, opts ...*options.CreateCollectionOptions) error {
	if !m.connected || m.database == nil {
		return database.ErrNotConnected
	}

	return m.database.CreateCollection(ctx, name, opts...)
}

// DropCollection drops a collection from the database
func (m *MongoDB) DropCollection(ctx context.Context, name string) error {
	if !m.connected || m.database == nil {
		return database.ErrNotConnected
	}

	collection := m.database.Collection(name)
	return collection.Drop(ctx)
}
