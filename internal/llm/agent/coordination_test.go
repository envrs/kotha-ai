package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFileLockAcquireAndRelease(t *testing.T) {
	fl := NewFileLock(10 * time.Second)
	ctx := context.Background()

	require.NoError(t, fl.Acquire(ctx, "s1", "file.go"))
	require.NoError(t, fl.Acquire(ctx, "s1", "file.go")) // same session re-acquires
	require.True(t, fl.IsLocked("s2", "file.go"))
	fl.Release("s1", "file.go")
	require.False(t, fl.IsLocked("s2", "file.go"))
}

func TestFileLockBlocksOtherSession(t *testing.T) {
	fl := NewFileLock(10 * time.Second)
	ctx := context.Background()

	require.NoError(t, fl.Acquire(ctx, "s1", "file.go"))

	errCh := make(chan error, 1)
	go func() {
		errCh <- fl.Acquire(ctx, "s2", "file.go")
	}()

	select {
	case err := <-errCh:
		t.Fatalf("expected blocking, got %v", err)
	case <-time.After(50 * time.Millisecond):
		// Expected: s2 is still waiting.
	}

	fl.Release("s1", "file.go")

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for s2 to acquire")
	}
}

func TestFileLockTimeout(t *testing.T) {
	fl := NewFileLock(10 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	require.NoError(t, fl.Acquire(ctx, "s1", "file.go"))
	err := fl.Acquire(ctx, "s2", "file.go")
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrFileLocked) || errors.Is(err, context.DeadlineExceeded))
}

func TestFileLockTTL(t *testing.T) {
	fl := NewFileLock(50 * time.Millisecond)
	ctx := context.Background()

	require.NoError(t, fl.Acquire(ctx, "s1", "file.go"))
	time.Sleep(100 * time.Millisecond) // wait for TTL

	// s2 should now be able to acquire because s1's lock expired.
	require.NoError(t, fl.Acquire(ctx, "s2", "file.go"))
}

func TestFileLockConcurrentSessions(t *testing.T) {
	fl := NewFileLock(100 * time.Millisecond)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			sess := fmt.Sprintf("s%d", id)
			require.NoError(t, fl.Acquire(ctx, sess, "shared.go"))
			fl.Release(sess, "shared.go")
		}(i)
	}
	wg.Wait()
}

func TestExtractFilePath(t *testing.T) {
	require.Equal(t, "foo.go", extractFilePath(`{"file_path":"foo.go","old_string":"a"}`))
	require.Equal(t, "", extractFilePath(`{"url":"http://example.com"}`))
	require.Equal(t, "", extractFilePath(""))
}