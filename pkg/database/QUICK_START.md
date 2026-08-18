# Database Package - Quick Start Guide

## Installation

```go
import (
    "base_project/pkg/database"
    _ "base_project/pkg/database/mysql"
    _ "base_project/pkg/database/postgresql"
    _ "base_project/pkg/database/mongodb"
)
```

## 5-Minute Quick Start

### MySQL

```go
package main

import (
    "context"
    "log"
    
    "base_project/pkg/database"
    _ "base_project/pkg/database/mysql"
)

func main() {
    // Create config
    config := database.DefaultConfig(database.MySQL)
    config.Username = "root"
    config.Password = "password"
    config.Database = "mydb"
    
    // Connect
    db, _ := database.New(config)
    ctx := context.Background()
    db.Connect(ctx)
    defer db.Close()
    
    // Query
    rows, _ := db.Query(ctx, "SELECT id, name FROM users")
    defer rows.Close()
    
    for rows.Next() {
        var id int
        var name string
        rows.Scan(&id, &name)
        log.Printf("%d: %s", id, name)
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
    config := database.DefaultConfig(database.PostgreSQL)
    config.Username = "postgres"
    config.Password = "password"
    config.Database = "mydb"
    config.SSLMode = "disable"
    
    db, _ := database.New(config)
    ctx := context.Background()
    db.Connect(ctx)
    defer db.Close()
    
    // PostgreSQL uses $1, $2 placeholders
    var count int
    db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE active = $1", true).Scan(&count)
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
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
)

func main() {
    config := database.DefaultConfig(database.MongoDB)
    config.Username = "admin"
    config.Password = "password"
    config.Database = "mydb"
    
    db, _ := database.New(config)
    ctx := context.Background()
    db.Connect(ctx)
    defer db.Close()
    
    // Get MongoDB client
    clientInterface, _ := db.GetMongoClient()
    client := clientInterface.(*mongo.Client)
    
    // Use collection
    collection := client.Database("mydb").Collection("users")
    collection.InsertOne(ctx, bson.M{"name": "John", "age": 30})
}
```

## Common Operations

### Insert

```go
// MySQL/PostgreSQL
result, err := db.Exec(ctx, "INSERT INTO users (name, email) VALUES (?, ?)", "John", "john@example.com")
if err != nil {
    log.Fatal(err)
}
id, _ := result.LastInsertId()
```

### Update

```go
result, err := db.Exec(ctx, "UPDATE users SET active = ? WHERE id = ?", false, 123)
rowsAffected, _ := result.RowsAffected()
```

### Delete

```go
db.Exec(ctx, "DELETE FROM users WHERE id = ?", 123)
```

### Transactions

```go
tx, err := db.Begin(ctx)
if err != nil {
    log.Fatal(err)
}

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

tx.Commit()
```

## Connection Pool

```go
pool := database.NewConnectionPool()

// Add databases
mysqlDB, _ := database.New(mysqlConfig)
mysqlDB.Connect(ctx)
pool.Add("mysql", mysqlDB)

pgDB, _ := database.New(pgConfig)
pgDB.Connect(ctx)
pool.Add("postgres", pgDB)

// Use
db, _ := pool.Get("mysql")
db.Query(ctx, "SELECT * FROM users")

// Cleanup
defer pool.CloseAll()
```

## Configuration Options

### Full Configuration Example

```go
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
    ConnMaxIdleTime: 10 * time.Minute,
    Timeout:         10 * time.Second,
    Options: map[string]string{
        "charset":   "utf8mb4",
        "parseTime": "true",
    },
}
```

### Using Direct DSN

```go
config := database.DefaultConfig(database.MySQL)
config.Options["dsn"] = "user:password@tcp(localhost:3306)/dbname?charset=utf8mb4"
```

## Error Handling

```go
db, err := database.New(config)
if err != nil {
    switch {
    case errors.Is(err, database.ErrInvalidConfig):
        log.Println("Invalid configuration")
    case errors.Is(err, database.ErrUnsupportedDriver):
        log.Println("Database type not supported")
    default:
        log.Printf("Error: %v", err)
    }
    return
}

if err := db.Connect(ctx); err != nil {
    if errors.Is(err, database.ErrConnectionFailed) {
        log.Println("Failed to connect - check credentials and host")
    }
    return
}

if !db.IsConnected() {
    log.Println("Database is not connected")
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
    log.Println("Database is healthy")
}

// Get database type
log.Printf("Connected to %s", db.GetType())
```

## Advanced: Direct Database Access

### SQL Databases

```go
sqlDB, err := db.GetSQLDB()
if err != nil {
    log.Fatal(err)
}

// Use standard database/sql features
stats := sqlDB.Stats()
log.Printf("Open connections: %d", stats.OpenConnections)

// Prepared statements
stmt, _ := sqlDB.Prepare("SELECT * FROM users WHERE id = ?")
defer stmt.Close()
```

### MongoDB

```go
clientInterface, err := db.GetMongoClient()
if err != nil {
    log.Fatal(err)
}

client := clientInterface.(*mongo.Client)

// Use MongoDB features
database := client.Database("mydb")
collections, _ := database.ListCollectionNames(ctx, bson.M{})
log.Printf("Collections: %v", collections)
```

## Testing Example

```go
func TestDatabase(t *testing.T) {
    config := database.DefaultConfig(database.MySQL)
    config.Database = "test_db"
    
    db, err := database.New(config)
    if err != nil {
        t.Fatalf("Failed to create database: %v", err)
    }
    
    ctx := context.Background()
    if err := db.Connect(ctx); err != nil {
        t.Skipf("Database not available: %v", err)
    }
    defer db.Close()
    
    // Run tests
    if err := db.Ping(ctx); err != nil {
        t.Errorf("Ping failed: %v", err)
    }
}
```

## Best Practices

1. **Always use context**: Pass context for timeouts and cancellation
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
   defer cancel()
   db.Query(ctx, "SELECT ...")
   ```

2. **Close resources**: Use defer to ensure cleanup
   ```go
   defer db.Close()
   defer rows.Close()
   ```

3. **Handle errors**: Check all errors
   ```go
   if err := db.Connect(ctx); err != nil {
       log.Fatal(err)
   }
   ```

4. **Use connection pooling**: Configure based on load
   ```go
   config.MaxOpenConns = 50
   config.MaxIdleConns = 10
   ```

5. **Set timeouts**: Always set appropriate timeouts
   ```go
   config.Timeout = 30 * time.Second
   ```

## Common Pitfalls

❌ **Don't**: Forget to close connections
```go
db, _ := database.New(config)
db.Connect(ctx)
// Missing: defer db.Close()
```

✅ **Do**: Always close connections
```go
db, _ := database.New(config)
db.Connect(ctx)
defer db.Close()
```

❌ **Don't**: Ignore errors
```go
db.Query(ctx, "SELECT ...")
```

✅ **Do**: Check errors
```go
rows, err := db.Query(ctx, "SELECT ...")
if err != nil {
    log.Fatal(err)
}
```

❌ **Don't**: Use operations not supported by database type
```go
// MongoDB doesn't support SQL queries
mongoDb.Query(ctx, "SELECT * FROM users") // Error!
```

✅ **Do**: Use appropriate methods for database type
```go
clientInterface, _ := mongoDb.GetMongoClient()
client := clientInterface.(*mongo.Client)
// Use MongoDB operations
```

## Troubleshooting

### Connection Failed
- Check host and port
- Verify username and password
- Ensure database exists
- Check firewall rules
- Verify SSL/TLS settings

### Timeout Errors
- Increase timeout value
- Check network connectivity
- Verify database server is responsive

### Pool Exhaustion
- Increase MaxOpenConns
- Check for connection leaks
- Ensure connections are properly closed

## Next Steps

- Read the full [README.md](README.md) for detailed documentation
- Check [examples/database_examples.go](../../examples/database_examples.go) for complete examples
- Review [IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md) for architecture details

## Support

For issues or questions:
1. Check the README.md documentation
2. Review example code
3. Check error messages and logs
4. Verify database server is running and accessible