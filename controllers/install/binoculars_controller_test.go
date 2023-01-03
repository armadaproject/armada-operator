package install

import (
	"context"

	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Binoculars controller", func() {
	const binocularsName = "test-binoculars"

	ctx := context.Background()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      binocularsName,
			Namespace: binocularsName,
		},
	}

	BeforeEach(func() {
		By("Creating the Namespace to perform the tests")
		err := k8sClient.Create(ctx, namespace)
		Expect(err).To(Not(HaveOccurred()))

	})

	AfterEach(func() {
		// TODO(user): Attention if you improve this code by adding other context test you MUST
		// be aware of the current delete namespace limitations. More info: https://book.kubebuilder.io/reference/envtest.html#testing-considerations
		By("Deleting the Namespace to perform the tests")
		_ = k8sClient.Delete(ctx, namespace)

	})

	It("should successfully reconcile a custom resource for Binoculars", func() {
		binoculars := installv1alpha1.Binoculars{
			ObjectMeta: metav1.ObjectMeta{Name: binocularsName, Namespace: binocularsName},
			Spec: installv1alpha1.BinocularsSpec{
				Name: "TestBinoculars",
				Image: installv1alpha1.Image{
					Repository: "armadaproject",
					Image:      "binoculars",
					Tag:        "latest",
				},
				ApplicationConfig: nil,
			},
		}
		err := k8sClient.Create(ctx, &binoculars)
		Expect(err).ToNot(HaveOccurred())

		namespaceName := types.NamespacedName{Namespace: binocularsName, Name: binocularsName}
		var binocularsGet installv1alpha1.Binoculars
		err = k8sClient.Get(ctx, namespaceName, &binocularsGet)
		Expect(err).ToNot(HaveOccurred())
	})
})
