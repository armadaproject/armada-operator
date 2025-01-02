package armadaclient

import (
	"github.com/armadaproject/armada/pkg/api"
)

//go:generate mockgen -destination=./mock_queue_client.go -package=armadaclient "github.com/armadaproject/armada-operator/test/armadaclient" QueueClient

type QueueClient interface {
	api.QueueServiceClient
}
