package install

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	"github.com/armadaproject/armada-operator/internal/k8sclient"
)

func TestLookoutReconciler_Reconcile(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "lookout"}
	expectedLookout := installv1alpha1.Lookout{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Lookout",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "lookout"},
		Spec: installv1alpha1.LookoutSpec{
			Labels: nil,
			Image: installv1alpha1.Image{
				Repository: "testrepo",
				Tag:        "1.0.0",
			},
			ApplicationConfig: runtime.RawExtension{},
		},
	}
	owner := metav1.OwnerReference{
		APIVersion: expectedLookout.APIVersion,
		Kind:       expectedLookout.Kind,
		Name:       expectedLookout.Name,
		UID:        expectedLookout.UID,
	}

	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	ownerReference := []metav1.OwnerReference{owner}
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&installv1alpha1.Lookout{})).
		Return(nil).
		SetArg(2, expectedLookout)

	expectedSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            expectedLookout.Name,
			Namespace:       expectedLookout.Namespace,
			OwnerReferences: ownerReference,
		},
	}
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1.Secret{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "executor"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Secret{})).
		Return(nil).
		SetArg(1, expectedSecret)

	expectedDeployment := v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            expectedLookout.Name,
			Namespace:       expectedLookout.Namespace,
			OwnerReferences: ownerReference,
		},
	}
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&v1.Deployment{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "executor"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&v1.Deployment{})).
		Return(nil).
		SetArg(1, expectedDeployment)

	expectedService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            expectedLookout.Name,
			Namespace:       expectedLookout.Namespace,
			OwnerReferences: ownerReference,
		},
	}
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&corev1.Service{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "executor"))
	mockK8sClient.
		EXPECT().
		Create(gomock.Any(), gomock.AssignableToTypeOf(&corev1.Service{})).
		Return(nil).
		SetArg(1, expectedService)
	scheme, err := installv1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	r := LookoutReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "lookout"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}
