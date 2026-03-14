package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gf-claude-quota/internal/api"
)

func sampleUsage() *api.UsageResponse {
	reset5h := "2025-11-04T04:59:59+00:00"
	reset7d := "2025-11-06T03:59:59+00:00"
	return &api.UsageResponse{
		FiveHour: &api.UsageWindow{
			Utilization: 42.0,
			ResetsAt:    &reset5h,
		},
		SevenDay: &api.UsageWindow{
			Utilization: 18.0,
			ResetsAt:    &reset7d,
		},
		SevenDayOpus: &api.UsageWindow{
			Utilization: 0.0,
			ResetsAt:    nil,
		},
	}
}

func TestFileCache_SetAndGet(t *testing.T) {
	dir := t.TempDir()
	fc := NewFileCache(dir, 60*time.Second)
	fixedNow := time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC)
	fc.nowFunc = func() time.Time { return fixedNow }

	usage := sampleUsage()

	// Set
	if err := fc.Set(usage); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(fc.path()); err != nil {
		t.Fatalf("cache file not created: %v", err)
	}

	// Get — should return cached data
	got, err := fc.Get()
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got == nil {
		t.Fatal("Get() returned nil, want cached data")
	}
	if got.FiveHour.Utilization != 42.0 {
		t.Errorf("FiveHour.Utilization = %v, want 42.0", got.FiveHour.Utilization)
	}
	if got.SevenDay.Utilization != 18.0 {
		t.Errorf("SevenDay.Utilization = %v, want 18.0", got.SevenDay.Utilization)
	}
}

func TestFileCache_GetStale(t *testing.T) {
	dir := t.TempDir()
	fc := NewFileCache(dir, 60*time.Second)

	writeTime := time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC)
	fc.nowFunc = func() time.Time { return writeTime }

	if err := fc.Set(sampleUsage()); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	// Advance time past TTL
	fc.nowFunc = func() time.Time { return writeTime.Add(61 * time.Second) }

	got, err := fc.Get()
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got != nil {
		t.Errorf("Get() returned data for stale cache, want nil")
	}
}

func TestFileCache_GetNotStale(t *testing.T) {
	dir := t.TempDir()
	fc := NewFileCache(dir, 60*time.Second)

	writeTime := time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC)
	fc.nowFunc = func() time.Time { return writeTime }

	if err := fc.Set(sampleUsage()); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	// Still within TTL
	fc.nowFunc = func() time.Time { return writeTime.Add(59 * time.Second) }

	got, err := fc.Get()
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got == nil {
		t.Error("Get() returned nil for fresh cache, want data")
	}
}

func TestFileCache_GetMissing(t *testing.T) {
	dir := t.TempDir()
	fc := NewFileCache(dir, 60*time.Second)

	got, err := fc.Get()
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got != nil {
		t.Errorf("Get() returned data for missing cache, want nil")
	}
}

func TestFileCache_GetCorruptJSON(t *testing.T) {
	dir := t.TempDir()
	fc := NewFileCache(dir, 60*time.Second)

	// Write corrupt data
	os.MkdirAll(dir, 0o700)
	os.WriteFile(fc.path(), []byte("not json"), 0o600)

	got, err := fc.Get()
	if err != nil {
		t.Fatalf("Get() error: %v (should treat corrupt as miss)", err)
	}
	if got != nil {
		t.Errorf("Get() returned data for corrupt cache, want nil")
	}
}

func TestFileCache_SetCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")
	fc := NewFileCache(dir, 60*time.Second)

	if err := fc.Set(sampleUsage()); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	if _, err := os.Stat(fc.path()); err != nil {
		t.Errorf("cache file not created in nested dir: %v", err)
	}
}

func TestFileCache_SetOverwrite(t *testing.T) {
	dir := t.TempDir()
	fc := NewFileCache(dir, 60*time.Second)
	fc.nowFunc = func() time.Time { return time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC) }

	usage1 := sampleUsage()
	if err := fc.Set(usage1); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	// Overwrite with different data
	usage2 := &api.UsageResponse{
		FiveHour: &api.UsageWindow{Utilization: 99.0},
	}
	if err := fc.Set(usage2); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	got, err := fc.Get()
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got.FiveHour.Utilization != 99.0 {
		t.Errorf("Utilization = %v, want 99.0", got.FiveHour.Utilization)
	}
}

func TestFileCache_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	fc := NewFileCache(dir, 60*time.Second)

	if err := fc.Set(sampleUsage()); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	info, err := os.Stat(fc.path())
	if err != nil {
		t.Fatalf("Stat() error: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("file permissions = %o, want 600", perm)
	}
}

func TestFileCache_CacheFileContents(t *testing.T) {
	dir := t.TempDir()
	fc := NewFileCache(dir, 60*time.Second)
	fixedNow := time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC)
	fc.nowFunc = func() time.Time { return fixedNow }

	if err := fc.Set(sampleUsage()); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	data, err := os.ReadFile(fc.path())
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if !entry.FetchedAt.Equal(fixedNow) {
		t.Errorf("FetchedAt = %v, want %v", entry.FetchedAt, fixedNow)
	}
	if entry.Usage == nil {
		t.Fatal("Usage is nil")
	}
	if entry.Usage.FiveHour.Utilization != 42.0 {
		t.Errorf("Usage.FiveHour.Utilization = %v, want 42.0", entry.Usage.FiveHour.Utilization)
	}
}

func TestFileCache_DefaultDir(t *testing.T) {
	fc := NewFileCache("", 60*time.Second)
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".cache", "gf-claude-quota")
	if fc.dir != expected {
		t.Errorf("dir = %q, want %q", fc.dir, expected)
	}
}

func TestFileCache_NullFields(t *testing.T) {
	dir := t.TempDir()
	fc := NewFileCache(dir, 60*time.Second)
	fc.nowFunc = func() time.Time { return time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC) }

	usage := &api.UsageResponse{
		FiveHour:      nil,
		SevenDay:      nil,
		SevenDayOAuth: nil,
		SevenDayOpus:  nil,
	}

	if err := fc.Set(usage); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	got, err := fc.Get()
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got == nil {
		t.Fatal("Get() returned nil")
	}
	if got.FiveHour != nil {
		t.Error("FiveHour should be nil")
	}
}

func TestFileCache_CustomTTL(t *testing.T) {
	dir := t.TempDir()
	fc := NewFileCache(dir, 5*time.Second)

	writeTime := time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC)
	fc.nowFunc = func() time.Time { return writeTime }

	if err := fc.Set(sampleUsage()); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	// 4 seconds — still fresh
	fc.nowFunc = func() time.Time { return writeTime.Add(4 * time.Second) }
	got, _ := fc.Get()
	if got == nil {
		t.Error("cache should be fresh at 4s with 5s TTL")
	}

	// 6 seconds — stale
	fc.nowFunc = func() time.Time { return writeTime.Add(6 * time.Second) }
	got, _ = fc.Get()
	if got != nil {
		t.Error("cache should be stale at 6s with 5s TTL")
	}
}

func TestFileCache_NoTokenInCache(t *testing.T) {
	dir := t.TempDir()
	fc := NewFileCache(dir, 60*time.Second)
	fc.nowFunc = func() time.Time { return time.Date(2025, 11, 4, 2, 30, 0, 0, time.UTC) }

	if err := fc.Set(sampleUsage()); err != nil {
		t.Fatalf("Set() error: %v", err)
	}

	data, err := os.ReadFile(fc.path())
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	content := string(data)
	// Ensure no token-like strings appear in cache
	for _, pattern := range []string{"sk-ant", "Bearer", "accessToken", "refreshToken"} {
		if contains(content, pattern) {
			t.Errorf("cache file contains %q — tokens must not be cached", pattern)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
