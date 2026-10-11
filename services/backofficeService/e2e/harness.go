// Package e2e is the black-box harness for backofficeService: build the real
// cmd/main.go binary, run it against an ephemeral Postgres (testcontainers) with
// every repo migration applied, and talk to it over plain HTTP. It mirrors
// services/appService/e2e, trimmed to what the backoffice needs.
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

const (
	cmdDir         = "../cmd"
	migrationsGlob = "../../../migrations/*.sql"
	baseConfigPath = "../config/config.dev.yaml"
)

var (
	buildDirOnce sync.Once
	buildDir     string
	buildOnce    sync.Once
	binPath      string
	buildErr     error
)

func requireE2E(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping E2E test in -short mode")
	}
}

func cleanupBuild() {
	if buildDir != "" {
		_ = os.RemoveAll(buildDir)
	}
}

func binaryPath(t *testing.T) string {
	t.Helper()
	buildDirOnce.Do(func() {
		dir, err := os.MkdirTemp("", "backoffice-e2e-")
		require.NoError(t, err)
		buildDir = dir
	})
	buildOnce.Do(func() {
		binPath = filepath.Join(buildDir, "service")
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

// startPostgres runs a throwaway Postgres with every migration applied.
func startPostgres(t *testing.T) (host, port string, cleanup func()) {
	t.Helper()
	ctx := context.Background()
	c, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("innerlamp"), postgres.WithUsername("postgres"), postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp").WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		t.Skipf("docker unavailable, skipping E2E: %v", err)
	}
	host, err = c.Host(ctx)
	require.NoError(t, err)
	mp, err := c.MappedPort(ctx, "5432")
	require.NoError(t, err)
	port = mp.Port()

	db := openDB(t, host, port)
	files, err := filepath.Glob(migrationsGlob)
	require.NoError(t, err)
	sort.Strings(files)
	for _, f := range files {
		b, err := os.ReadFile(f)
		require.NoError(t, err)
		_, err = db.Exec(ctx, string(b))
		require.NoErrorf(t, err, "applying %s", f)
	}
	_ = db.Close()
	return host, port, func() { _ = c.Terminate(ctx) }
}

func openDB(t *testing.T, host, port string) database.Database {
	t.Helper()
	n, err := strconv.Atoi(port)
	require.NoError(t, err)
	db, err := database.New(&database.Config{Type: database.PostgreSQL, Host: host, Port: n,
		Username: "postgres", Password: "postgres", Database: "innerlamp", SSLMode: "disable"})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	require.NoError(t, db.Connect(ctx))
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func exec1(t *testing.T, db database.Database, q string, args ...any) {
	t.Helper()
	_, err := db.Exec(context.Background(), q, args...)
	require.NoError(t, err, q)
}

func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer func() { _ = l.Close() }()
	_, p, err := net.SplitHostPort(l.Addr().String())
	require.NoError(t, err)
	return p
}

func writeTempConfig(t *testing.T, dbHost, dbPort, appPort string) string {
	t.Helper()
	raw, err := os.ReadFile(baseConfigPath)
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &doc))
	n, err := strconv.Atoi(dbPort)
	require.NoError(t, err)
	d := doc["database"].(map[string]any)
	d["host"], d["port"] = dbHost, n
	s := doc["server"].(map[string]any)
	s["addr"] = ":" + appPort
	// A fixed origin makes the image URL assertion independent of the Host header.
	s["public_base_url"] = "https://cdn.test"
	out, err := yaml.Marshal(doc)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "config.e2e.yaml")
	require.NoError(t, os.WriteFile(path, out, 0o600))
	return path
}

func startServer(t *testing.T, dbHost, dbPort string) (baseURL string, cleanup func()) {
	t.Helper()
	bin := binaryPath(t)
	port := freePort(t)
	cmd := exec.Command(bin)
	cmd.Env = append(os.Environ(), "CONFIG_PATH="+writeTempConfig(t, dbHost, dbPort, port))
	var out bytes.Buffer
	cmd.Stderr, cmd.Stdout = &out, &out
	require.NoError(t, cmd.Start())
	baseURL = "http://127.0.0.1:" + port
	kill := func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }

	addr := strings.TrimPrefix(baseURL, "http://")
	ready := false
	for deadline := time.Now().Add(30 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		if c, err := net.DialTimeout("tcp", addr, 200*time.Millisecond); err == nil {
			_ = c.Close()
			ready = true
			break
		}
	}
	if !ready {
		kill()
		t.Fatalf("server did not become ready; output:\n%s", out.String())
	}
	return baseURL, func() {
		kill()
		if t.Failed() {
			t.Logf("server output:\n%s", out.String())
		}
	}
}

// apiClient is a tiny JSON client over the shared response envelope.
type apiClient struct {
	base  string
	token string
	http  *http.Client
}

type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   struct {
		Code string `json:"code"`
	} `json:"error"`
	Details map[string]any `json:"details"`
}

func (c *apiClient) do(t *testing.T, method, path string, body any) (int, envelope) {
	t.Helper()
	var rd io.Reader
	if body != nil {
		switch b := body.(type) {
		case string:
			rd = strings.NewReader(b)
		default:
			j, err := json.Marshal(b)
			require.NoError(t, err)
			rd = bytes.NewReader(j)
		}
	}
	req, err := http.NewRequest(method, c.base+path, rd)
	require.NoError(t, err)
	if rd != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	cl := c.http
	if cl == nil {
		cl = &http.Client{Timeout: 15 * time.Second}
	}
	resp, err := cl.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var env envelope
	require.NoErrorf(t, json.Unmarshal(raw, &env), "body: %s", raw)
	return resp.StatusCode, env
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
