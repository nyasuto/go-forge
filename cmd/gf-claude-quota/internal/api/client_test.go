package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchUsage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer test-token")
		}
		if got := r.Header.Get("anthropic-beta"); got != anthropicBeta {
			t.Errorf("anthropic-beta header = %q, want %q", got, anthropicBeta)
		}
		if got := r.Header.Get("User-Agent"); got != userAgent {
			t.Errorf("User-Agent header = %q, want %q", got, userAgent)
		}
		if r.Method != "GET" {
			t.Errorf("method = %q, want GET", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"five_hour": {"utilization": 42.0, "resets_at": "2025-11-04T04:59:59.943648+00:00"},
			"seven_day": {"utilization": 35.0, "resets_at": "2025-11-06T03:59:59.943679+00:00"},
			"seven_day_oauth_apps": null,
			"seven_day_opus": {"utilization": 0.0, "resets_at": null}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.SetEndpoint(server.URL)

	usage, err := client.FetchUsage("test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if usage.FiveHour == nil {
		t.Fatal("FiveHour is nil")
	}
	if usage.FiveHour.Utilization != 42.0 {
		t.Errorf("FiveHour.Utilization = %v, want 42.0", usage.FiveHour.Utilization)
	}
	if usage.FiveHour.ResetsAt == nil {
		t.Fatal("FiveHour.ResetsAt is nil")
	}
	if *usage.FiveHour.ResetsAt != "2025-11-04T04:59:59.943648+00:00" {
		t.Errorf("FiveHour.ResetsAt = %q, want %q", *usage.FiveHour.ResetsAt, "2025-11-04T04:59:59.943648+00:00")
	}

	if usage.SevenDay == nil {
		t.Fatal("SevenDay is nil")
	}
	if usage.SevenDay.Utilization != 35.0 {
		t.Errorf("SevenDay.Utilization = %v, want 35.0", usage.SevenDay.Utilization)
	}

	if usage.SevenDayOAuth != nil {
		t.Errorf("SevenDayOAuth = %v, want nil", usage.SevenDayOAuth)
	}

	if usage.SevenDayOpus == nil {
		t.Fatal("SevenDayOpus is nil")
	}
	if usage.SevenDayOpus.Utilization != 0.0 {
		t.Errorf("SevenDayOpus.Utilization = %v, want 0.0", usage.SevenDayOpus.Utilization)
	}
	if usage.SevenDayOpus.ResetsAt != nil {
		t.Errorf("SevenDayOpus.ResetsAt = %v, want nil", usage.SevenDayOpus.ResetsAt)
	}
}

func TestFetchUsage_Errors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    string
	}{
		{
			name:       "401 Unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"error": "unauthorized"}`,
			wantErr:    "authentication failed (401)",
		},
		{
			name:       "429 Rate Limited",
			statusCode: http.StatusTooManyRequests,
			body:       `{"error": "rate limited"}`,
			wantErr:    "rate limited (429)",
		},
		{
			name:       "500 Server Error",
			statusCode: http.StatusInternalServerError,
			body:       `{"error": "internal"}`,
			wantErr:    "server error (500)",
		},
		{
			name:       "503 Service Unavailable",
			statusCode: http.StatusServiceUnavailable,
			body:       `service unavailable`,
			wantErr:    "server error (503)",
		},
		{
			name:       "403 Forbidden",
			statusCode: http.StatusForbidden,
			body:       `forbidden`,
			wantErr:    "unexpected status 403",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := NewClient(server.Client())
			client.SetEndpoint(server.URL)

			_, err := client.FetchUsage("test-token")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if got := err.Error(); !contains(got, tt.wantErr) {
				t.Errorf("error = %q, want to contain %q", got, tt.wantErr)
			}
		})
	}
}

func TestFetchUsage_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.SetEndpoint(server.URL)

	_, err := client.FetchUsage("test-token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); !contains(got, "parsing response JSON") {
		t.Errorf("error = %q, want to contain %q", got, "parsing response JSON")
	}
}

func TestFetchUsage_AllFieldsNull(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"five_hour": null, "seven_day": null, "seven_day_oauth_apps": null, "seven_day_opus": null}`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.SetEndpoint(server.URL)

	usage, err := client.FetchUsage("test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.FiveHour != nil {
		t.Errorf("FiveHour = %v, want nil", usage.FiveHour)
	}
	if usage.SevenDay != nil {
		t.Errorf("SevenDay = %v, want nil", usage.SevenDay)
	}
}

func TestFetchUsage_HighUtilization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"five_hour": {"utilization": 99.5, "resets_at": "2025-11-04T04:59:59+00:00"},
			"seven_day": {"utilization": 100.0, "resets_at": "2025-11-06T03:59:59+00:00"}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.Client())
	client.SetEndpoint(server.URL)

	usage, err := client.FetchUsage("test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.FiveHour.Utilization != 99.5 {
		t.Errorf("FiveHour.Utilization = %v, want 99.5", usage.FiveHour.Utilization)
	}
	if usage.SevenDay.Utilization != 100.0 {
		t.Errorf("SevenDay.Utilization = %v, want 100.0", usage.SevenDay.Utilization)
	}
}

func TestNewClient_NilHTTPClient(t *testing.T) {
	client := NewClient(nil)
	if client.httpClient != http.DefaultClient {
		t.Error("expected http.DefaultClient when nil is passed")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
