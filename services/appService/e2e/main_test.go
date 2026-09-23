package e2e

import (
	"os"
	"testing"
)

// TestMain removes the binary shared by every test in this package once they
// have all run — see binaryPath.
func TestMain(m *testing.M) {
	code := m.Run()
	cleanupBuild()
	os.Exit(code)
}
