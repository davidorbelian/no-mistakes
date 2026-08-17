package db

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/kunchenguid/no-mistakes/internal/paths"
	"github.com/oklog/ulid/v2"
	_ "modernc.org/sqlite"
)

var (
	entropyMu sync.Mutex
	entropy   = ulid.Monotonic(rand.Reader, 0)
)

// DB wraps a SQLite database connection.
type DB struct {
	sql *sql.DB
}

// Open opens (or creates) the SQLite database at path and runs migrations.
func Open(path string) (*DB, error) {
	// SQLite creates the database, and the WAL and shared-memory files beside
	// it, with the process umask, which usually leaves them readable by every
	// local account. Pre-creating the database owner-only makes the siblings
	// inherit that mode, and makePrivate below tightens a database written
	// before this behavior existed. Both are best effort: the app-state root is
	// owner-only on its own, so these modes are defense in depth and must never
	// fail an otherwise healthy open.
	precreatePrivate(path)
	sqlDB, err := sql.Open("sqlite", path+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if _, err := sqlDB.Exec(schemaSQL); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("migrate db: %w", err)
	}
	for _, stmt := range migrationStatements {
		if _, err := sqlDB.Exec(stmt); err != nil && !isDuplicateColumnErr(err) {
			sqlDB.Close()
			return nil, fmt.Errorf("migrate db: %w", err)
		}
	}
	makePrivate(path)
	return &DB{sql: sqlDB}, nil
}

// precreatePrivate creates an absent database file owner-only so SQLite's WAL
// and shared-memory siblings inherit the mode. A failure is ignored on purpose,
// leaving SQLite to report the real problem with the path.
func precreatePrivate(path string) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, paths.FileMode)
	if err != nil {
		return
	}
	_ = f.Close()
}

// makePrivate tightens the database and its siblings after migrations, which is
// what upgrades a database an older version left world-readable.
func makePrivate(path string) {
	for _, suffix := range []string{"", "-wal", "-shm"} {
		_ = paths.MakeFilePrivate(path + suffix)
	}
}

// OpenReadOnly opens an existing database without creating or migrating it.
// It is used by pre-mutation authorization, where even schema repair would be
// an unacceptable side effect before the caller is classified.
func OpenReadOnly(path string) (*DB, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	sqlDB, err := sql.Open("sqlite", "file:"+path+"?mode=ro&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open db read-only: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("open db read-only: %w", err)
	}
	return &DB{sql: sqlDB}, nil
}

// isDuplicateColumnErr reports whether err is SQLite's "duplicate column name"
// error, which ALTER TABLE ADD COLUMN emits when the column already exists.
// Treating this as a no-op keeps migrations idempotent without a version table.
func isDuplicateColumnErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "duplicate column name")
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.sql.Close()
}

// newID generates a new ULID with monotonic ordering.
func newID() string {
	entropyMu.Lock()
	defer entropyMu.Unlock()
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}

// now returns the current unix timestamp in seconds.
func now() int64 {
	return time.Now().Unix()
}
