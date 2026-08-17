package paths

import (
	"errors"
	"io/fs"
	"os"
	"runtime"
)

// DirMode and FileMode are the permissions for everything no-mistakes creates
// under its app-state root, and this package is their single owner.
//
// The root is owner-only because the state below it is sensitive on a machine
// with more than one local account: state.sqlite carries run intent, review
// findings, and PR URLs, the logs carry agent output, the worktrees carry the
// source under review, and each gate's .git/config carries the credentialled
// upstream URL in plain text. An owner-only root is the load-bearing control,
// since it makes the whole subtree unreachable for another account whatever the
// modes inside it are. The owner-only file mode on the sensitive files is cheap
// insurance for the case where a copy of one escapes the root.
//
// Files no-mistakes writes OUTSIDE the app-state root keep the ordinary
// 0o755/0o644: installed skills and repository files are meant to be readable,
// and the service unit has its own rule (see internal/daemon/service.go, which
// drops to 0o600 only when it carries proxy credentials).
const (
	DirMode  fs.FileMode = 0o700
	FileMode fs.FileMode = 0o600
)

// EnsurePrivateDir creates dir with its parents and makes an existing one
// private. The second step matters on upgrade: MkdirAll leaves the mode of a
// directory that already exists alone, so a root an older version created
// world-readable would stay that way forever.
func EnsurePrivateDir(dir string) error {
	if err := os.MkdirAll(dir, DirMode); err != nil {
		return err
	}
	return chmodIfEnforced(dir, DirMode)
}

// OpenPrivateAppend opens an app-state log for appending, creating it
// owner-only. It also tightens a file that already exists, because an open mode
// applies only to a file being created and a log written by an older version is
// still world-readable.
func OpenPrivateAppend(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, FileMode)
	if err != nil {
		return nil, err
	}
	if err := MakeFilePrivate(path); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

// MakeFilePrivate tightens an existing app-state file. A missing file is not an
// error: callers pass paths a component may not have created yet, such as a
// SQLite WAL sibling or a log that has never been written.
func MakeFilePrivate(path string) error {
	err := chmodIfEnforced(path, FileMode)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// chmodIfEnforced applies mode where POSIX modes mean something. On Windows they
// do not: os.Chmod there only toggles the read-only attribute, and the user
// profile directory that holds the app state already excludes other accounts
// through its ACL.
func chmodIfEnforced(path string, mode fs.FileMode) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	return os.Chmod(path, mode)
}
