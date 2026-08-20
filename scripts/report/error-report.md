# InnerLamp Banking Harness — Error Report

| Field | Value |
|---|---|
| Timestamp | 2026-08-18T16:55:25Z |
| Schema | not provided |
| Total violations | 18 |

## Result: FAIL ❌

⚠️ **18 violations detected. Review and fix before deployment.**

---

## Violations

### ❌ Violation #1: MISSING_TEST

**Detail:**
```
Test file missing for services/appService/cmd/main.go
```

**How to Fix:**
Create services/appService/cmd/main_test.go with table-driven tests

**Affected File:** `services/appService/cmd/main.go`

---

### ❌ Violation #2: MISSING_TEST

**Detail:**
```
Test file missing for services/appService/delivery/http/auth_context.go
```

**How to Fix:**
Create services/appService/delivery/http/auth_context_test.go with table-driven tests

**Affected File:** `services/appService/delivery/http/auth_context.go`

---

### ❌ Violation #3: MISSING_TEST

**Detail:**
```
Test file missing for services/appService/domain/mock_user_repository.go
```

**How to Fix:**
Create services/appService/domain/mock_user_repository_test.go with table-driven tests

**Affected File:** `services/appService/domain/mock_user_repository.go`

---

### ❌ Violation #4: GOSEC_VIOLATION

**Detail:**
```
/Users/chjpcoj/Documents/Programming/Go/workspaces/src/innerlamp/services/appService/delivery/http/reset_password_handler.go:17 [HIGH] G101: Potential hardcoded credentials
```

**How to Fix:**
Review gosec output and fix security issue

**Affected File:** `services/appService/delivery/http/reset_password_handler.go`

---

### ❌ Violation #5: GOSEC_VIOLATION

**Detail:**
```
/Users/chjpcoj/Documents/Programming/Go/workspaces/src/innerlamp/services/appService/delivery/http/forgot_password_handler.go:18 [HIGH] G101: Potential hardcoded credentials
```

**How to Fix:**
Review gosec output and fix security issue

**Affected File:** `services/appService/delivery/http/forgot_password_handler.go`

---

### ❌ Violation #6: GOSEC_VIOLATION

**Detail:**
```
/Users/chjpcoj/Documents/Programming/Go/workspaces/src/innerlamp/pkg/config/loader.go:63 [MEDIUM] G304: Potential file inclusion via variable
```

**How to Fix:**
Review gosec output and fix security issue

---

### ❌ Violation #7: GOSEC_VIOLATION

**Detail:**
```
/Users/chjpcoj/Documents/Programming/Go/workspaces/src/innerlamp/pkg/utils/http.go:214 [LOW] G104: Errors unhandled
```

**How to Fix:**
Review gosec output and fix security issue

---

### ❌ Violation #8: GOSEC_VIOLATION

**Detail:**
```
/Users/chjpcoj/Documents/Programming/Go/workspaces/src/innerlamp/pkg/utils/http.go:196 [LOW] G104: Errors unhandled
```

**How to Fix:**
Review gosec output and fix security issue

---

### ❌ Violation #9: STALE_GO_MOD

**Detail:**
```
go.mod or go.sum not in sync with imports: go.mod
```

**How to Fix:**
Run 'go mod tidy' and commit the updated go.mod and go.sum files

---

### ❌ Violation #10: HARDCODED_CONFIG

**Detail:**
```
Hardcoded config value (port/host/DSN) in source code | File: services/appService/cmd/main.go:115
```

**How to Fix:**
Load all operational config from config/config.yaml — never hardcode host, port, or connection strings. Reference: goalng.md Section 10

**Affected File:** `services/appService/cmd/main.go`

---

### ❌ Violation #11: TRANSACTION_IN_REPOSITORY

**Detail:**
```
Repository layer is managing DB transactions — transaction ownership belongs to usecase | File: services/appService/repository/postgres_user_repository.go
```

**How to Fix:**
Move BeginTx/Commit/Rollback to the usecase layer. Repository methods should accept an optional *sql.Tx or mongo.Session parameter injected from usecase.

**Affected File:** `services/appService/repository/postgres_user_repository.go`

---

### ❌ Violation #12: HARDCODED_GCP_PROJECT

**Detail:**
```
Hardcoded GCP project ID in source code | File: services/appService/cmd/main.go:30
```

**How to Fix:**
Load from config: cfg.GCP.ProjectID — never hardcode project IDs. Wrong project in production = data written to wrong environment.

**Affected File:** `services/appService/cmd/main.go`

---

### ❌ Violation #13: HARDCODED_GCS_BUCKET

**Detail:**
```
Hardcoded Cloud Storage bucket name | File: services/appService/cmd/main.go:74
```

**How to Fix:**
Load from config: cfg.Storage.DocumentsBucket. Using prod bucket in dev = real user documents in test environment.

**Affected File:** `services/appService/cmd/main.go`

---

### ❌ Violation #14: UNFINISHED_TODO

**Detail:**
```
TODO/FIXME comment in production code | File: services/appService/cmd/main.go:5 | Code: // TODO before deploying:
```

**How to Fix:**
Resolve the TODO before merging to production. If intentional, track in Jira and remove from code.

**Affected File:** `services/appService/cmd/main.go`

---

### ❌ Violation #15: MOCK_IN_PRODUCTION

**Detail:**
```
Mock/fake/stub pattern found in production code | File: services/appService/cmd/main.go:10 | Code: // MOCK_DATA_MARKER: dbConfig, jwtConfig placeholder values
```

**How to Fix:**
Replace mock implementation with a real implementation. See pkg/ for available clients (pkg/sms/, pkg/email/, etc.).

**Affected File:** `services/appService/cmd/main.go`

---

### ❌ Violation #16: MOCK_IN_PRODUCTION

**Detail:**
```
Mock/fake/stub pattern found in production code | File: services/appService/delivery/http/login_handler.go:26 | Code: // mock instead of the real use-case.
```

**How to Fix:**
Replace mock implementation with a real implementation. See pkg/ for available clients (pkg/sms/, pkg/email/, etc.).

**Affected File:** `services/appService/delivery/http/login_handler.go`

---

### ❌ Violation #17: MOCK_IN_PRODUCTION

**Detail:**
```
Mock/fake/stub pattern found in production code | File: services/appService/delivery/http/avatar_handler.go:15 | Code: // mock instead of the real use-case.
```

**How to Fix:**
Replace mock implementation with a real implementation. See pkg/ for available clients (pkg/sms/, pkg/email/, etc.).

**Affected File:** `services/appService/delivery/http/avatar_handler.go`

---

### ❌ Violation #18: MOCK_IN_PRODUCTION

**Detail:**
```
Mock/fake/stub pattern found in production code | File: services/appService/delivery/http/profile_handler.go:41 | Code: // mock instead of the real use-case.
```

**How to Fix:**
Replace mock implementation with a real implementation. See pkg/ for available clients (pkg/sms/, pkg/email/, etc.).

**Affected File:** `services/appService/delivery/http/profile_handler.go`

---

## Next Steps

1. Review each violation above
2. Fix violations manually OR run: `@Fix Agent` in Copilot Chat
3. Re-run: `./scripts/run-all.sh`
4. Repeat until all checks pass ✅

## Fix Agent Usage

```
1. Open GitHub Copilot Chat in VS Code
2. Select agent: Fix Agent
3. Paste path to this error report or say 'fix all violations'
```
