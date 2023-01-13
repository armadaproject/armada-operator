package builders

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"k8s.io/apimachinery/pkg/runtime"
)

func Test_GenerateSecret(t *testing.T) {

	tests := []struct {
		title   string
		input   runtime.RawExtension
		wantErr bool
	}{
		{
			title: "it converts runtime.RawExtension json to yaml",
			input: runtime.RawExtension{Raw: []byte(`{ "test": { "foo": "bar" }}`)},
		},
		{
			title: "it converts complex runtime.RawExtension json to yaml",
			input: runtime.RawExtension{Raw: []byte(`{ "test": {"foo": "bar"}, "test1": {"foo1": { "foo2": "bar2" }}}`)},
		},
		{
			title:   "it errors if runtime.RawExtension raw is malformed json",
			input:   runtime.RawExtension{Raw: []byte(`{ "foo": "bar" `)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			output, err := CreateSecret(tt.input, "secret", "default")
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
