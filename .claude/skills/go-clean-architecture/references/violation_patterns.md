# Go Clean Architecture - Common Violation Patterns

## Pattern Detection Guide

Use this guide to identify architectural violations during code review.

---

## Category 1: Layer Boundary Violations

### 1.1 Domain Importing Outer Layers

**Pattern**: Domain imports delivery, repository, or infrastructure packages

**Detection**:
```bash
# In domain/ packages, check for imports like:
import "yourapp/delivery"    # ❌
import "yourapp/repository"  # ❌
import "yourapp/infrastructure" # ❌
```

**Example Violation**:
```go
// domain/user.go
package domain

import "yourapp/repository"  // ❌ VIOLATION

type User struct {
  ID   string
  repo repository.UserRepository
}
```

**Impact**: HIGH - Domain is no longer framework-agnostic
**Fix**: Move dependency to usecase layer via constructor injection

---

### 1.2 Usecase Importing Concrete Repository Implementations

**Pattern**: Usecase imports repository concrete types instead of domain interfaces

**Detection**:
```bash
# In usecase/ packages, check for:
import "yourapp/repository"
import "postgres" or "mysql" or "mongodb"

# And usage like:
repo := &repository.PostgresUserRepository{}  # ❌
```

**Example Violation**:
```go
// usecase/create_user.go
package usecase

import "yourapp/repository"

type CreateUserUseCase struct {
  repo *repository.PostgresUserRepository  // ❌ Concrete type
}
```

**Impact**: HIGH - Usecase tightly coupled to specific database
**Fix**: Use domain.UserRepository interface instead

---

### 1.3 Usecase Importing Infrastructure Directly

**Pattern**: Usecase imports infrastructure packages (database, cache, logging)

**Detection**:
```bash
# In usecase/ packages, check for:
import "yourapp/infrastructure"
import "database/sql"        # ❌ Direct DB in usecase
import "redis"              # ❌ Direct cache in usecase
import "github.com/sirupsen/logrus"  # ❌ Logging directly in usecase
```

**Example Violation**:
```go
// usecase/create_user.go
package usecase

import (
  "yourapp/infrastructure"
  "database/sql"
)

func (uc *CreateUserUseCase) Execute(ctx context.Context) error {
  db := infrastructure.GetDatabase()  // ❌ VIOLATION
  return db.Exec(...)
}
```

**Impact**: HIGH - Usecase knows about database/infrastructure details
**Fix**: Inject repository interface into usecase

---

### 1.4 Repository Importing Delivery

**Pattern**: Repository imports HTTP/gRPC handlers or delivery packages

**Detection**:
```bash
# In repository/ packages, check for:
import "yourapp/delivery"
import "net/http"  # ❌ HTTP in repository
import "google.golang.org/grpc"  # ❌ gRPC in repository
```

**Example Violation**:
```go
// repository/user_repository.go
package repository

import (
  "yourapp/delivery"
  "net/http"
)

type PostgresUserRepository struct {
  handler *delivery.UserHandler  // ❌ VIOLATION
}
```

**Impact**: CRITICAL - Repository shouldn't know about HTTP/gRPC
**Fix**: Remove delivery dependencies from repository

---

## Category 2: Circular Dependency Issues

### 2.1 Circular Imports Between Layers

**Pattern**: Package A imports B, and B imports A (directly or indirectly)

**Detection**:
```bash
# Check for cycles:
delivery → usecase → repository → delivery (❌ Cycle!)
domain ← repository ← usecase ← delivery (❌ Reverse dependency!)
```

**Example Violation**:
```go
// delivery/handler.go
package delivery

import "yourapp/usecase"

type UserHandler struct {
  uc *usecase.CreateUserUseCase
}

// usecase/create_user.go
package usecase

import "yourapp/delivery"  // ❌ Circular: delivery ← usecase ← delivery

type CreateUserUseCase struct {
  handler *delivery.UserHandler
}
```

**Impact**: CRITICAL - Won't compile, breaks architecture
**Fix**: Remove circular dependency; dependencies flow one direction only

---

## Category 3: Coupling Issues

### 3.1 Package-Level Coupling (Tight Dependencies)

**Pattern**: Layer depends on concrete implementations instead of interfaces

**Detection**:
```bash
# Look for:
repo := &repository.PostgresUserRepository{}  # ❌ Concrete
service := &infrastructure.SMTPEmailService{}  # ❌ Concrete

# Instead of:
var repo domain.UserRepository  # ✅ Interface
var emailSvc domain.EmailService  # ✅ Interface
```

**Example Violation**:
```go
// usecase/create_user.go
package usecase

type CreateUserUseCase struct {
  repo *repository.PostgresUserRepository  // ❌ Tight coupling
}

// Hard to test, can't swap implementations
```

**Impact**: MEDIUM - Tight coupling, hard to test, inflexible
**Fix**: Depend on domain interfaces, inject via constructor

---

### 3.2 Global State / Package-Level Singletons

**Pattern**: Layers use global singletons instead of dependency injection

**Detection**:
```bash
# Look for:
var globalDB *sql.DB  // ❌
var globalCache *redis.Client  // ❌

func init() {
  globalDB = initDB()  // ❌
}
```

**Example Violation**:
```go
// infrastructure/database.go
package infrastructure

var DB *sql.DB  // ❌ Global state

func init() {
  DB = sql.Open(...)
}

// usecase/create_user.go
package usecase

import "yourapp/infrastructure"

func (uc *CreateUserUseCase) Execute() error {
  return infrastructure.DB.Exec(...)  // ❌ Depends on global
}
```

**Impact**: MEDIUM - Hard to test, global state, inflexible
**Fix**: Inject dependencies via constructor or dependency container

---

## Category 4: Missing Layer Violations

### 4.1 Business Logic in Delivery Layer

**Pattern**: HTTP/gRPC handlers contain business logic instead of delegating to usecase

**Detection**:
```bash
# In delivery/ packages, look for:
- Complex conditional logic
- Database queries
- Business rule validation
- State mutations
```

**Example Violation**:
```go
// delivery/handler.go
package delivery

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
  var req CreateUserRequest
  json.NewDecoder(r.Body).Decode(&req)
  
  // ❌ Business logic in delivery!
  if req.Email == "" {
    http.Error(w, "invalid email", http.StatusBadRequest)
    return
  }
  
  user := &User{Email: req.Email, Name: req.Name}
  
  // ❌ Direct database access in delivery!
  h.db.Exec("INSERT INTO users ...")
  
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(user)
}
```

**Impact**: HIGH - Business logic in wrong layer, hard to reuse, hard to test
**Fix**: Move logic to usecase, keep delivery as thin HTTP handler

---

### 4.2 Missing Repository Abstraction

**Pattern**: Direct database access in usecase or delivery

**Detection**:
```bash
# Look for direct database imports in usecase/delivery:
import "database/sql"
import "gorm.io/gorm"
import "go.mongodb.org/mongo-driver"

# And usage like:
db.QueryContext(...)  # ❌
db.Exec(...)          # ❌
```

**Example Violation**:
```go
// usecase/create_user.go
package usecase

import "database/sql"

func (uc *CreateUserUseCase) Execute(ctx context.Context, email string) error {
  // ❌ Direct SQL in usecase!
  _, err := uc.db.ExecContext(ctx, "INSERT INTO users (email) VALUES ($1)", email)
  return err
}
```

**Impact**: HIGH - Usecase coupled to database, hard to test, can't swap DB
**Fix**: Create repository interface, implement in repository layer

---

## Category 5: Testing-Related Violations

### 5.1 Concrete Dependencies Instead of Interfaces

**Pattern**: Using concrete types instead of interfaces makes mocking impossible

**Detection**:
```bash
# Hard to test:
type CreateUserUseCase struct {
  repo *repository.PostgresUserRepository  // Can't mock this
}

# Easy to test:
type CreateUserUseCase struct {
  repo domain.UserRepository  // Easy to mock
}
```

**Impact**: MEDIUM - Hard to write unit tests, integration tests required
**Fix**: Always use interfaces for external dependencies

---

### 5.2 Missing Dependency Injection

**Pattern**: Dependencies created inside functions instead of injected

**Detection**:
```bash
# Hard to test:
func (uc *CreateUserUseCase) Execute() error {
  repo := repository.NewPostgresUserRepository(...)  // ❌ Created inside
}

# Easy to test:
type CreateUserUseCase struct {
  repo domain.UserRepository  // ✅ Injected
}
```

**Impact**: MEDIUM - Hard to test with mocked dependencies
**Fix**: Always inject dependencies via constructor

---

## Category 6: Naming & Organization Violations

### 6.1 Wrong Package Structure

**Pattern**: Layers organized incorrectly or named unclearly

**Detection**:
```bash
❌ WRONG:
myapp/
  ├── models/          (unclear what's here)
  ├── handlers/        (unclear from where)
  ├── services/        (too generic)
  └── utils/           (catch-all)

✅ CORRECT:
myapp/
  ├── domain/
  ├── repository/
  ├── usecase/
  ├── delivery/
  ├── infrastructure/
  └── adapter/
```

**Impact**: LOW - Makes code navigation harder
**Fix**: Rename packages to follow clean architecture layers

---

### 6.2 Generic Package Names

**Pattern**: Packages named `utils`, `helpers`, `common` (too vague)

**Detection**:
```bash
❌ myapp/utils/         (what's actually here?)
❌ myapp/helpers/       (too generic)
❌ myapp/common/        (unclear)

✅ myapp/infrastructure/config/
✅ myapp/adapter/mapper/
✅ myapp/domain/entity/
```

**Impact**: LOW - Code harder to navigate
**Fix**: Use layer names + specific purpose

---

## Review Checklist

When reviewing Go code for clean architecture, check for:

- [ ] Domain layer has NO imports from other internal packages
- [ ] Repository only imports domain interfaces, not usecase/delivery
- [ ] Usecase depends on domain interfaces, not concrete implementations
- [ ] Delivery handlers call usecases, not repositories directly
- [ ] Infrastructure wires up all dependencies
- [ ] No circular dependencies between layers
- [ ] Dependencies injected via constructor, not global singletons
- [ ] All external dependencies are interfaces
- [ ] Business logic is in usecase, not delivery
- [ ] Database access only in repository layer
- [ ] Package structure matches clean architecture layers

