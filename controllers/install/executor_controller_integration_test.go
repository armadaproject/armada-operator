package install

import (
	"github.com/armadaproject/armada-operator/apis/common"
	"github.com/armadaproject/armada-operator/apis/install/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Executor controller", func() {
	Context("When creating Executor", func() {
		It("should create Executor Kubernetes resources", func() {
			By("applying the Executor CRD", func() {
				executor := v1alpha1.Executor{
					ObjectMeta: metav1.ObjectMeta{Name: "executor", Namespace: "default"},
					Spec: v1alpha1.ExecutorSpec{
						Name: "test",
						Image: common.Image{
							Repository: "testrepo",
							Image:      "executor",
							Tag:        "1.0.2",
						},
						ApplicationConfig: nil,
					},
				}
				Expect(k8sClient.Create(ctx, &executor)).Should(Succeed())
			})
		})
	})
})
