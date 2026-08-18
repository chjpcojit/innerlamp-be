# Go Clean Architecture - REST API Template

This is a production-ready template for building REST APIs in Go following clean architecture principles.

## Project Structure

```
.
├── domain/              # Domain entities and interfaces
├── repository/          # Data access implementations
├── usecase/             # Business logic/application services
├── delivery/http/       # HTTP handlers and routers
├── infrastructure/      # External dependencies setup
├── adapter/             # DTOs and mappers
├── cmd/                 # Application entry point
├── tests/               # Test files
├── go.mod              # Go module definition
├── Dockerfile          # Docker configuration
└── Makefile            # Build automation
```

## Layer Descriptions

### domain/
Contains pure business logic and entity definitions. No framework dependencies.

Example: `User` entity, `UserRepository` interface

### repository/
Implements domain interfaces for data access.

Example: `PostgresUserRepository` implementing `domain.UserRepository`

### usecase/
Application business logic orchestrating domain and repository.

Example: `CreateUserUseCase` orchestrating user creation

### delivery/http/
HTTP handlers translating requests to usecases.

Example: `UserHandler` handling HTTP requests

### infrastructure/
External dependencies setup (database, server, config).

Example: Database connection, HTTP server configuration

### adapter/
DTOs and mappers between layers.

Example: `CreateUserRequest`, `UserResponse`, mapping functions

## Getting Started

### 1. Setup

```bash
# Clone/copy this template
cp -r template_rest_api myapp
cd myapp

# Update module name in go.mod
sed -i 's/yourapp/myapp/g' go.mod

# Update import paths in all files
find . -name "*.go" -exec sed -i 's|yourapp|myapp|g' {} \;

# Download dependencies
go mod download
```

### 2. Configure Database

```bash
# Create PostgreSQL database
createdb myapp

# Set environment variables
export DB_HOST=localhost
export DB_USER=postgres
export DB_PASSWORD=password
export DB_NAME=myapp
export DB_SSLMODE=disable
```

### 3. Run

```bash
# Build and run
make run

# Or run directly
go run ./cmd/main.go
```

### 4. Test

```bash
# Run all tests
make test

# Run with coverage
go test -cover ./...
```

## API Endpoints

### Create User
```bash
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "name": "John Doe"}'
```

## Clean Architecture Rules

**Dependencies flow inward only:**

```
delivery → usecase → domain ← repository ← infrastructure
                ↓
             adapter
```

**Key Rules:**
- ✅ Domain has no dependencies on other layers
- ✅ Repository implements domain interfaces
- ✅ Usecase depends on domain interfaces, not concrete types
- ✅ Delivery handlers delegate to usecases
- ✅ Infrastructure wires up all dependencies

## Extending the Template

### Adding a New Entity

1. Define in `domain/entity.go`
2. Add repository interface to domain
3. Implement in `repository/entity_repository.go`
4. Create usecase in `usecase/create_entity.go`
5. Add HTTP handler in `delivery/http/entity_handler.go`
6. Add DTOs in `adapter/entity_dto.go`

### Adding Middleware

1. Create middleware function
2. Add to `infrastructure/server.go` HTTP route setup

### Adding Tests

1. Create `*_test.go` files alongside implementation
2. Mock domain interfaces for unit tests
3. Use integration tests for repository layer

## Best Practices

- Always use dependency injection (constructor functions)
- Implement domain interfaces, don't import concrete types
- Keep handlers thin (delegate to usecases)
- Write tests for business logic in usecase layer
- Use context for cancellation and timeouts

## Troubleshooting

### Circular Import Errors

Check your import statements. Verify no inner layers import outer layers.

### "user not found" errors

Ensure database is initialized and running.

### Handler not working

Verify routes are registered in `infrastructure/server.go`

## References

- [Clean Architecture by Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Go Project Layout](https://github.com/golang-standards/project-layout)
