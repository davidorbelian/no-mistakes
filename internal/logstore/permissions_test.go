package logstore

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

// TestOpenKeepsTheLogPrivate proves daemon logs and their directory are
// owner-only. Logs carry agent output and command lines, so another local
// account must not be able to read them.
func TestOpenKeepsTheLogPrivate(t *testing.T) {
	skipWithoutPOSIXModes(t)

	dir := filepath.Join(t.TempDir(), "logs")
	path := filepath.Join(dir, "daemon.log")
	w, err := Open(path, Policy{MaxBytes: 4, Backups: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	if info, err := os.Stat(dir); err != nil {
		t.Fatal(err)
	} else if got := info.Mode().Perm(); got != paths.DirMode {
		t.Errorf("log directory mode = %o, want %o", got, paths.DirMode)
	}
	if info, err := os.Stat(path); err != nil {
		t.Fatal(err)
	} else if got := info.Mode().Perm(); got != paths.FileMode {
		t.Errorf("log mode = %o, want %o", got, paths.FileMode)
	}

	// Rotation backups carry the same output, so they carry the same mode.
	if _, err := w.Write([]byte("AAAABBBB")); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(path + ".1"); err != nil {
		t.Fatal(err)
	} else if got := info.Mode().Perm(); got != paths.FileMode {
		t.Errorf("backup mode = %o, want %o", got, paths.FileMode)
	}
}

// TestOpenTightensAWorldReadableLog covers the upgrade path: a log an older
// version created at 0644 must be tightened when the daemon next opens it,
// because the open mode only applies to a file being created.
func TestOpenTightensAWorldReadableLog(t *testing.T) {
	skipWithoutPOSIXModes(t)

	path := filepath.Join(t.TempDir(), "daemon.log")
	if err := os.WriteFile(path, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	w, err := Open(path, Policy{MaxBytes: 1 << 20, Backups: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != paths.FileMode {
		t.Errorf("mode = %o, want %o (an existing world-readable log must be tightened)", got, paths.FileMode)
	}
}
