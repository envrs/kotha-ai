package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitDialogFlow(t *testing.T) {
	old := defaultProviderInstance
	defer func() { defaultProviderInstance = old }()

	dir := t.TempDir()
	SetDefaultProvider(NewDefaultProvider(&Config{Data: Data{Directory: dir}}))

	show, err := ShouldShowInitDialog()
	require.NoError(t, err)
	require.True(t, show)

	require.NoError(t, MarkProjectInitialized())

	show, err = ShouldShowInitDialog()
	require.NoError(t, err)
	require.False(t, show)
}

func TestInitDialogNoConfig(t *testing.T) {
	old := defaultProviderInstance
	defer func() { defaultProviderInstance = old }()

	SetDefaultProvider(NewDefaultProvider(nil))
	_, err := ShouldShowInitDialog()
	require.Error(t, err)
	require.Error(t, MarkProjectInitialized())
}
