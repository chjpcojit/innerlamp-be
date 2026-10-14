package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReport(t *testing.T) {
	assert.Contains(t, report(0, ""), "already seeded")
	assert.Equal(t, "seeded 5 dharma talks from the embedded data", report(5, ""))
	assert.Equal(t, "seeded 5 dharma talks from talks.json", report(5, "talks.json"))
}

func TestSource_EmbeddedWhenNoFile(t *testing.T) {
	got, err := source("")
	require.NoError(t, err)
	assert.NotEmpty(t, got, "the embedded dharma_talks.json")
}

func TestSource_ReadsTheFileAndReportsAMissingOne(t *testing.T) {
	path := filepath.Join(t.TempDir(), "talks.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"talks":[]}`), 0o600))

	got, err := source(path)
	require.NoError(t, err)
	assert.JSONEq(t, `{"talks":[]}`, string(got))

	_, err = source(filepath.Join(t.TempDir(), "missing.json"))
	assert.Error(t, err)
}
