package credentials

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestGetToken_EnvVar(t *testing.T) {
	const envToken = "sk-ant-oat01-env-test-token"
	t.Setenv("CLAUDE_OAUTH_TOKEN", envToken)

	token, err := GetToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != envToken {
		t.Errorf("token = %q, want %q", token, envToken)
	}
}

func TestGetToken_EnvVarEmpty(t *testing.T) {
	t.Setenv("CLAUDE_OAUTH_TOKEN", "")

	// With empty env var, it falls through to platform-specific method.
	// On macOS CI without keychain, this will error — that's expected.
	_, _ = GetToken()
}

func TestGetToken_EnvVarPriority(t *testing.T) {
	const envToken = "sk-ant-oat01-env-priority"
	t.Setenv("CLAUDE_OAUTH_TOKEN", envToken)

	// Even if platform-specific method would succeed, env var takes priority
	token, err := GetToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != envToken {
		t.Errorf("token = %q, want %q", token, envToken)
	}
}

func TestGetTokenFromFile_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/credentials.json"
	content := `{
		"claudeAiOauth": {
			"accessToken": "sk-ant-oat01-file-token",
			"refreshToken": "rt-test",
			"expiresAt": 1234567890,
			"scopes": ["user:read"],
			"subscriptionType": "pro"
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	token, err := GetTokenFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "sk-ant-oat01-file-token" {
		t.Errorf("token = %q, want %q", token, "sk-ant-oat01-file-token")
	}
}

func TestGetTokenFromFile_MaxPlan(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/credentials.json"
	content := `{
		"claudeAiOauth": {
			"accessToken": "sk-ant-oat01-max-plan",
			"refreshToken": "rt",
			"expiresAt": 9999999999,
			"scopes": ["user:read", "usage:read"],
			"subscriptionType": "max"
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	token, err := GetTokenFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "sk-ant-oat01-max-plan" {
		t.Errorf("token = %q, want %q", token, "sk-ant-oat01-max-plan")
	}
}

func TestGetTokenFromFile_FileNotFound(t *testing.T) {
	_, err := GetTokenFromFile("/nonexistent/path/credentials.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestGetTokenFromFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/credentials.json"
	if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := GetTokenFromFile(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestGetTokenFromFile_MissingAccessToken(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/credentials.json"
	content := `{"claudeAiOauth": {"accessToken": "", "refreshToken": "rt"}}`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := GetTokenFromFile(path)
	if err == nil {
		t.Fatal("expected error for empty accessToken, got nil")
	}
}

func TestGetTokenFromFile_MissingOAuthField(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/credentials.json"
	content := `{"otherField": "value"}`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := GetTokenFromFile(path)
	if err == nil {
		t.Fatal("expected error for missing claudeAiOauth, got nil")
	}
}

func TestGetTokenFromFile_DefaultPath(t *testing.T) {
	// GetTokenFromFile with empty path uses default ~/.config/claude-code/credentials.json
	// This will likely fail (file doesn't exist) but should not panic
	_, err := GetTokenFromFile("")
	if err == nil {
		// If it succeeds, the user has a real credentials file — that's fine
		return
	}
	// Error is expected when the file doesn't exist
}

func TestGetTokenFromFile_WhitespaceInFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/credentials.json"
	content := `
	{
		"claudeAiOauth": {
			"accessToken": "sk-ant-oat01-whitespace",
			"refreshToken": "rt",
			"expiresAt": 0,
			"scopes": [],
			"subscriptionType": "pro"
		}
	}
	`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	token, err := GetTokenFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "sk-ant-oat01-whitespace" {
		t.Errorf("token = %q, want %q", token, "sk-ant-oat01-whitespace")
	}
}

// fakeLock implements lockReleaser for getTokenWithDeps tests.
type fakeLock struct {
	released bool
}

func (l *fakeLock) Release() error {
	l.released = true
	return nil
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
