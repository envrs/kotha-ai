package shell

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/kothagpt/kotha/internal/config"
	"github.com/stretchr/testify/require"
)

func newTestShell(t *testing.T) *PersistentShell {
	t.Helper()
	ResetPersistentShell()
	dir := t.TempDir()
	config.SetDefaultProvider(config.NewDefaultProvider(&config.Config{WorkingDir: dir}))
	_ = os.Setenv("SHELL", "/bin/bash")
	t.Cleanup(func() { _ = os.Unsetenv("SHELL") })
	return GetPersistentShell(dir)
}

func TestStartBackgroundAndRead(t *testing.T) {
	shell := newTestShell(t)
	defer shell.Close()

	bc, err := shell.StartBackground(context.Background(), "echo hello", 5000)
	require.NoError(t, err)
	require.NotEmpty(t, bc.ID)
	require.NotEmpty(t, bc.OutputFile)
	require.NotEmpty(t, bc.ErrorFile)

	// Wait for completion.
	require.Eventually(t, func() bool {
		return bc.IsDone()
	}, 2*time.Second, 10*time.Millisecond)

	res := bc.Result()
	require.NotNil(t, res)
	require.Contains(t, res.Stdout, "hello")
}

func TestStartBackgroundCancellation(t *testing.T) {
	shell := newTestShell(t)
	defer shell.Close()

	ctx, cancel := context.WithCancel(context.Background())
	bc, err := shell.StartBackground(ctx, "sleep 30", 60000)
	require.NoError(t, err)

	cancel()
	require.Eventually(t, func() bool {
		return bc.IsDone()
	}, 2*time.Second, 10*time.Millisecond)

	res := bc.Result()
	require.NotNil(t, res)
	require.True(t, res.Interrupted)
}

func TestGetBackgroundNotFound(t *testing.T) {
	shell := newTestShell(t)
	defer shell.Close()
	require.Nil(t, shell.GetBackground("nonexistent"))
}

func TestListBackgrounds(t *testing.T) {
	shell := newTestShell(t)
	defer shell.Close()

	bc1, _ := shell.StartBackground(context.Background(), "echo one", 5000)
	bc2, _ := shell.StartBackground(context.Background(), "echo two", 5000)

	list := shell.ListBackgrounds()
	require.Len(t, list, 2)
	ids := map[string]bool{bc1.ID: true, bc2.ID: true}
	for _, bc := range list {
		require.True(t, ids[bc.ID])
	}
}

func TestCancelBackground(t *testing.T) {
	shell := newTestShell(t)
	defer shell.Close()

	bc, err := shell.StartBackground(context.Background(), "sleep 300", 60000)
	require.NoError(t, err)

	// First cancel should succeed.
	require.True(t, shell.CancelBackground(bc.ID))
	// Second cancel should fail (already cancelled/done).
	require.False(t, shell.CancelBackground(bc.ID))
}

func TestReadFileOrEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0o600))
	require.Equal(t, "hello", ReadFileOrEmpty(path))
	require.Equal(t, "", ReadFileOrEmpty(filepath.Join(dir, "missing.txt")))
}

func TestConcurrentBackgroundCommands(t *testing.T) {
	shell := newTestShell(t)
	defer shell.Close()

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bc, err := shell.StartBackground(context.Background(), "echo ok", 5000)
			if err != nil {
				return
			}
			require.Eventually(t, func() bool {
				return bc.IsDone()
			}, 2*time.Second, 10*time.Millisecond)
		}()
	}
	wg.Wait()
}