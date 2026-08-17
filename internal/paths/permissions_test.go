package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// skipWithoutPOSIXModes keeps the mode assertions on the platforms that enforce
// them. Windows carries no POSIX mode bits, and the user profile directory
// already excludes other accounts there.
func skipWithoutPOSIXModes(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes are not enforced on Windows")
	}
}

// TestEnsureDirsCreatesOwnerOnlyAppState proves the app-state tree is private to
// its owner. The root holds the state database, the run logs, and every gate's
// .git/config with its credentialled upstream URL, so another local account must
// not be able to walk into it.
func TestEnsureDirsCreatesOwnerOnlyAppState(t *testing.T) {
	skipWithoutPOSIXModes(t)

	p := WithRoot(filepath.Join(t.TempDir(), "nm"))
	if err := p.EnsureDirs(); err != nil {
		t.Fatal(err)
	}

	for _, d := range []string{p.Root(), p.ReposDir(), p.WorktreesDir(), p.LogsDir(), p.ServerPIDsDir()} {
		info, err := os.Stat(d)
		if err != nil {
			t.Fatalf("stat %q: %v", d, err)
		}
		if got := info.Mode().Perm(); got != DirMode {
			t.Errorf("%q mode = %o, want %o", d, got, DirMode)
		}
	}
}

// TestEnsureDirsTightensAWorldReadableRoot covers the upgrade path: a root an
// older version created at 0755 must be tightened, because MkdirAll leaves the
// mode of an existing directory alone.
func TestEnsureDirsTightensAWorldReadableRoot(t *testing.T) {
	skipWithoutPOSIXModes(t)

	p := WithRoot(filepath.Join(t.TempDir(), "nm"))
	if err := os.MkdirAll(p.LogsDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p.Root(), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := p.EnsureDirs(); err != nil {
		t.Fatal(err)
	}

	for _, d := range []string{p.Root(), p.LogsDir()} {
		info, err := os.Stat(d)
		if err != nil {
			t.Fatalf("stat %q: %v", d, err)
		}
		if got := info.Mode().Perm(); got != DirMode {
			t.Errorf("%q mode = %o, want %o (an existing world-readable directory must be tightened)", d, got, DirMode)
		}
	}
}

// TestEnsurePrivateDirAndFile proves the shared helpers create and tighten with
// the app-state modes, so callers outside this package never spell the modes out
// themselves.
func TestEnsurePrivateDirAndFile(t *testing.T) {
	skipWithoutPOSIXModes(t)

	root := t.TempDir()
	dir := filepath.Join(root, "nested", "dir")
	if err := EnsurePrivateDir(dir); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != DirMode {
		t.Errorf("EnsurePrivateDir mode = %o, want %o", got, DirMode)
	}

	file := filepath.Join(dir, "state")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := MakeFilePrivate(file); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(file); err != nil {
		t.Fatal(err)
	} else if got := info.Mode().Perm(); got != FileMode {
		t.Errorf("MakeFilePrivate mode = %o, want %o", got, FileMode)
	}

	// A file that is not there is not a failure: callers tighten best effort on
	// paths a component may not have created yet (a WAL sibling, an old log).
	if err := MakeFilePrivate(filepath.Join(dir, "absent")); err != nil {
		t.Errorf("MakeFilePrivate on a missing file = %v, want nil", err)
	}
}

// TestOpenPrivateAppendCreatesAndTightens proves an app-state log is owner-only
// whether this version created it or an older one did.
func TestOpenPrivateAppendCreatesAndTightens(t *testing.T) {
	skipWithoutPOSIXModes(t)

	dir := t.TempDir()

	fresh := filepath.Join(dir, "fresh.log")
	f, err := OpenPrivateAppend(fresh)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if info, err := os.Stat(fresh); err != nil {
		t.Fatal(err)
	} else if got := info.Mode().Perm(); got != FileMode {
		t.Errorf("fresh log mode = %o, want %o", got, FileMode)
	}

	existing := filepath.Join(dir, "existing.log")
	if err := os.WriteFile(existing, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err = OpenPrivateAppend(existing)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if info, err := os.Stat(existing); err != nil {
		t.Fatal(err)
	} else if got := info.Mode().Perm(); got != FileMode {
		t.Errorf("existing log mode = %o, want %o", got, FileMode)
	}
}
