package util

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCmdHandler(t *testing.T) {
	cmd := CmdHandler("hello")
	require.Equal(t, "hello", cmd())
}

func TestReportHelpers(t *testing.T) {
	require.Equal(t, InfoMsg{Type: InfoTypeError, Msg: "boom"}, ReportError(errors.New("boom"))())
	require.Equal(t, InfoMsg{Type: InfoTypeInfo, Msg: "hi"}, ReportInfo("hi")())
	require.Equal(t, InfoMsg{Type: InfoTypeWarn, Msg: "w"}, ReportWarn("w")())
}

func TestClamp(t *testing.T) {
	require.Equal(t, 5, Clamp(5, 0, 10))
	require.Equal(t, 0, Clamp(-3, 0, 10))
	require.Equal(t, 10, Clamp(99, 0, 10))
	require.Equal(t, 5, Clamp(5, 10, 0)) // swapped bounds
	require.Equal(t, 3, Clamp(3, 3, 3))
}
