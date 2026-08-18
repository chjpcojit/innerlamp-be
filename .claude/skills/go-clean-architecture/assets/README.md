# Go Clean Architecture - Template Guide

This directory contains production-ready templates for implementing clean architecture in Go projects.

## Templates Available

### 1. template_rest_api/
Complete REST API implementation using standard Go `net/http` package.

**Use this when:**
- Building REST APIs with HTTP handlers
- Starting a new web service
- Want comprehensive example with middleware, testing, Docker

**Includes:**
- Complete layer implementation (domain, repository, usecase, delivery, infrastructure, adapter)
- PostgreSQL database integration
- HTTP handlers with proper request/response handling
- Docker and Makefile setup
- Tests structure

**Start here:** Copy this template and customize for your project.

### 2. template_grpc_service/
gRPC microservice implementation with protocol buffers.

**Use this when:**
- Building gRPC microservices
- Need high-performance RPC communication
- Integrating with other gRPC services

**Includes:**
- gRPC service definitions (proto files)
- Clean architecture layer structure
- Proto message to domain entity mapping
- Docker and deployment setup

**Note:** This is a reference structure. You'll need to define your own .proto files.

### 3. template_microservice/
Hybrid template combining REST API and gRPC patterns.

**Use this when:**
- Building services with both REST and gRPC endpoints
- Want flexibility to support multiple protocols
- Need single codebase with multiple interfaces

**Includes:**
- Both HTTP and gRPC delivery layers
- Shared domain, usecase, and repository layers
- Combined infrastructure setup

## How to Use a Template

### Step 1: Copy the Template

```bash
cp -r template_rest_api my-new-project
cd my-new-project
```

### Step 2: Update Module Name

```bash
# Update go.mod
vim go.mod

# Update all import paths
find . -name "*.go" -exec sed -i 's/yourapp/my-new-project/g' {} \;
```

### Step 3: Customize Layers

- **domain/**: Define your business entities and interfaces
- **repository/**: Implement data access for your database
- **usecase/**: Add business logic use cases
- **delivery/**: Add HTTP/gRPC handlers
- **adapter/**: Add DTOs for your endpoints
- **infrastructure/**: Configure your external dependencies

### Step 4: Run Tests & Build

```bash
make test
make build
make run
```

## Clean Architecture Principles

All templates follow these core principles:

```
Layer Dependencies (arrows point inward):

  delivery → usecase → repository → infrastructure
         ↓       ↓            ↓
       adapter ← domain ←(interfaces)
```

**Key Rules:**
- ✅ Domain layer has NO external dependencies
- ✅ Each layer depends on interfaces, not concrete implementations
- ✅ Dependencies injected via constructor functions
- ✅ Business logic in usecase, not delivery
- ✅ Database access only in repository

## Common Customizations

### Add a New Entity/Domain

1. Create entity struct in `domain/`
2. Add repository interface to domain
3. Implement repository in `repository/`
4. Create usecase in `usecase/`
5. Add delivery handler in `delivery/`
6. Add DTOs in `adapter/`

### Switch Database (PostgreSQL → MongoDB)

1. Update `infrastructure/database.go`
2. Reimplement `repository/` for MongoDB
3. Domain interfaces stay the same

### Add Authentication/Authorization

1. Add middleware in `infrastructure/`
2. Add auth interfaces to domain
3. Inject into handlers that need it

### Add Logging/Monitoring

1. Create logger interface in domain (or infrastructure)
2. Inject into usecases and handlers
3. Structured logging in usecase/repository

## Testing Strategy

All templates support this testing pyramid:

```
                    /\
                   /  \
                  / E2E \     (REST/gRPC clients)
                 /________\
                /          \
               / Integration \  (Repository + Database)
              /________________\
            /                    \
           /      Unit Tests       \  (Usecase + Mocked Repos)
          /____________________________\
```

- **Unit Tests**: Mock domain interfaces, test business logic
- **Integration Tests**: Test with real database
- **E2E Tests**: Test full HTTP/gRPC flows

## Deployment

All templates include:

- **Dockerfile**: Multi-stage build for minimal image size
- **Makefile**: Build automation
- **.env support**: Configuration via environment variables
- **Health checks**: Ready for Kubernetes/orchestration

## Questions?

Refer to the `references/` directory in the skill root for:
- `clean_architecture_rules.md` - Detailed layer descriptions
- `dependency_rules.md` - Import rules and constraints
- `violation_patterns.md` - Common mistakes and fixes
- `review_checklist.md` - Code review guidelines

