package core

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/armadaproject/armada/pkg/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	corev1alpha1 "github.com/armadaproject/armada-operator/api/core/v1alpha1"
	"github.com/armadaproject/armada-operator/test/armadaclient"
	"github.com/armadaproject/armada-operator/test/k8sclient"

	"github.com/golang/mock/gomock"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/armadaproject/armada-operator/api/install/v1alpha1"
)

const (
	queueName = "test"
)

func TestQueueReconciler_ReconcileNoQueue(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: queueName}

	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// queue
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1alpha1.Queue{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "executor"))

	mockQueueClient := armadaclient.NewMockQueueClient(mockCtrl)

	r := QueueReconciler{
		Client:      mockK8sClient,
		Scheme:      scheme,
		QueueClient: mockQueueClient,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: queueName},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}

func TestQueueReconciler_ReconcileNewQueueSuccess(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: queueName}
	expectedQueue := corev1alpha1.Queue{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Queue",
			APIVersion: "core.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: queueName},
		Spec:       corev1alpha1.QueueSpec{},
	}

	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// queue
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1alpha1.Queue{})).
		Return(nil).
		SetArg(2, expectedQueue)
	// queue finalizer
	mockK8sClient.
		EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&corev1alpha1.Queue{})).
		Return(nil)

	createQueueRequest := &api.Queue{
		Name:           queueName,
		PriorityFactor: 1,
	}

	mockQueueClient := armadaclient.NewMockQueueClient(mockCtrl)
	mockQueueClient.
		EXPECT().
		GetQueue(gomock.Any(), &api.QueueGetRequest{Name: queueName}).
		Return(nil, status.Error(codes.NotFound, ""))

	mockQueueClient.
		EXPECT().
		CreateQueue(gomock.Any(), createQueueRequest).Return(nil, nil)

	r := QueueReconciler{
		Client:      mockK8sClient,
		Scheme:      scheme,
		QueueClient: mockQueueClient,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: queueName},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}

func TestQueueReconciler_ReconcileNewQueueServerErrorOnCreate(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: queueName}
	expectedQueue := corev1alpha1.Queue{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Queue",
			APIVersion: "core.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: queueName},
		Spec:       corev1alpha1.QueueSpec{},
	}

	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// queue
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1alpha1.Queue{})).
		Return(nil).
		SetArg(2, expectedQueue)
	// queue finalizer
	mockK8sClient.
		EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&corev1alpha1.Queue{})).
		Return(nil)

	createQueueRequest := &api.Queue{
		Name:           queueName,
		PriorityFactor: 1,
	}

	mockQueueClient := armadaclient.NewMockQueueClient(mockCtrl)
	mockQueueClient.
		EXPECT().
		GetQueue(gomock.Any(), &api.QueueGetRequest{Name: queueName}).
		Return(nil, status.Error(codes.NotFound, ""))

	mockQueueClient.
		EXPECT().
		CreateQueue(gomock.Any(), createQueueRequest).Return(nil, status.Error(codes.Internal, ""))

	r := QueueReconciler{
		Client:      mockK8sClient,
		Scheme:      scheme,
		QueueClient: mockQueueClient,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: queueName},
	}

	_, err = r.Reconcile(context.Background(), req)
	require.ErrorContains(t, err, "creating queue")
}

func TestQueueReconciler_ReconcileNewQueueServerErrorOnGet(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: queueName}
	expectedQueue := corev1alpha1.Queue{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Queue",
			APIVersion: "core.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: queueName},
		Spec:       corev1alpha1.QueueSpec{},
	}

	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// queue
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1alpha1.Queue{})).
		Return(nil).
		SetArg(2, expectedQueue)
	// queue finalizer
	mockK8sClient.
		EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&corev1alpha1.Queue{})).
		Return(nil)

	mockQueueClient := armadaclient.NewMockQueueClient(mockCtrl)
	mockQueueClient.
		EXPECT().
		GetQueue(gomock.Any(), &api.QueueGetRequest{Name: queueName}).
		Return(nil, status.Error(codes.Internal, ""))

	r := QueueReconciler{
		Client:      mockK8sClient,
		Scheme:      scheme,
		QueueClient: mockQueueClient,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: queueName},
	}

	_, err = r.Reconcile(context.Background(), req)
	require.ErrorContains(t, err, "getting queue")
}

func TestQueueReconciler_ReconcileExistingQueueNoopSuccess(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: queueName}
	expectedQueue := corev1alpha1.Queue{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Queue",
			APIVersion: "core.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: queueName},
		Spec:       corev1alpha1.QueueSpec{},
	}

	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// queue
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1alpha1.Queue{})).
		Return(nil).
		SetArg(2, expectedQueue)
	// queue finalizer
	mockK8sClient.
		EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&corev1alpha1.Queue{})).
		Return(nil)

	existingQueue := &api.Queue{
		Name:           queueName,
		PriorityFactor: 1,
	}

	mockQueueClient := armadaclient.NewMockQueueClient(mockCtrl)
	mockQueueClient.
		EXPECT().
		GetQueue(gomock.Any(), &api.QueueGetRequest{Name: queueName}).
		Return(existingQueue, nil)

	r := QueueReconciler{
		Client:      mockK8sClient,
		Scheme:      scheme,
		QueueClient: mockQueueClient,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: queueName},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}

func TestQueueReconciler_ReconcileExistingQueueUpdateSuccess(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	priorityFactor := resource.MustParse("5")
	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: queueName}
	expectedQueue := corev1alpha1.Queue{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Queue",
			APIVersion: "core.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: queueName},
		Spec: corev1alpha1.QueueSpec{
			PriorityFactor: &priorityFactor,
			Permissions: []corev1alpha1.QueuePermissions{
				{
					Subjects: []corev1alpha1.PermissionSubject{
						{
							Kind: "Group",
							Name: "Admin",
						},
					},
					Verbs: []string{"submit"},
				},
			},
		},
	}

	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// queue
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1alpha1.Queue{})).
		Return(nil).
		SetArg(2, expectedQueue)
	// queue finalizer
	mockK8sClient.
		EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&corev1alpha1.Queue{})).
		Return(nil)

	existingQueue := &api.Queue{
		Name:           queueName,
		PriorityFactor: 1,
	}

	updateQueueRequest := &api.Queue{
		Name:           queueName,
		PriorityFactor: 5,
		Permissions: []*api.Queue_Permissions{
			{
				Subjects: []*api.Queue_Permissions_Subject{
					{
						Kind: "Group",
						Name: "Admin",
					},
				},
				Verbs: []string{
					"submit",
				},
			},
		},
	}

	mockQueueClient := armadaclient.NewMockQueueClient(mockCtrl)
	mockQueueClient.
		EXPECT().
		GetQueue(gomock.Any(), &api.QueueGetRequest{Name: queueName}).
		Return(existingQueue, nil)
	mockQueueClient.
		EXPECT().
		UpdateQueue(gomock.Any(), updateQueueRequest).
		Return(nil, nil)

	r := QueueReconciler{
		Client:      mockK8sClient,
		Scheme:      scheme,
		QueueClient: mockQueueClient,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: queueName},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}

func TestQueueReconciler_ReconcileDeleteQueueSuccess(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: queueName}
	expectedQueue := corev1alpha1.Queue{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Queue",
			APIVersion: "core.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "default",
			Name:              queueName,
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
			Finalizers:        []string{operatorFinalizer}},
		Spec: corev1alpha1.QueueSpec{},
	}

	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	// queue
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1alpha1.Queue{})).
		Return(nil).
		SetArg(2, expectedQueue)
	// remove finalizer
	mockK8sClient.
		EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&corev1alpha1.Queue{})).
		Return(nil)

	mockQueueClient := armadaclient.NewMockQueueClient(mockCtrl)
	mockQueueClient.
		EXPECT().
		DeleteQueue(gomock.Any(), &api.QueueDeleteRequest{Name: queueName}).
		Return(nil, nil)

	r := QueueReconciler{
		Client:      mockK8sClient,
		Scheme:      scheme,
		QueueClient: mockQueueClient,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: queueName},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}
