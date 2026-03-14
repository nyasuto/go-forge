package credentials

import (
	"os"
	"testing"
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
