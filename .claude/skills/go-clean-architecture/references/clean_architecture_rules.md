# Go Clean Architecture Rules

## Layer Structure & Contracts

### 1. Domain Layer (`domain/`)

**Responsibility**: Contains pure business logic and entity definitions. No framework dependencies, no knowledge of external systems.

**Key Rules**:
- Define core entities (e.g., User, Order, Product)
- Define interfaces that outer layers must implement (e.g., UserRepository, EmailService)
- Contains business logic that's framework-agnostic
- No imports from other internal layers
- Only imports: `context`, standard library, external domain libraries

**Example**:
```go
package domain

type User struct {
  ID    string
  Email string
  Name  string
}

type UserRepository interface {
  Create(ctx context.Context, user *User) error
  GetByID(ctx context.Context, id string) (*User, error)
  Update(ctx context.Context, user *User) error
}

type EmailService interface {
  Send(ctx context.Context, to, subject, body string) error
}
```

---

### 2. Repository Layer (`repository/`)

**Responsibility**: Implements data access interfaces defined in domain. Communicates with databases, caches, external data sources.

**Key Rules**:
- Implements interfaces from `domain/`
- Concrete database logic (SQL, queries, transactions)
- Depends on infrastructure (database connections)
- Does NOT know about HTTP, gRPC, or delivery details
- Only imports: domain, infrastructure, database drivers, standard library

**Example**:
```go
package repository

import "yourapp/domain"

type PostgresUserRepository struct {
  db *sql.DB
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *domain.User) error {
  // SQL INSERT
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
  // SQL SELECT
}
```

---

### 3. Usecase Layer (`usecase/`)

**Responsibility**: Application business logic. Orchestrates domain entities and repositories. Implements use cases/features.

**Key Rules**:
- Depends on domain interfaces, not concrete implementations
- Contains business rules orchestration
- Does NOT know about HTTP, gRPC, databases
- Does NOT know about how data is presented
- Only imports: domain, standard library, context

**Example**:
```go
package usecase

import "yourapp/domain"

type CreateUserUseCase struct {
  userRepo domain.UserRepository
  emailSvc domain.EmailService
}

func (uc *CreateUserUseCase) Execute(ctx context.Context, email, name string) (*domain.User, error) {
  // Business logic: validation, create entity, save, send email
  user := &domain.User{ID: uuid.New().String(), Email: email, Name: name}
  if err := uc.userRepo.Create(ctx, user); err != nil {
    return nil, err
  }
  uc.emailSvc.Send(ctx, email, "Welcome", "...")
  return user, nil
}
```

---

### 4. Delivery Layer (`delivery/`)

**Responsibility**: HTTP/gRPC handlers, controllers. Translates external requests to usecases.

**Key Rules**:
- HTTP handler functions or gRPC service implementations
- Parses incoming requests (HTTP body, query params, gRPC messages)
- Calls usecases
- Formats responses (JSON, protobuf)
- Does NOT contain business logic
- Only imports: domain, usecase, adapter (DTOs), standard library, net/http or grpc

**Example (HTTP)**:
```go
package delivery

import "yourapp/usecase"

type UserHandler struct {
  createUserUC *usecase.CreateUserUseCase
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
  var req CreateUserRequest
  json.NewDecoder(r.Body).Decode(&req)
  
  user, err := h.createUserUC.Execute(r.Context(), req.Email, req.Name)
  if err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
  }
  
  json.NewEncoder(w).Encode(user)
}
```

---

### 5. Infrastructure Layer (`infrastructure/`)

**Responsibility**: External dependencies, configuration, frameworks. Database connections, logging, caching.

**Key Rules**:
- Establishes connections to external systems (DB, cache, message queues)
- Configuration loading
- Logging, monitoring setup
- Framework-specific setup (HTTP server, gRPC server)
- Does NOT implement domain logic
- Imports: everything (framework, drivers, domain)

**Example**:
```go
package infrastructure

func NewPostgresDB(dsn string) (*sql.DB, error) {
  return sql.Open("postgres", dsn)
}

func NewHTTPServer(router *http.ServeMux) *http.Server {
  return &http.Server{
    Addr:    ":8080",
    Handler: router,
  }
}
```

---

### 6. Adapter Layer (`adapter/`)

**Responsibility**: DTOs (Data Transfer Objects) and mappers between layers.

**Key Rules**:
- HTTP request/response DTOs
- gRPC message mappings
- Domain-to-DTO and DTO-to-domain mappers
- Does NOT contain business logic
- Only imports: domain, standard library

**Example**:
```go
package adapter

type CreateUserRequest struct {
  Email string `json:"email"`
  Name  string `json:"name"`
}

type UserResponse struct {
  ID    string `json:"id"`
  Email string `json:"email"`
  Name  string `json:"name"`
}

func MapUserToDomain(req CreateUserRequest) domain.User {
  return domain.User{Email: req.Email, Name: req.Name}
}

func MapUserToResponse(user domain.User) UserResponse {
  return UserResponse{ID: user.ID, Email: user.Email, Name: user.Name}
}
```

---

## Dependency Flow (Critical)

```
Infrastructure (frameworks, DB, external services)
        ↑
Repository (implements domain interfaces, uses infrastructure)
        ↑
Usecase (orchestrates domain, uses repository interfaces)
        ↑
Domain (pure logic, defines interfaces)
        ↑
Delivery (handlers, calls usecases)
        ↑
Adapter (DTOs, mappers)
```

**Key Rule**: Inner layers (domain, usecase, repository) do NOT import outer layers (delivery, adapter, infrastructure). Only outer layers import inward.

**Valid Imports**:
- ✅ delivery → usecase → domain
- ✅ repository → domain
- ✅ infrastructure → (anything)
- ✅ adapter → domain

**Invalid Imports**:
- ❌ domain → delivery
- ❌ domain → repository
- ❌ usecase → delivery
- ❌ usecase → infrastructure

