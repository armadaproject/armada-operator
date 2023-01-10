package install

import (
	"github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

var _ = Describe("EventIngester controller", func() {
	Context("When creating EventIngester", func() {
		It("should create EventIngester Kubernetes resources", func() {
			By("applying the EventIngester CRD", func() {
				applicationConfig := map[string]interface{}{
					"armadaUrl": "localhost:50001",
					"foo": map[string]interface{}{
						"baz": "bar",
						"xxx": "yyy",
					},
				}
				applicationConfigYAML, err := yaml.Marshal(applicationConfig)
				Expect(err).NotTo(HaveOccurred())
				eventIngester := v1alpha1.EventIngester{
					ObjectMeta: metav1.ObjectMeta{Name: "eventingester", Namespace: "default"},
					Spec: v1alpha1.EventIngesterSpec{
						Name: "test",
						Image: v1alpha1.Image{
							Repository: "testrepo",
							Tag:        "1.0.2",
						},
						ApplicationConfig: runtime.RawExtension{Raw: applicationConfigYAML},
					},
				}
				Expect(k8sClient.Create(ctx, &eventIngester)).Should(Succeed())
			})
		})
	})
})
