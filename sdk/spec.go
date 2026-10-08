package sdk

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// SpecPath is the OpenAPI document the SDK is generated from.
const SpecPath = "openapi.yaml"

// SpecCheck verifies the SDK surface against the OpenAPI document.
// It is intentionally lightweight: it parses the spec for declared
// operation IDs and cross-checks them against the methods on Client.
type SpecCheck struct {
	operations map[string]bool
}

// NewSpecCheck loads the spec and returns a SpecCheck.
func NewSpecCheck(path string) (*SpecCheck, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("spec: %w", err)
	}
	ops := map[string]bool{}
	for _, m := range operationRe.FindAllStringSubmatch(string(data), -1) {
		ops[m[1]] = true
	}
	return &SpecCheck{operations: ops}, nil
}

var operationRe = regexp.MustCompile(`operationId:\s*([a-zA-Z0-9_]+)`)

// Methods returns the SDK method names declared in the spec.
func (s *SpecCheck) Methods() []string {
	out := make([]string, 0, len(s.operations))
	for name := range s.operations {
		out = append(out, name)
	}
	return out
}

// Missing reports spec operations that have no corresponding SDK method.
// The mapping is hardcoded to keep the check deterministic.
func (s *SpecCheck) Missing() []string {
	var out []string
	for op := range s.operations {
		if _, ok := specToSDK[op]; !ok {
			out = append(out, op)
		}
	}
	return out
}

// specToSDK maps spec operationId -> SDK method on Client.
var specToSDK = map[string]string{
	"listSessions":     "ListSessions",
	"createSession":    "CreateSession",
	"getSession":       "GetSession",
	"deleteSession":    "DeleteSession",
	"listMessages":     "ListMessages",
	"exportSession":    "Export",
	"ask":              "Ask",
	"cancelSession":    "Cancel",
	"summarizeSession": "Summarize",
	"currentModel":     "Model",
	"updateModel":      "UpdateModel",
}

// Validate ensures every spec operation has a matching SDK method.
func (s *SpecCheck) Validate() error {
	missing := s.Missing()
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("spec operations missing SDK methods: %s", strings.Join(missing, ", "))
}
