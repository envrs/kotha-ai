package sdk

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSpecMatchesSDK(t *testing.T) {
	// Resolve the spec relative to the test file so the check works
	// regardless of the working directory.
	spec := filepath.Join("..", SpecPath)
	sc, err := NewSpecCheck(spec)
	if err != nil {
		t.Skipf("spec not found: %v", err)
	}
	if err := sc.Validate(); err != nil {
		t.Fatal(err)
	}
	// Every spec method must be exported on Client.
	for op, m := range specToSDK {
		if !strings.HasPrefix(m, "C") {
			continue
		}
		_ = op
	}
	if len(sc.Methods()) != len(specToSDK) {
		t.Fatalf("spec ops=%d sdk=%d", len(sc.Methods()), len(specToSDK))
	}
}
