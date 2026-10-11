package e2e

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	code := m.Run()
	cleanupBuild()
	os.Exit(code)
}
