package format

import (
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	if got := Text.String(); got != "text" {
		t.Errorf("Text.String() = %q, want %q", got, "text")
	}
	if got := JSON.String(); got != "json" {
		t.Errorf("JSON.String() = %q, want %q", got, "json")
	}
}

func TestSupportedFormats(t *testing.T) {
	if len(SupportedFormats) != 2 {
		t.Fatalf("expected 2 supported formats, got %d", len(SupportedFormats))
	}
	if SupportedFormats[0] != string(Text) || SupportedFormats[1] != string(JSON) {
		t.Errorf("unexpected supported formats: %v", SupportedFormats)
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    OutputFormat
		wantErr bool
	}{
		{"text", "text", Text, false},
		{"uppercase", "TEXT", Text, false},
		{"json", "json", JSON, false},
		{"whitespace", "  json  ", JSON, false},
		{"invalid", "yaml", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Parse(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsValid(t *testing.T) {
	if !IsValid("text") {
		t.Error("IsValid(text) = false, want true")
	}
	if !IsValid("json") {
		t.Error("IsValid(json) = false, want true")
	}
	if IsValid("yaml") {
		t.Error("IsValid(yaml) = true, want false")
	}
}

func TestGetHelpText(t *testing.T) {
	h := GetHelpText()
	if !strings.Contains(h, "text") || !strings.Contains(h, "json") {
		t.Errorf("GetHelpText() = %q, should mention text and json", h)
	}
}

func TestFormatOutput(t *testing.T) {
	content := "hello world"

	if got := FormatOutput(content, "text"); got != content {
		t.Errorf("FormatOutput(text) = %q, want %q", got, content)
	}

	if got := FormatOutput(content, "invalid"); got != content {
		t.Errorf("FormatOutput(invalid) = %q, want %q (fallback to text)", got, content)
	}

	jsonOut := FormatOutput(content, "json")
	if !strings.Contains(jsonOut, `"response"`) || !strings.Contains(jsonOut, content) {
		t.Errorf("FormatOutput(json) = %q, does not contain response wrapper", jsonOut)
	}
}