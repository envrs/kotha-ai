package diff

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const sampleDiff = `--- a/foo.go
+++ b/foo.go
@@ -1,3 +1,3 @@
 line1
-old
+new
 line3
`

func TestParseUnifiedDiff(t *testing.T) {
	res, err := ParseUnifiedDiff(sampleDiff)
	require.NoError(t, err)
	require.Equal(t, "foo.go", res.OldFile)
	require.Equal(t, "foo.go", res.NewFile)
	require.Len(t, res.Hunks, 1)
	require.Len(t, res.Hunks[0].Lines, 5)

	kinds := []LineType{LineContext, LineRemoved, LineAdded, LineContext, LineContext}
	for i, k := range kinds {
		require.Equal(t, k, res.Hunks[0].Lines[i].Kind)
	}
}

func TestParseEmpty(t *testing.T) {
	res, err := ParseUnifiedDiff("")
	require.NoError(t, err)
	require.Empty(t, res.Hunks)
}

func TestSideBySideOptions(t *testing.T) {
	c := NewSideBySideConfig(WithTotalWidth(100))
	require.Equal(t, 100, c.TotalWidth)
	require.NotNil(t, WithContextSize(3))
}

func TestFormatDiff(t *testing.T) {
	out, err := FormatDiff(sampleDiff)
	require.NoError(t, err)
	require.Contains(t, out, "line1")

	_, err = FormatDiff("")
	require.NoError(t, err)
}

func TestDiffErrorTypes(t *testing.T) {
	require.Error(t, NewDiffError("x"))
	require.Error(t, fileError("op", "why", "p"))
	require.Error(t, contextError(3, "ctx", false))
	require.Error(t, contextError(3, "ctx", true))
}

func TestParser(t *testing.T) {
	p := NewParser(map[string]string{"f.txt": "hello\n"}, []string{"some line"})
	require.False(t, p.isDone([]string{"zzz"}))
	require.NotPanics(t, func() { _ = p.Parse() })
}
