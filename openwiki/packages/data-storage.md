---
type: Reference
title: Data Persistence & Blob Storage
description: Unified abstractions for relational and NoSQL databases alongside AWS S3 and GCP Cloud Storage engines.
tags: [database, postgresql, mysql, mongodb, s3, gcs, blob-storage]
openwiki:
  roles: [domain, integration]
  change_kinds: [lifecycle, public-api]
  source_paths: [pkg/database/database.go, pkg/storage/storage.go]
  symbols: [Database, Config, DBType, Client, Backend, StorageClass, ACL]
  test_paths: [pkg/database/database_test.go, pkg/storage/storage_test.go]
  invariants: [Standard SQL operations throw errors on NoSQL, GCS/S3 object paths must not be empty.]
  validation_commands: [go test -v ./pkg/database/... ./pkg/storage/...]
---

# Data Persistence & Blob Storage

The `innerlamp` framework provides enterprise-ready storage layers dividing relational or document persistence (`pkg/database`) from standard file-based cloud object storage (`pkg/storage`).

---

## 🗄️ Relational & NoSQL Databases (`pkg/database`)

The database module provides a unified interface `Database` capable of representing traditional SQL engines (**MySQL**, **PostgreSQL**) and NoSQL document databases (**MongoDB**).

### The `Database` Interface

Consuming applications query, mutate, and manage transactions through this general-purpose contract:

```go
type Database interface {
    // Connection Management
    Connect(ctx context.Context) error
    Close() error
    Ping(ctx context.Context) error
    IsConnected() bool

    // SQL operations (Throws ErrOperationNotSupported on MongoDB)
    Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)
    QueryRow(ctx context.Context, query string, args ...any) *sql.Row
    Exec(ctx context.Context, query string, args ...any) (sql.Result, error)

    // Transaction support (SQL only)
    Begin(ctx context.Context) (*sql.Tx, error)

    // Escape Hatches for native drivers
    GetSQLDB() (*sql.DB, error)
    GetMongoClient() (any, error)

    // Metadata
    GetType() DBType
    GetConfig() *Config
}
```

### Registered DB Drivers

Specific database engines reside in dedicated subpackages and register themselves via package initializers (`init()`):
* **MySQL**: Implemented in `/pkg/database/mysql/mysql.go` (backed by `github.com/go-sql-driver/mysql`).
* **PostgreSQL**: Implemented in `/pkg/database/postgresql/postgresql.go` (backed by `github.com/lib/pq`).
* **MongoDB**: Implemented in `/pkg/database/mongodb/mongodb.go` (backed by `go.mongodb.org/mongo-driver/mongo`).

Applications must activate a driver using a blank import:
```go
import _ "base_project/pkg/database/postgresql"
```

---

## ☁️ Object Storage (`pkg/storage`)

The `pkg/storage` package wraps cloud-native blob storage providers (**GCS** and **S3**) behind a single, consistent client interface. It is specifically built for highly concurrent, serverless architectures, managing internal connection pooling automatically.

### The `Client` Interface

The storage engine exposes rich file-manipulation primitives, including Signed URL generation and Access Control List (ACL) configurations.

```go
type Client interface {
    // Lifecycle
    Connect(ctx context.Context) error
    Close() error
    IsConnected() bool

    // Bucket Operations
    CreateBucket(ctx context.Context, name string, opts *BucketOptions) error
    DeleteBucket(ctx context.Context, name string) error
    BucketExists(ctx context.Context, name string) (bool, error)

    // Object Operations
    UploadObject(ctx context.Context, bucket, object string, content io.Reader, opts *UploadOptions) (*ObjectMetadata, error)
    DownloadObject(ctx context.Context, bucket, object string) (io.ReadCloser, error)
    DeleteObject(ctx context.Context, bucket, object string) error
    ObjectExists(ctx context.Context, bucket, object string) (bool, error)
    GetObjectMetadata(ctx context.Context, bucket, object string) (*ObjectMetadata, error)
    
    // Copy, Move, Signed URLs
    CopyObject(ctx context.Context, srcBucket, srcObject, dstBucket, dstObject string, opts *CopyOptions) (*ObjectMetadata, error)
    GenerateSignedURL(ctx context.Context, bucket, object string, opts *SignedURLOptions) (string, error)
    ListObjects(ctx context.Context, bucket string, opts *ListOptions) (*ListResult, error)
}
```

### Config & Standard Storage Classes

Both backends utilize standard `StorageClass` values mapped to their respective cloud equivalents:
* **GCS Storage Classes**: `STANDARD`, `NEARLINE`, `COLDLINE`, `ARCHIVE`.
* **S3 Storage Classes**: `STANDARD`, `INTELLIGENT_TIERING`, `STANDARD_IA`, `ONEZONE_IA`, `GLACIER`, `DEEP_ARCHIVE`.

---

## 🔗 Semantic Relationships (OKF Mapped)

* **Session Storage Persistence**: The MongoDB client instance created via `pkg/database` [depends on](security.md) `pkg/session` (which uses it to perform session CRUD, JWT rotation histories, and device fingerprint tracking).
* **Configuration Storage Location**: Storage configurations can resolve configuration files loaded via `pkg/config` (by downloading from `s3://` or `gs://` targets).

---

## 🧪 Focused Validation

The database and storage validation engines reside under `/pkg/database/database_test.go` and `/pkg/storage/storage_test.go`.

To validate configurations and connection pools without executing heavy cloud-hosted calls:
```bash
# Quietly validate database & storage configurations
go test -v ./pkg/database/... ./pkg/storage/...
```
*(To run full integration scenarios, ensure your environment has active local MySQL/PostgreSQL/MongoDB containers and active GCS/AWS sandbox credentials).*
