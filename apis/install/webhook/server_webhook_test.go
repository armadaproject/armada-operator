package webhook

import (
	"context"
	"testing"

	v1alpha "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestServerDefaultWebhook(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "correctly initialize defaults",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			serverWebHook := ArmadaServerWebhook{}
			server := &v1alpha.Server{}
			assert.NoError(t, serverWebHook.Default(context.Background(), server))
		})
	}
}
func TestServerValidateDelete(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "server validate delete",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			serverWebHook := &ArmadaServerWebhook{}
			server := &v1alpha.Server{}
			assert.NoError(t, serverWebHook.ValidateDelete(context.Background(), server))
		})
	}
}
func TestServerValidateCreate(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "server validate create",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			serverWebHook := &ArmadaServerWebhook{}
			server := &v1alpha.Server{}
			assert.NoError(t, serverWebHook.ValidateCreate(context.Background(), server))
		})
	}
}

func TestServerValidateUpdate(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "server validate update",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			serverWebHook := &ArmadaServerWebhook{}
			server := &v1alpha.Server{}
			assert.NoError(t, serverWebHook.ValidateUpdate(context.Background(), server, server))
		})
	}

}
