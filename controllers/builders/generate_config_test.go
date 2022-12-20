package builders

import (
	"testing"
)

func compareMaps(actual map[string][]byte, expected map[string][]byte) bool {
	if actual == nil && expected == nil {
		return true
	}
	// Not Implemented
	return false
}
func TestGenerateConfig(t *testing.T) {
	testcases := map[string]struct {
		config          map[string]any
		yamlifed        map[string][]byte
		serializedError error
	}{
		"Nil Call": {
			config:          nil,
			yamlifed:        nil,
			serializedError: nil,
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			got, err := generateArmadaConfig(tc.config)
			if compareMaps(tc.yamlifed, got) {
				t.Errorf("Unexpected response (want: %v, got: %v)", tc.yamlifed, got)
			}
			if err != tc.serializedError {
				t.Errorf("Unexpected response (want: %v, got: %v)", err, got)

			}
		})
	}
}
