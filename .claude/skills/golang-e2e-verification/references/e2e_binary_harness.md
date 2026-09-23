# E2E binary harness — design and reasoning

Location: `services/<service>/e2e/harness.go`, package `e2e`. One harness per service, shared by every E2E test file in that package. Scaffold from `assets/e2e_harness.go.template` the first time a service needs it; every later feature's E2E tests just call the existing harness.

## Why a compiled binary instead of `httptest.NewServer` over an in-process handler

The bug this skill exists to catch — a route mounted without its middleware, a handler built with the wrong dependency — lives in `cmd/main.go`'s `func main()`. Any harness that reconstructs the mux/middleware/handler wiring in test code is testing a *second, hand-maintained copy* of that wiring, which can drift from the real one silently (the exact failure mode this skill exists to prevent). Building the actual binary and running it as a subprocess means the test exercises byte-for-byte what `go build ./services/<service>/cmd` produces — there is no second copy to drift.

This costs more than `httptest`: process spawn/teardown, port allocation, a readiness poll. Accept that cost; it's what makes the test trustworthy instead of merely convenient.

## Harness responsibilities

### 1. Build the binary once per test run

```go
var (
	buildOnce sync.Once
	binPath   string
	buildErr  error
)

func binaryPath(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "appservice-e2e-") // NOT t.TempDir() — see below
		if err != nil {
			buildErr = err
			return
		}
		buildDir = dir // removed by TestMain after the whole run
		binPath = filepath.Join(dir, "appservice")
		cmd := exec.Command("go", "build", "-o", binPath, "./cmd")
		cmd.Dir = repoRootRelativeToService // e.g. "../.." from services/appService/e2e
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		buildErr = cmd.Run()
		if buildErr != nil {
			buildErr = fmt.Errorf("go build failed: %w\n%s", buildErr, stderr.String())
		}
	})
	require.NoError(t, buildErr)
	return binPath
}
```

`sync.Once` at package scope, not `t.TempDir()`-per-test — rebuilding the binary for every `t.Run` inside a table-driven test wastes real wall-clock time for no benefit; the binary doesn't change between subtests in the same `go test` invocation. **Do not build into `t.TempDir()`.** Go removes a `t.TempDir()` as soon as the test that created it finishes, not at the end of the run. With one E2E test function this goes unnoticed. With a second one, that function's `exec` fails with `fork/exec …/service: no such file or directory`. This was hit in `services/appService/e2e` on 2026-09-21, when `register_e2e_test.go` joined `login_e2e_test.go`. Build into `os.MkdirTemp` and remove it from a `TestMain` in a `_test.go` file of the package:

```go
func TestMain(m *testing.M) {
	code := m.Run()
	cleanupBuild() // os.RemoveAll(buildDir)
	os.Exit(code)
}
```

### 2. Start a Postgres container and apply every migration

```go
func startPostgres(t *testing.T) (host string, port string, cleanup func()) {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("innerlamp"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp").WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Skipf("docker unavailable, skipping E2E: %v", err)
	}

	host, err = pgContainer.Host(ctx)
	require.NoError(t, err)
	mappedPort, err := pgContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	applyAllMigrations(t, host, mappedPort.Port())

	return host, mappedPort.Port(), func() { _ = pgContainer.Terminate(ctx) }
}
```

**Skip, don't fail, when Docker is unavailable** — `postgres.Run` returning an error almost always means no Docker daemon, not a real test failure. Failing the suite on a machine with no Docker would make `go test ./...` (without `-short`) block anyone without Docker installed; skipping keeps the E2E package's absence-of-Docker behavior symmetric with the repository-layer tests' absence-of-Postgres behavior (`connectTestDB` in `services/<service>/repository/*_test.go` skips the same way).

**Apply every file under `migrations/*.sql`, sorted by filename**, not just one — the numbered filename prefix (`0001_`, `0002_`, ...) is the ordering contract already used by the migrations directory; glob and sort rather than hardcoding a single filename, so this harness doesn't need editing every time a new migration file is added.

```go
func applyAllMigrations(t *testing.T, host, port string) {
	t.Helper()
	db, err := database.New(&database.Config{
		Type: database.PostgreSQL, Host: host, Port: mustAtoi(t, port),
		Username: "postgres", Password: "postgres", Database: "innerlamp", SSLMode: "disable",
	})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	require.NoError(t, db.Connect(ctx))
	defer func() { _ = db.Close() }()

	files, err := filepath.Glob(migrationsGlobRelativeToService) // e.g. "../../../migrations/*.sql"
	require.NoError(t, err)
	sort.Strings(files)
	for _, f := range files {
		sqlBytes, err := os.ReadFile(f)
		require.NoErrorf(t, err, "reading %s", f)
		_, err = db.Exec(ctx, string(sqlBytes))
		require.NoErrorf(t, err, "applying %s", f)
	}
}
```

### 3. Generate a temp config pointing at the container, start the binary, poll for readiness

The service reads `CONFIG_PATH` (defaults to `services/<service>/config/config.dev.yaml` — see `cmd/main.go`). Don't edit that file; write a temp copy with `database.host`/`database.port` and `server.addr` overridden, and point `CONFIG_PATH` at the copy. Keep every other field (jwt, otp, sms, dev_secrets) identical to the real dev config, by loading it as YAML, mutating only those fields, and re-marshaling — do not hand-write a second config from scratch, or the E2E config silently drifts from what actually ships in dev.

```go
func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer l.Close()
	_, port, _ := net.SplitHostPort(l.Addr().String())
	return port
}

func startServer(t *testing.T, dbHost, dbPort string) (baseURL string, cleanup func()) {
	t.Helper()
	bin := binaryPath(t)
	appPort := freePort(t)
	cfgPath := writeTempConfig(t, dbHost, dbPort, appPort) // loads config.dev.yaml, overrides, writes to t.TempDir()

	cmd := exec.Command(bin)
	cmd.Env = append(os.Environ(), "CONFIG_PATH="+cfgPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	require.NoError(t, cmd.Start())

	baseURL = "http://127.0.0.1:" + appPort
	waitForReady(t, baseURL, 10*time.Second)

	return baseURL, func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		if t.Failed() {
			t.Logf("server stderr:\n%s", stderr.String())
		}
	}
}

func waitForReady(t *testing.T, baseURL string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", strings.TrimPrefix(baseURL, "http://"), 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("server at %s did not become ready within %s", baseURL, timeout)
}
```

`freePort` has a benign TCP race (port could theoretically be grabbed between `Listen`/`Close` and the server binding it) — acceptable here because it's test-local and retried implicitly by the whole test failing loudly rather than hanging; do not over-engineer this with a port-locking scheme unless it's observed to actually flake in this repo.

A plain TCP dial is enough for "ready" — this service has no `/healthz` yet. If one gets added later, switch `waitForReady` to poll it with a real GET instead of a bare dial.

### 4. Skip convention

```go
func requireE2E(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping E2E test in -short mode")
	}
}
```

Call `requireE2E(t)` first in every E2E test — same rationale as the repository-layer tests: `go test -short ./...` must stay green with nothing running.

## Composing it in a test

The template wraps `requireE2E` + `startPostgres` + `startServer` into a single `newE2EServer(t)` entry point (see `assets/e2e_harness.go.template`) so test files never call the pieces individually:

```go
func TestRegisterFlow_RequiresValidToken(t *testing.T) {
	baseURL, client := newE2EServer(t)

	resp, err := client.Get(baseURL + "/v1/users/me") // no Authorization header
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode) // fails today if the route isn't wrapped in Authenticate(...)
}
```

## Adjusting the template for a new service

`assets/e2e_harness.go.template` hardcodes relative paths (`./cmd`, `../../../migrations/*.sql`, `../config/config.dev.yaml`) assuming `services/<service>/e2e/harness.go`. When scaffolding for a service, update those three relative paths to match its actual directory depth — nothing else in the template should need to change.
