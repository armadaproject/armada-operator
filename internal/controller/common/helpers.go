package common

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CleanupFunc is a function that will clean up additional resources which are not deleted by owner references.
type CleanupFunc func(context.Context) error

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

// CheckAndHandleObjectDeletion handles the deletion of the resource by adding/removing the finalizer.
// If the resource is being deleted, it will remove the finalizer.
// If the resource is not being deleted, it will add the finalizer.
// If finish is true, the reconciliation should finish early.
func CheckAndHandleObjectDeletion(
	ctx context.Context,
	r client.Client,
	object client.Object,
	finalizer string,
	cleanupF CleanupFunc,
	logger logr.Logger,
) (finish bool, err error) {
	logger = logger.WithValues("finalizer", finalizer)
	deletionTimestamp := object.GetDeletionTimestamp()
	if deletionTimestamp.IsZero() {
		// The object is not being deleted as deletionTimestamp.
		// In this case, we should add the finalizer if it is not already present.
		if err := addFinalizerIfNeeded(ctx, r, object, finalizer, logger); err != nil {
			return true, err
		}
	} else {
		// The object is being deleted so we should run the cleanup function if needed and remove the finalizer.
		return handleObjectDeletion(ctx, r, object, finalizer, cleanupF, logger)
	}
	// The object is not being deleted, continue reconciliation
	return false, nil
}

// addFinalizerIfNeeded will add the finalizer to the object if it is not already present.
func addFinalizerIfNeeded(
	ctx context.Context,
	client client.Client,
	object client.Object,
	finalizer string,
	logger logr.Logger,
) error {
	if !controllerutil.ContainsFinalizer(object, finalizer) {
		logger.Info("Attaching cleanup finalizer because object does not have a deletion timestamp set")
		controllerutil.AddFinalizer(object, finalizer)
		return client.Update(ctx, object)
	}
	return nil
}

func handleObjectDeletion(
	ctx context.Context,
	client client.Client,
	object client.Object,
	finalizer string,
	cleanupF CleanupFunc,
	logger logr.Logger,
) (finish bool, err error) {
	deletionTimestamp := object.GetDeletionTimestamp()
	logger.Info(
		"Object is being deleted as it has a non-zero deletion timestamp set",
		"deletionTimestamp", deletionTimestamp,
	)
	logger.Info(
		"Namespace-scoped objects will be deleted by Kubernetes based on their OwnerReference",
		"deletionTimestamp", deletionTimestamp,
	)
	// The object is being deleted
	if controllerutil.ContainsFinalizer(object, finalizer) {
		// Run additional cleanup function if it is provided
		if cleanupF != nil {
			if err := cleanupF(ctx); err != nil {
				return true, err
			}
		}
		// Remove our finalizer from the list and update it.
		logger.Info("Removing cleanup finalizer from object")
		controllerutil.RemoveFinalizer(object, finalizer)
		if err := client.Update(ctx, object); err != nil {
			return true, err
		}
	}

	// Stop reconciliation as the item is being deleted
	return true, nil
}
