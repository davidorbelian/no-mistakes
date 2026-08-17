package gate

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/paths"
)

// TestInitKeepsTheGateConfigPrivate proves the one file that stores a plaintext
// credential is owner-only. The gate's own config keeps the full credentialled
// upstream URL on purpose (worktrees carved from it authenticate with it), so its
// mode is the last line of defense if a copy of it ever leaves the owner-only
// app-state root.
func TestInitKeepsTheGateConfigPrivate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file modes are not enforced on Windows")
	}

	workDir := setupTestRepo(t)
	const token = "ghp_secret_DO_NOT_LEAK"
	credURL := "https://x-access-token:" + token + "@127.0.0.1:1/o/r.git"
	if out, err := exec.Command("git", "-C", workDir, "remote", "set-url", "origin", credURL).CombinedOutput(); err != nil {
		t.Fatalf("set credentialled origin: %v: %s", err, out)
	}

	p := paths.WithRoot(t.TempDir())
	if err := p.EnsureDirs(); err != nil {
		t.Fatalf("ensure dirs: %v", err)
	}
	d := openTestDB(t, p)

	repo, _, err := Init(context.Background(), d, p, workDir)
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	configPath := filepath.Join(p.RepoDir(repo.ID), "config")
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read gate config: %v", err)
	}
	if !strings.Contains(string(raw), token) {
		t.Fatalf("gate config no longer holds the credential; this test guards the wrong file")
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != paths.FileMode {
		t.Errorf("gate config mode = %o, want %o", got, paths.FileMode)
	}
}
