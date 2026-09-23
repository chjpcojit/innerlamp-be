# Test Coverage Report — login

AC coverage audits and E2E wiring findings for this feature, run by `golang-e2e-verification`.
Newest run first. This file is an audit log, not part of the requirements→design→tasks cascade —
it needs no version number and no `changelog.md` entry of its own.

## 2026-09-21

First run. The E2E harness was created in this pass: `services/appService/e2e/harness.go`
(testcontainers-go v0.44.0, `postgres:16-alpine`, all `migrations/*.sql`, the real binary built
from `services/appService/cmd`). Test: `services/appService/e2e/login_e2e_test.go`
`TestLoginFlow_E2E`, 10 subtests, all passing (`go test -count=1 ./services/appService/e2e/` →
`ok … 13.4s`). It is skipped under `-short`.

Paths below are relative to `services/appService/`. `E2E:<line>` is the subtest's line in
`e2e/login_e2e_test.go`.

### AC coverage

| AC | Requirement (short) | Covered by | Status |
|----|---------------------|-----------|--------|
| AC1 | Correct phone+password+device → 200, tokens + user, session for the device; local phone form normalises | `usecase/login_usecase_test.go:166` `TestLogin_HappyPath`; `repository/postgres_login_repository_test.go:220`; `delivery/http/login_handler_test.go:31`; **E2E:36** (login with `0…` form, `user.phone_number` is `+84…`) | ✅ covered |
| AC2 | Wrong password / unknown phone / pending-only → same 401 `INVALID_CREDENTIALS`, no timing leak | `usecase/login_usecase_test.go:277` `TestLogin_InvalidCredentials`, `:451` `TestLogin_NoEnumeration` (timing); `repository/…:81` (pending-only); **E2E:64** (three bodies byte-identical) | ✅ covered |
| AC3 | New device → separate session, other devices untouched | `repository/postgres_login_repository_test.go:287`; **E2E:86** | ✅ covered |
| AC4 | Same device again → only that device's previous session revoked | `repository/postgres_login_repository_test.go:261`; **E2E:86** | ✅ covered |
| AC5 | Missing device / `platform` / `device_model` → 400 `INVALID_DEVICE`, not counted | `usecase/login_usecase_test.go:340` `TestLogin_Validation`; `delivery/http/login_handler_test.go:117`; **E2E:99** (5 rejections, then login still 200) | ✅ covered |
| AC6 | Bad phone / empty password → 400 `INVALID_PHONE` / `INVALID_PASSWORD`, not counted | `usecase/login_usecase_test.go:340`; `delivery/http/login_handler_test.go:118-119`; **E2E:99** | ✅ covered |
| AC7 | 5th failure locks for 15 min; locked rejects even the correct password with 429 `LOGIN_LOCKED` + `locked_until`; applies to phones without an account; reset rules | `usecase/login_usecase_test.go:488` `TestLogin_FailurePolicy` (reset window); `repository/…:314` (reset on success); `delivery/http/login_handler_test.go:116`; **E2E:122** (the config threshold and duration reach the running server; registered and unregistered phones) | ✅ covered |
| AC8 | Access token 15m, refresh 7d, only the SHA-256 hash stored, access token carries `sid` | `usecase/login_usecase_test.go:214` `TestLogin_TokenIssuerContract`; `repository/…:220` (hash stored); **E2E:36** (`exp − iat` = 900s / 604800s on real tokens, `extra.sid` present) | ✅ covered |
| AC9 | Refresh → new pair with the same `sid`, old token unusable | `usecase/login_usecase_test.go:687`; `repository/…:486`; `delivery/http/login_handler_test.go:139`; **E2E:143** | ✅ covered |
| AC10 | Any unusable refresh token → 401 `REFRESH_TOKEN_INVALID` | `usecase/login_usecase_test.go:794` `TestRefresh_Invalid`; `repository/…:452`; **E2E:143** (garbage, access token as refresh), **E2E:190** (after logout) | ✅ covered |
| AC11 | Reuse of the previous refresh token → 401 and revokes that device's session only | `usecase/login_usecase_test.go:767`; `repository/…:438`; **E2E:143** (rotated token also dies, other device refreshes) | ✅ covered |
| AC12 | Device-less `verify-otp` session refreshes and logs out like any other | `usecase/login_usecase_test.go:855`; `repository/…:425`; **E2E:173** (real US-001 token carries `sid`, refresh 200, logout 200, refresh after 401) | ✅ covered |
| AC13 | Logout with a valid access token revokes that session, 200 `{"success": true}`, later refresh fails | `usecase/login_usecase_test.go:939`; `repository/…:593`; `delivery/http/login_handler_test.go:212`; **E2E:190** | ✅ covered |
| AC14 | Logout of an already-gone/unknown/foreign `sid` → 200, no state change | `usecase/login_usecase_test.go:954`; `repository/…:614`, `:629`; **E2E:190** (second call), **E2E:207** (validly signed token for user A naming user B's `sid`: B's and A's sessions both survive) | ✅ covered |
| AC15 | No header / invalid / expired / refresh-type / no-`sid` token → 401 `UNAUTHORIZED`, no state change | `delivery/http/login_handler_test.go:225`, `:244` (handler); **E2E:229** (all five cases over HTTP; the session still refreshes afterwards) | ✅ covered, with the caveat below |

### E2E findings

No wiring bug in `cmd/main.go` this run.

To check that the harness really exercises the wiring, the logout route was temporarily registered
without `authMiddleware.AuthenticateFunc(...)` (then restored, diff clean):

- **AC12 and AC13 failed**, which is what catches a missing wrap: a valid access token gets 401
  because no claims reach the handler.
- **AC15 still passed.** `LoginHandler.Logout` itself answers 401 `UNAUTHORIZED` when the context
  has no claims, so the rejection cases look the same with or without the middleware.
- Likewise, `RequireTokenType: "access"` cannot be observed over HTTP: a refresh token has no
  `sid`, so the handler rejects it anyway.

Both points are written in the comment above the AC15 subtest.

Test-only problem found and fixed: `pkg/jwt.Config.Validate` refuses a non-positive TTL, so the
"expired access token" case now mints with a 1s TTL and sleeps 2s.

### Follow-ups

- `RequireTokenType: "access"` on the logout middleware is defence in depth that no test observes
  end-to-end. It is covered only by `pkg/jwt/middleware`'s own tests. Worth an assertion if a
  future route accepts tokens without a `sid`-style check.
- The harness depends on the dev OTP constant `"111111"` (`usecase.generateOTP`, `⚠️ DEV ONLY`).
  When that becomes random, `verifyOTP` in the test needs another way to read the code.
- Dependency added: `github.com/testcontainers/testcontainers-go` +
  `modules/postgres` v0.44.0 (`go mod tidy` + `go mod vendor`; `vendor/` 60M → 67M). Side effect:
  indirect `github.com/klauspost/compress` bumped from v1.16.7 to v1.18.6.
