package util

import (
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func HasOwnerReference(owner metav1.Object, child metav1.Object, scheme *runtime.Scheme) (bool, error) {
	childOwners := child.GetOwnerReferences()

	ro, ok := owner.(runtime.Object)
	if !ok {
		return false, errors.Errorf("%T is not a runtime.Object, cannot call check OwnerReference", owner)
	}

	gvk, err := apiutil.GVKForObject(ro, scheme)
	if err != nil {
		return false, err
	}

	for _, o := range childOwners {
		if o.Kind == gvk.Kind &&
			o.APIVersion == gvk.GroupVersion().String() &&
			o.Name == owner.GetName() &&
			o.UID == owner.GetUID() {
			return true, nil
		}
	}
	return false, nil
}
