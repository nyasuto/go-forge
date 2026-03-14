package credentials

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseKeychainJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{
			name: "valid credentials",
			input: `{
				"claudeAiOauth": {
					"accessToken": "sk-ant-oat01-test-token",
					"refreshToken": "rt-test",
					"expiresAt": 1234567890,
					"scopes": ["user:read"],
					"subscriptionType": "pro"
				}
			}`,
			want: "sk-ant-oat01-test-token",
		},
		{
			name: "max plan credentials",
			input: `{
				"claudeAiOauth": {
					"accessToken": "sk-ant-oat01-max-token-abc",
					"refreshToken": "rt-max",
					"expiresAt": 9999999999,
					"scopes": ["user:read", "usage:read"],
					"subscriptionType": "max"
				}
			}`,
			want: "sk-ant-oat01-max-token-abc",
		},
		{
			name:    "invalid JSON",
			input:   `not json`,
			wantErr: "parsing keychain credentials",
		},
		{
			name:    "missing claudeAiOauth",
			input:   `{"otherField": "value"}`,
			wantErr: "missing claudeAiOauth field",
		},
		{
			name:    "empty accessToken",
			input:   `{"claudeAiOauth": {"accessToken": "", "refreshToken": "rt"}}`,
			wantErr: "missing accessToken",
		},
		{
			name:    "null claudeAiOauth",
			input:   `{"claudeAiOauth": null}`,
			wantErr: "missing claudeAiOauth field",
		},
		{
			name:    "empty object",
			input:   `{}`,
			wantErr: "missing claudeAiOauth field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseKeychainJSON(tt.input)
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
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetTokenFromKeychain_Success(t *testing.T) {
	runner := func(name string, args ...string) ([]byte, error) {
		if name != "security" {
			t.Errorf("command = %q, want security", name)
		}
		return []byte(`{"claudeAiOauth":{"accessToken":"sk-ant-oat01-mock","refreshToken":"rt","expiresAt":123,"scopes":[],"subscriptionType":"pro"}}` + "\n"), nil
	}

	token, err := GetTokenFromKeychain(runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "sk-ant-oat01-mock" {
		t.Errorf("token = %q, want %q", token, "sk-ant-oat01-mock")
	}
}

func TestGetTokenFromKeychain_CommandFailure(t *testing.T) {
	runner := func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("exit status 44")
	}

	_, err := GetTokenFromKeychain(runner)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read keychain") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "failed to read keychain")
	}
}

func TestGetTokenFromKeychain_InvalidJSON(t *testing.T) {
	runner := func(name string, args ...string) ([]byte, error) {
		return []byte("not-json\n"), nil
	}

	_, err := GetTokenFromKeychain(runner)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "parsing keychain credentials") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "parsing keychain credentials")
	}
}

func TestGetTokenFromKeychain_WhitespaceHandling(t *testing.T) {
	runner := func(name string, args ...string) ([]byte, error) {
		return []byte(`  {"claudeAiOauth":{"accessToken":"sk-token","refreshToken":"rt","expiresAt":0,"scopes":[],"subscriptionType":"pro"}}  ` + "\n"), nil
	}

	token, err := GetTokenFromKeychain(runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "sk-token" {
		t.Errorf("token = %q, want %q", token, "sk-token")
	}
}
