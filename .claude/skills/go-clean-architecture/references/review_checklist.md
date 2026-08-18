# Go Clean Architecture - Code Review Checklist

Use this comprehensive checklist when reviewing Go projects for clean architecture compliance.

---

## Phase 1: Structure & Organization

- [ ] **Directory Structure**: Project follows 6-layer clean architecture (domain, repository, usecase, delivery, infrastructure, adapter)
- [ ] **Package Names**: Layers clearly named, no generic `utils/`, `helpers/`, `common/`
- [ ] **File Organization**: Related functionality grouped logically within each layer
- [ ] **No Orphan Packages**: All packages clearly belong to one of the 6 layers
- [ ] **Main Entry Point**: Clear `main.go` or `cmd/` folder with project initialization

---

## Phase 2: Domain Layer Review

**File**: `domain/` packages

- [ ] **No External Dependencies**: Domain imports only standard library and external domain libraries
- [ ] **No Framework Imports**: No imports of `net/http`, `database/sql`, `gorm`, Redis, etc.
- [ ] **Entity Definitions**: Clear entity/model structs defined
- [ ] **Interface Contracts**: Interfaces defined for external dependencies (UserRepository, EmailService, etc.)
- [ ] **Business Logic**: Entity methods contain business rules (e.g., User.IsValid())
- [ ] **No Database Imports**: Domain doesn't import database drivers or ORM libraries
- [ ] **No HTTP/gRPC**: No HTTP or gRPC libraries imported
- [ ] **Immutability**: Entities designed for clean separation (getters/setters if needed)

---

## Phase 3: Repository Layer Review

**File**: `repository/` packages

- [ ] **Implements Domain Interfaces**: Repository structs implement interfaces from domain/
- [ ] **Only Imports Domain**: Repository imports domain package for interfaces and entities
- [ ] **Infrastructure Imports**: Can import infrastructure package for DB connections
- [ ] **Database Logic**: SQL queries, database operations are here
- [ ] **No Delivery Imports**: Repository doesn't know about HTTP/gRPC
- [ ] **No Usecase Imports**: Repository doesn't import usecase layer
- [ ] **Dependency Injection**: Database connection injected via constructor
- [ ] **Error Handling**: Proper error handling and wrapping

---

## Phase 4: Usecase Layer Review

**File**: `usecase/` packages

- [ ] **Orchestrates Domain & Repository**: Calls domain entities and repository interfaces
- [ ] **Interface Dependencies**: Depends on domain interfaces (UserRepository), not concrete types
- [ ] **No Framework Imports**: No HTTP, database, or infrastructure imports
- [ ] **No Delivery Imports**: Doesn't know about HTTP/gRPC handlers
- [ ] **No Infrastructure Imports**: Doesn't access database, cache, or external services directly
- [ ] **Business Logic**: Clear business logic flow (validation, creation, updates)
- [ ] **Constructor Injection**: All dependencies injected via constructor functions
- [ ] **Error Handling**: Business logic errors properly handled and propagated
- [ ] **Testability**: Easy to mock dependencies for unit testing

**Example**:
```go
type CreateUserUseCase struct {
  repo domain.UserRepository    // ✅ Interface
  emailSvc domain.EmailService  // ✅ Interface
}

func NewCreateUserUseCase(
  repo domain.UserRepository,
  emailSvc domain.EmailService,
) *CreateUserUseCase {
  return &CreateUserUseCase{repo: repo, emailSvc: emailSvc}
}

func (uc *CreateUserUseCase) Execute(ctx context.Context, email, name string) error {
  // Business logic
}
```

---

## Phase 5: Delivery Layer Review

**File**: `delivery/` packages (typically `delivery/http/` or `delivery/grpc/`)

### HTTP Handlers

- [ ] **Thin Handlers**: Business logic delegated to usecases, handlers are thin
- [ ] **Request Parsing**: Properly parses HTTP request body/query params
- [ ] **Error Handling**: Returns appropriate HTTP status codes
- [ ] **Response Formatting**: Returns JSON/protobuf responses
- [ ] **No Business Logic**: Complex logic not in handler functions
- [ ] **No Direct Database**: No SQL queries or direct database access
- [ ] **Usecase Injection**: Usecases injected via constructor
- [ ] **DTO Usage**: Adapter DTOs used for request/response mapping

**Example**:
```go
type UserHandler struct {
  createUserUC *usecase.CreateUserUseCase
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
  var req adapter.CreateUserRequest
  json.NewDecoder(r.Body).Decode(&req)
  
  err := h.createUserUC.Execute(r.Context(), req.Email, req.Name)
  if err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
  }
  
  w.Header().Set("Content-Type", "application/json")
  w.WriteHeader(http.StatusCreated)
  json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}
```

### gRPC Services

- [ ] **Service Implementation**: Implements generated gRPC service interface
- [ ] **Proto Mapping**: Properly maps protobuf messages to domain entities
- [ ] **Usecase Calls**: Delegates to usecases, not repositories
- [ ] **No Business Logic**: Complex logic in usecases, not service implementation
- [ ] **Error Handling**: Returns appropriate gRPC status codes

---

## Phase 6: Infrastructure Layer Review

**File**: `infrastructure/` packages

- [ ] **Dependency Wiring**: Sets up all external dependencies
- [ ] **Database Initialization**: Creates database connections
- [ ] **Framework Setup**: Initializes HTTP/gRPC servers
- [ ] **Configuration**: Loads config from env, files, etc.
- [ ] **Can Import Everything**: Infrastructure can import all other layers
- [ ] **Factory Functions**: Provides factory functions for creating repositories/services
- [ ] **No Business Logic**: No business logic here, only setup

**Example**:
```go
func NewPostgresUserRepository(db *sql.DB) domain.UserRepository {
  return &repository.PostgresUserRepository{db: db}
}

func NewHTTPServer(handler *delivery.UserHandler) *http.Server {
  mux := http.NewServeMux()
  mux.HandleFunc("/users", handler.CreateUser)
  return &http.Server{Addr: ":8080", Handler: mux}
}
```

---

## Phase 7: Adapter Layer Review

**File**: `adapter/` packages

- [ ] **DTOs Defined**: Request/response DTOs match HTTP/gRPC contracts
- [ ] **Mappers Exist**: DTO-to-domain and domain-to-DTO mappers
- [ ] **No Business Logic**: Adapters only map, no business logic
- [ ] **Domain Imports Only**: Only imports domain layer
- [ ] **Marshaling**: Proper JSON tags for marshaling/unmarshaling

**Example**:
```go
type CreateUserRequest struct {
  Email string `json:"email"`
  Name  string `json:"name"`
}

type UserResponse struct {
  ID    string `json:"id"`
  Email string `json:"email"`
  Name  string `json:"name"`
}

func MapCreateUserRequestToDomain(req CreateUserRequest) *domain.User {
  return &domain.User{Email: req.Email, Name: req.Name}
}

func MapUserToResponse(user *domain.User) UserResponse {
  return UserResponse{ID: user.ID, Email: user.Email, Name: user.Name}
}
```

---

## Phase 8: Dependency Analysis

### Import Direction

- [ ] **No Reverse Dependencies**: Outer layers don't import inner layers
- [ ] **Correct Flow**: delivery → usecase → repository → domain
- [ ] **Interface-Based**: Dependencies use domain interfaces, not concrete implementations
- [ ] **No Circular Imports**: No package A imports B and B imports A

### Dependency Injection

- [ ] **Constructor Injection**: Dependencies passed via constructor functions
- [ ] **No Global Singletons**: No global `var DB`, `var Config`, etc.
- [ ] **No Service Locator**: No global registry/container in layers
- [ ] **Explicit Dependencies**: All dependencies visible in type signature

**Check with**:
```bash
go mod graph | grep cycles  # Check for circular dependencies
grep -r "import \"yourapp/" <layer>  # Check what each layer imports
```

---

## Phase 9: Testing & Code Quality

- [ ] **Mockable Interfaces**: External dependencies are interfaces (easy to mock)
- [ ] **Test Files**: Test files exist alongside implementation (`*_test.go`)
- [ ] **Unit Tests**: Usecase layer has good unit test coverage
- [ ] **Integration Tests**: Integration tests for repository/database layer
- [ ] **Error Cases**: Tests cover error scenarios and edge cases
- [ ] **No Test Pyramid Inversion**: Not all integration tests, proper unit tests

**Check**:
```bash
go test ./... -v  # Run all tests
go test -cover ./...  # Check coverage
```

---

## Phase 10: Middleware & Cross-Cutting Concerns

- [ ] **HTTP Middleware**: Logging, CORS, auth middleware in delivery layer
- [ ] **Error Handling Middleware**: Global error handler for HTTP/gRPC
- [ ] **Request Logging**: Structured logging for requests/responses
- [ ] **Authentication**: Auth/authorization middleware not mixed with business logic
- [ ] **Validation**: Input validation in adapter/delivery, business validation in usecase

---

## Phase 11: Configuration & Environment

- [ ] **Environment Variables**: Config loaded from env vars, not hardcoded
- [ ] **Config Structure**: Clear config types (AppConfig, DatabaseConfig, etc.)
- [ ] **Secrets Management**: Sensitive data not logged or exposed
- [ ] **Multiple Environments**: Config supports dev/staging/production

---

## Phase 12: Error Handling

- [ ] **Consistent Error Types**: Domain defines error types
- [ ] **Error Wrapping**: Errors wrapped with context (using `fmt.Errorf`)
- [ ] **HTTP Error Codes**: Usecases return errors, delivery translates to HTTP codes
- [ ] **Logging**: Errors logged appropriately (not duplicated)
- [ ] **User Messages**: User-facing errors vs internal debug errors separated

---

## Phase 13: Documentation

- [ ] **README**: Project structure documented
- [ ] **Layer Explanation**: Each layer has clear responsibility
- [ ] **API Documentation**: Endpoints/services documented
- [ ] **Setup Instructions**: How to run, test, deploy documented
- [ ] **Example Usage**: Examples of how to use the project

---

## Phase 14: Performance & Optimization

- [ ] **N+1 Queries**: Repository uses batch operations where appropriate
- [ ] **Database Connections**: Connection pooling configured
- [ ] **Caching Strategy**: Cache layer considered (though can be in infrastructure)
- [ ] **Goroutines**: Concurrent operations properly managed
- [ ] **Context Usage**: Context timeouts respected throughout

---

## Summary Checklist

- [ ] All 14 phases reviewed
- [ ] No high-severity violations found
- [ ] Medium violations documented with remediation plan
- [ ] Code follows Go conventions (naming, formatting)
- [ ] Project is testable and maintainable
- [ ] Ready for production (security, error handling, logging)

---

## Output Format

When reviewing, document findings as:

```
Layer: [domain/repository/usecase/delivery/infrastructure/adapter]
Severity: [CRITICAL/HIGH/MEDIUM/LOW]
Finding: [What was found]
Location: [File path and line number]
Rule Violated: [From dependency_rules.md or violation_patterns.md]
Recommendation: [How to fix]
```

