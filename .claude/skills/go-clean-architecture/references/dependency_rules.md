# Go Clean Architecture - Dependency Rules Matrix

## Import Rules by Layer

### Domain Layer (`domain/`)

**Can Import**:
- ✅ Standard library (context, fmt, errors, sync, etc.)
- ✅ External domain-specific libraries (e.g., UUID, crypto)
- ✅ Other domain packages

**Cannot Import**:
- ❌ Repository layer
- ❌ Usecase layer
- ❌ Delivery layer (HTTP, gRPC)
- ❌ Infrastructure layer (database, cache, logging frameworks)
- ❌ Adapter layer

**Reasoning**: Domain should be framework-agnostic and reusable. No knowledge of how data is accessed or presented.

---

### Repository Layer (`repository/`)

**Can Import**:
- ✅ Domain layer (interfaces and entities)
- ✅ Infrastructure layer (database connections, drivers)
- ✅ Standard library
- ✅ External database libraries (database/sql, sqlc, etc.)

**Cannot Import**:
- ❌ Usecase layer
- ❌ Delivery layer (HTTP, gRPC)
- ❌ Adapter layer (DTOs)

**Reasoning**: Repository implements domain contracts and uses infrastructure but should not know about business orchestration or HTTP/gRPC details.

---

### Usecase Layer (`usecase/`)

**Can Import**:
- ✅ Domain layer (interfaces and entities)
- ✅ Standard library
- ✅ Context package
- ✅ Logging libraries (for usecase-level logging only)

**Cannot Import**:
- ❌ Repository layer (concrete implementations)
  - ✅ BUT CAN use domain repository interfaces
- ❌ Delivery layer (HTTP, gRPC)
- ❌ Infrastructure layer (databases, cache directly)
- ❌ Adapter layer (DTOs)

**Reasoning**: Usecase depends on domain interfaces, not concrete implementations. This enables testing and flexibility.

---

### Delivery Layer (`delivery/`)

**Can Import**:
- ✅ Domain layer (entities for responses)
- ✅ Usecase layer (to call business logic)
- ✅ Adapter layer (DTOs, mappers)
- ✅ Standard library (net/http, encoding/json)
- ✅ gRPC libraries (google.golang.org/grpc)
- ✅ Middleware libraries (CORS, auth, etc.)

**Cannot Import**:
- ❌ Repository layer (concrete implementations)
- ❌ Infrastructure layer directly

**Reasoning**: Delivery is the HTTP/gRPC entry point. It calls usecases and formats responses using adapters.

---

### Infrastructure Layer (`infrastructure/`)

**Can Import**:
- ✅ Everything (framework, database, logging, monitoring)
- ✅ Domain, repository, usecase, delivery (to wire up)
- ✅ Standard library
- ✅ All external packages

**Reasoning**: Infrastructure is the outermost layer and knows about frameworks and external systems. It wires everything together.

---

### Adapter Layer (`adapter/`)

**Can Import**:
- ✅ Domain layer (entities)
- ✅ Standard library
- ✅ JSON/protobuf libraries for marshaling

**Cannot Import**:
- ❌ Repository layer
- ❌ Usecase layer
- ❌ Delivery layer
- ❌ Infrastructure layer

**Reasoning**: Adapters are stateless mappers. They only know about domain entities and DTOs.

---

## Common Violations & Fixes

### Violation 1: Domain imports repository
```go
// ❌ WRONG - domain/user.go
package domain

import "yourapp/repository"

type User struct {
  repo repository.UserRepository  // ❌ Domain shouldn't know about concrete repo
}
```

**Fix**:
```go
// ✅ CORRECT - domain/user.go
package domain

type UserRepository interface {  // Define interface in domain
  Create(ctx context.Context, user *User) error
}

type User struct {
  ID    string
  Email string
}
```

---

### Violation 2: Usecase imports repository concrete type
```go
// ❌ WRONG - usecase/create_user.go
package usecase

import "yourapp/repository"

type CreateUserUseCase struct {
  repo *repository.PostgresUserRepository  // ❌ Concrete type
}
```

**Fix**:
```go
// ✅ CORRECT - usecase/create_user.go
package usecase

import "yourapp/domain"

type CreateUserUseCase struct {
  repo domain.UserRepository  // Interface from domain
}
```

---

### Violation 3: Repository imports delivery
```go
// ❌ WRONG - repository/user_repository.go
package repository

import "yourapp/delivery"

type PostgresUserRepository struct {
  handler *delivery.UserHandler  // ❌ Repository shouldn't know about HTTP
}
```

**Fix**: Remove the dependency. Repository doesn't need to know about delivery.

---

### Violation 4: Usecase imports infrastructure directly
```go
// ❌ WRONG - usecase/create_user.go
package usecase

import "yourapp/infrastructure"

func (uc *CreateUserUseCase) Execute(ctx context.Context, email string) error {
  db := infrastructure.GetDatabase()  // ❌ Direct dependency
  return db.Exec(...)
}
```

**Fix**:
```go
// ✅ CORRECT - usecase/create_user.go
type CreateUserUseCase struct {
  repo domain.UserRepository  // Injected interface
}

func (uc *CreateUserUseCase) Execute(ctx context.Context, email string) error {
  return uc.repo.Create(ctx, &User{Email: email})
}
```

---

## Dependency Injection Pattern

All dependencies should be injected via constructor functions, not imported directly:

```go
// ✅ CORRECT - usecase/create_user.go
type CreateUserUseCase struct {
  userRepo  domain.UserRepository
  emailSvc  domain.EmailService
}

func NewCreateUserUseCase(
  userRepo domain.UserRepository,
  emailSvc domain.EmailService,
) *CreateUserUseCase {
  return &CreateUserUseCase{
    userRepo:  userRepo,
    emailSvc:  emailSvc,
  }
}
```

**Benefits**:
- ✅ Easy to test (mock dependencies)
- ✅ Flexible (swap implementations)
- ✅ Clear dependencies
- ✅ Avoids circular imports

