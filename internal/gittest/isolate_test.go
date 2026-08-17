package gittest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsolateConfigHidesAmbientConfiguration(t *testing.T) {
	ambient := writeAmbientConfig(t)
	t.Setenv("GIT_CONFIG_GLOBAL", ambient)
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "safe.bareRepository")
	t.Setenv("GIT_CONFIG_VALUE_0", "explicit")

	restore := IsolateConfig()
	defer restore()

	repo := t.TempDir()
	gitRun(t, repo, "init")

	if out := gitOut(t, repo, "remote", "-v"); out != "" {
		t.Fatalf("isolated repo reports remotes %q, want none", out)
	}
	if out, err := gitTry(repo, "config", "--get", "remote.upstream.pushurl"); err == nil {
		t.Fatalf("ambient remote pushurl still readable as %q", out)
	}
	if out, _ := gitTry(repo, "config", "--get", "user.name"); strings.TrimSpace(out) == "Ambient" {
		t.Fatal("ambient user.name still reaches git")
	}
	if got := os.Getenv("GIT_CONFIG_COUNT"); got != "" {
		t.Fatalf("GIT_CONFIG_COUNT = %q, want it dropped", got)
	}
	if got := os.Getenv("GIT_CONFIG_NOSYSTEM"); got != "1" {
		t.Fatalf("GIT_CONFIG_NOSYSTEM = %q, want \"1\"", got)
	}
}

func TestIsolateConfigRestoresTheEnvironmentItFound(t *testing.T) {
	ambient := writeAmbientConfig(t)
	t.Setenv("GIT_CONFIG_GLOBAL", ambient)
	t.Setenv("GIT_CONFIG_COUNT", "1")
	os.Unsetenv("GIT_CONFIG_NOSYSTEM")

	restore := IsolateConfig()
	isolated := os.Getenv("GIT_CONFIG_GLOBAL")
	restore()

	if got := os.Getenv("GIT_CONFIG_GLOBAL"); got != ambient {
		t.Fatalf("GIT_CONFIG_GLOBAL = %q, want the ambient %q", got, ambient)
	}
	if got := os.Getenv("GIT_CONFIG_COUNT"); got != "1" {
		t.Fatalf("GIT_CONFIG_COUNT = %q, want \"1\"", got)
	}
	if _, ok := os.LookupEnv("GIT_CONFIG_NOSYSTEM"); ok {
		t.Fatal("GIT_CONFIG_NOSYSTEM was unset before isolation and must be unset again")
	}
	if _, err := os.Stat(isolated); !os.IsNotExist(err) {
		t.Fatalf("isolated config %q still exists after restore (err %v)", isolated, err)
	}
}

func writeAmbientConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "gitconfig")
	content := "[remote \"upstream\"]\n\tpushurl = DISABLED-ambient\n[user]\n\tname = Ambient\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write ambient config: %v", err)
	}
	return path
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	if out, err := gitTry(dir, args...); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
	}
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := gitTry(dir, args...)
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(out)
}

func gitTry(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}
