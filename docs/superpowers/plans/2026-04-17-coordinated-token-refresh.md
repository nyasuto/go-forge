# Coordinated OAuth Token Refresh Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `gf-claude-quota` survive >24 h of idle use by refreshing the OAuth token automatically — but only when no `claude` process is running, so Claude Code's in-memory refresh token is never invalidated.

**Architecture:** Introduce two new internal files (`process.go`, `lock.go`) and rewrite `GetToken()` in `credentials.go` to orchestrate: (1) check expiry, (2) if expired, probe for running `claude` process via `pgrep`, (3) if not running, take an exclusive file lock, re-read the Keychain inside the lock, then refresh via the existing `TokenRefresher` and persist. All refactoring done via dependency injection so the new flow is table-driven testable.

**Tech Stack:** Go standard library only (`os/exec`, `syscall`, `path/filepath`). Existing modules: `internal/credentials/{keychain.go,store.go,refresh.go,provider_*.go}`. Spec: `docs/superpowers/specs/2026-04-17-coordinated-refresh-design.md`.

---

## File Structure

| File | Responsibility | Status |
|------|----------------|--------|
| `cmd/gf-claude-quota/internal/credentials/process.go` | `IsClaudeRunning()` — `pgrep` probe, fails safe to `true` | Create |
| `cmd/gf-claude-quota/internal/credentials/process_test.go` | Table-driven tests with stubbed `CommandRunner` | Create |
| `cmd/gf-claude-quota/internal/credentials/lock.go` | `FileLock` via `syscall.Flock` | Create |
| `cmd/gf-claude-quota/internal/credentials/lock_test.go` | Concurrency & path resolution tests | Create |
| `cmd/gf-claude-quota/internal/credentials/credentials.go` | Rewrite `GetToken()`, add `ErrClaudeRunning`, add testable `getTokenWithDeps` | Modify |
| `cmd/gf-claude-quota/internal/credentials/credentials_test.go` | Add orchestration tests with stubs | Modify |

---

## Task 1: Process Detection (`process.go`)

**Files:**
- Create: `cmd/gf-claude-quota/internal/credentials/process.go`
- Create: `cmd/gf-claude-quota/internal/credentials/process_test.go`

**Behavior:**

- `IsClaudeRunning()` returns `true` iff at least one process has `claude` as its binary basename (excludes `gf-claude-quota`, `claude-code-proxy`, etc.).
- Uses `pgrep -af claude` (substring-matches command lines containing "claude") then post-filters in Go to check the binary basename. This is more portable than relying on specific pgrep regex dialects.
- Fail-safe: on any unexpected `pgrep` error (binary missing, permission denied, unknown exit code), return `true`. Rationale: we'd rather skip a refresh opportunity than risk breaking Claude Code.

### Steps

- [ ] **Step 1: Write the failing tests**

Write `cmd/gf-claude-quota/internal/credentials/process_test.go` with the full content below:

```go
package credentials

import (
	"errors"
	"testing"
)

// stubExitErr satisfies the exitCoder interface used by isClaudeRunningWithRunner.
type stubExitErr struct{ code int }

func (s *stubExitErr) Error() string { return "exit" }
func (s *stubExitErr) ExitCode() int { return s.code }

func TestIsClaudeRunningWithRunner(t *testing.T) {
	tests := []struct {
		name    string
		runner  CommandRunner
		want    bool
	}{
		{
			name: "single bare claude match",
			runner: func(name string, args ...string) ([]byte, error) {
				return []byte("2062 claude --dangerously-skip-permissions\n"), nil
			},
			want: true,
		},
		{
			name: "absolute path claude match",
			runner: func(name string, args ...string) ([]byte, error) {
				return []byte("3001 /usr/local/bin/claude chat\n"), nil
			},
			want: true,
		},
		{
			name: "multiple matches include at least one real claude",
			runner: func(name string, args ...string) ([]byte, error) {
				return []byte("100 gf-claude-quota\n200 /usr/local/bin/claude\n"), nil
			},
			want: true,
		},
		{
			name: "exit code 1 means no matches at all",
			runner: func(name string, args ...string) ([]byte, error) {
				return nil, &stubExitErr{code: 1}
			},
			want: false,
		},
		{
			name: "only non-claude binaries match substring",
			runner: func(name string, args ...string) ([]byte, error) {
				return []byte("100 gf-claude-quota\n200 claude-code-proxy\n"), nil
			},
			want: false,
		},
		{
			name: "pgrep binary missing — fail-safe to running",
			runner: func(name string, args ...string) ([]byte, error) {
				return nil, errors.New("exec: \"pgrep\": executable file not found")
			},
			want: true,
		},
		{
			name: "pgrep unexpected exit code — fail-safe to running",
			runner: func(name string, args ...string) ([]byte, error) {
				return nil, &stubExitErr{code: 2}
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isClaudeRunningWithRunner(tt.runner)
			if got != tt.want {
				t.Errorf("isClaudeRunningWithRunner() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchesClaudeBinary(t *testing.T) {
	tests := []struct {
		name    string
		cmdline string
		want    bool
	}{
		{"bare claude with args", "claude --dangerously-skip-permissions", true},
		{"absolute path", "/usr/local/bin/claude chat", true},
		{"just claude", "claude", true},
		{"gf-claude-quota excluded", "gf-claude-quota", false},
		{"claude-code-proxy excluded", "/opt/claude-code-proxy serve", false},
		{"empty", "", false},
		{"random process", "node /some/script.js", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesClaudeBinary(tt.cmdline)
			if got != tt.want {
				t.Errorf("matchesClaudeBinary(%q) = %v, want %v", tt.cmdline, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd cmd/gf-claude-quota && go test ./internal/credentials/ -run 'TestIsClaudeRunning|TestMatchesClaude' -v`
Expected: **FAIL** — `undefined: isClaudeRunningWithRunner`, `undefined: matchesClaudeBinary`

- [ ] **Step 3: Write the implementation**

Create `cmd/gf-claude-quota/internal/credentials/process.go` with this full content:

```go
package credentials

import (
	"strings"
)

// exitCoder is the interface implemented by *exec.ExitError (and test stubs)
// that exposes the process exit code.
type exitCoder interface {
	ExitCode() int
}

// IsClaudeRunning reports whether any `claude` binary is currently running.
// Fails safe to true on any probe error, to avoid accidentally refreshing the
// OAuth token while Claude Code holds a live copy of the refresh_token in
// memory (which would force the user to re-login).
func IsClaudeRunning() bool {
	return isClaudeRunningWithRunner(DefaultCommandRunner)
}

func isClaudeRunningWithRunner(runner CommandRunner) bool {
	output, err := runner("pgrep", "-af", "claude")
	if err != nil {
		if ec, ok := err.(exitCoder); ok && ec.ExitCode() == 1 {
			return false // pgrep exit 1 = no match
		}
		return true // fail-safe for missing binary / perm errors / etc.
	}

	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		// Each line: "<pid> <cmdline>"
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		if matchesClaudeBinary(parts[1]) {
			return true
		}
	}
	return false
}

// matchesClaudeBinary reports whether the first token of cmdline is a path
// whose basename equals "claude".
func matchesClaudeBinary(cmdline string) bool {
	cmdline = strings.TrimSpace(cmdline)
	if cmdline == "" {
		return false
	}
	first := cmdline
	if idx := strings.Index(cmdline, " "); idx >= 0 {
		first = cmdline[:idx]
	}
	base := first
	if idx := strings.LastIndex(first, "/"); idx >= 0 {
		base = first[idx+1:]
	}
	return base == "claude"
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd cmd/gf-claude-quota && go test ./internal/credentials/ -run 'TestIsClaudeRunning|TestMatchesClaude' -v`
Expected: **PASS** — all sub-tests green.

- [ ] **Step 5: Run the full credentials test suite (should still pass)**

Run: `cd cmd/gf-claude-quota && go test ./internal/credentials/ -v`
Expected: **PASS** — all new + existing tests green.

- [ ] **Step 6: Commit**

```bash
cd /Users/yast/git/go-forge
git add cmd/gf-claude-quota/internal/credentials/process.go cmd/gf-claude-quota/internal/credentials/process_test.go
git commit -m "$(cat <<'EOF'
feat(gf-claude-quota): Claude Codeプロセス検出を追加

pgrepで`claude`バイナリ（basename完全一致）の稼働を検出する
IsClaudeRunning()を追加。gf-claude-quotaやclaude-code-proxy等は
basename不一致により自動除外。

pgrep不在/権限エラー時はfail-safeでtrueを返し、refreshによる
Claude Code再ログインを未然に防ぐ。

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: File Lock (`lock.go`)

**Files:**
- Create: `cmd/gf-claude-quota/internal/credentials/lock.go`
- Create: `cmd/gf-claude-quota/internal/credentials/lock_test.go`

**Behavior:**

- `NewFileLock(path string) *FileLock` — constructor.
- `(*FileLock).Acquire() error` — opens/creates the lock file and takes an exclusive `flock`. Blocking (waits if another holder exists). Creates parent directories as needed.
- `(*FileLock).Release() error` — unlocks and closes the file descriptor.
- `defaultLockPath() (string, error)` — returns `os.UserCacheDir() + "/gf-claude-quota/refresh.lock"`.

### Steps

- [ ] **Step 1: Write the failing tests**

Create `cmd/gf-claude-quota/internal/credentials/lock_test.go` with:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd cmd/gf-claude-quota && go test ./internal/credentials/ -run 'TestFileLock|TestDefaultLockPath' -v`
Expected: **FAIL** — `undefined: FileLock`, `undefined: NewFileLock`, `undefined: defaultLockPath`.

- [ ] **Step 3: Write the implementation**

Create `cmd/gf-claude-quota/internal/credentials/lock.go`:

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd cmd/gf-claude-quota && go test ./internal/credentials/ -run 'TestFileLock|TestDefaultLockPath' -v`
Expected: **PASS** — all sub-tests green.

- [ ] **Step 5: Commit**

```bash
cd /Users/yast/git/go-forge
git add cmd/gf-claude-quota/internal/credentials/lock.go cmd/gf-claude-quota/internal/credentials/lock_test.go
git commit -m "$(cat <<'EOF'
feat(gf-claude-quota): refresh直列化用のFileLockを追加

syscall.Flockによる排他ロック実装。複数のgf-claude-quota起動が
同時にrefreshすることを防ぐ。ロックファイルは
~/Library/Caches/gf-claude-quota/refresh.lock (macOS) /
~/.cache/gf-claude-quota/refresh.lock (Linux)に配置。

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Orchestrate the New Flow in `GetToken()`

**Files:**
- Modify: `cmd/gf-claude-quota/internal/credentials/credentials.go` (full rewrite)
- Modify: `cmd/gf-claude-quota/internal/credentials/credentials_test.go` (append new tests)

**Behavior:**

- Introduce sentinel error `ErrClaudeRunning`.
- Factor the flow into a `getTokenWithDeps(deps *getTokenDeps) (string, error)` helper that takes all side-effecting operations as function-typed fields. `GetToken()` becomes a thin wrapper binding platform-real functions.
- Flow matches the spec exactly:
  1. Env var bypass
  2. Read credentials
  3. Not expired → return
  4. Expired + Claude running → return `ErrClaudeRunning`
  5. Expired + Claude not running → acquire lock → re-read → if now valid return → else refresh → save (best-effort) → return new token

### Steps

- [ ] **Step 1: Write failing orchestration tests**

Append to `cmd/gf-claude-quota/internal/credentials/credentials_test.go`:

```go
import (
	"errors"
	"time"
)

// fakeLock implements the lock interface used by getTokenWithDeps.
type fakeLock struct {
	released bool
	acquireErr error
	releaseErr error
}

func (l *fakeLock) Acquire() error {
	if l.acquireErr != nil {
		return l.acquireErr
	}
	return nil
}
func (l *fakeLock) Release() error {
	l.released = true
	return l.releaseErr
}

func TestGetTokenWithDeps_EnvBypass(t *testing.T) {
	t.Setenv("CLAUDE_OAUTH_TOKEN", "env-token")
	// Deps fields should never be invoked.
	d := &getTokenDeps{}
	tok, err := getTokenWithDeps(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "env-token" {
		t.Errorf("token = %q, want env-token", tok)
	}
}

func TestGetTokenWithDeps_ValidToken(t *testing.T) {
	t.Setenv("CLAUDE_OAUTH_TOKEN", "")
	future := time.Now().Add(1 * time.Hour).UnixMilli()
	d := &getTokenDeps{
		loadCredentials: func() (*FullCredentials, error) {
			return &FullCredentials{AccessToken: "valid", ExpiresAt: future}, nil
		},
		refresher: NewTokenRefresher(nil),
	}
	tok, err := getTokenWithDeps(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "valid" {
		t.Errorf("token = %q, want valid", tok)
	}
}

func TestGetTokenWithDeps_ExpiredAndClaudeRunning(t *testing.T) {
	t.Setenv("CLAUDE_OAUTH_TOKEN", "")
	past := time.Now().Add(-1 * time.Hour).UnixMilli()
	d := &getTokenDeps{
		loadCredentials: func() (*FullCredentials, error) {
			return &FullCredentials{AccessToken: "stale", ExpiresAt: past, RefreshToken: "rt"}, nil
		},
		isClaudeRunning: func() bool { return true },
		refresher:       NewTokenRefresher(nil),
	}
	_, err := getTokenWithDeps(d)
	if !errors.Is(err, ErrClaudeRunning) {
		t.Errorf("err = %v, want ErrClaudeRunning", err)
	}
}

func TestGetTokenWithDeps_ExpiredNotRunningRefreshes(t *testing.T) {
	t.Setenv("CLAUDE_OAUTH_TOKEN", "")
	past := time.Now().Add(-1 * time.Hour).UnixMilli()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.FormValue("refresh_token") != "rt-original" {
			t.Errorf("refresh_token = %q", r.FormValue("refresh_token"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"new-access","refresh_token":"new-rt","expires_in":3600}`))
	}))
	defer server.Close()

	refresher := NewTokenRefresher(server.Client())
	refresher.SetEndpoint(server.URL)

	loadCount := 0
	var saved *FullCredentials
	d := &getTokenDeps{
		loadCredentials: func() (*FullCredentials, error) {
			loadCount++
			return &FullCredentials{
				AccessToken: "stale", ExpiresAt: past,
				RefreshToken: "rt-original", Scopes: []string{"user:profile"},
				SubscriptionType: "max",
			}, nil
		},
		saveCredentials: func(c *FullCredentials) error { saved = c; return nil },
		isClaudeRunning: func() bool { return false },
		acquireLock:     func() (lockReleaser, error) { return &fakeLock{}, nil },
		refresher:       refresher,
	}

	tok, err := getTokenWithDeps(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "new-access" {
		t.Errorf("token = %q, want new-access", tok)
	}
	if loadCount != 2 {
		t.Errorf("loadCredentials called %d times, want 2 (initial + re-read inside lock)", loadCount)
	}
	if saved == nil || saved.AccessToken != "new-access" || saved.RefreshToken != "new-rt" {
		t.Errorf("saved credentials mismatch: %+v", saved)
	}
	if saved.Scopes[0] != "user:profile" || saved.SubscriptionType != "max" {
		t.Errorf("scopes/subscriptionType not preserved: %+v", saved)
	}
}

func TestGetTokenWithDeps_LockReReadSkipsRefresh(t *testing.T) {
	// Simulates another gf-claude-quota instance refreshing while we waited for the lock.
	t.Setenv("CLAUDE_OAUTH_TOKEN", "")
	past := time.Now().Add(-1 * time.Hour).UnixMilli()
	future := time.Now().Add(1 * time.Hour).UnixMilli()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("refresher should NOT be called when re-read shows valid token")
	}))
	defer server.Close()
	refresher := NewTokenRefresher(server.Client())
	refresher.SetEndpoint(server.URL)

	loadCount := 0
	d := &getTokenDeps{
		loadCredentials: func() (*FullCredentials, error) {
			loadCount++
			if loadCount == 1 {
				return &FullCredentials{AccessToken: "stale", ExpiresAt: past, RefreshToken: "rt"}, nil
			}
			return &FullCredentials{AccessToken: "refreshed-by-other", ExpiresAt: future, RefreshToken: "rt2"}, nil
		},
		saveCredentials: func(c *FullCredentials) error { t.Error("save should not be called"); return nil },
		isClaudeRunning: func() bool { return false },
		acquireLock:     func() (lockReleaser, error) { return &fakeLock{}, nil },
		refresher:       refresher,
	}

	tok, err := getTokenWithDeps(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "refreshed-by-other" {
		t.Errorf("token = %q, want refreshed-by-other", tok)
	}
}

func TestGetTokenWithDeps_RefreshFailure(t *testing.T) {
	t.Setenv("CLAUDE_OAUTH_TOKEN", "")
	past := time.Now().Add(-1 * time.Hour).UnixMilli()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer server.Close()
	refresher := NewTokenRefresher(server.Client())
	refresher.SetEndpoint(server.URL)

	d := &getTokenDeps{
		loadCredentials: func() (*FullCredentials, error) {
			return &FullCredentials{AccessToken: "stale", ExpiresAt: past, RefreshToken: "rt"}, nil
		},
		saveCredentials: func(*FullCredentials) error { return nil },
		isClaudeRunning: func() bool { return false },
		acquireLock:     func() (lockReleaser, error) { return &fakeLock{}, nil },
		refresher:       refresher,
	}

	_, err := getTokenWithDeps(d)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetTokenWithDeps_LockAcquireFailure(t *testing.T) {
	t.Setenv("CLAUDE_OAUTH_TOKEN", "")
	past := time.Now().Add(-1 * time.Hour).UnixMilli()
	d := &getTokenDeps{
		loadCredentials: func() (*FullCredentials, error) {
			return &FullCredentials{AccessToken: "stale", ExpiresAt: past, RefreshToken: "rt"}, nil
		},
		isClaudeRunning: func() bool { return false },
		acquireLock:     func() (lockReleaser, error) { return nil, errors.New("lock denied") },
		refresher:       NewTokenRefresher(nil),
	}
	_, err := getTokenWithDeps(d)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetTokenWithDeps_SaveFailureStillReturnsToken(t *testing.T) {
	// If Keychain save fails, we still return the refreshed access token
	// for the current caller (best-effort persistence).
	t.Setenv("CLAUDE_OAUTH_TOKEN", "")
	past := time.Now().Add(-1 * time.Hour).UnixMilli()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"new","refresh_token":"rt2","expires_in":3600}`))
	}))
	defer server.Close()
	refresher := NewTokenRefresher(server.Client())
	refresher.SetEndpoint(server.URL)

	d := &getTokenDeps{
		loadCredentials: func() (*FullCredentials, error) {
			return &FullCredentials{AccessToken: "stale", ExpiresAt: past, RefreshToken: "rt"}, nil
		},
		saveCredentials: func(*FullCredentials) error { return errors.New("keychain denied") },
		isClaudeRunning: func() bool { return false },
		acquireLock:     func() (lockReleaser, error) { return &fakeLock{}, nil },
		refresher:       refresher,
	}
	tok, err := getTokenWithDeps(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "new" {
		t.Errorf("token = %q, want new", tok)
	}
}
```

NOTE: if the existing test file already imports some of `errors`, `http`, `httptest`, `time`, merge the imports — don't duplicate. Run `gofmt -w` after editing.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd cmd/gf-claude-quota && go test ./internal/credentials/ -run 'TestGetTokenWithDeps' -v`
Expected: **FAIL** — `undefined: getTokenWithDeps`, `undefined: getTokenDeps`, `undefined: lockReleaser`, `undefined: ErrClaudeRunning`.

- [ ] **Step 3: Rewrite `credentials.go`**

Replace the full contents of `cmd/gf-claude-quota/internal/credentials/credentials.go` with:

```go
package credentials

import (
	"errors"
	"fmt"
	"os"
)

// CredentialProvider retrieves an OAuth access token.
type CredentialProvider interface {
	GetToken() (string, error)
}

// ErrClaudeRunning is returned when the stored token is expired and a
// `claude` process is currently running. Callers should surface this as a
// short-lived state (1 minute menu-bar poll resolves it automatically once
// Claude Code refreshes on its next API call).
var ErrClaudeRunning = errors.New("Claude Code is running; its next API call will refresh the token — retry in a moment")

// lockReleaser is the subset of FileLock used by getTokenWithDeps.
type lockReleaser interface {
	Release() error
}

// getTokenDeps bundles the side-effecting operations GetToken orchestrates.
// All fields are injected for testability; GetToken binds the real ones.
type getTokenDeps struct {
	loadCredentials func() (*FullCredentials, error)
	saveCredentials func(*FullCredentials) error
	isClaudeRunning func() bool
	acquireLock     func() (lockReleaser, error)
	refresher       *TokenRefresher
}

// GetToken retrieves a valid OAuth access token.
//
// Resolution order:
//   1. CLAUDE_OAUTH_TOKEN environment variable (CI / tests)
//   2. Platform credential store (macOS Keychain / Linux credentials.json)
//
// If the stored token is expired:
//   - If a `claude` process is running, return ErrClaudeRunning (Claude Code
//     will refresh on its next API call; our refresh would invalidate its
//     in-memory refresh_token and force re-login).
//   - Otherwise, take an exclusive refresh lock, re-read credentials, and if
//     still expired, refresh via the OAuth endpoint and persist.
func GetToken() (string, error) {
	return getTokenWithDeps(&getTokenDeps{
		loadCredentials: getFullPlatformCredentials,
		saveCredentials: savePlatformCredentials,
		isClaudeRunning: IsClaudeRunning,
		acquireLock:     acquireRefreshLock,
		refresher:       NewTokenRefresher(nil),
	})
}

func getTokenWithDeps(d *getTokenDeps) (string, error) {
	if token := os.Getenv("CLAUDE_OAUTH_TOKEN"); token != "" {
		return token, nil
	}

	creds, err := d.loadCredentials()
	if err != nil {
		return "", err
	}

	if !d.refresher.IsExpired(creds.ExpiresAt) {
		return creds.AccessToken, nil
	}

	// Expired.
	if d.isClaudeRunning() {
		return "", ErrClaudeRunning
	}

	lock, err := d.acquireLock()
	if err != nil {
		return "", fmt.Errorf("acquiring refresh lock: %w", err)
	}
	defer lock.Release()

	// Re-read inside lock in case another instance just refreshed.
	creds, err = d.loadCredentials()
	if err != nil {
		return "", err
	}
	if !d.refresher.IsExpired(creds.ExpiresAt) {
		return creds.AccessToken, nil
	}

	if creds.RefreshToken == "" {
		return "", fmt.Errorf("token expired and no refresh token available — run `claude login`")
	}

	newEntry, err := d.refresher.Refresh(creds.RefreshToken)
	if err != nil {
		return "", err
	}

	updated := &FullCredentials{
		AccessToken:      newEntry.AccessToken,
		RefreshToken:     newEntry.RefreshToken,
		ExpiresAt:        newEntry.ExpiresAt,
		Scopes:           creds.Scopes,
		SubscriptionType: creds.SubscriptionType,
	}
	// Best-effort save. If it fails, the current caller still gets a valid
	// token; the next invocation will prompt re-login via `claude login`.
	_ = d.saveCredentials(updated)
	return updated.AccessToken, nil
}

// acquireRefreshLock opens and locks the default refresh lock file,
// returning a lockReleaser for deferred release.
func acquireRefreshLock() (lockReleaser, error) {
	path, err := defaultLockPath()
	if err != nil {
		return nil, err
	}
	lock := NewFileLock(path)
	if err := lock.Acquire(); err != nil {
		return nil, err
	}
	return lock, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd cmd/gf-claude-quota && go test ./internal/credentials/ -v`
Expected: **PASS** — new orchestration tests + all existing tests.

- [ ] **Step 5: Run the full test suite for the binary**

Run: `cd cmd/gf-claude-quota && go test ./...`
Expected: **PASS** — every package.

- [ ] **Step 6: Build the binary locally to verify no compilation errors**

Run: `cd cmd/gf-claude-quota && go build -o /tmp/gf-claude-quota .`
Expected: exit 0, binary produced.

- [ ] **Step 7: Commit**

```bash
cd /Users/yast/git/go-forge
git add cmd/gf-claude-quota/internal/credentials/credentials.go cmd/gf-claude-quota/internal/credentials/credentials_test.go
git commit -m "$(cat <<'EOF'
feat(gf-claude-quota): Coordinated refresh実装

GetToken()をCoordinated Refresh方式に書き換え:
- 期限切れ + Claude Code稼働中 → ErrClaudeRunning返却（次のAPI呼び出しで本体がrefreshするのを待つ）
- 期限切れ + Claude Code非稼働 → FileLock取得 → 再読み込み → refresh → 保存

refresh token rotationによるre-login問題をプロセス検出で回避しつつ、
24h以上の放置シナリオでも自動回復する。

設計: docs/superpowers/specs/2026-04-17-coordinated-refresh-design.md

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Final Quality Check & End-to-End Verification

**Files:** none modified

### Steps

- [ ] **Step 1: Run project-wide quality check**

Run: `cd /Users/yast/git/go-forge && make quality`
Expected: exit 0. If golangci-lint reports issues in new files, fix them inline (re-run tests after each fix) and amend the most recent commit — but **only if the lint fix is non-behavioral** (naming, imports, formatting). Behavioral fixes warrant a new commit.

- [ ] **Step 2: Run project-wide tests**

Run: `cd /Users/yast/git/go-forge && make test`
Expected: all packages PASS.

- [ ] **Step 3: Manual smoke test — valid (non-expired) token path**

Run: `cd /Users/yast/git/go-forge/cmd/gf-claude-quota && go build -o gf-claude-quota . && ./gf-claude-quota`
Expected: quota data prints normally (today's Keychain token is still valid).

- [ ] **Step 4: Manual smoke test — env var bypass**

Run: `CLAUDE_OAUTH_TOKEN='sk-ant-oat01-dummy' ./gf-claude-quota 2>&1 | head -2`
Expected: HTTP error (the dummy token is rejected by the usage endpoint), **but** no Keychain-related error — confirming the env var short-circuited platform lookup.

- [ ] **Step 5: Verify ErrClaudeRunning path manually (optional)**

While a `claude` process is running (the current session counts), inspect the flow by temporarily setting Keychain credentials to expired via direct code inspection — or simply trust the orchestration test suite, which already covers this branch exhaustively with deterministic stubs.

- [ ] **Step 6: Push the branch and open a PR**

```bash
cd /Users/yast/git/go-forge
git push -u origin feat/coordinated-token-refresh
gh pr create --title "feat(gf-claude-quota): Coordinated OAuth token refresh" --body "$(cat <<'EOF'
## Summary
- 24h以上の放置で動作不能になる問題を解消
- Claude Code非稼働時のみrefreshすることでrefresh token rotation衝突を回避
- File-lockで複数gf-claude-quotaインスタンスの同時refreshを直列化

## Design
設計ドキュメント: `docs/superpowers/specs/2026-04-17-coordinated-refresh-design.md`

## Rejected alternatives (tracked as issues)
- #10 Option B (claude.ai cookie)
- #11 Option D/K (Claude Code subprocess)
- #12 Option C (UX改善のみ)

## Test plan
- [x] unit tests (process detection, file lock, GetToken orchestration)
- [x] `make quality` passes
- [x] `make test` passes
- [ ] 手動: 通常起動でクォータ表示確認
- [ ] 手動: 24h放置後の自動回復確認（時間を要する）

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

---

## Self-Review

**1. Spec coverage**

| Spec section | Task |
|---|---|
| Core flow (env / load / expiry / running / lock / re-read / refresh / save) | Task 3 |
| `isClaudeRunning` via pgrep + basename check | Task 1 |
| File lock under user cache dir | Task 2 |
| `ErrClaudeRunning` sentinel | Task 3 |
| Best-effort save on Keychain failure | Task 3 (verified by `TestGetTokenWithDeps_SaveFailureStillReturnsToken`) |
| Re-read inside lock | Task 3 (verified by `TestGetTokenWithDeps_LockReReadSkipsRefresh`) |
| Fail-safe process detection | Task 1 (verified by binary-missing and unexpected-exit tests) |
| Scopes / SubscriptionType preserved | Task 3 (verified inside refresh test) |

All spec requirements have at least one implementing task.

**2. Placeholder scan**

None. Every step includes either exact code or exact commands.

**3. Type consistency**

- `CommandRunner` — matches existing type in `keychain.go`.
- `FullCredentials`, `FullCredentials.AccessToken`, `.RefreshToken`, `.ExpiresAt`, `.Scopes`, `.SubscriptionType` — match `store.go`.
- `NewTokenRefresher`, `IsExpired`, `Refresh`, `SetEndpoint` — match `refresh.go`.
- `FileLock`, `NewFileLock`, `Acquire`, `Release` — defined in Task 2, used identically in Task 3.
- `lockReleaser` interface matches `FileLock`'s `Release() error` signature.
- `getTokenDeps`, `getTokenWithDeps`, `ErrClaudeRunning` — defined and used consistently in Task 3.
- `IsClaudeRunning`, `isClaudeRunningWithRunner`, `matchesClaudeBinary` — defined in Task 1, `IsClaudeRunning` called from Task 3.
- `defaultLockPath`, `acquireRefreshLock` — defined in Task 2/3, used in Task 3.

All names and signatures consistent across tasks.

---
