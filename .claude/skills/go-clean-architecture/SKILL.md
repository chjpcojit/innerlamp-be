---
name: go-clean-architecture
description: This skill should be used when reviewing Go projects for clean architecture compliance or scaffolding new Go microservices, REST APIs, and gRPC services following clean architecture patterns with comprehensive setup including middleware, testing, and Docker configuration.
---
# Go Clean Architecture Skill

## Purpose

Review Go codebases vs clean architecture principles. Scaffold production Go projects w/ proper layer separation, dependency mgmt, comprehensive infrastructure setup.

## When to Use

- **Code Review**: Detect clean architecture violations (layer separation, dependency direction, naming)
- **Project Scaffolding**: Generate REST API, gRPC, or microservice projects w/ correct architecture
- **Refactoring**: Restructure Go projects to follow clean architecture
- **Architecture Validation**: Verify Go codebase adheres to clean architecture rules

## How to Use This Skill

### 1. Reviewing Existing Code

When reviewing Go project for clean architecture:

1. Reference `references/clean_architecture_rules.md` for Go-specific layer contracts & dependencies
2. Reference `references/dependency_rules.md` — understand what each layer can/cannot import
3. Reference `references/violation_patterns.md` — identify common violations
4. Use `scripts/analyze_architecture.go` to scan project (file or directory) & generate violation report
5. Use `references/review_checklist.md` for systematic review

**Example Review Workflow:**
```
Input: "Review this Go REST API project for clean architecture issues"
→ Use analyzer script on the project directory
→ Reference violation patterns to understand issues
→ Use checklist to ensure comprehensive review
→ Output: Detailed findings with layer violations, fixes, and recommendations
```

### 2. Scaffolding New Projects

When creating new Go projects:

1. Reference appropriate template in `assets/`:
   - `template_rest_api/` for REST APIs using standard `net/http`
   - `template_grpc_service/` for gRPC microservices
   - `template_microservice/` for services combining patterns
2. Use templates as reference for correct layer structure, middleware setup, testing patterns, Docker config
3. Reference `references/clean_architecture_rules.md` for implementation guidance

**Example Scaffolding Workflow:**
```
Input: "Create a new Go REST API project with clean architecture"
→ Review template_rest_api/ structure and patterns
→ Generate project with all 6 layers properly organized
→ Include middleware, testing, Docker setup
→ Output: Production-ready project following clean architecture
```

## Architecture Layers

Go clean architecture: 6 layers (inside to outside):

1. **domain/** — Core business logic, entities, interfaces (no framework deps)
2. **repository/** — Data access (SQL, NoSQL, cache)
3. **usecase/** — Business logic orchestrating domain & repository
4. **delivery/** — HTTP/gRPC handlers (external → usecases)
5. **infrastructure/** — External deps (database, cache, logging, config)
6. **adapter/** — DTOs, mappers, port adapters between layers

**Critical Rule**: Dependencies flow inward. Outer layers depend on inner interfaces, never reverse.

## Bundled Resources

- **Scripts**: `analyze_architecture.go` — Detect clean architecture violations
- **References**: Rules, patterns, violations, review criteria
- **Assets**: Production-ready templates (REST API, gRPC, microservices)