package storage

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	pkgstorage "chjpcoj.com.innerlamp/pkg/storage"
	_ "chjpcoj.com.innerlamp/pkg/storage/gcs"
	"chjpcoj.com.innerlamp/services/appService/domain"
)

// startEmulator runs fake-gcs-server and returns "host:port". The test is
// skipped (not failed) under -short or when Docker is unreachable, like the
// repository tests do for Postgres.
func startEmulator(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping storage emulator test in -short mode")
	}
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
		t.Skipf("docker unavailable, skipping storage emulator test: %v", err)
	}
	t.Cleanup(func() { _ = c.Terminate(ctx) })

	host, err := c.Host(ctx)
	require.NoError(t, err)
	port, err := c.MappedPort(ctx, "4443/tcp")
	require.NoError(t, err)
	return host + ":" + port.Port()
}

// do sends one request exactly as a client holding a ticket would.
func do(t *testing.T, method, url string, headers map[string]string, body []byte) (int, []byte, http.Header) {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	require.NoError(t, err)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, b, resp.Header
}

// TestAvatarStorage_AgainstEmulator walks the adapter through a real GCS client
// and a real HTTP upload, the way the profile flow uses it: ticket → client
// upload → Stat → Copy → read URL → Delete.
//
// What it proves: the emulator-mode URLs and the method are ones a store
// accepts, the key layout round-trips, and Stat/Copy/Delete map their results
// onto the domain. What it cannot prove: anything about signing (TD-001).
func TestAvatarStorage_AgainstEmulator(t *testing.T) {
	host := startEmulator(t)
	t.Setenv("STORAGE_EMULATOR_HOST", host)
	ctx := context.Background()

	client, err := pkgstorage.New(&pkgstorage.Config{
		Backend: pkgstorage.BackendGCS,
		Timeout: 10 * time.Second,
		GCS:     &pkgstorage.GCSConfig{ProjectID: "test", UseDefaultCredentials: true},
	})
	require.NoError(t, err)
	require.NoError(t, client.Connect(ctx))
	t.Cleanup(func() { _ = client.Close() })
	const bucket = "avatars-test"
	require.NoError(t, client.CreateBucket(ctx, bucket, &pkgstorage.BucketCreateOptions{}))

	a := NewAvatarStorage(client, bucket, WithEmulator(host))
	png := []byte("\x89PNG not really, but bytes")
	tmp, final := "avatars/tmp/42/0b9c4f3e", "avatars/42/7f3a.png"

	// Ticket → upload exactly as told.
	up, err := a.SignUpload(ctx, tmp, "image/png", 5<<20, time.Minute)
	require.NoError(t, err)
	status, body, _ := do(t, up.Method, up.URL, up.Headers, png)
	require.Less(t, status, 300, string(body))

	meta, err := a.Stat(ctx, tmp)
	require.NoError(t, err)
	assert.Equal(t, &domain.ObjectMeta{ContentType: "image/png", Size: int64(len(png))}, meta)

	require.NoError(t, a.Copy(ctx, tmp, final))
	meta, err = a.Stat(ctx, final)
	require.NoError(t, err)
	assert.Equal(t, int64(len(png)), meta.Size)
	_, err = a.Stat(ctx, tmp)
	assert.NoError(t, err, "Copy leaves the source in place")

	readURL, err := a.SignRead(ctx, final, time.Minute)
	require.NoError(t, err)
	status, got, header := do(t, http.MethodGet, readURL, nil, nil)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, png, got)
	assert.Equal(t, "image/png", header.Get("Content-Type"))

	require.NoError(t, a.Delete(ctx, final))
	_, err = a.Stat(ctx, final)
	assert.ErrorIs(t, err, domain.ErrObjectNotFound)
	assert.NoError(t, a.Delete(ctx, final), "deleting what is already gone is not an error")
	assert.ErrorIs(t, a.Copy(ctx, "avatars/missing", "avatars/x"), domain.ErrObjectNotFound)
}
