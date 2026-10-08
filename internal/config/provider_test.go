package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProviderWorkingDirectory(t *testing.T) {
	p := NewDefaultProvider(&Config{WorkingDir: "/tmp/wd"})
	require.Equal(t, "/tmp/wd", p.WorkingDirectory())
	require.Equal(t, "/tmp/wd", p.Get().WorkingDir)
}

func TestProviderNilConfigPanics(t *testing.T) {
	p := NewDefaultProvider(nil)
	require.Panics(t, func() { _ = p.WorkingDirectory() })
}

func TestSetGetDefaultProvider(t *testing.T) {
	old := defaultProviderInstance
	defer func() { defaultProviderInstance = old }()

	SetDefaultProvider(NewDefaultProvider(&Config{WorkingDir: "/x"}))
	require.Equal(t, "/x", WorkingDirectory())
	require.NotNil(t, Get())
}
