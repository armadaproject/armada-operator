package webhook

import (
	"context"
	"testing"

	v1alpha "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestEventIngesterDefaultWebhook(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "correctly initialize defaults",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			eventWebHook := EventIngesterWebhook{}
			server := &v1alpha.EventIngester{}
			assert.NoError(t, eventWebHook.Default(context.Background(), event))
		})
	}
}

func TestEventIngesterValidateCreate(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "event ingester validate create",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			eventIngesterWebhook := &EventIngesterWebhook{}
			event := &v1alpha.EventIngester{}
			assert.NoError(t, eventIngesterWebhook.ValidateCreate(context.Background(), event))
		})
	}
}

func TestEventIngesterValidateDelete(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "event ingester validate delete",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			eventIngesterWebhook := &EventIngesterWebhook{}
			event := &v1alpha.EventIngester{}
			assert.NoError(t, eventIngesterWebhook.ValidateDelete(context.Background(), event))
		})
	}
}

func TestEventIngesterValidateUpdate(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "server validate update",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			eventIngesterWebhook := &EventIngesterWebhook{}
			eventIngester := &v1alpha.EventIngester{}
			assert.NoError(t, eventIngesterWebhook.ValidateUpdate(context.Background(), eventIngester, eventIngester))
		})
	}

}
