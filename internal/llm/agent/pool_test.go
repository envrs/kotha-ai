package agent

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPoolAcquireAndRelease(t *testing.T) {
	calls := 0
	factory := func() (int, error) {
		calls++
		return calls, nil
	}
	pool := NewPool[int](factory, nil, 4)

	r1, err := pool.Acquire()
	require.NoError(t, err)
	require.Equal(t, 1, r1)

	r2, err := pool.Acquire()
	require.NoError(t, err)
	require.Equal(t, 2, r2)

	pool.Release(r1)
	pool.Release(r2)

	// Re-acquire should reuse the released resources.
	r3, err := pool.Acquire()
	require.NoError(t, err)
	require.Equal(t, 1, r3)

	r4, err := pool.Acquire()
	require.NoError(t, err)
	require.Equal(t, 2, r4)

	require.Equal(t, 2, calls) // factory only called twice
}

func TestPoolMaxSize(t *testing.T) {
	calls := 0
	factory := func() (int, error) {
		calls++
		return calls, nil
	}
	pool := NewPool[int](factory, nil, 2)

	r1, _ := pool.Acquire()
	_, _ = pool.Acquire()
	require.Equal(t, 2, calls)

	// Acquire a third resource; should block until one is released.
	errCh := make(chan error, 1)
	go func() {
		_, err := pool.Acquire()
		errCh <- err
	}()

	select {
	case <-errCh:
		t.Fatal("expected blocking acquire")
	case <-time.After(50 * time.Millisecond):
		// Expected.
	}

	pool.Release(r1)
	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for acquire")
	}
}

func TestPoolFactoryError(t *testing.T) {
	factory := func() (int, error) {
		return 0, errors.New("factory failed")
	}
	pool := NewPool[int](factory, nil, 4)

	_, err := pool.Acquire()
	require.Error(t, err)
}

func TestPoolClose(t *testing.T) {
	closed := false
	factory := func() (int, error) {
		return 1, nil
	}
	close := func(r int) error {
		closed = true
		return nil
	}
	pool := NewPool[int](factory, close, 4)

	r, _ := pool.Acquire()
	pool.Release(r)
	require.NoError(t, pool.Close())
	require.True(t, closed)
}

func TestPoolConcurrent(t *testing.T) {
	calls := 0
	var mu sync.Mutex
	factory := func() (int, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		return calls, nil
	}
	pool := NewPool[int](factory, nil, 4)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := pool.Acquire()
			if err != nil {
				return
			}
			time.Sleep(time.Millisecond)
			pool.Release(r)
		}()
	}
	wg.Wait()
}