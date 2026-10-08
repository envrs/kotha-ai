package logging

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLevelFunctionsDoNotPanic(t *testing.T) {
	require.NotPanics(t, func() {
		Info("i", "k", "v")
		Debug("d")
		Warn("w")
		Error("e")
		InfoPersist("ip")
		DebugPersist("dp")
		WarnPersist("wp")
		ErrorPersist("ep")
	})
}

func TestRecoverPanic(t *testing.T) {
	// no panic: no-op
	require.NotPanics(t, func() { RecoverPanic("test", nil) })

	// with panic + cleanup; log file goes to os.TempDir(), never CWD
	called := false
	require.NotPanics(t, func() {
		defer RecoverPanic("test", func() { called = true })
		panic("boom")
	})
	require.True(t, called)
	matches, _ := filepath.Glob(filepath.Join(os.TempDir(), "kotha-panic-test-*.log"))
	require.NotEmpty(t, matches)
	// cleanup the panic log we created
	for _, m := range matches {
		_ = os.Remove(m)
	}

	// panicking cleanup must not propagate
	require.NotPanics(t, func() {
		defer RecoverPanic("test-cleanup", func() { panic("cleanup-boom") })
		panic("boom")
	})
}

func TestSessionLogGuards(t *testing.T) {
	old := MessageDir
	defer func() { MessageDir = old }()
	MessageDir = ""
	require.Equal(t, "", AppendToSessionLogFile("sess", "f", "c"))
	require.Equal(t, "", WriteRequestMessage("s", 1, "m"))
	require.Equal(t, "", WriteRequestMessageJson("s", 1, map[string]string{"a": "b"}))
	require.Equal(t, "", AppendToStreamSessionLog("s", 1, "c"))
	require.Equal(t, "", AppendToStreamSessionLogJson("s", 1, "c"))
	require.Equal(t, "", WriteChatResponseJson("s", 1, "r"))
	require.Equal(t, "", WriteToolResultsJson("s", 1, "r"))
	require.Equal(t, "", WriteRequestMessage("", 0, ""))
}

func TestSessionLogWrites(t *testing.T) {
	old := MessageDir
	defer func() { MessageDir = old }()
	MessageDir = t.TempDir()
	sid := "1234567890abcdef"

	p := WriteRequestMessage(sid, 1, "hello")
	require.NotEmpty(t, p)
	data, err := os.ReadFile(p)
	require.NoError(t, err)
	require.Equal(t, "hello", string(data))

	p2 := WriteRequestMessageJson(sid, 2, map[string]string{"a": "b"})
	require.NotEmpty(t, p2)

	p3 := AppendToStreamSessionLog(sid, 1, "chunk")
	require.NotEmpty(t, p3)
	p4 := AppendToStreamSessionLogJson(sid, 1, map[string]int{"n": 1})
	require.NotEmpty(t, p4)

	p5 := WriteChatResponseJson(sid, 1, map[string]string{"r": "ok"})
	require.NotEmpty(t, p5)
	p6 := WriteToolResultsJson(sid, 1, []string{"x"})
	require.NotEmpty(t, p6)

	require.Equal(t, "12345678", GetSessionPrefix(sid))
}
