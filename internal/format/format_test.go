package format

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	f, err := Parse("text")
	require.NoError(t, err)
	require.Equal(t, Text, f)

	f, err = Parse(" JSON ")
	require.NoError(t, err)
	require.Equal(t, JSON, f)

	_, err = Parse("yaml")
	require.Error(t, err)
}

func TestIsValidAndString(t *testing.T) {
	require.True(t, IsValid("text"))
	require.True(t, IsValid("json"))
	require.False(t, IsValid("xml"))
	require.Equal(t, "json", JSON.String())
	require.Contains(t, SupportedFormats, "text")
	require.Contains(t, GetHelpText(), "json")
}

func TestFormatOutput(t *testing.T) {
	require.Equal(t, "hi", FormatOutput("hi", "text"))
	require.Equal(t, "hi", FormatOutput("hi", "bogus"))

	out := FormatOutput("hi", "json")
	var v map[string]string
	require.NoError(t, json.Unmarshal([]byte(out), &v))
	require.Equal(t, "hi", v["response"])

	out = FormatOutput("a\"b\nc", "json")
	require.NoError(t, json.Unmarshal([]byte(out), &v))
	require.Equal(t, "a\"b\nc", v["response"])
	require.True(t, strings.Contains(out, "response"))
}
