---
type: Reference
title: OpenWiki Quickstart
description: The master entrypoint and task-routing map for the innerlamp Go shared backend framework.
tags: [onboarding, guide, fast-path]
---

# innerlamp Code Wiki Quickstart

Welcome to the **innerlamp** (module: `chjpcoj.com.innerlamp`) code wiki. This repository is a highly optimized, enterprise-grade Go shared backend package framework (`pkg/`). It provides standard abstractions and reusable implementations for configuration management, database connections, secure secrets handling, authentication, tracing, metrics, logging, structured storage, rate limiting, and input validation.

This framework is specifically engineered with **serverless and containerized environments** in mind (such as Google Cloud Run, GKE, AWS Fargate). It emphasizes internal connection management, single-instance lifecycle patterns, and dynamic, environment-aware configuration.

---

## 🗺️ Wiki Navigation Directory

- **[System Architecture & Design Principles](architecture/overview.md)**: High-level overview of the library's design principles, package design patterns, integration boundaries, and the `base_project` import structure.
- **[Configuration & Secrets Management](packages/config-secrets.md)**: Standard mechanisms to load configs from files, S3, or GCS, and safely resolve environment/cloud secrets via GCP and AWS Secret Managers.
- **[Data Persistence & Blob Storage](packages/data-storage.md)**: SQL database pools (MySQL, PostgreSQL), MongoDB client engines, and GCS/S3 object storage client APIs.
- **[Resilient Networking & Middlewares](packages/networking.md)**: Custom HTTP Client with builtin exponential backoffs and circuit breakers, alongside standard HTTP/gRPC routing middlewares.
- **[Authentication & Session Lifecycle](packages/security.md)**: Custom Claims JWT signing services and Mongo-backed Session tracking with automated rotation and revocation.
- **[Observability: Logging, Metrics, & Tracing](packages/observability.md)**: Context-aware JSON loggers, Prometheus/GCP metrics collection registries, and OpenTelemetry-based tracing wrappers.
- **[Rate Limiting Stores & Policies](packages/ratelimit.md)**: High-performance limiters supporting Token Bucket, Sliding/Fixed Window, and Leaky Bucket with local or Redis storage backends.
- **[Validation, Sanity, & GCP Helpers](packages/validation-utils.md)**: Specialized validation routines (such as Mexican phone formats and CURP codes), XSS sanitization, and cloud environment discovery.

---

## ⚡ Task-Routing Table

If you need to make changes to a component or extend the framework, use this quick task-routing map to locate the implementation, tests, and minimal non-destructive validation checks.

| Change Area / Intent | Relevant Wiki Page | Exact Source Entrypoints | Important Symbols / Types | Focused Tests | Minimal Validation Command |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Extend Config / Secret Provider** | **[Config & Secrets](packages/config-secrets.md)** | `/pkg/config/loader.go`<br>`/pkg/secrets/secrets.go` | `LoadSource`, `secrets.Client`, `Provider` | `/pkg/secrets/secrets_test.go` | `go test -v ./pkg/secrets/...` |
| **Add DB Driver / DB Config** | **[Data & Storage](packages/data-storage.md)** | `/pkg/database/database.go` | `Database`, `Config`, `DBType` | `/pkg/database/database_test.go` | `go test -v ./pkg/database/...` |
| **Add Object Storage Class** | **[Data & Storage](packages/data-storage.md)** | `/pkg/storage/storage.go` | `Client`, `StorageClass`, `Config` | `/pkg/storage/storage_test.go` | `go test -v ./pkg/storage/...` |
| **Change HTTP Client / Retry** | **[Networking](packages/networking.md)** | `/pkg/httpclient/httpclient.go` | `Client`, `Config`, `circuitBreaker` | `/pkg/httpclient/httpclient_test.go` | `go test -v ./pkg/httpclient/...` |
| **Modify JWT Generation / Claims** | **[Security](packages/security.md)** | `/pkg/jwt/jwt.go`<br>`/pkg/jwt/claims/claims.go` | `Service`, `CustomClaims` | `/pkg/jwt/jwt_test.go`<br>`/pkg/jwt/middleware/http_test.go` | `go test -v ./pkg/jwt/...` |
| **Change Session Rotation Lifecycle** | **[Security](packages/security.md)** | `/pkg/session/session.go` | `Manager`, `Session`, `DeviceInfo` | `/pkg/session/session_test.go` | `go test -v ./pkg/session/...` |
| **Format Structured Logs / Fields** | **[Observability](packages/observability.md)** | `/pkg/logging/logger.go` | `Logger`, `Entry`, `Formatter`, `Fields` | *(Integrated across packages)* | `go test -v ./pkg/logging/...` |
| **Register Custom Metric Type** | **[Observability](packages/observability.md)** | `/pkg/metrics/metrics.go` | `Collector`, `MetricDefinition` | `/pkg/metrics/factory.go` | `go test -v ./pkg/metrics/...` |
| **Add Rate Limiter Algorithm / Store**| **[Rate Limiting](packages/ratelimit.md)** | `/pkg/ratelimit/ratelimit.go` | `Store`, `Config`, `Algorithm` | `/pkg/ratelimit/store/memory_test.go` | `go test -v ./pkg/ratelimit/...` |
| **Add Regulatory Formats / Regex** | **[Validation & Utils](packages/validation-utils.md)** | `/pkg/validation/validation.go` | `ValidationError`, `EmailRegex` | `/pkg/validation/validation_test.go` | `go test -v ./pkg/validation/...` |

---

## 📋 Backlog

Here is a list of known documentation or integration tasks deferred for future development cycles:

| Target Backlog Area | Primary Source Anchor | Expected Disposition / Context |
| :--- | :--- | :--- |
| **Refactor import path names** | `/go.mod` | The module is defined as `chjpcoj.com.innerlamp` but internal package imports currently reference `base_project/pkg/...`. This discrepancy should be aligned in future cleanup. |
| **Integrate appService implementation** | `/services/appService/cmd/main.go` | Implement the main entrypoint by tying together the config loader, database connections, logging, and tracing services. |
| **Add Redis Session Store** | `/pkg/session/session.go` | Introduce a Redis-backed session manager as an alternative to MongoDB for faster cache lookups. |
