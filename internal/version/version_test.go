package version

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionSet(t *testing.T) {
	// init() resolves Version from build info or leaves the ldflags value;
	// either way it must be non-empty.
	require.NotEmpty(t, Version)
}
