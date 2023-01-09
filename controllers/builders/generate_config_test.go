package builders

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"k8s.io/apimachinery/pkg/runtime"
)

func Test_convertRawExtensionToYaml(t *testing.T) {

	tests := []struct {
		name     string
		input    runtime.RawExtension
		expected string
		wantErr  bool
	}{
		{
			name:     "it converts runtime.RawExtension json to yaml",
			input:    runtime.RawExtension{Raw: []byte(`{ "test": { "foo": "bar" }}`)},
			expected: "test:\n  foo: bar\n",
		},
		{
			name:     "it converts complex runtime.RawExtension json to yaml",
			input:    runtime.RawExtension{Raw: []byte(`{ "test": {"foo": "bar"}, "test1": {"foo1": { "foo2": "bar2" }}}`)},
			expected: "test:\n  foo: bar\ntest1:\n  foo1:\n    foo2: bar2\n",
		},
		{
			name:     "it ghacks if runtime.RawExtension raw is malformed json",
			input:    runtime.RawExtension{Raw: []byte(`{ "foo": "bar" `)},
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := convertRawExtensionToYaml(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, tt.expected, output)
		})
	}
}
