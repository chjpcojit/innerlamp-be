# Files

- [Configuration & Secrets Management](config-secrets.md) - Detailed usage, symbols, and architectural integration for loading structured configuration and managing secrets via GCP/AWS.
- [Data Persistence & Blob Storage](data-storage.md) - Unified abstractions for relational and NoSQL databases alongside AWS S3 and GCP Cloud Storage engines.
- [Resilient Networking & Middlewares](networking.md) - Design and implementation details for the robust HTTP client (circuit breaker, retries) and standard gRPC/HTTP request interceptors.
- [Observability: Logging, Metrics, & Tracing](observability.md) - Comprehensive details on structured log formatting, GCP Cloud Monitoring, Prometheus, and OpenTelemetry tracing integrations.
- [Rate Limiting Stores & Policies](ratelimit.md) - Dynamic request throttling via Token Bucket, Sliding Window, Fixed Window, and Leaky Bucket using Redis or In-Memory backends.
- [Authentication & Session Lifecycle](security.md) - Production-grade JWT generation/parsing, symmetric and asymmetric signature validations, and MongoDB-backed session token rotations.
- [Validation, Sanity, & GCP Helpers](validation-utils.md) - Struct tag validations, strict XSS filters, Mexican compliance formatting (CURP/Phones), and serverless runtime detectors.
