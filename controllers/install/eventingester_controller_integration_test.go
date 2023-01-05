package install

import (
	"github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("EventIngester controller", func() {
	Context("When creating EventIngester", func() {
		It("should create EventIngester Kubernetes resources", func() {
			By("applying the EventIngester CRD", func() {
				executor := v1alpha1.EventIngester{
					ObjectMeta: metav1.ObjectMeta{Name: "eventingester", Namespace: "default"},
					Spec: v1alpha1.EventIngesterSpec{
						Name: "test",
						Image: v1alpha1.Image{
							Repository: "testrepo",
							Image:      "eventingester",
							Tag:        "1.0.2",
						},
						ApplicationConfig: map[string]runtime.RawExtension{},
					},
				}
				Expect(k8sClient.Create(ctx, &executor)).Should(Succeed())
			})
		})
	})
})
