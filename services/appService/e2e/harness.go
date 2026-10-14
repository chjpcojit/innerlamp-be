// Package e2e provides the shared black-box HTTP test harness for this
// service: build the real cmd/main.go binary once, run it against a real
// ephemeral Postgres (testcontainers-go), and hit it over plain HTTP.
//
// Every test in this package proves behavior of the actual shipped binary —
// no route/middleware wiring is reconstructed here. See
// .claude/skills/golang-e2e-verification/references/e2e_binary_harness.md
// for the reasoning behind each piece below.
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gopkg.in/yaml.v3"

	"chjpcoj.com.innerlamp/pkg/database"
	_ "chjpcoj.com.innerlamp/pkg/database/postgresql"
)

// cmdDir is services/appService/cmd, relative to this package.
const cmdDir = "../cmd"

// migrationsGlob matches every repo migration, relative to this package.
const migrationsGlob = "../../../migrations/*.sql"

// deitySeedPath is the deity data cmd/seeddeities loads, relative to this package.
const deitySeedPath = "../../../assets/data/phat_botat.json"

// baseConfigPath is the dev config cmd/main.go loads by default — copied and
// overridden per test run rather than pointed at directly.
const baseConfigPath = "../config/config.dev.yaml"

// requireE2E skips the calling test under `go test -short`, matching the
// repository-layer integration tests' convention so `-short` stays green
// with nothing running.
func requireE2E(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping E2E test in -short mode")
	}
}

// --- binary build (once per test run) ---------------------------------

var (
	buildDirOnce sync.Once
	buildDir     string
	buildDirErr  error

	buildOnce sync.Once
	binPath   string
	buildErr  error

	seedSutrasOnce sync.Once
	seedSutrasPath string
	seedSutrasErr  error

	seederMu   sync.Mutex
	seederBins = map[string]string{}
)

// cleanupBuild removes the shared build directory. TestMain calls it after
// every test in the package has run.
func cleanupBuild() {
	if buildDir != "" {
		_ = os.RemoveAll(buildDir)
	}
	if emulatorContainer != nil {
		_ = emulatorContainer.Terminate(context.Background())
	}
}

// sharedBuildDir creates, once, the directory every built binary goes into. Not
// t.TempDir(): that directory is removed when the test that created it finishes,
// and the binaries are shared by every test in the package. TestMain removes this
// one after the whole run.
func sharedBuildDir(t *testing.T) string {
	t.Helper()
	buildDirOnce.Do(func() {
		dir, err := os.MkdirTemp("", "appservice-e2e-")
		if err != nil {
			buildDirErr = fmt.Errorf("create build dir: %w", err)
			return
		}
		buildDir = dir
	})
	require.NoError(t, buildDirErr)
	return buildDir
}

// goBuild compiles the main package in pkgDir (relative to this package) to out.
func goBuild(pkgDir, out string) error {
	cmd := exec.Command("go", "build", "-o", out, ".")
	cmd.Dir = pkgDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build %s failed: %w\n%s", pkgDir, err, stderr.String())
	}
	return nil
}

func binaryPath(t *testing.T) string {
	t.Helper()
	dir := sharedBuildDir(t)
	buildOnce.Do(func() {
		binPath = filepath.Join(dir, "service")
		buildErr = goBuild(cmdDir, binPath)
	})
	require.NoError(t, buildErr)
	return binPath
}

// seederBinaryPath builds cmd/<name> once per test run, like binaryPath does for the
// service. It is for the seed commands other than seedsutras/seedadminunits, which
// keep their own helpers.
func seederBinaryPath(t *testing.T, name string) string {
	t.Helper()
	dir := sharedBuildDir(t)
	seederMu.Lock()
	defer seederMu.Unlock()
	if p, ok := seederBins[name]; ok {
		return p
	}
	out := filepath.Join(dir, name)
	require.NoError(t, goBuild(filepath.Join(cmdDir, name), out))
	seederBins[name] = out
	return out
}

// seedContent loads the content the service reads (articles, dharma talks, the sutra
// library) the way a deployment does: by running the seed commands. appService seeds
// nothing on start, so a test that reads this content must call it before starting the
// server. The deities must already be there (startPostgres does that).
func seedContent(t *testing.T, host, port string) {
	t.Helper()
	cfg := writeTempConfig(t, host, port, "0", nil)
	for _, bin := range []string{
		seederBinaryPath(t, "seedarticles"),
		seederBinaryPath(t, "seeddharmatalks"),
		seedSutrasBinaryPath(t),
	} {
		res := runUntilExit(t, bin, cfg, 3*time.Minute)
		require.False(t, res.timedOut, "%s timed out:\n%s", filepath.Base(bin), res.stderr)
		require.Equal(t, 0, res.exitCode, "%s failed:\n%s", filepath.Base(bin), res.stderr)
	}
}

// seedSutrasBinaryPath builds cmd/seedsutras once, like binaryPath does for the service.
func seedSutrasBinaryPath(t *testing.T) string {
	t.Helper()
	dir := sharedBuildDir(t)
	seedSutrasOnce.Do(func() {
		seedSutrasPath = filepath.Join(dir, "seedsutras")
		seedSutrasErr = goBuild(filepath.Join(cmdDir, "seedsutras"), seedSutrasPath)
	})
	require.NoError(t, seedSutrasErr)
	return seedSutrasPath
}

// startPostgres starts an ephemeral Postgres via testcontainers and applies
// every migration under migrationsGlob, sorted by filename (the numbered
// prefix is the ordering contract). Skips (not fails) the test when Docker
// isn't reachable, mirroring connectTestDB's behavior when no local
// Postgres answers.
func startPostgres(t *testing.T) (host, port string, cleanup func()) {
	t.Helper()
	return startPostgresSeeded(t, true)
}

// startPostgresWithoutDeities is startPostgres minus the deities seed, for the tests
// that prove the service refuses its first start without them.
func startPostgresWithoutDeities(t *testing.T) (host, port string, cleanup func()) {
	t.Helper()
	return startPostgresSeeded(t, false)
}

func startPostgresSeeded(t *testing.T, withDeities bool) (host, port string, cleanup func()) {
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
	port = mappedPort.Port()

	applyAllMigrations(t, host, port)
	if withDeities {
		seedDeities(t, host, port)
	}

	return host, port, func() { _ = pgContainer.Terminate(ctx) }
}

// seedDeities loads the Phật / Bồ tát catalogue the way a real deployment does
// before the first start (cmd/seeddeities): the service's first-run sutra seed links
// every sutra to deities and refuses to start without them. It inserts the columns the
// links and cards need (id, type, order, name, thumbnail), not the full deity record.
func seedDeities(t *testing.T, host, port string) {
	t.Helper()
	raw, err := os.ReadFile(deitySeedPath)
	require.NoError(t, err)
	var deities []struct {
		ID           string `json:"id"`
		Type         string `json:"type"`
		NameVI       string `json:"name_vi"`
		ThumbnailURL string `json:"thumbnail_url"`
	}
	require.NoError(t, json.Unmarshal(raw, &deities))
	require.NotEmpty(t, deities, "no deities in %s", deitySeedPath)

	portNum, err := strconv.Atoi(port)
	require.NoError(t, err)
	db, err := database.New(&database.Config{
		Type: database.PostgreSQL, Host: host, Port: portNum,
		Username: "postgres", Password: "postgres", Database: "innerlamp", SSLMode: "disable",
	})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	require.NoError(t, db.Connect(ctx))
	defer func() { _ = db.Close() }()

	for i, d := range deities {
		_, err := db.Exec(ctx,
			`INSERT INTO deities (id, type, sort_order, name_vi, thumbnail_url) VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (id) DO NOTHING`, d.ID, d.Type, i, d.NameVI, d.ThumbnailURL)
		require.NoErrorf(t, err, "seeding deity %s", d.ID)
	}
}

func applyAllMigrations(t *testing.T, host, port string) {
	t.Helper()
	portNum, err := strconv.Atoi(port)
	require.NoError(t, err)

	db, err := database.New(&database.Config{
		Type:     database.PostgreSQL,
		Host:     host,
		Port:     portNum,
		Username: "postgres",
		Password: "postgres",
		Database: "innerlamp",
		SSLMode:  "disable",
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	require.NoError(t, db.Connect(ctx))
	defer func() { _ = db.Close() }()

	files, err := filepath.Glob(migrationsGlob)
	require.NoError(t, err)
	sort.Strings(files)
	for _, f := range files {
		sqlBytes, err := os.ReadFile(f)
		require.NoErrorf(t, err, "reading %s", f)
		_, err = db.Exec(ctx, string(sqlBytes))
		require.NoErrorf(t, err, "applying %s", f)
	}
}

// --- server process -------------------------------------------------

func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer func() { _ = l.Close() }()
	_, port, err := net.SplitHostPort(l.Addr().String())
	require.NoError(t, err)
	return port
}

// configOverrides replaces individual keys of top-level config blocks, e.g.
// {"otp": {"ttl": "2s"}}. Only a test that must observe time passing (an
// expiry) should use it; everything else runs on the dev config as shipped.
type configOverrides map[string]map[string]any

// writeTempConfig loads baseConfigPath, overrides database.host/port,
// server.addr and any explicit overrides, and writes the result to a temp
// file — every other field (jwt, otp, sms, dev_secrets) stays identical to
// the real dev config so the E2E run never silently drifts from it.
func writeTempConfig(t *testing.T, dbHost, dbPort, appPort string, overrides configOverrides) string {
	t.Helper()

	raw, err := os.ReadFile(baseConfigPath)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &doc))

	dbPortNum, err := strconv.Atoi(dbPort)
	require.NoError(t, err)

	database, _ := doc["database"].(map[string]any)
	database["host"] = dbHost
	database["port"] = dbPortNum
	doc["database"] = database

	server, _ := doc["server"].(map[string]any)
	server["addr"] = ":" + appPort
	// The dev config's static_images_dir is relative to the repo root, but the binary
	// starts elsewhere: an absolute path keeps the /static/images route mounted, and
	// exercised, under test (a configured-but-missing folder stops the service).
	imagesDir, err := filepath.Abs("../../../assets/images")
	require.NoError(t, err)
	server["static_images_dir"] = imagesDir
	doc["server"] = server

	// config.dev.yaml points storage at the compose emulator on localhost:4443. A test must
	// never touch whatever dev data lives there, so every server gets the shared container.
	if storage, ok := doc["storage"].(map[string]any); ok {
		storage["emulator_host"] = sharedStorageEmulator(t)
	}

	// config.dev.yaml counts rate limits in the compose Redis. Each test server gets its own
	// in-memory counters instead: tests must not depend on, or leave counts in, dev Redis,
	// and two servers sharing one Redis would spend each other's budgets.
	if rls, ok := doc["rate_limit_store"].(map[string]any); ok {
		rls["backend"] = "memory"
	}

	for block, keys := range overrides {
		section, ok := doc[block].(map[string]any)
		require.Truef(t, ok, "config has no %q block to override", block)
		for key, value := range keys {
			_, exists := section[key]
			require.Truef(t, exists, "config block %q has no key %q to override", block, key)
			section[key] = value
		}
	}

	out, err := yaml.Marshal(doc)
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "config.e2e.yaml")
	require.NoError(t, os.WriteFile(path, out, 0o600))
	return path
}

// waitForReady polls until baseURL accepts a TCP connection and reports whether it did
// within timeout. The caller decides how to fail, so it can attach the server's stderr.
func waitForReady(t *testing.T, baseURL string, timeout time.Duration) bool {
	t.Helper()
	addr := strings.TrimPrefix(baseURL, "http://")
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// startServer builds (if needed) and runs the service binary against the
// given Postgres, returning the base URL to hit and a cleanup func that
// kills the process and (on test failure) logs its stderr.
func startServer(t *testing.T, dbHost, dbPort string, overrides configOverrides) (baseURL string, cleanup func()) {
	t.Helper()
	bin := binaryPath(t)
	appPort := freePort(t)
	cfgPath := writeTempConfig(t, dbHost, dbPort, appPort, overrides)

	cmd := exec.Command(bin)
	cmd.Env = append(os.Environ(), "CONFIG_PATH="+cfgPath)
	// The service logs JSON to stdout; keep it with stderr so a startup failure
	// carries its own reason.
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stderr
	require.NoError(t, cmd.Start())

	baseURL = "http://127.0.0.1:" + appPort
	kill := func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
	// 30s, not 10: with storage.emulator_host pointed at a port nothing listens on,
	// startup spends its own 10s connect budget retrying the bucket check before it
	// serves, so a 10s wait raced it and failed every E2E test.
	if !waitForReady(t, baseURL, 30*time.Second) {
		// A binary that exits at startup (bad config, missing seed) would otherwise
		// surface only as a timeout; its stderr is the actual reason.
		kill()
		t.Fatalf("server at %s did not become ready within 30s; server stderr:\n%s", baseURL, stderr.String())
	}

	return baseURL, func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		if t.Failed() {
			t.Logf("server stderr:\n%s", stderr.String())
		}
	}
}

// newE2EServer is the single entry point E2E test files should call: it
// starts Postgres, applies migrations, starts the real binary against it,
// and returns an *http.Client-ready base URL plus one combined cleanup.
func newE2EServer(t *testing.T) (baseURL string, client *http.Client) {
	t.Helper()
	return newE2EServerWithConfig(t, nil)
}

// newE2EServerWithConfig is newE2EServer with some config keys replaced —
// see configOverrides. Unknown blocks/keys fail the test rather than being
// silently added, so a renamed config field cannot turn an override into a
// no-op.
func newE2EServerWithConfig(t *testing.T, overrides configOverrides) (baseURL string, client *http.Client) {
	t.Helper()
	requireE2E(t)

	dbHost, dbPort, cleanupDB := startPostgres(t)
	t.Cleanup(cleanupDB)

	url, cleanupSrv := startServer(t, dbHost, dbPort, overrides)
	t.Cleanup(cleanupSrv)

	return url, &http.Client{Timeout: 5 * time.Second}
}

// newE2EServerSeeded is newE2EServer for the tests that read articles, dharma talks or
// sutras: the content is loaded by the seed commands before the service starts, because
// the service itself seeds nothing.
func newE2EServerSeeded(t *testing.T) (baseURL string, client *http.Client) {
	t.Helper()
	requireE2E(t)

	dbHost, dbPort, cleanupDB := startPostgres(t)
	t.Cleanup(cleanupDB)
	seedContent(t, dbHost, dbPort)

	url, cleanupSrv := startServer(t, dbHost, dbPort, nil)
	t.Cleanup(cleanupSrv)

	return url, &http.Client{Timeout: 5 * time.Second}
}

// openDB opens a connection to the E2E Postgres so a test can assert on, or change,
// stored state the API does not expose. It is closed when the test ends.
func openDB(t *testing.T, host, port string) database.Database {
	t.Helper()
	portNum, err := strconv.Atoi(port)
	require.NoError(t, err)
	db, err := database.New(&database.Config{
		Type: database.PostgreSQL, Host: host, Port: portNum,
		Username: "postgres", Password: "postgres", Database: "innerlamp", SSLMode: "disable",
	})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	require.NoError(t, db.Connect(ctx))
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// countRows returns count(*) of a table.
func countRows(t *testing.T, db database.Database, table string) int {
	t.Helper()
	var n int
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(t, db.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&n))
	return n
}

// execSQL runs a statement against the E2E Postgres.
func execSQL(t *testing.T, db database.Database, query string, args ...any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := db.Exec(ctx, query, args...)
	require.NoError(t, err)
}

// runResult is what a process that was expected to stop on its own left behind.
type runResult struct {
	stderr   string
	exitCode int
	timedOut bool
}

// runUntilExit runs bin with CONFIG_PATH=cfgPath and waits for it to exit, killing it
// after timeout. It is for the commands and startup failures that end by themselves:
// a service that is meant to keep running is startServer's job, not this one's.
func runUntilExit(t *testing.T, bin, cfgPath string, timeout time.Duration) runResult {
	t.Helper()
	cmd := exec.Command(bin)
	cmd.Env = append(os.Environ(), "CONFIG_PATH="+cfgPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	require.NoError(t, cmd.Start())

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		code := 0
		if err != nil {
			exitErr, ok := err.(*exec.ExitError)
			require.Truef(t, ok, "waiting for %s: %v", bin, err)
			code = exitErr.ExitCode()
		}
		return runResult{stderr: stderr.String(), exitCode: code}
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		<-done
		return runResult{stderr: stderr.String(), exitCode: -1, timedOut: true}
	}
}

// sharedStorageEmulator starts, once per test process, a fake-gcs-server container (the
// image docker-compose.yml uses) and returns "host:port". Every server the harness starts
// points at it, so no test ever touches a dev emulator running on localhost:4443. Each test
// that stores objects picks its own bucket (profile.bucket) so they cannot see each other's.
//
// It does not sign or verify URLs, so a green run says nothing about signing (TD-001).
// Skips the calling test when Docker is unreachable, like startPostgres.
func sharedStorageEmulator(t *testing.T) string {
	t.Helper()
	emulatorOnce.Do(func() {
		ctx := context.Background()
		c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:        "fsouza/fake-gcs-server:1.56.1",
				ExposedPorts: []string{"4443/tcp"},
				Cmd:          []string{"-scheme", "http", "-port", "4443", "-backend", "memory"},
				WaitingFor:   wait.ForListeningPort("4443/tcp").WithStartupTimeout(30 * time.Second),
			},
			Started: true,
		})
		if err != nil {
			emulatorErr = err
			return
		}
		hostPort, err := c.PortEndpoint(ctx, "4443/tcp", "")
		if err != nil {
			emulatorErr = err
			_ = c.Terminate(ctx)
			return
		}
		emulatorHostPort = hostPort
		emulatorContainer = c
	})
	if emulatorErr != nil {
		t.Skipf("docker unavailable, skipping E2E: %v", emulatorErr)
	}
	return emulatorHostPort
}

var (
	emulatorOnce      sync.Once
	emulatorHostPort  string
	emulatorContainer testcontainers.Container
	emulatorErr       error
)
