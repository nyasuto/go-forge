package credentials

import (
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestFileLock_AcquireRelease(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "refresh.lock")

	lock := NewFileLock(path)
	if err := lock.Acquire(); err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
}

func TestFileLock_ReAcquireAfterRelease(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "refresh.lock")

	lock1 := NewFileLock(path)
	if err := lock1.Acquire(); err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	if err := lock1.Release(); err != nil {
		t.Fatalf("first Release: %v", err)
	}

	lock2 := NewFileLock(path)
	if err := lock2.Acquire(); err != nil {
		t.Fatalf("second Acquire: %v", err)
	}
	if err := lock2.Release(); err != nil {
		t.Fatalf("second Release: %v", err)
	}
}

func TestFileLock_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "subdir", "refresh.lock")

	lock := NewFileLock(path)
	if err := lock.Acquire(); err != nil {
		t.Fatalf("Acquire with nested path: %v", err)
	}
	defer lock.Release()
}

func TestFileLock_Concurrent(t *testing.T) {
	// Two goroutines contending on the same lock path must serialize.
	dir := t.TempDir()
	path := filepath.Join(dir, "refresh.lock")

	var entered atomic.Int32
	var maxEntered atomic.Int32
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		lock := NewFileLock(path)
		if err := lock.Acquire(); err != nil {
			t.Errorf("Acquire: %v", err)
			return
		}
		defer lock.Release()

		n := entered.Add(1)
		if n > maxEntered.Load() {
			maxEntered.Store(n)
		}
		time.Sleep(20 * time.Millisecond)
		entered.Add(-1)
	}

	wg.Add(2)
	go worker()
	go worker()
	wg.Wait()

	if got := maxEntered.Load(); got != 1 {
		t.Errorf("max concurrent holders = %d, want 1", got)
	}
}

func TestFileLock_InvalidPath(t *testing.T) {
	// A path that cannot be created (e.g., under a non-existent device root)
	// should error out cleanly.
	lock := NewFileLock("/proc/invalid/cannot/create/here/lock")
	err := lock.Acquire()
	if err == nil {
		_ = lock.Release()
		t.Fatal("expected error for unwritable path, got nil")
	}
}

func TestDefaultLockPath(t *testing.T) {
	p, err := defaultLockPath()
	if err != nil {
		t.Fatalf("defaultLockPath: %v", err)
	}
	if p == "" {
		t.Error("defaultLockPath returned empty string")
	}
	if filepath.Base(p) != "refresh.lock" {
		t.Errorf("basename = %q, want refresh.lock", filepath.Base(p))
	}
}
