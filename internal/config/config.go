// Package config holds the configuration that the operator reads from its environment.
// It is the only package that reads environment variables.
// The other packages receive the values as parameters, so that tests can set them directly.
package config

import "os"

const (
	webhookCertDirEnvVar       = "WEBHOOK_CERT_DIR"
	enableWebhooksEnvVar       = "ENABLE_WEBHOOKS"
	defaultAlpineImageEnvVar   = "DEFAULT_ALPINE_IMAGE"
	defaultPostgresImageEnvVar = "DEFAULT_POSTGRES_IMAGE"

	builtInAlpineImage   = "alpine:3.20"
	builtInPostgresImage = "postgres:15.2-alpine"
)

// Config is the configuration of the operator process.
type Config struct {
	// WebhookCertDir is the directory that contains the TLS certificate of the webhook server.
	// An empty value selects the default directory of controller-runtime.
	WebhookCertDir string
	// EnableWebhooks is false only when ENABLE_WEBHOOKS has the value "false".
	EnableWebhooks bool
	// Images contains the default images for the containers that the operator adds.
	Images Images
}

// Images contains the overrides of the built-in default images,
// for clusters that pull images from an internal mirror.
// An empty field selects the built-in default, so the zero value is safe to use.
type Images struct {
	// AlpineOverride replaces the image for init containers that only need the nc command.
	AlpineOverride string
	// PostgresOverride replaces the image for init containers that need the psql client and the nc command.
	PostgresOverride string
}

// FromEnv reads the configuration from the environment variables of the process.
func FromEnv() Config {
	return load(os.Getenv)
}

// load builds the configuration from getenv.
// Tests supply their own getenv, so they do not change the environment of the process.
func load(getenv func(string) string) Config {
	return Config{
		WebhookCertDir: getenv(webhookCertDirEnvVar),
		EnableWebhooks: getenv(enableWebhooksEnvVar) != "false",
		Images: Images{
			AlpineOverride:   getenv(defaultAlpineImageEnvVar),
			PostgresOverride: getenv(defaultPostgresImageEnvVar),
		},
	}
}

// AlpineImage returns the override when it is set, and the built-in default otherwise.
func (i Images) AlpineImage() string {
	if i.AlpineOverride != "" {
		return i.AlpineOverride
	}
	return builtInAlpineImage
}

// PostgresImage returns the override when it is set, and the built-in default otherwise.
func (i Images) PostgresImage() string {
	if i.PostgresOverride != "" {
		return i.PostgresOverride
	}
	return builtInPostgresImage
}
