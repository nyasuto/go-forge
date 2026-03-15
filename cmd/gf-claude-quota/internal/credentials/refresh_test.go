package credentials

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIsExpired(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	r := NewTokenRefresher(nil)
	r.nowFunc = func() time.Time { return now }

	tests := []struct {
		name      string
		expiresMs int64
		want      bool
	}{
		{
			name:      "expired in the past",
			expiresMs: now.Add(-1 * time.Hour).UnixMilli(),
			want:      true,
		},
		{
			name:      "expires within buffer",
			expiresMs: now.Add(3 * time.Minute).UnixMilli(),
			want:      true,
		},
		{
			name:      "expires exactly at buffer boundary",
			expiresMs: now.Add(expiryBuffer).UnixMilli(),
			want:      false, // After is strict: equal times => not after
		},
		{
			name:      "expires well after buffer",
			expiresMs: now.Add(1 * time.Hour).UnixMilli(),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.IsExpired(tt.expiresMs)
			if got != tt.want {
				t.Errorf("IsExpired(%d) = %v, want %v", tt.expiresMs, got, tt.want)
			}
		})
	}
}

func TestRefresh_Success(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", ct)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.FormValue("grant_type"); got != "refresh_token" {
			t.Errorf("grant_type = %q, want refresh_token", got)
		}
		if got := r.FormValue("refresh_token"); got != "rt-test-123" {
			t.Errorf("refresh_token = %q, want rt-test-123", got)
		}
		if got := r.FormValue("client_id"); got != clientID {
			t.Errorf("client_id = %q, want %q", got, clientID)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"access_token": "new-access-token",
			"refresh_token": "new-refresh-token",
			"expires_in": 3600,
			"token_type": "Bearer"
		}`))
	}))
	defer server.Close()

	r := NewTokenRefresher(server.Client())
	r.SetEndpoint(server.URL)
	r.nowFunc = func() time.Time { return now }

	entry, err := r.Refresh("rt-test-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if entry.AccessToken != "new-access-token" {
		t.Errorf("AccessToken = %q, want %q", entry.AccessToken, "new-access-token")
	}
	if entry.RefreshToken != "new-refresh-token" {
		t.Errorf("RefreshToken = %q, want %q", entry.RefreshToken, "new-refresh-token")
	}

	wantExpires := now.UnixMilli() + 3600*1000
	if entry.ExpiresAt != wantExpires {
		t.Errorf("ExpiresAt = %d, want %d", entry.ExpiresAt, wantExpires)
	}
}

func TestRefresh_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid_grant"}`))
	}))
	defer server.Close()

	r := NewTokenRefresher(server.Client())
	r.SetEndpoint(server.URL)

	_, err := r.Refresh("invalid-token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "token refresh failed (400)") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "token refresh failed (400)")
	}
	if !strings.Contains(err.Error(), "claude login") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "claude login")
	}
}

func TestRefresh_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	r := NewTokenRefresher(server.Client())
	r.SetEndpoint(server.URL)

	_, err := r.Refresh("some-token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "parsing refresh response") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "parsing refresh response")
	}
}

func TestRefresh_EmptyAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"access_token": "", "refresh_token": "rt", "expires_in": 3600}`))
	}))
	defer server.Close()

	r := NewTokenRefresher(server.Client())
	r.SetEndpoint(server.URL)

	_, err := r.Refresh("some-token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "missing access_token") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "missing access_token")
	}
}
