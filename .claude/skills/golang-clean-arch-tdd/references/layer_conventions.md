# Layer conventions

Concrete rules per layer, grounded in `services/appService`. Follow these exactly when the target service shares this layout; when a future service diverges, match that service's own established pattern instead of forcing this one.

## `domain/`

Contains, and only contains:

- **Entity structs** — one file per aggregate (`user.go`, `content.go`...). Plain Go structs mapping to DB tables. No `json:"..."` tag, no `db:"..."` tag, no framework import. If a struct in `domain/` grows a `json` tag, that's a delivery concern leaking inward — stop and move it out.
- **Repository interfaces (ports)** — one interface per aggregate, e.g. `UserRepository`, `SutraRepository`, declared next to the entity it serves. Each method's doc comment should cite the user story / acceptance criterion it implements and the sentinel error it returns on failure, e.g.:
  ```go
  // FindByEmail looks up a user by their (already normalized) email.
  // Returns ErrUserNotFound if no such user exists — US-002 AC3.
  FindByEmail(ctx context.Context, email string) (*User, error)
  ```
- **Use-case boundary structs** — `RegisterInput`, `LoginInput`, `ListSutrasInput`, `RegisterResult`... These are not pure Entities (Uncle Bob would call them the use case's Request/Response model) but live in `domain` rather than `usecase` specifically so `delivery` can reference them without importing a concrete `usecase` type.
- **Sentinel errors** — `errors.go`, one `var ErrX = errors.New("...")` block per feature area, each with a comment on which layer detects it and why it exists (e.g. `ErrInvalidCredentials` exists so login doesn't leak whether the failure was "unknown email" vs "wrong password").
- **Callback-port types**, when an outer layer's write must call back into a use-case-level capability at a specific moment (see below).

Must NOT contain: `net/http`, `database/sql`, any DB driver, any concrete `pkg/*` backend, any method that opens a network connection. `domain` package imports should be stdlib only (`context`, `time`, `errors`) plus `github.com/stretchr/testify/mock` in `mock_*.go` files only (test-support code, not real domain logic).

### The callback-port pattern (e.g. `TokenIssuer`)

Sometimes a repository write needs a value that can only be produced by use-case-level logic (which JWT claims, which TTL), but that value is only needed *mid-transaction*, after a DB-generated ID becomes known and before a NOT NULL column depending on it is inserted in the same transaction. Splitting this into two round-trips would break atomicity. The fix: `domain` declares a plain function type —

```go
// TokenIssuer generates an access/refresh token pair for a persisted user
// ID. It is supplied by the use-case so the repository can mint tokens
// after the user row's ID is known, inside the same transaction that also
// creates the device and session rows.
type TokenIssuer func(userID string) (accessToken, refreshToken string, err error)
```

— and the port method takes it as a parameter (`Register(ctx, user, device, issueTokens TokenIssuer) (*Session, error)`). `usecase` supplies the real closure (closing over `jwtService`); `repository` calls `issueTokens(...)` at the right point inside its `tx`, without ever importing the JWT package. Reach for this only when there's a genuine atomicity/ordering constraint forcing outer-layer logic to run *during* an inner-layer-triggered write — not as a general substitute for a repository interface.

## `usecase/`

- One file per use case (`register_usecase.go`, `login_usecase.go`...), one exported struct with a constructor taking interfaces:
  ```go
  func NewRegisterUsecase(users domain.UserRepository, jwtService jwt.Service, logger logging.Logger) *RegisterUsecase
  ```
- Depends on `domain` interfaces and reusable `pkg/*` abstractions (`pkg/jwt`, `pkg/logging`, `pkg/validation`) — never a concrete `repository` or `delivery` type. `usecase/*.go` must not import `.../services/<svc>/repository` or `.../services/<svc>/delivery`.
- Does business-rule validation/normalization only (e.g. lowercasing email for case-insensitive uniqueness) — input *format* validation (email shape, required fields) belongs to `delivery`'s request struct tags and runs before the use case is ever called; say so in a comment if it's non-obvious.
- Maps a repository's internal sentinel error to a different caller-facing one when the business rule requires it (e.g. `login` maps `domain.ErrUserNotFound` → `domain.ErrInvalidCredentials` so unknown-email and wrong-password look identical to the caller).

## `repository/`

- One file per aggregate: `postgres_<name>_repository.go`. Constructor returns the **interface** type, not the concrete struct:
  ```go
  func NewUserRepository(db database.Database) domain.UserRepository {
      return &userRepository{db: db}
  }
  ```
- Multi-statement writes that must be atomic wrap in a transaction: `tx, err := r.db.Begin(ctx)` → `defer func() { _ = tx.Rollback() }()` → ... → `tx.Commit()`.
- Translates DB-specific failures into `domain` sentinel errors at the boundary — a caller of this package should never see a raw `sql.ErrNoRows` or a Postgres unique-violation code:
  ```go
  if errors.Is(err, sql.ErrNoRows) {
      return nil, domain.ErrUserNotFound
  }
  if isUniqueViolation(err, "users_email_key") {
      return nil, domain.ErrEmailTaken
  }
  ```
- Test this layer against real infrastructure (existing `*_repository_test.go` files show the pattern already used in the target service) — a repository test that mocks the DB doesn't validate the SQL.

## `delivery/http/`

- One handler struct per endpoint (`RegisterHandler`, `ProfileHandler`...).
- **Consumer-defined interface**: the handler declares its own narrow interface for just the usecase method(s) it calls, instead of depending on `*usecase.RegisterUsecase` directly:
  ```go
  // registerUsecase is the subset of *usecase.RegisterUsecase this handler
  // depends on, defined here (the consumer) so the handler can be unit
  // tested with a mock instead of the real use-case.
  type registerUsecase interface {
      Register(ctx context.Context, input domain.RegisterInput) (*domain.RegisterResult, error)
  }
  ```
  This is Dependency Inversion applied a second time, at the delivery boundary — it lets the handler be unit-tested with a hand-written mock without touching the real use case or its dependencies.
- **Request structs** carry `json` tags and `validate` tags for format checking (`validate:"required,email"`) — this is where input format is rejected, before the use case ever runs.
- **Response structs are never a `domain` entity.** Define a dedicated `xxxResponse` struct per handler (or reuse one across handlers when the shape is genuinely identical, e.g. `login` reuses `register`'s `userResponse`) and map field-by-field. Two handlers are free to name the same `domain` field differently in JSON — e.g. `profileResponse.PhoneNumber` serializes as `"phone"` while `registerResponse`'s equivalent serializes as `"phone_number"` — because each is answering to a different API contract, not to the DB shape. Putting the response struct in `domain` instead would force one JSON shape on every consumer and make the API contract change every time `domain` changes for an unrelated reason.
- **Parsing and writing both go through `pkg/utils`, never a hand-rolled decoder/encoder.** This is the one wire contract shared across every handler in the service, not a per-feature choice:
  - Decode the body with `utils.DecodeJSONBody(r, &req, opts)` or `utils.DecodeJSONBodyStrict(r, &req)` (rejects unknown fields) — never `json.NewDecoder(r.Body).Decode(...)` inline. On error, `utils.GetDecodeErrorDetails(err)` gives the field/message to report.
  - Validate the struct with `utils.GetValidator().Validate(req, nil)` (wraps `go-playground/validator/v10`) unless the feature already threads its own `*validation.Validator` for a tag `pkg/utils`' singleton doesn't register (e.g. `phone`) — don't run two different validator instances against the same struct for no reason.
  - Write success with `utils.WriteSuccessResponse(w, data)` / `WriteSuccessResponseWithStatus(w, status, data)`, and errors with `utils.WriteValidationError` (400), `utils.WriteBusinessError` (caller-chosen status), or `utils.WriteSystemError` (500) — all three wrap `WriteErrorResponse`, which serializes the shared `utils.ApiResponse{Success, Data, Error, Details}` envelope.
  - Do not define a second response envelope (a local `errorResponse`/`writeJSON` pair with its own JSON shape) — every handler in the service must reply with the same `success/data/error` contract so a client can parse any endpoint the same way. If an existing handler already has its own `writeJSON`/`writeError` helpers instead of calling into `pkg/utils`, that is drift from this rule, not a second accepted pattern — flag it and migrate to `pkg/utils` rather than copying it into a new handler.

## `config/` — `services/<service>/config/`

Not a Clean Architecture layer, but the rule that keeps one honest: **no layer below `cmd` names a value that can differ between environments.** The config package owns `AppConfig` and its sub-structs (`config.go`), and everything is injected down as constructor parameters. A layer that reads a package-level config singleton has broken dependency injection — the value must arrive as an argument.

What it holds, and what stays a constant:

| Belongs in `config/` | Stays a `const` in its layer |
|---|---|
| TTLs, cooldowns, caps, lockouts, timeouts, page sizes | Enum members whose set the schema CHECKs (`domain.OTPPurpose*`) |
| DB host/port/database/SSL mode, bucket names, project IDs | Sentinel error messages — they are the API contract |
| `storage.backend`, SMS provider type — anything selecting an implementation | A column's CHECK list — changing it is a migration, not a config edit |
| Credentials, as a `*_secret` **name**, resolved at load time | |

Operational limits sit firmly on the left: a TTL is a number a dev environment shortens to test expiry and prod lengthens under load, which is the definition of configuration. "It is a business rule" is not a reason to hardcode — a business rule a different environment may tune is configuration.

Adding one:

1. Field on the relevant sub-struct in `config.go`, with a `yaml:` tag. Any `time.Duration` uses the package's `Duration` type — yaml.v3 has no built-in support, and `Duration.UnmarshalYAML` is what parses `"90s"` / `"15m"` / `"168h"`.
2. Value in **every** `config.<env>.yaml`. A missing key decodes to the Go zero value, which for a TTL means "already expired", not "fall back to a default".
3. Thread it through `cmd/main.go` into the constructor that needs it. When a whole sub-struct is tunable together, pass the sub-struct rather than exploding it into one parameter per field — `usecase.NewRegisterUsecase(..., cfg.OTP)`.
4. A secret gets two fields: the `*_secret` name (yaml-tagged) and the resolved value (`yaml:"-"`, filled by `resolveSecrets` via the injected `SecretGetter`) — see `DatabaseConfig.PasswordSecret` / `Password`. The YAML never holds a literal credential.

**The layer names the config type; it does not redeclare it.** `usecase` may import `config` and declare a field or parameter as `config.OTPConfig`. It may not *read* config — no package-level singleton, no `config.Get()`, no `os.Getenv`. The distinction is between naming a shape and reaching for a value.

A local mirror struct in `usecase` looks like the safer middle ground and is worse than either extreme: the composition root then maps field-by-field with nothing checking that mapping, and a field missed there decodes to a zero value — a TTL of 0 is a code that expires the instant it is issued. `config/` is a leaf that imports nothing from the service, so `usecase → config` creates no cycle; `scripts/check-architecture.sh` rejects infrastructure, DB drivers and `net/http` in `usecase`, not `config`.

Worked example already in the tree: `JWTConfig.RefreshTokenExpiration` → `usecase.NewRegisterUsecase(..., sessionTTL)` → `CreateUserAndSession(ctx, ..., sessionTTL, ...)`. The stored session expiry and the signed refresh token's expiry agree by construction because both read the one config value instead of each restating `168h`.

Second worked example, added 2026-09-17: the OTP policy (`otpTTL`, `otpResendCooldown`, `otpMaxResend`, `otpMaxAttempts`, `otpLockDuration`, `pendingRegistrationTTL`) was a `const` block in `usecase/register_usecase.go`, justified in-comment as "business decisions, not configuration". It is now `config.OTPConfig` in `config.go`, with every value in `config.dev.yaml` and the struct passed to the constructor. Those six numbers were the ones a dev environment shortens to exercise expiry — which is what made the justification wrong.

**When a value looks like configuration and `config/` has no field for it, ask — don't decide alone.** Neither hardcode it with a comment explaining why it is "really a business rule", nor quietly promote it. Name the specific value and what makes it ambiguous; the two failure modes are both silent, so the bar for asking is low.

## `cmd/main.go`

The composition root — the only file in the service allowed to import every concrete type (`repository.NewUserRepository`, `usecase.NewRegisterUsecase`, `http.NewRegisterHandler`) and wire them together with `mux.Handle(...)`. If a handler needs middleware (auth, rate limiting), it gets wrapped here — a handler that's documented as requiring auth but isn't wrapped with the auth middleware in `cmd/main.go` is a wiring bug, not a layering violation, and won't be caught by `scripts/check-architecture.sh` (that script only checks import direction, not what's actually wired at runtime) — check `cmd/main.go`'s route table by hand for that.
