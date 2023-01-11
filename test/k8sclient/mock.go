package k8sclient

import "sigs.k8s.io/controller-runtime/pkg/client"

//go:generate mockgen -destination=./mock_client.go -package=k8sclient "github.com/armadaproject/armada-operator/test/k8sclient" Client

type Client interface {
	client.Client
}
