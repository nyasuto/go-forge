package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gf-claude-quota/internal/api"
)

const defaultCacheDir = ".cache/gf-claude-quota"
const defaultCacheFile = "usage.json"

// cacheEntry wraps the usage response with metadata for TTL management.
type cacheEntry struct {
	Usage     *api.UsageResponse `json:"usage"`
	FetchedAt time.Time          `json:"fetched_at"`
}

// FileCache provides file-based caching for API responses.
type FileCache struct {
	dir     string
	ttl     time.Duration
	nowFunc func() time.Time
}

// NewFileCache creates a new FileCache with the given TTL.
// If dir is empty, uses ~/.cache/gf-claude-quota/.
func NewFileCache(dir string, ttl time.Duration) *FileCache {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		dir = filepath.Join(home, defaultCacheDir)
	}
	return &FileCache{
		dir:     dir,
		ttl:     ttl,
		nowFunc: time.Now,
	}
}

func (fc *FileCache) path() string {
	return filepath.Join(fc.dir, defaultCacheFile)
}

// Get returns cached usage data if available and not stale.
// Returns nil, nil if cache is missing or expired.
func (fc *FileCache) Get() (*api.UsageResponse, error) {
	data, err := os.ReadFile(fc.path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading cache: %w", err)
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		// Corrupt cache — treat as miss
		return nil, nil
	}

	if fc.isStale(entry.FetchedAt) {
		return nil, nil
	}

	return entry.Usage, nil
}

// Set writes usage data to the cache file.
func (fc *FileCache) Set(usage *api.UsageResponse) error {
	if err := os.MkdirAll(fc.dir, 0o700); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	entry := cacheEntry{
		Usage:     usage,
		FetchedAt: fc.nowFunc(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling cache: %w", err)
	}

	// Write atomically via temp file + rename
	tmp := fc.path() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("writing cache: %w", err)
	}
	if err := os.Rename(tmp, fc.path()); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("renaming cache: %w", err)
	}

	return nil
}

func (fc *FileCache) isStale(fetchedAt time.Time) bool {
	return fc.nowFunc().Sub(fetchedAt) > fc.ttl
}
