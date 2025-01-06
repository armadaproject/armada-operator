/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package core

import (
	"context"
	"fmt"
	"time"

	"github.com/armadaproject/armada-operator/internal/controller/common"

	"github.com/armadaproject/armada/pkg/api"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1alpha1 "github.com/armadaproject/armada-operator/api/core/v1alpha1"
)

// QueueReconciler reconciles a Queue object
type QueueReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	QueueClient api.QueueServiceClient
}

//+kubebuilder:rbac:groups=core.armadaproject.io,resources=queues,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.armadaproject.io,resources=queues/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.armadaproject.io,resources=queues/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *QueueReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("namespace", req.Namespace, "name", req.Name)

	started := time.Now()

	logger.Info("Reconciling object")

	var queue corev1alpha1.Queue
	if miss, err := common.GetObject(ctx, r.Client, &queue, req.NamespacedName, logger); err != nil || miss {
		return ctrl.Result{}, err
	}

	cleanupF := func(ctx context.Context) error { return r.deleteQueue(ctx, queue) }
	finish, err := common.CheckAndHandleObjectDeletion(ctx, r.Client, &queue, operatorFinalizer, cleanupF, logger)
	if err != nil || finish {
		return ctrl.Result{}, err
	}

	err = r.upsertQueue(ctx, queue)
	if err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled Queue object", "durationMillis", time.Since(started).Milliseconds())

	return ctrl.Result{}, nil
}

func (r *QueueReconciler) upsertQueue(ctx context.Context, queue corev1alpha1.Queue) error {

	createQueueRequest := queueToProto(queue)

	existingQueue, getErr := r.QueueClient.GetQueue(ctx, &api.QueueGetRequest{Name: queue.Name})
	if getErr != nil {
		// See if this is a gRPC error
		e, ok := status.FromError(getErr)
		if !ok {
			return fmt.Errorf("getting queue: %w", getErr)
		}

		// Check if not found
		if e.Code() == codes.NotFound {
			// Queue not found so create it
			_, createErr := r.QueueClient.CreateQueue(ctx, createQueueRequest)
			if createErr != nil {
				return fmt.Errorf("creating queue: %w", createErr)
			}

			return nil
		}

		return fmt.Errorf("getting queue: %w", getErr)
	}

	if cmp.Equal(existingQueue, createQueueRequest, protocmp.Transform()) {
		// nothing to do
		return nil
	}

	_, getErr = r.QueueClient.UpdateQueue(ctx, createQueueRequest)
	return getErr
}

func queueToProto(queue corev1alpha1.Queue) *api.Queue {
	queueRequest := &api.Queue{
		Name: queue.Name,
	}

	if queue.Spec.PriorityFactor != nil {
		queueRequest.PriorityFactor = queue.Spec.PriorityFactor.AsApproximateFloat64()
	} else {
		// Value must be at least 1 or the API will get upset
		queueRequest.PriorityFactor = 1
	}

	protoPermissions := []*api.Queue_Permissions{}
	for _, p := range queue.Spec.Permissions {
		protoSubjects := []*api.Queue_Permissions_Subject{}

		for _, s := range p.Subjects {
			protoSubjects = append(protoSubjects, &api.Queue_Permissions_Subject{
				Name: s.Name,
				Kind: s.Kind,
			})
		}

		protoPermission := &api.Queue_Permissions{
			Subjects: protoSubjects,
			Verbs:    p.Verbs,
		}

		protoPermissions = append(protoPermissions, protoPermission)
	}

	if len(protoPermissions) > 0 {
		queueRequest.Permissions = protoPermissions
	}

	return queueRequest
}

func (r *QueueReconciler) deleteQueue(ctx context.Context, queue corev1alpha1.Queue) error {
	_, err := r.QueueClient.DeleteQueue(ctx, &api.QueueDeleteRequest{Name: queue.Name})
	if err != nil {
		return fmt.Errorf("deleting queue: %w", err)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *QueueReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.Queue{}).
		Complete(r)
}
