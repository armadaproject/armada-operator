package common

import (
	"context"

	"github.com/go-logr/logr"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetObject will get the object from Kubernetes and return if it is missing or an error.
func GetObject(
	ctx context.Context,
	client client.Client,
	object client.Object,
	namespacedName types.NamespacedName,
	logger logr.Logger,
) (miss bool, err error) {
	logger.Info("Fetching object from cache")
	if err := client.Get(ctx, namespacedName, object); err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Info("Object not found in cache, ending reconcile...")
			return true, nil
		}
		return true, err
	}
	return false, nil
}
