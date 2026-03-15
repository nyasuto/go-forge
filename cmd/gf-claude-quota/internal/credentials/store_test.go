package credentials

import (
	"os"
	"strings"
	"testing"
)

func TestParseFullCredentials(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *FullCredentials
		wantErr string
	}{
		{
			name: "valid credentials",
			input: `{
				"claudeAiOauth": {
					"accessToken": "sk-ant-oat01-test",
					"refreshToken": "rt-test",
					"expiresAt": 1234567890000,
					"scopes": ["user:read"],
					"subscriptionType": "pro"
				}
			}`,
			want: &FullCredentials{
				AccessToken:      "sk-ant-oat01-test",
				RefreshToken:     "rt-test",
				ExpiresAt:        1234567890000,
				Scopes:           []string{"user:read"},
				SubscriptionType: "pro",
			},
		},
		{
			name:    "invalid JSON",
			input:   "not json",
			wantErr: "parsing credentials",
		},
		{
			name:    "missing claudeAiOauth",
			input:   `{"other": "field"}`,
			wantErr: "missing claudeAiOauth",
		},
		{
			name:    "empty accessToken",
			input:   `{"claudeAiOauth": {"accessToken": "", "refreshToken": "rt"}}`,
			wantErr: "missing accessToken",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFullCredentials(tt.input)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.AccessToken != tt.want.AccessToken {
				t.Errorf("AccessToken = %q, want %q", got.AccessToken, tt.want.AccessToken)
			}
			if got.RefreshToken != tt.want.RefreshToken {
				t.Errorf("RefreshToken = %q, want %q", got.RefreshToken, tt.want.RefreshToken)
			}
			if got.ExpiresAt != tt.want.ExpiresAt {
				t.Errorf("ExpiresAt = %d, want %d", got.ExpiresAt, tt.want.ExpiresAt)
			}
			if got.SubscriptionType != tt.want.SubscriptionType {
				t.Errorf("SubscriptionType = %q, want %q", got.SubscriptionType, tt.want.SubscriptionType)
			}
		})
	}
}

func TestGetFullCredentialsFromKeychain(t *testing.T) {
	runner := func(name string, args ...string) ([]byte, error) {
		return []byte(`{"claudeAiOauth":{"accessToken":"sk-ant-oat01-mock","refreshToken":"rt-mock","expiresAt":9999999999999,"scopes":["user:read"],"subscriptionType":"pro"}}` + "\n"), nil
	}

	creds, err := GetFullCredentialsFromKeychain(runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.AccessToken != "sk-ant-oat01-mock" {
		t.Errorf("AccessToken = %q, want %q", creds.AccessToken, "sk-ant-oat01-mock")
	}
	if creds.RefreshToken != "rt-mock" {
		t.Errorf("RefreshToken = %q, want %q", creds.RefreshToken, "rt-mock")
	}
}

func TestGetFullCredentialsFromFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/credentials.json"
	content := `{
		"claudeAiOauth": {
			"accessToken": "sk-ant-oat01-file",
			"refreshToken": "rt-file",
			"expiresAt": 9999999999999,
			"scopes": ["user:read"],
			"subscriptionType": "max"
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	creds, err := GetFullCredentialsFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.AccessToken != "sk-ant-oat01-file" {
		t.Errorf("AccessToken = %q, want %q", creds.AccessToken, "sk-ant-oat01-file")
	}
	if creds.SubscriptionType != "max" {
		t.Errorf("SubscriptionType = %q, want %q", creds.SubscriptionType, "max")
	}
}

func TestSaveToFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/credentials.json"

	// Write initial file.
	initial := `{"claudeAiOauth":{"accessToken":"old","refreshToken":"rt-old","expiresAt":100,"scopes":[],"subscriptionType":"pro"},"extraField":"keep"}`
	if err := os.WriteFile(path, []byte(initial), 0600); err != nil {
		t.Fatal(err)
	}

	creds := &FullCredentials{
		AccessToken:      "new-token",
		RefreshToken:     "new-rt",
		ExpiresAt:        200,
		Scopes:           []string{"user:read"},
		SubscriptionType: "pro",
	}

	if err := SaveToFile(path, creds); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}

	// Read back and verify.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "new-token") {
		t.Errorf("saved file missing new-token: %s", content)
	}
	if !strings.Contains(content, "extraField") {
		t.Errorf("saved file lost extraField: %s", content)
	}

	// Verify we can read it back.
	readBack, err := GetFullCredentialsFromFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if readBack.AccessToken != "new-token" {
		t.Errorf("read back AccessToken = %q, want %q", readBack.AccessToken, "new-token")
	}
}

func TestSaveToKeychain(t *testing.T) {
	var savedArgs []string
	runner := func(name string, args ...string) ([]byte, error) {
		savedArgs = append(savedArgs, name)
		savedArgs = append(savedArgs, args...)
		return nil, nil
	}

	creds := &FullCredentials{
		AccessToken:      "new-token",
		RefreshToken:     "new-rt",
		ExpiresAt:        200,
		Scopes:           []string{"user:read"},
		SubscriptionType: "pro",
	}

	original := `{"claudeAiOauth":{"accessToken":"old","refreshToken":"old-rt","expiresAt":100,"scopes":[],"subscriptionType":"pro"}}`
	err := SaveToKeychain(runner, creds, original)
	if err != nil {
		t.Fatalf("SaveToKeychain: %v", err)
	}

	// Verify the security command was called with -U flag.
	found := false
	for _, arg := range savedArgs {
		if arg == "-U" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected -U flag in security command args: %v", savedArgs)
	}
}

func TestTrimSpace(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"  hello  ", "hello"},
		{"\n\thello\n\t", "hello"},
		{"hello", "hello"},
		{"", ""},
		{"  ", ""},
	}

	for _, tt := range tests {
		got := trimSpace(tt.input)
		if got != tt.want {
			t.Errorf("trimSpace(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
