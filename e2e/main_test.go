// Package e2e contains end-to-end tests for cplugins.
// Tests use a real 'claude' binary and a fake HOME directory.
// Auth is not required: only 'claude plugin' subcommands are used, which are fully local.
//
// To run:
//
//	make e2e-tests
//
// Or directly:
//
//	go test ./e2e/... -v -timeout 120s
package e2e_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

var (
	cpluginsBin string // path to the built cplugins binary
	claudeBin   string // path to the claude CLI
)

func TestMain(m *testing.M) {
	// Locate project root (parent of the e2e/ directory)
	root := projectRoot()

	// Build into a unique temp dir so concurrent `go test ./e2e/` runs each get
	// their own binary (TestMain removes it on exit).
	binDir, err := os.MkdirTemp("", "cplugins-e2e-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "tempdir: %s\n", err)
		os.Exit(1)
	}
	bin := filepath.Join(binDir, "cplugins")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %s\n%s\n", err, out)
		os.RemoveAll(binDir)
		os.Exit(1)
	}
	cpluginsBin = bin

	// Require claude in PATH
	path, err := exec.LookPath("claude")
	if err != nil {
		fmt.Fprintln(os.Stderr, "claude not found in PATH — skipping E2E tests")
		os.RemoveAll(binDir)
		os.Exit(0)
	}
	claudeBin = path

	code := m.Run()
	os.RemoveAll(binDir)
	os.Exit(code)
}

// projectRoot returns the absolute path to the cplugins project root.
func projectRoot() string {
	_, file, _, _ := runtime.Caller(0)
	// file is .../cplugins/e2e/main_test.go → parent of parent
	return filepath.Dir(filepath.Dir(file))
}
