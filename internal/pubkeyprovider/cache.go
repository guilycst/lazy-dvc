package pubkeyprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

const (
	DefaultCacheTTL     = 5 * time.Minute
	DefaultLockDuration = 3 * time.Second
	DefaultCacheDir     = "/var/cache/lazy-dvc"
)

type CacheEntry struct {
	Keys       []string  `json:"keys"`
	ValidUntil time.Time `json:"valid_until"`
}

type LockEntry struct {
	PID        int       `json:"pid"`
	ValidUntil time.Time `json:"valid_until"`
}

type CachedProvider struct {
	delegate Provider
	cacheDir string
	ttl      time.Duration
	disabled bool
}

type CachedProviderOption func(*CachedProvider)

func WithCacheDir(dir string) CachedProviderOption {
	return func(c *CachedProvider) {
		c.cacheDir = dir
	}
}

func WithCacheTTL(ttl time.Duration) CachedProviderOption {
	return func(c *CachedProvider) {
		c.ttl = ttl
	}
}

func WithCacheDisabled(disabled bool) CachedProviderOption {
	return func(c *CachedProvider) {
		c.disabled = disabled
	}
}

func NewCachedProvider(delegate Provider, opts ...CachedProviderOption) *CachedProvider {
	c := &CachedProvider{
		delegate: delegate,
		cacheDir: DefaultCacheDir,
		ttl:      DefaultCacheTTL,
		disabled: false,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *CachedProvider) GetUsersPublicKeys(ctx context.Context, orgName string, opts ...UsersPublicKeysOption) ([]string, error) {
	if c.disabled {
		slog.Info("Fetching keys from GitHub API (cache disabled)")
		return c.delegate.GetUsersPublicKeys(ctx, orgName, opts...)
	}

	cacheFile := filepath.Join(c.cacheDir, "keys.json")
	lockFile := filepath.Join(c.cacheDir, "keys.lock")

	if err := os.MkdirAll(c.cacheDir, 0755); err != nil {
		slog.Info("Fetching keys from GitHub API (cache unavailable)", "error", err)
		return c.delegate.GetUsersPublicKeys(ctx, orgName, opts...)
	}

	if keys, err := c.readCache(cacheFile); err == nil {
		slog.Info("Cache hit", "num_keys", len(keys))
		return keys, nil
	}

	slog.Info("Cache miss, fetching keys from GitHub API")

	if err := c.acquireLock(lockFile); err != nil {
		slog.Debug("Lock acquisition failed, rechecking cache", "error", err)
		if keys, err := c.readCache(cacheFile); err == nil {
			slog.Info("Cache hit (after lock contention)", "num_keys", len(keys))
			return keys, nil
		}
		return nil, fmt.Errorf("failed to acquire lock and no valid cache: %w", err)
	}
	defer c.releaseLock(lockFile)

	keys, err := c.delegate.GetUsersPublicKeys(ctx, orgName, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch keys from GitHub API: %w", err)
	}

	if err := c.writeCache(cacheFile, keys); err != nil {
		slog.Info("Fetched keys from GitHub API (cache write failed)", "num_keys", len(keys))
	} else {
		slog.Info("Fetched keys from GitHub API and cached", "num_keys", len(keys))
	}

	return keys, nil
}

func (c *CachedProvider) readCache(cacheFile string) ([]string, error) {
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache: %w", err)
	}

	if time.Now().After(entry.ValidUntil) {
		return nil, fmt.Errorf("cache expired")
	}

	return entry.Keys, nil
}

func (c *CachedProvider) writeCache(cacheFile string, keys []string) error {
	entry := CacheEntry{
		Keys:       keys,
		ValidUntil: time.Now().Add(c.ttl),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	tmpFile := cacheFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	if err := os.Rename(tmpFile, cacheFile); err != nil {
		return fmt.Errorf("failed to rename cache file: %w", err)
	}

	return nil
}

func (c *CachedProvider) acquireLock(lockFile string) error {
	data, err := os.ReadFile(lockFile)
	if err == nil {
		var lock LockEntry
		if err := json.Unmarshal(data, &lock); err == nil {
			if !c.isLockStale(&lock) {
				return fmt.Errorf("lock held by process %d", lock.PID)
			}
			slog.Debug("Lock is stale, taking over", "pid", lock.PID)
		}
	}

	//TODO: Should error here if error is not "file not found"

	lock := LockEntry{
		PID:        os.Getpid(),
		ValidUntil: time.Now().Add(DefaultLockDuration),
	}

	data, err = json.Marshal(lock)
	if err != nil {
		return fmt.Errorf("failed to marshal lock: %w", err)
	}

	if err := os.WriteFile(lockFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}

	return nil
}

func (c *CachedProvider) isLockStale(lock *LockEntry) bool {
	if time.Now().After(lock.ValidUntil) {
		return true
	}

	process, err := os.FindProcess(lock.PID)
	if err != nil {
		return true
	}

	if err := process.Signal(syscall.Signal(0)); err != nil {
		return true
	}

	return false
}

func (c *CachedProvider) releaseLock(lockFile string) {
	if err := os.Remove(lockFile); err != nil {
		slog.Debug("Failed to remove lock file", "error", err)
	}
}

var _ Provider = (*CachedProvider)(nil)
