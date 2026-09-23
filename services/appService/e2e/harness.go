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
	buildOnce sync.Once
	buildDir  string
	binPath   string
	buildErr  error
)

// cleanupBuild removes the shared build directory. TestMain calls it after
// every test in the package has run.
func cleanupBuild() {
	if buildDir != "" {
		_ = os.RemoveAll(buildDir)
	}
}

func binaryPath(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		// Not t.TempDir(): that directory is removed when the test that
		// created it finishes, and the binary is shared by every test in the
		// package. TestMain removes this one after the whole run.
		dir, err := os.MkdirTemp("", "appservice-e2e-")
		if err != nil {
			buildErr = fmt.Errorf("create build dir: %w", err)
			return
		}
		buildDir = dir
		binPath = filepath.Join(dir, "service")
		cmd := exec.Command("go", "build", "-o", binPath, ".")
		cmd.Dir = cmdDir
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			buildErr = fmt.Errorf("go build failed: %w\n%s", err, stderr.String())
		}
	})
	require.NoError(t, buildErr)
	return binPath
}

// --- Postgres container -------------------------------------------------

// startPostgres starts an ephemeral Postgres via testcontainers and applies
// every migration under migrationsGlob, sorted by filename (the numbered
// prefix is the ordering contract). Skips (not fails) the test when Docker
// isn't reachable, mirroring connectTestDB's behavior when no local
// Postgres answers.
func startPostgres(t *testing.T) (host, port string, cleanup func()) {
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

	return host, port, func() { _ = pgContainer.Terminate(ctx) }
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
	doc["server"] = server

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

func waitForReady(t *testing.T, baseURL string, timeout time.Duration) {
	t.Helper()
	addr := strings.TrimPrefix(baseURL, "http://")
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("server at %s did not become ready within %s", baseURL, timeout)
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
