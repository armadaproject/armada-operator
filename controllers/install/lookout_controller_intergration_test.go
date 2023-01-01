package install

import (
	"github.com/armadaproject/armada-operator/apis/common"
	"github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Lookout controller", func() {
	Context("When creating Lookout", func() {
		It("should create Lookout Kubernetes resources", func() {
			By("applying the Lookout CRD", func() {
				lookout := v1alpha1.Lookout{
					ObjectMeta: metav1.ObjectMeta{Name: "lookout", Namespace: "default"},
					Spec: v1alpha1.LookoutSpec{
						Name: "test",
						Image: common.Image{
							Repository: "testrepo",
							Image:      "lookout",
							Tag:        "1.0.2",
						},
						ApplicationConfig: nil,
					},
				}
				Expect(k8sClient.Create(ctx, &lookout)).Should(Succeed())
			})
		})
	})
})
