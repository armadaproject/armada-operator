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
			serverWebHook := EventIngesterWebhook{}
			server := &v1alpha.EventIngester{}
			assert.NoError(t, serverWebHook.Default(context.Background(), server))
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
			server := &v1alpha.EventIngester{}
			assert.NoError(t, eventIngesterWebhook.ValidateDelete(context.Background(), server))
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
