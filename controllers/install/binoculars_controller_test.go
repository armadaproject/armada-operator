package install

import (
	"context"
	"testing"

	"github.com/armadaproject/armada-operator/test/k8sclient"

	"github.com/golang/mock/gomock"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
)

func TestBinocularsReconciler_Reconcile(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expectedNamespacedName := types.NamespacedName{Namespace: "default", Name: "binoculars"}
	expectedBinoculars := v1alpha1.Binoculars{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Binoculars",
			APIVersion: "install.armadaproject.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "binoculars"},
		Spec: v1alpha1.BinocularsSpec{
			Labels: nil,
			Image: installv1alpha1.Image{
				Repository: "testrepo",
				Tag:        "1.0.0",
			},
			ApplicationConfig: runtime.RawExtension{},
		},
	}
	owner := metav1.OwnerReference{
		APIVersion: expectedBinoculars.APIVersion,
		Kind:       expectedBinoculars.Kind,
		Name:       expectedBinoculars.Name,
		UID:        expectedBinoculars.UID,
	}

	mockK8sClient := k8sclient.NewMockClient(mockCtrl)
	ownerReference := []metav1.OwnerReference{owner}
	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&v1alpha1.Binoculars{})).
		Return(nil).
		SetArg(2, expectedBinoculars)

	mockK8sClient.
		EXPECT().
		Get(gomock.Any(), expectedNamespacedName, gomock.AssignableToTypeOf(&rbacv1.ClusterRole{})).
		Return(errors.NewNotFound(schema.GroupResource{}, "binoculars"))

	mockK8sClient.EXPECT().Create(gomock.Any(), gomock.AssignableToTypeOf(&rbacv1.ClusterRole{})).Return(nil)

	expectedSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            expectedBinoculars.Name,
			Namespace:       expectedBinoculars.Namespace,
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
			Name:            expectedBinoculars.Name,
			Namespace:       expectedBinoculars.Namespace,
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
			Name:            expectedBinoculars.Name,
			Namespace:       expectedBinoculars.Namespace,
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
	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("should not return error when building schema")
	}

	r := BinocularsReconciler{
		Client: mockK8sClient,
		Scheme: scheme,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: "default", Name: "binoculars"},
	}

	_, err = r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("reconcile should not return error")
	}
}
