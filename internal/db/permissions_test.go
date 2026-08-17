package db

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/paths"
)

func skipWithoutPOSIXModes(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes are not enforced on Windows")
	}
}

// TestOpenKeepsTheDatabasePrivate proves the state database and its WAL siblings
// are owner-only. They hold run intent, review findings, and PR URLs, so another
// local account must not be able to read them even if a copy leaves the
// owner-only app-state root.
func TestOpenKeepsTheDatabasePrivate(t *testing.T) {
	skipWithoutPOSIXModes(t)

	path := filepath.Join(t.TempDir(), "state.sqlite")
	database, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()

	// A write forces the WAL and shared-memory siblings into existence.
	if _, err := database.sql.Exec(`INSERT INTO repos (id, working_path, upstream_url, default_branch, created_at) VALUES ('r1', '/tmp/r1', 'https://example.com/r1.git', 'main', 1)`); err != nil {
		t.Fatalf("write: %v", err)
	}

	for _, suffix := range []string{"", "-wal", "-shm"} {
		info, err := os.Stat(path + suffix)
		if err != nil {
			// SQLite may not keep every sibling on every platform; only assert
			// the ones that exist.
			continue
		}
		if got := info.Mode().Perm(); got != paths.FileMode {
			t.Errorf("%q mode = %o, want %o", path+suffix, got, paths.FileMode)
		}
	}
}

// TestOpenTightensAWorldReadableDatabase covers the upgrade path: a database an
// older version created at 0644 must be tightened when it is next opened.
func TestOpenTightensAWorldReadableDatabase(t *testing.T) {
	skipWithoutPOSIXModes(t)

	path := filepath.Join(t.TempDir(), "state.sqlite")
	database, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != paths.FileMode {
		t.Errorf("mode = %o, want %o (an existing world-readable database must be tightened)", got, paths.FileMode)
	}
}
