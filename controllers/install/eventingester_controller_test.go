package install

import (
	"context"
	"testing"
	"time"

	"github.com/armadaproject/armada-operator/test/k8sclient"

	"github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	"github.com/golang/mock/gomock"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestEventIngesterReconciler_Reconcile(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "EventIngester"}
	expectedEventIngester := v1alpha1.EventIngester{
		TypeMeta: metav1.TypeMeta{
			Kind:       "EventIngester",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "EventIngester"},
		Spec: v1alpha1.EventIngesterSpec{
			Labels: nil,
			Image: v1alpha1.Image{
				Repository: "testrepo",
				Tag:        "1.0.0",
			},
			ApplicationConfig: runtime.RawExtension{},
		},
	}
	owner := metav1.OwnerReference{
		APIVersion: expectedEventIngester.APIVersion,
		Kind:       expectedEventIngester.Kind,
		Name:       expectedEventIngester.Name,
		UID:        expectedEventIngester.UID,
	}
	ownerReference := []metav1.OwnerReference{owner}

	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&v1alpha1.EventIngester{})).
		Return(nil).
		SetArg(2, expectedEventIngester)

	expectedSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            expectedEventIngester.Name,
			Namespace:       expectedEventIngester.Namespace,
			OwnerReferences: ownerReference,
		},
	}
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1.Secret{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "EventIngesterSecret"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Secret{})).
		Return(nil).
		SetArg(1, expectedSecret)

	expectedDeployment := v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            expectedEventIngester.Name,
			Namespace:       expectedEventIngester.Namespace,
			OwnerReferences: ownerReference,
		},
	}
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&v1.Deployment{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "EventIngesterDeployment"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&v1.Deployment{})).
		Return(nil).
		SetArg(1, expectedDeployment)

	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	r := EventIngesterReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "EventIngester"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}

func TestEventIngesterReconciler_ReconcileNoEventIngester(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "EventIngester"}

	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&v1alpha1.EventIngester{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "EventIngester"))

	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	r := EventIngesterReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "EventIngester"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}

func TestEventIngesterReconciler_ReconcileDelete(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "EventIngester"}
	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	expectedEventIngester := v1alpha1.EventIngester{
		TypeMeta: metav1.TypeMeta{
			Kind:       "EventIngester",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "default",
			Name:              "EventIngester",
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
			Finalizers:        []string{"batch.tutorial.kubebuilder.io/finalizer"},
		},
		Spec: v1alpha1.EventIngesterSpec{
			Labels: nil,
			Image: v1alpha1.Image{
				Repository: "testrepo",
				Tag:        "1.0.0",
			},
			ApplicationConfig: runtime.RawExtension{},
		},
	}
	// Executor
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&installv1alpha1.Executor{})).
		Return(nil).
		SetArg(2, expectedEventIngester)
	
	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	r := EventIngesterReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "EventIngester"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}
