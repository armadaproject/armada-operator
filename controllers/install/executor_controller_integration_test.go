package install

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/armadaproject/armada-operator/apis/install/v1alpha1"
)

var _ = Describe("Executor controller", func() {
	When("Executor is created using k8s go-client", func() {
		It("Kubernetes should create Executor Kubernetes resources", func() {
			By("calling the Executor Controller Reconcile function", func() {
				applicationConfig := map[string]interface{}{
					"armadaUrl": "localhost:50001",
					"foo": map[string]interface{}{
						"baz": "bar",
						"xxx": "yyy",
					},
				}
				applicationConfigJSON, err := json.Marshal(applicationConfig)
				Expect(err).NotTo(HaveOccurred())
				executor := v1alpha1.Executor{
					ObjectMeta: metav1.ObjectMeta{Name: "executor", Namespace: "default"},
					Spec: v1alpha1.ExecutorSpec{
						Image: v1alpha1.Image{
							Repository: "executor",
							Tag:        "1.0.2",
						},
						ApplicationConfig: runtime.RawExtension{Raw: applicationConfigJSON},
					},
				}
				Expect(k8sClient.Create(ctx, &executor)).Should(Succeed())

				time.Sleep(5 * time.Second)

				secret := corev1.Secret{}
				secretKey := kclient.ObjectKey{Namespace: "default", Name: "executor-e2e"}
				err = k8sClient.Get(ctx, secretKey, &secret)
				Expect(err).NotTo(HaveOccurred())
				Expect(secret.Data["armada-config.yaml"]).NotTo(BeEmpty())

				deployment := appsv1.Deployment{}
				deploymentKey := kclient.ObjectKey{Namespace: "default", Name: "executor-e2e"}
				err = k8sClient.Get(ctx, deploymentKey, &deployment)
				Expect(err).NotTo(HaveOccurred())
				Expect(deployment.Spec.Selector.MatchLabels["app"]).To(Equal("executor-e2e"))

				service := corev1.Service{}
				serviceKey := kclient.ObjectKey{Namespace: "default", Name: "executor-e2e"}
				err = k8sClient.Get(ctx, serviceKey, &service)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
