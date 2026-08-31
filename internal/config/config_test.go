package config

import (
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// TestFromEnv is the only test that changes the environment of the process.
// It cannot run in parallel, because it uses t.Setenv.
func TestFromEnv(t *testing.T) {
	t.Setenv("DEFAULT_ALPINE_IMAGE", "my-mirror.example.com/alpine:3.20")

	assert.Equal(t, "my-mirror.example.com/alpine:3.20", FromEnv().Images.AlpineOverride)
}

func Test_load(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		env      map[string]string
		expected Config
	}{
		{
			name:     "Webhooks are enabled and the overrides are empty when no variable is set",
			env:      map[string]string{},
			expected: Config{EnableWebhooks: true},
		},
		{
			name: "Each variable sets its field",
			env: map[string]string{
				"WEBHOOK_CERT_DIR":       "/tmp/certs",
				"ENABLE_WEBHOOKS":        "false",
				"DEFAULT_ALPINE_IMAGE":   "my-mirror.example.com/alpine:3.20",
				"DEFAULT_POSTGRES_IMAGE": "my-mirror.example.com/postgres:15.2-alpine",
			},
			expected: Config{
				WebhookCertDir: "/tmp/certs",
				EnableWebhooks: false,
				Images: Images{
					AlpineOverride:   "my-mirror.example.com/alpine:3.20",
					PostgresOverride: "my-mirror.example.com/postgres:15.2-alpine",
				},
			},
		},
		{
			name:     "Webhooks stay enabled when ENABLE_WEBHOOKS has a value other than false",
			env:      map[string]string{"ENABLE_WEBHOOKS": "no"},
			expected: Config{EnableWebhooks: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual := load(func(key string) string { return tt.env[key] })
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestImages_AlpineImage(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "alpine:3.20", Images{}.AlpineImage())
	assert.Equal(t, "my-mirror.example.com/alpine:3.20", Images{AlpineOverride: "my-mirror.example.com/alpine:3.20"}.AlpineImage())
}

func TestImages_PostgresImage(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "postgres:15.2-alpine", Images{}.PostgresImage())
	assert.Equal(t, "my-mirror.example.com/postgres:15.2-alpine", Images{PostgresOverride: "my-mirror.example.com/postgres:15.2-alpine"}.PostgresImage())
}

// TestManagerManifest_DefaultImages keeps the manager manifest and the built-in defaults equal.
// The manifest always sets the two variables, so a change of only the built-in default
// has no effect on a deployed operator.
// The Helm chart values come from this manifest, so this test also covers the chart.
func TestManagerManifest_DefaultImages(t *testing.T) {
	t.Parallel()

	file, err := os.Open("../../config/manager/manager.yaml")
	require.NoError(t, err)
	defer func() { _ = file.Close() }()

	env := map[string]string{}
	decoder := yaml.NewYAMLOrJSONDecoder(file, 4096)
	for {
		var deployment appsv1.Deployment
		err := decoder.Decode(&deployment)
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		for _, container := range deployment.Spec.Template.Spec.Containers {
			for _, envVar := range container.Env {
				env[envVar.Name] = envVar.Value
			}
		}
	}

	assert.Equal(t, builtInAlpineImage, env[defaultAlpineImageEnvVar])
	assert.Equal(t, builtInPostgresImage, env[defaultPostgresImageEnvVar])
}
