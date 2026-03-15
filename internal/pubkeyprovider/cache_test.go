package pubkeyprovider

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockProvider struct {
	keys    []string
	err     error
	called  int
	callErr bool
}

func (m *MockProvider) GetUsersPublicKeys(ctx context.Context, orgName string, opts ...UsersPublicKeysOption) ([]string, error) {
	m.called++
	if m.callErr {
		return nil, errors.New("mock error")
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.keys, nil
}

func TestCachedProvider_CacheHit(t *testing.T) {
	// ARRANGE
	tempDir := t.TempDir()
	keys := []string{"ssh-rsa AAAA...", "ssh-ed25519 BBB..."}

	delegate := &MockProvider{keys: keys}
	cached := NewCachedProvider(delegate, WithCacheDir(tempDir))

	cacheFile := filepath.Join(tempDir, "keys.json")
	entry := CacheEntry{
		Keys:       keys,
		ValidUntil: time.Now().Add(5 * time.Minute),
	}
	data, _ := json.Marshal(entry)
	_ = os.WriteFile(cacheFile, data, 0644)

	// ACT
	result, err := cached.GetUsersPublicKeys(context.Background(), "test-org")

	// ASSERT
	require.NoError(t, err)
	assert.Equal(t, keys, result)
	assert.Equal(t, 0, delegate.called, "should not call delegate on cache hit")
}

func TestCachedProvider_CacheMiss(t *testing.T) {
	// ARRANGE
	tempDir := t.TempDir()
	keys := []string{"ssh-rsa AAAA...", "ssh-ed25519 BBB..."}

	delegate := &MockProvider{keys: keys}
	cached := NewCachedProvider(delegate, WithCacheDir(tempDir))

	// ACT
	result, err := cached.GetUsersPublicKeys(context.Background(), "test-org")

	// ASSERT
	require.NoError(t, err)
	assert.Equal(t, keys, result)
	assert.Equal(t, 1, delegate.called, "should call delegate on cache miss")

	// Verify cache was written
	cacheFile := filepath.Join(tempDir, "keys.json")
	data, err := os.ReadFile(cacheFile)
	require.NoError(t, err)

	var entry CacheEntry
	require.NoError(t, json.Unmarshal(data, &entry))
	assert.Equal(t, keys, entry.Keys)
}

func TestCachedProvider_CacheExpired(t *testing.T) {
	// ARRANGE
	tempDir := t.TempDir()
	staleKeys := []string{"ssh-rsa OLD..."}
	freshKeys := []string{"ssh-rsa NEW..."}

	cacheFile := filepath.Join(tempDir, "keys.json")
	entry := CacheEntry{
		Keys:       staleKeys,
		ValidUntil: time.Now().Add(-1 * time.Minute), // Expired
	}
	data, _ := json.Marshal(entry)
	_ = os.WriteFile(cacheFile, data, 0644)

	delegate := &MockProvider{keys: freshKeys}
	cached := NewCachedProvider(delegate, WithCacheDir(tempDir))

	// ACT
	result, err := cached.GetUsersPublicKeys(context.Background(), "test-org")

	// ASSERT
	require.NoError(t, err)
	assert.Equal(t, freshKeys, result)
	assert.Equal(t, 1, delegate.called, "should call delegate when cache expired")
}

func TestCachedProvider_CacheDisabled(t *testing.T) {
	// ARRANGE
	tempDir := t.TempDir()
	keys := []string{"ssh-rsa AAAA...", "ssh-ed25519 BBB..."}

	// Write a valid cache
	cacheFile := filepath.Join(tempDir, "keys.json")
	entry := CacheEntry{
		Keys:       keys,
		ValidUntil: time.Now().Add(5 * time.Minute),
	}
	data, _ := json.Marshal(entry)
	_ = os.WriteFile(cacheFile, data, 0644)

	delegate := &MockProvider{keys: keys}
	cached := NewCachedProvider(delegate, WithCacheDir(tempDir), WithCacheDisabled(true))

	// ACT
	result, err := cached.GetUsersPublicKeys(context.Background(), "test-org")

	// ASSERT
	require.NoError(t, err)
	assert.Equal(t, keys, result)
	assert.Equal(t, 1, delegate.called, "should call delegate when cache disabled")
}

func TestCachedProvider_DelegateError(t *testing.T) {
	// ARRANGE
	tempDir := t.TempDir()

	delegate := &MockProvider{err: errors.New("api error")}
	cached := NewCachedProvider(delegate, WithCacheDir(tempDir))

	// ACT
	result, err := cached.GetUsersPublicKeys(context.Background(), "test-org")

	// ASSERT
	require.Error(t, err)
	assert.Nil(t, result)
}

func TestLockFile_Acquire(t *testing.T) {
	// ARRANGE
	tempDir := t.TempDir()
	lockFile := filepath.Join(tempDir, "keys.lock")

	delegate := &MockProvider{keys: []string{"ssh-rsa AAAA..."}}
	cached := NewCachedProvider(delegate, WithCacheDir(tempDir))

	// ACT
	err := cached.acquireLock(lockFile)

	// ASSERT
	require.NoError(t, err)

	data, err := os.ReadFile(lockFile)
	require.NoError(t, err)

	var lock LockEntry
	require.NoError(t, json.Unmarshal(data, &lock))
	assert.Equal(t, os.Getpid(), lock.PID)
	assert.True(t, lock.ValidUntil.After(time.Now()))

	// Cleanup
	cached.releaseLock(lockFile)
}

func TestLockFile_Stale_Expired(t *testing.T) {
	// ARRANGE
	tempDir := t.TempDir()
	lockFile := filepath.Join(tempDir, "keys.lock")

	staleLock := LockEntry{
		PID:        99999,
		ValidUntil: time.Now().Add(-1 * time.Minute), // Expired
	}
	data, _ := json.Marshal(staleLock)
	_ = os.WriteFile(lockFile, data, 0644)

	delegate := &MockProvider{keys: []string{"ssh-rsa AAAA..."}}
	cached := NewCachedProvider(delegate, WithCacheDir(tempDir))

	// ACT
	err := cached.acquireLock(lockFile)

	// ASSERT
	require.NoError(t, err, "should take over expired lock")

	data, err = os.ReadFile(lockFile)
	require.NoError(t, err)

	var lock LockEntry
	require.NoError(t, json.Unmarshal(data, &lock))
	assert.Equal(t, os.Getpid(), lock.PID)

	// Cleanup
	cached.releaseLock(lockFile)
}

func TestLockFile_Stale_DeadProcess(t *testing.T) {
	// ARRANGE
	tempDir := t.TempDir()
	lockFile := filepath.Join(tempDir, "keys.lock")

	staleLock := LockEntry{
		PID:        99999, // Non-existent PID
		ValidUntil: time.Now().Add(5 * time.Minute),
	}
	data, _ := json.Marshal(staleLock)
	_ = os.WriteFile(lockFile, data, 0644)

	delegate := &MockProvider{keys: []string{"ssh-rsa AAAA..."}}
	cached := NewCachedProvider(delegate, WithCacheDir(tempDir))

	// ACT
	err := cached.acquireLock(lockFile)

	// ASSERT
	require.NoError(t, err, "should take over lock from dead process")

	// Cleanup
	cached.releaseLock(lockFile)
}

func TestLockFile_Blocked_ByValidLock(t *testing.T) {
	// ARRANGE
	tempDir := t.TempDir()
	lockFile := filepath.Join(tempDir, "keys.lock")

	validLock := LockEntry{
		PID:        os.Getpid(), // Current process (always running)
		ValidUntil: time.Now().Add(5 * time.Minute),
	}
	data, _ := json.Marshal(validLock)
	_ = os.WriteFile(lockFile, data, 0644)

	delegate := &MockProvider{keys: []string{"ssh-rsa AAAA..."}}
	cached := NewCachedProvider(delegate, WithCacheDir(tempDir))

	// ACT
	err := cached.acquireLock(lockFile)

	// ASSERT
	require.Error(t, err, "should fail to acquire when lock held by running process")
}
