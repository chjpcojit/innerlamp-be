# Database Package

A comprehensive database abstraction layer for Go that supports multiple database systems with a unified interface.

## Features

- **Multi-Database Support**: MySQL, PostgreSQL, and MongoDB
- **Unified Interface**: Common interface for all database operations
- **Factory Pattern**: Runtime database selection based on configuration
- **Connection Management**: Built-in connection pooling and lifecycle management
- **Error Handling**: Consistent error handling across all database types
- **Connection Pool**: Manage multiple database connections simultaneously
- **Health Checks**: Built-in health check functionality
- **Simple API**: Easy to use with minimal boilerplate

## Supported Databases

- **MySQL** - Using `github.com/go-sql-driver/mysql`
- **PostgreSQL** - Using `github.com/lib/pq`
- **MongoDB** - Using `go.mongodb.org/mongo-driver`

## Installation

```bash
go get base_project/pkg/database
```

## Quick Start

### MySQL

```go
package main

import (
    "context"
    "log"
    "time"
    
    "base_project/pkg/database"
    _ "base_project/pkg/database/mysql"
)

func main() {
    // Create configuration
    config := &database.Config{
        Type:            database.MySQL,
        Host:            "localhost",
        Port:            3306,
        Username:        "root",
        Password:        "password",
        Database:        "mydb",
        MaxOpenConns:    25,
        MaxIdleConns:    5,
        ConnMaxLifetime: 5 * time.Minute,
        Timeout:         10 * time.Second,
    }

    // Create database instance
    db, err := database.New(config)
    if err != nil {
        log.Fatal(err)
    }

    // Connect
    ctx := context.Background()
    if err := db.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Query
    rows, err := db.Query(ctx, "SELECT id, name FROM users WHERE active = ?", true)
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()

    for rows.Next() {
        var id int
        var name string
        if err := rows.Scan(&id, &name); err != nil {
            log.Fatal(err)
        }
        log.Printf("User: %d - %s", id, name)
    }
}
```

### PostgreSQL

```go
package main

import (
    "context"
    "log"
    
    "base_project/pkg/database"
    _ "base_project/pkg/database/postgresql"
)

func main() {
    // Create configuration
    config := database.DefaultConfig(database.PostgreSQL)
    config.Username = "postgres"
    config.Password = "password"
    config.Database = "mydb"
    config.SSLMode = "disable"

    // Create and connect
    db, err := database.New(config)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()
    if err := db.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // PostgreSQL uses $1, $2 placeholders
    var count int
    err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE active = $1", true).Scan(&count)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Active users: %d", count)
}
```

### MongoDB

```go
package main

import (
    "context"
    "log"
    
    "base_project/pkg/database"
    _ "base_project/pkg/database/mongodb"
    "go.mongodb.org/mongo-driver/mongo"
)

func main() {
    // Create configuration
    config := &database.Config{
        Type:     database.MongoDB,
        Host:     "localhost",
        Port:     27017,
        Username: "admin",
        Password: "password",
        Database: "mydb",
        Timeout:  10 * time.Second,
    }

    // Create and connect
    db, err := database.New(config)
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()
    if err := db.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Get MongoDB client
    clientInterface, err := db.GetMongoClient()
    if err != nil {
        log.Fatal(err)
    }

    // Type assert to MongoDB client
    client := clientInterface.(*mongo.Client)
    collection := client.Database("mydb").Collection("users")
    
    // Use MongoDB operations
    // ...
}
```

## Configuration

### Config Structure

```go
type Config struct {
    Type            DBType            // Database type (MySQL, PostgreSQL, MongoDB)
    Host            string            // Database host
    Port            int               // Database port
    Username        string            // Username
    Password        string            // Password
    Database        string            // Database name
    SSLMode         string            // SSL mode (PostgreSQL)
    MaxOpenConns    int               // Maximum open connections
    MaxIdleConns    int               // Maximum idle connections
    ConnMaxLifetime time.Duration     // Connection max lifetime
    ConnMaxIdleTime time.Duration     // Connection max idle time
    Timeout         time.Duration     // Connection timeout
    Options         map[string]string // Additional options
}
```

### Default Configuration

Use `DefaultConfig()` to get sensible defaults:

```go
config := database.DefaultConfig(database.MySQL)
config.Username = "root"
config.Password = "password"
config.Database = "mydb"
```

Default values:
- **Port**: 3306 (MySQL), 5432 (PostgreSQL), 27017 (MongoDB)
- **MaxOpenConns**: 25
- **MaxIdleConns**: 5
- **ConnMaxLifetime**: 5 minutes
- **ConnMaxIdleTime**: 10 minutes
- **Timeout**: 10 seconds
- **SSLMode**: "disable" (PostgreSQL)

## Interface

### Database Interface

```go
type Database interface {
    // Connection management
    Connect(ctx context.Context) error
    Close() error
    Ping(ctx context.Context) error
    IsConnected() bool

    // SQL database operations (MySQL and PostgreSQL)
    Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)
    QueryRow(ctx context.Context, query string, args ...any) *sql.Row
    Exec(ctx context.Context, query string, args ...any) (sql.Result, error)

    // Transaction support (SQL databases)
    Begin(ctx context.Context) (*sql.Tx, error)

    // Get underlying connection
    GetSQLDB() (*sql.DB, error)
    GetMongoClient() (any, error)

    // Metadata
    GetType() DBType
    GetConfig() *Config
}
```

## Transactions

### SQL Databases (MySQL, PostgreSQL)

```go
// Begin transaction
tx, err := db.Begin(ctx)
if err != nil {
    log.Fatal(err)
}

// Execute queries
_, err = tx.ExecContext(ctx, "UPDATE accounts SET balance = balance - 100 WHERE id = ?", 1)
if err != nil {
    tx.Rollback()
    log.Fatal(err)
}

_, err = tx.ExecContext(ctx, "UPDATE accounts SET balance = balance + 100 WHERE id = ?", 2)
if err != nil {
    tx.Rollback()
    log.Fatal(err)
}

// Commit
if err := tx.Commit(); err != nil {
    log.Fatal(err)
}
```

### MongoDB Sessions

```go
// For MongoDB, you need to access the client directly
clientInterface, _ := db.GetMongoClient()
client := clientInterface.(*mongo.Client)

session, err := client.StartSession()
if err != nil {
    log.Fatal(err)
}
defer session.EndSession(ctx)

// Use session for transactions
// ...
```

## Connection Pool

Manage multiple database connections:

```go
// Create pool
pool := database.NewConnectionPool()

// Add MySQL connection
mysqlDB, _ := database.New(mysqlConfig)
mysqlDB.Connect(ctx)
pool.Add("mysql_primary", mysqlDB)

// Add PostgreSQL connection
pgDB, _ := database.New(pgConfig)
pgDB.Connect(ctx)
pool.Add("postgres_analytics", pgDB)

// Get connection
db, err := pool.Get("mysql_primary")
if err != nil {
    log.Fatal(err)
}

// Use connection
db.Query(ctx, "SELECT * FROM users")

// List all connections
connections := pool.List() // ["mysql_primary", "postgres_analytics"]

// Remove specific connection
pool.Remove("mysql_primary")

// Close all connections
pool.CloseAll()
```

## Error Handling

The package defines common errors:

```go
var (
    ErrUnsupportedDriver     = errors.New("unsupported database driver")
    ErrNotConnected          = errors.New("database not connected")
    ErrInvalidConfig         = errors.New("invalid database configuration")
    ErrConnectionFailed      = errors.New("failed to connect to database")
    ErrOperationNotSupported = errors.New("operation not supported for this database type")
)
```

Example error handling:

```go
if err := db.Connect(ctx); err != nil {
    switch {
    case errors.Is(err, database.ErrConnectionFailed):
        log.Println("Connection failed - check credentials")
    case errors.Is(err, database.ErrInvalidConfig):
        log.Println("Invalid configuration")
    default:
        log.Printf("Unexpected error: %v", err)
    }
}
```

## Health Checks

```go
// Simple ping
if err := db.Ping(ctx); err != nil {
    log.Printf("Database unhealthy: %v", err)
}

// Check connection status
if db.IsConnected() {
    log.Println("Database is connected")
}

// Get database type
dbType := db.GetType()
log.Printf("Database type: %s", dbType)
```

## Advanced Usage

### Custom Options

```go
config := database.DefaultConfig(database.MySQL)
config.Options["charset"] = "utf8mb4"
config.Options["parseTime"] = "true"
config.Options["loc"] = "Local"
```

### Direct DSN/URI

```go
// MySQL
config.Options["dsn"] = "user:password@tcp(localhost:3306)/dbname?charset=utf8mb4"

// PostgreSQL
config.Options["dsn"] = "host=localhost port=5432 user=postgres password=secret dbname=mydb"

// MongoDB
config.Options["uri"] = "mongodb://user:password@localhost:27017/dbname"
```

### Getting Underlying Connection

```go
// For SQL databases (MySQL, PostgreSQL)
sqlDB, err := db.GetSQLDB()
if err != nil {
    log.Fatal(err)
}

// Now you can use sql.DB directly
stats := sqlDB.Stats()
log.Printf("Open connections: %d", stats.OpenConnections)

// For MongoDB
clientInterface, err := db.GetMongoClient()
if err != nil {
    log.Fatal(err)
}

mongoClient := clientInterface.(*mongo.Client)
// Use MongoDB client directly
```

## Best Practices

1. **Always close connections**: Use `defer db.Close()`
2. **Use contexts**: Always pass context for cancellation support
3. **Handle errors**: Check all errors, especially connection errors
4. **Connection pooling**: Configure appropriate pool sizes for your workload
5. **Health checks**: Implement regular health checks in production
6. **Transactions**: Use transactions for operations that must be atomic
7. **Timeouts**: Set appropriate timeouts based on your requirements

## Examples

See `example_usage.go` for complete examples:
- `ExampleMySQL()` - MySQL usage
- `ExamplePostgreSQL()` - PostgreSQL usage
- `ExampleMongoDB()` - MongoDB usage
- `ExampleConnectionPool()` - Connection pool usage
- `ExampleWithDefaultConfig()` - Using default configuration
- `ExampleErrorHandling()` - Error handling patterns

## Testing

```go
func TestDatabaseConnection(t *testing.T) {
    config := database.DefaultConfig(database.MySQL)
    config.Username = "test"
    config.Password = "test"
    config.Database = "test_db"

    db, err := database.New(config)
    if err != nil {
        t.Fatalf("Failed to create database: %v", err)
    }

    ctx := context.Background()
    if err := db.Connect(ctx); err != nil {
        t.Fatalf("Failed to connect: %v", err)
    }
    defer db.Close()

    if !db.IsConnected() {
        t.Error("Database should be connected")
    }

    if err := db.Ping(ctx); err != nil {
        t.Errorf("Ping failed: %v", err)
    }
}
```

## Contributing

When adding a new database implementation:

1. Create a new package under `pkg/database/`
2. Implement the `Database` interface
3. Register the factory function in `init()`
4. Add tests and documentation

Example:

```go
package mydb

import "base_project/pkg/database"

func init() {
    database.Register(database.DBType("mydb"), NewMyDB)
}

type MyDB struct {
    // implementation
}

func NewMyDB(config *database.Config) (database.Database, error) {
    // create and return instance
}
```

## License

See the project's main LICENSE file.