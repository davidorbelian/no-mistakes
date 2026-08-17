//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/paths"
)

// TestAppStateIsPrivateToItsOwner proves the state another local account must
// never read is unreachable after a real init: the app-state root is owner-only,
// and so are the database, the CLI log, and the gate config that keeps the
// credentialled upstream URL in plain text.
//
// The harness deliberately creates NM_HOME world-readable, the way an older
// version did, so this also covers the upgrade path.
func TestAppStateIsPrivateToItsOwner(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes are not enforced on Windows; the user profile directory excludes other accounts there")
	}

	h := NewHarness(t, SetupOpts{Agent: "claude", Scenario: cleanReviewScenario(t)})

	if err := os.Chmod(h.NMHome, 0o755); err != nil {
		t.Fatalf("loosen NM_HOME to model an older install: %v", err)
	}
	if out, err := h.Run("init"); err != nil {
		t.Fatalf("nm init: %v\n%s", err, out)
	}
	// The CLI log writer opens before init creates the logs directory, so the
	// log first appears on the next command.
	if out, err := h.Run("status"); err != nil {
		t.Fatalf("nm status: %v\n%s", err, out)
	}

	p := paths.WithRoot(h.NMHome)
	assertMode(t, h.NMHome, paths.DirMode)
	assertMode(t, p.LogsDir(), paths.DirMode)
	assertMode(t, p.DB(), paths.FileMode)
	// The harness writes the global config world-readable, the way an older
	// version did, so this also proves an existing config is tightened.
	assertMode(t, p.ConfigFile(), paths.FileMode)
	assertMode(t, p.CLILog(), paths.FileMode)

	gateConfig := filepath.Join(p.RepoDir(h.repoID()), "config")
	assertMode(t, gateConfig, paths.FileMode)
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Errorf("%s mode = %o, want %o", path, got, want)
	}
}
