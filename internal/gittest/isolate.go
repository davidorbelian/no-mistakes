// Package gittest isolates tests that shell out to git from the configuration
// of the machine they run on.
package gittest

import (
	"os"
	"path/filepath"
)

// isolatedEnv lists every variable IsolateConfig owns, so restore puts back
// exactly what it found, including "was not set".
var isolatedEnv = []string{"GIT_CONFIG_GLOBAL", "GIT_CONFIG_NOSYSTEM", "GIT_CONFIG_COUNT"}

// IsolateConfig points git at an empty global configuration, disables system
// configuration, and drops injected GIT_CONFIG_* entries for the whole test
// binary. Call it first in TestMain and call the returned restore before
// os.Exit.
//
// Ambient configuration otherwise reaches every git call a test makes, and
// `remote.<name>.*` is the sharp edge: such a key in global config applies to
// every repository, so git reports a remote the test never created, and a
// global `pushurl` silently replaces the URL the test wants to push to. Agent
// harnesses separately inject safe.bareRepository=explicit through
// GIT_CONFIG_COUNT/KEY_n/VALUE_n (issue #362); a test that needs it re-sets it
// with t.Setenv.
func IsolateConfig() (restore func()) {
	previous := make(map[string]*string, len(isolatedEnv))
	for _, key := range isolatedEnv {
		if value, ok := os.LookupEnv(key); ok {
			previous[key] = &value
		} else {
			previous[key] = nil
		}
	}

	dir, err := os.MkdirTemp("", "no-mistakes-gitconfig-")
	if err != nil {
		panic(err)
	}
	mustSet("GIT_CONFIG_GLOBAL", filepath.Join(dir, "gitconfig"))
	mustSet("GIT_CONFIG_NOSYSTEM", "1")
	mustUnset("GIT_CONFIG_COUNT")

	return func() {
		for key, value := range previous {
			if value == nil {
				mustUnset(key)
				continue
			}
			mustSet(key, *value)
		}
		_ = os.RemoveAll(dir)
	}
}

func mustSet(key, value string) {
	if err := os.Setenv(key, value); err != nil {
		panic(err)
	}
}

func mustUnset(key string) {
	if err := os.Unsetenv(key); err != nil {
		panic(err)
	}
}
