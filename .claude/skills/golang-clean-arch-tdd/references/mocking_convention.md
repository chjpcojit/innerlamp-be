# Mocking convention

This repo hand-writes mocks with `testify/mock`. It does **not** use `gomock`/`mockery` codegen — there is no `go:generate` directive anywhere in `services/appService` or `pkg/` (verified by grepping both trees). Do not introduce mockgen/mockery tooling for a new interface without being asked; follow the existing hand-written pattern instead.

## Where the mock lives

**In the same package as the interface it mocks**, not a separate `mocks/` package. `domain.UserRepository` is mocked by `domain/mock_user_repository.go`'s `domain.MockUserRepository` — same `package domain`. This is called out explicitly in `services/appService/domain/README.md`:

> 2 file `mock_*.go` nằm cùng package `domain` (không tách package `mocks` riêng) — khi thêm method mới vào `UserRepository`/`SutraRepository`, phải cập nhật mock tương ứng thủ công, không có `go:generate` tự sinh.

Same pattern applies outside `domain` too — e.g. `pkg/logging` ships its own hand-written `logging.MockLogger` for the same reason: use cases take `logging.Logger` as a constructor dependency and tests need a mock for it.

## The pattern

```go
package domain

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockUserRepository is a mock implementation of UserRepository for testing.
type MockUserRepository struct {
	mock.Mock
}

// Register mocks the Register method.
func (m *MockUserRepository) Register(ctx context.Context, user *User, device *Device, issueTokens TokenIssuer) (*Session, error) {
	args := m.Called(ctx, user, device, issueTokens)
	if s := args.Get(0); s != nil {
		return s.(*Session), args.Error(1)
	}
	return nil, args.Error(1)
}
```

Every method on the mock:
1. Forwards its arguments to `m.Called(...)`.
2. Type-asserts the first return value only if it's non-nil (avoids a panic when a test stubs an error-only return).
3. Returns `args.Error(N)` for the error slot.

## Using it in a usecase test

```go
repo := &domain.MockUserRepository{}
jwtSvc := &jwt.MockService{}
logger := newLogger() // local test helper, see below

repo.On("Register",
	mock.Anything,
	mock.AnythingOfType("*domain.User"),
	mock.AnythingOfType("*domain.Device"),
	mock.AnythingOfType("domain.TokenIssuer"),
).Run(func(args mock.Arguments) {
	// mutate the *domain.User arg here if the code under test relies on
	// the repository having assigned an ID, mirroring what Postgres does
}).Return(&domain.Session{ /* ... */ }, nil)
```

For a chainable interface like `logging.Logger` (`WithContext(...).WithFields(...).Info(...)`), stub every method the chain calls in a small local helper so every test doesn't repeat the same five `.On(...)` calls:

```go
func newLogger() *logging.MockLogger {
	l := &logging.MockLogger{}
	l.On("WithContext", mock.Anything).Return(l)
	l.On("WithFields", mock.Anything).Return(l)
	l.On("WithError", mock.Anything).Return(l)
	l.On("Info", mock.Anything, mock.Anything).Return()
	l.On("Error", mock.Anything, mock.Anything).Return()
	return l
}
```

## When a port's interface changes

If a method is added to (or removed from, or has its signature changed on) a `domain` repository interface, update the corresponding `mock_*.go` file **by hand, in the same change** — there is no codegen step to run instead. Forgetting this fails the build (the mock no longer satisfies the interface), which is the intended safety net: a compile error here is cheaper than a silently-stale mock.

## Delivery-layer handlers follow the same idea, without a shared mock file

A handler's consumer-defined interface (see `references/layer_conventions.md`) is usually small enough (often one method) that its test file defines a tiny inline mock directly in the `_test.go` file rather than a separate `mock_*.go` — check the sibling `*_handler_test.go` in the target service for the exact local convention before adding a new one.
