package main

import (
	"os/exec"
	"strings"
	"testing"
)

const selfUpdaterPackage = "github.com/kunchenguid/no-mistakes/internal/update"

// This fork is built and installed from source by the machine that declares it,
// so no shipped code path may check for or install a release. The package is
// kept in the tree to keep upstream merges trivial; this test is what makes it
// unreachable from the binary.
func TestCLIBinaryDoesNotLinkTheSelfUpdater(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", "./cmd/no-mistakes")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list -deps ./cmd/no-mistakes: %v\n%s", err, out)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == selfUpdaterPackage {
			t.Fatalf("cmd/no-mistakes depends on %s; this fork never checks for or installs releases", selfUpdaterPackage)
		}
	}
}
