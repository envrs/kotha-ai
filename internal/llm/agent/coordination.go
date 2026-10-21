package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Coordination errors.
var (
	ErrFileLocked = errors.New("file is locked by another agent session")
)

// FileLock coordinates concurrent edits to the same file across multiple
// agent sessions. A lock is held for a bounded duration and is released
// when the owning session's context is cancelled.
type FileLock struct {
	mu        sync.Mutex
	locks     map[string]*fileLock
	defaultTTL time.Duration
}

type fileLock struct {
	sessionID string
	deadline  time.Time
	released  bool
}

// NewFileLock returns a FileLock with the given default TTL.
func NewFileLock(ttl time.Duration) *FileLock {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &FileLock{
		locks:     make(map[string]*fileLock),
		defaultTTL: ttl,
	}
}

// Acquire tries to acquire a lock for the given file. It blocks until the
// lock is available or ctx is cancelled. Returns ErrFileLocked only when
// the file is already held by another session and the context is done.
func (fl *FileLock) Acquire(ctx context.Context, sessionID, filePath string) error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	for {
		now := time.Now()
		// Purge expired locks.
		for path, l := range fl.locks {
			if l.released || now.After(l.deadline) {
				delete(fl.locks, path)
			}
		}
		existing, ok := fl.locks[filePath]
		if !ok || existing.sessionID == sessionID {
			fl.locks[filePath] = &fileLock{
				sessionID: sessionID,
				deadline:  now.Add(fl.defaultTTL),
			}
			return nil
		}
		// Another session holds the lock; wait or bail.
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w: %s", ErrFileLocked, filePath)
		case <-time.After(100 * time.Millisecond):
			// Re-check the lock state.
		}
	}
}

// Release frees the lock held by the given session for the file.
func (fl *FileLock) Release(sessionID, filePath string) {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	if l, ok := fl.locks[filePath]; ok && l.sessionID == sessionID {
		l.released = true
		delete(fl.locks, filePath)
	}
}

// IsLocked reports whether the file is currently held by another session.
func (fl *FileLock) IsLocked(sessionID, filePath string) bool {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	l, ok := fl.locks[filePath]
	if !ok {
		return false
	}
	return l.sessionID != sessionID
}