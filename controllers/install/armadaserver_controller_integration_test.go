package install

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	"github.com/armadaproject/armada-operator/apis/install/v1alpha1"
)

var _ = Describe("ArmadaServer controller", func() {
	When("ArmadaServer is created using k8s go-client", func() {
		It("Kubernetes should create ArmadaServer Kubernetes resources", func() {
			By("calling the ArmadaServer Controller Reconcile function", func() {
				applicationConfig := map[string]interface{}{
					"armadaUrl": "localhost:50001",
					"foo": map[string]interface{}{
						"baz": "bar",
						"xxx": "yyy",
					},
				}
				applicationConfigYAML, err := yaml.Marshal(applicationConfig)
				Expect(err).NotTo(HaveOccurred())
				armadaserver := v1alpha1.ArmadaServer{
					ObjectMeta: metav1.ObjectMeta{Name: "armadaserver", Namespace: "default"},
					Spec: v1alpha1.ArmadaServerSpec{
						Image: v1alpha1.Image{
							Repository: "armadaserver",
							Tag:        "1.0.2",
						},
						ApplicationConfig: runtime.RawExtension{Raw: applicationConfigYAML},
					},
				}
				Expect(k8sClient.Create(ctx, &armadaserver)).Should(Succeed())
			})
		})
	})
})
