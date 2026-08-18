# Database Package - Documentation Index

## Overview

Welcome to the multi-database abstraction layer for Go. This package provides a unified interface for working with MySQL, PostgreSQL, and MongoDB databases.

## Quick Navigation

### 📚 Documentation Files

1. **[QUICK_START.md](QUICK_START.md)** - ⚡ Start here!
   - 5-minute quick start guide
   - Simple examples for each database type
   - Common operations and patterns
   - Best practices and troubleshooting

2. **[README.md](README.md)** - 📖 Complete documentation
   - Comprehensive API reference
   - Detailed feature descriptions
   - Configuration guide
   - Advanced usage patterns
   - Testing guidelines

3. **[IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md)** - 🏗️ Architecture
   - Design patterns used
   - Implementation details
   - File structure overview
   - Extensibility guide
   - Performance and security considerations

4. **[INDEX.md](INDEX.md)** - 📑 This file
   - Documentation overview
   - Quick reference to all resources

### 💻 Source Files

#### Core Package
- **[database.go](database.go)** - Main interface, factory pattern, and connection pool
- **[database_test.go](database_test.go)** - Unit tests

#### Database Implementations
- **[mysql/mysql.go](mysql/mysql.go)** - MySQL implementation
- **[postgresql/postgresql.go](postgresql/postgresql.go)** - PostgreSQL implementation
- **[mongodb/mongodb.go](mongodb/mongodb.go)** - MongoDB implementation

#### Examples
- **[../../examples/database_examples.go](../../examples/database_examples.go)** - Complete usage examples

## Quick Reference

### Supported Databases

| Database   | Driver Package                   | Port  | Status |
|------------|----------------------------------|-------|--------|
| MySQL      | github.com/go-sql-driver/mysql   | 3306  | ✅     |
| PostgreSQL | github.com/lib/pq                | 5432  | ✅     |
| MongoDB    | go.mongodb.org/mongo-driver      | 27017 | ✅     |

### Key Features

- ✅ Unified Database Interface
- ✅ Factory Pattern for Runtime Selection
- ✅ Connection Pooling
- ✅ Transaction Support (SQL databases)
- ✅ Context-based Operations
- ✅ Connection Pool Manager
- ✅ Health Checks
- ✅ Comprehensive Error Handling
- ✅ Fully Tested
- ✅ Production Ready

### Basic Usage

```go
import (
    "base_project/pkg/database"
    _ "base_project/pkg/database/mysql"
)

config := database.DefaultConfig(database.MySQL)
config.Username = "root"
config.Password = "password"
config.Database = "mydb"

db, err := database.New(config)
if err != nil {
    log.Fatal(err)
}

ctx := context.Background()
if err := db.Connect(ctx); err != nil {
    log.Fatal(err)
}
defer db.Close()

rows, err := db.Query(ctx, "SELECT * FROM users")
// Use rows...
```

## Getting Started Path

### For New Users
1. Read [QUICK_START.md](QUICK_START.md) - Get up and running in 5 minutes
2. Try the examples in [database_examples.go](../../examples/database_examples.go)
3. Reference [README.md](README.md) for specific features

### For Developers
1. Review [IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md) - Understand the architecture
2. Read the source code in order:
   - [database.go](database.go) - Core interface
   - [mysql/mysql.go](mysql/mysql.go) - Example implementation
   - [database_test.go](database_test.go) - Testing patterns
3. Check [README.md](README.md) for extending the package

### For Contributors
1. Study [IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md) - Design patterns
2. Review [database_test.go](database_test.go) - Testing requirements
3. Read "Contributing" section in [README.md](README.md)

## Common Tasks

### Create New Database Connection
→ See [QUICK_START.md - 5-Minute Quick Start](QUICK_START.md#5-minute-quick-start)

### Perform Transactions
→ See [QUICK_START.md - Transactions](QUICK_START.md#transactions)

### Use Connection Pool
→ See [QUICK_START.md - Connection Pool](QUICK_START.md#connection-pool)

### Handle Errors
→ See [QUICK_START.md - Error Handling](QUICK_START.md#error-handling)

### Configure Database
→ See [README.md - Configuration](README.md#configuration)

### Add New Database Type
→ See [IMPLEMENTATION_SUMMARY.md - Extensibility](IMPLEMENTATION_SUMMARY.md#extensibility)

### Run Tests
```bash
go test ./pkg/database -v
```
→ See [README.md - Testing](README.md#testing)

## API Quick Reference

### Core Functions
- `database.New(config)` - Create database instance
- `database.DefaultConfig(dbType)` - Get default configuration
- `database.NewConnectionPool()` - Create connection pool
- `database.Register(dbType, factory)` - Register database factory

### Database Interface Methods
- `Connect(ctx)` - Establish connection
- `Close()` - Close connection
- `Ping(ctx)` - Check connection health
- `IsConnected()` - Get connection status
- `Query(ctx, query, args...)` - Execute query
- `QueryRow(ctx, query, args...)` - Query single row
- `Exec(ctx, query, args...)` - Execute statement
- `Begin(ctx)` - Start transaction
- `GetSQLDB()` - Get raw SQL connection
- `GetMongoClient()` - Get MongoDB client
- `GetType()` - Get database type
- `GetConfig()` - Get configuration

### Connection Pool Methods
- `Add(name, db)` - Add connection
- `Get(name)` - Get connection
- `Remove(name)` - Remove connection
- `CloseAll()` - Close all connections
- `List()` - List connection names

## Error Types

- `ErrUnsupportedDriver` - Unknown database type
- `ErrNotConnected` - Database not connected
- `ErrInvalidConfig` - Invalid configuration
- `ErrConnectionFailed` - Connection failed
- `ErrOperationNotSupported` - Operation not supported

## Database Types

- `database.MySQL` - MySQL database
- `database.PostgreSQL` - PostgreSQL database
- `database.MongoDB` - MongoDB database

## Configuration Structure

```go
type Config struct {
    Type            DBType            // Database type
    Host            string            // Host address
    Port            int               // Port number
    Username        string            // Username
    Password        string            // Password
    Database        string            // Database name
    SSLMode         string            // SSL mode (PostgreSQL)
    MaxOpenConns    int               // Max open connections
    MaxIdleConns    int               // Max idle connections
    ConnMaxLifetime time.Duration     // Connection max lifetime
    ConnMaxIdleTime time.Duration     // Connection max idle time
    Timeout         time.Duration     // Operation timeout
    Options         map[string]string // Additional options
}
```

## Examples Directory

The [examples](../../examples/) directory contains:
- Complete working examples for each database type
- Connection pool usage
- Transaction patterns
- Error handling examples
- Configuration examples

## Testing

Run all tests:
```bash
go test ./pkg/database/...
```

Run with coverage:
```bash
go test ./pkg/database -cover
```

Run verbose:
```bash
go test ./pkg/database -v
```

## Support & Resources

### Documentation Order
1. **Quick Start** → [QUICK_START.md](QUICK_START.md)
2. **Full Documentation** → [README.md](README.md)
3. **Architecture** → [IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md)

### Example Code
- Simple examples → [QUICK_START.md](QUICK_START.md)
- Complete examples → [../../examples/database_examples.go](../../examples/database_examples.go)

### Troubleshooting
- Common issues → [QUICK_START.md - Troubleshooting](QUICK_START.md#troubleshooting)
- Error handling → [README.md - Error Handling](README.md#error-handling)

## Version Information

- **Go Version**: 1.24.0+
- **Package Path**: `base_project/pkg/database`
- **Status**: Production Ready ✅

## License

See the project's main LICENSE file.

---

**Need Help?**
- Start with [QUICK_START.md](QUICK_START.md) for immediate usage
- Read [README.md](README.md) for comprehensive documentation
- Check [IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md) for architecture details
- Review [examples](../../examples/) for working code

**Last Updated**: 2025