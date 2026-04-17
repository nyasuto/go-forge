---
title: Coordinated OAuth Token Refresh for gf-claude-quota
date: 2026-04-17
status: approved
tool: gf-claude-quota
---

# Coordinated OAuth Token Refresh (Option J)

## Problem

`gf-claude-quota` reads Claude Code's OAuth credentials from the macOS Keychain (or `~/.claude/.credentials.json` on Linux) to query the usage endpoint `https://api.anthropic.com/api/oauth/usage`. The stored access token typically expires within ~24 hours.

**Historical context:**

- **PR #5 (`c8cd009`)** added automatic refresh. Anthropic's OAuth server enforces **refresh token rotation** (single-use refresh tokens). When `gf-claude-quota` refreshed the token, Claude Code's in-memory `refresh_token` became invalid on the server, and Claude Code's next refresh attempt forced the user to re-login.
- **PR #9 (`ee6f0c9`)** disabled refresh to protect Claude Code's lifecycle. Current behavior: if Claude Code is idle for more than ~24h, the menu-bar tool errors out until the user invokes any `claude` command.

**User pain:** the menu-bar quota display silently dies after a day of inactivity, even though Claude Code itself recovers seamlessly on next use.

## Constraints

- Anthropic has disabled third-party OAuth tokens (issue [#28091](https://github.com/anthropics/claude-code/issues/28091)). The `claude setup-token` command generates inference-only tokens that lack the `user:profile` scope required by the usage endpoint (verified experimentally: HTTP 403 `OAuth token does not meet scope requirement user:profile`).
- Must not break Claude Code's own token lifecycle (no forced re-logins).
- Project rules: standard library only, no external dependencies.
- Must keep existing `CLAUDE_OAUTH_TOKEN` env-var bypass intact (CI/testing path).

## Design

### Core Insight

The refresh-token rotation collision happens only when Claude Code has a **live in-memory copy** of the current `refresh_token`. This only occurs while a `claude` process is running. When no `claude` process exists, refreshing from `gf-claude-quota` is safe: Claude Code will re-read the Keychain on its next startup and pick up the rotated token.

### Flow

```
GetToken():
  1. if env CLAUDE_OAUTH_TOKEN is set → return it (CI/test bypass)
  2. read Keychain credentials
  3. if !expired(creds) → return creds.AccessToken
  4. (expired path)
     a. if isClaudeRunning() → return user-facing error:
        "Claude Code is running; its next API call will refresh the token.
         Retry in a moment."
     b. acquire file lock (refresh.lock)
        - re-read Keychain inside lock (another gf-claude-quota instance
          may have just refreshed)
        - if now valid → release lock, return access_token
        - else POST to token endpoint, save to Keychain, release lock,
          return new access_token
```

### Component Design

| Component | Responsibility | New/Modified |
|-----------|----------------|--------------|
| `internal/credentials/process.go` | Detect running `claude` processes via `pgrep` | New |
| `internal/credentials/lock.go` | File lock wrapper using `syscall.Flock` | New |
| `internal/credentials/credentials.go` | Orchestrate new `GetToken()` flow | Modified |
| `internal/credentials/refresh.go` | Refresh via OAuth endpoint | Existing, unchanged |
| `internal/credentials/store.go` | Keychain read/write | Existing, unchanged |

### Process Detection (`isClaudeRunning`)

- Command: `pgrep -f '(^|/)claude($| )'`
  - Matches `claude` at the start of cmdline or as basename of an absolute path
  - Followed by space or end-of-line → excludes `claude-*` (`gf-claude-quota`, `claude-code-proxy`, etc.)
- Exit code handling:
  - `0` → running (≥1 process matched)
  - `1` → not running (no matches)
  - other → assume running (fail-safe: don't risk breaking Claude Code)
- If `pgrep` binary is missing → assume running (same fail-safe default)
- Works identically on macOS and Linux

### File Lock

- Path: `filepath.Join(os.UserCacheDir(), "gf-claude-quota", "refresh.lock")`
  - macOS: `~/Library/Caches/gf-claude-quota/refresh.lock`
  - Linux: `~/.cache/gf-claude-quota/refresh.lock`
- `syscall.Flock(fd, LOCK_EX)` — advisory exclusive lock
- Released via `defer` on function exit
- Purpose: serialize multiple `gf-claude-quota` invocations (e.g., menu-bar poll + shell check)

### Re-read Inside Lock

Prevents "double refresh" when two `gf-claude-quota` instances contend for refresh:

```go
acquire lock
creds := reloadFromKeychain()
if !expired(creds) {
    return creds.AccessToken  // another instance already refreshed
}
newCreds := refresh(creds.RefreshToken)
save(newCreds)
return newCreds.AccessToken
```

### Error Messages

| Situation | Message |
|-----------|---------|
| Claude running + expired | `"Claude Code is running; its next API call will refresh the token. Retry in a moment."` |
| Claude running + expired (xbar short form) | `"Token refresh pending"` |
| Refresh HTTP failure | Existing message preserved (points user to `claude login`) |
| Lock acquisition failure | `"refresh already in progress; try again"` |

## Testing

Project rules require table-driven tests with 3+ normal, 2+ error, 2+ edge cases per unit.

**`isClaudeRunning`**
- Normal: pgrep returns one pid (stub `CommandRunner`)
- Normal: pgrep returns zero pids (exit code 1)
- Normal: multiple claude processes
- Error: pgrep binary missing → returns `true` (fail-safe)
- Error: pgrep unexpected non-zero exit → returns `true` (fail-safe)
- Edge: absolute path `/usr/local/bin/claude --foo` matches
- Edge: `claude-code-proxy` or `gf-claude-quota` does NOT match

**File lock**
- Normal: acquire succeeds on a fresh lock file
- Normal: acquire then release, second acquire succeeds
- Error: lock file path in non-writable location → error
- Edge: concurrent acquires in same process serialize correctly

**`GetToken` orchestration** (integration-style with stubs)
- Normal: `CLAUDE_OAUTH_TOKEN` env set → bypass path
- Normal: valid Keychain token → return directly (no process check)
- Normal: expired + Claude not running → refresh + save + return
- Error: expired + Claude running → specific user-facing error
- Error: expired + refresh endpoint fails → error propagates
- Edge: expired, acquire lock, re-read shows fresh token → return without refresh
- Edge: Keychain parse error → wrapped error with guidance

## Rejected Alternatives

Tracked in GitHub issues so future-us can revisit without re-deriving the reasoning.

- **Option A — Long-lived token via `claude setup-token`**
  Verified dead: HTTP 403 `user:profile` scope missing. The `setup-token` command issues inference-only tokens.
- **Option B — `claude.ai` session cookie**
  Heavy implementation: requires WebView login, cookie extraction, and a completely different auth path. Deferred as a backup if Option J reveals unforeseen issues.
- **Option C — Status-quo + UX improvement**
  Leaves the core pain unresolved; only surfaces it more clearly.
- **Option D/K — Trigger Claude Code via subprocess to refresh**
  No known lightweight `claude` subcommand that forces refresh without making a billable API call. Would waste quota.
- **Option E — Refresh + file lock only (no process detection)**
  Cannot eliminate the race with Claude Code's in-memory `refresh_token` through file locking alone; Claude Code does not participate in the lock protocol.

## Known Residual Risks

- **Micro-window between `isClaudeRunning()` and refresh**: Claude Code could launch in the few milliseconds between check and HTTP POST. If it happens, the user's next Claude Code invocation errors and prompts re-login. This is the same failure mode as the pre-PR-#9 behavior, but astronomically less likely. Not a regression.
- **Claude Code running 24h+ without any API call**: token expires while the process is active but idle. `gf-claude-quota` will show the "Claude Code running; retry in a moment" message indefinitely. Extremely rare in practice; mitigated by the clear user-facing message.
- **Non-standard Claude Code installs** (alternative binary names): detection misses them, so `gf-claude-quota` proceeds with refresh and could cause a re-login. Documented as a known limitation.

## Out of Scope

- Windows support (not currently targeted by `gf-claude-quota`).
- Switching to `claude.ai` cookie auth.
- Coordinating with Claude Code via IPC / shared memory.
