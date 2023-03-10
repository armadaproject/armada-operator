package builders

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"k8s.io/apimachinery/pkg/runtime"
)

func Test_GenerateSecret(t *testing.T) {

	tests := map[string]struct {
		input   runtime.RawExtension
		wantErr bool
	}{
		"it converts runtime.RawExtension json to yaml": {
			input: runtime.RawExtension{Raw: []byte(`{ "test": { "foo": "bar" }}`)},
		},
		"it converts complex runtime.RawExtension json to yaml": {
			input: runtime.RawExtension{Raw: []byte(`{ "test": {"foo": "bar"}, "test1": {"foo1": { "foo2": "bar2" }}}`)},
		},
		"it errors if runtime.RawExtension raw is malformed json": {
			input:   runtime.RawExtension{Raw: []byte(`{ "foo": "bar" `)},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			output, err := CreateSecret(tt.input, "secret", "default", "config.yaml")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, "secret", output.Name)
				assert.Equal(t, "default", output.Namespace)
			}
		})
	}
}
