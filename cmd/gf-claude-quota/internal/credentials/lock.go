package credentials

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// FileLock is an advisory exclusive lock backed by syscall.Flock.
type FileLock struct {
	path string
	f    *os.File
}

// NewFileLock creates a FileLock bound to the given file path.
// The file and its parent directories are created on Acquire if missing.
func NewFileLock(path string) *FileLock {
	return &FileLock{path: path}
}

// Acquire takes an exclusive flock on the underlying file, blocking until
// any other holder releases. The file is created if it does not exist.
func (l *FileLock) Acquire() error {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o700); err != nil {
		return fmt.Errorf("creating lock dir: %w", err)
	}
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("opening lock file: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return fmt.Errorf("flock: %w", err)
	}
	l.f = f
	return nil
}

// Release unlocks and closes the lock file. Safe to call on an
// unacquired lock (returns nil).
func (l *FileLock) Release() error {
	if l.f == nil {
		return nil
	}
	_ = syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	err := l.f.Close()
	l.f = nil
	return err
}

// defaultLockPath returns the default refresh lock path under the user's
// cache dir: ~/Library/Caches/gf-claude-quota/refresh.lock on macOS,
// ~/.cache/gf-claude-quota/refresh.lock on Linux.
func defaultLockPath() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("locating user cache dir: %w", err)
	}
	return filepath.Join(cacheDir, "gf-claude-quota", "refresh.lock"), nil
}
