package integration

import (
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"
)

var _ = Describe("Armada Operator", func() {
	When("User applies a new ArmadaServer YAML using kubectl", func() {
		It("Kubernetes should create ArmadaServer Kubernetes resources", func() {
			By("Calling the ArmadaServer Controller Reconcile function", func() {
				f, err := os.Open("./resources/server1.yaml")
				Expect(err).ToNot(HaveOccurred())
				defer f.Close()
				Expect(err).ToNot(HaveOccurred())
				defer f.Close()

				k, err := testUser.Kubectl()
				Expect(err).ToNot(HaveOccurred())
				stdout, stderr, err := k.Run("create", "-f", f.Name())
				if err != nil {
					stderrBytes, err := io.ReadAll(stderr)
					Expect(err).ToNot(HaveOccurred())
					Fail(string(stderrBytes))
				}
				stdoutBytes, err := io.ReadAll(stdout)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(stdoutBytes)).To(Equal("armadaserver.install.armadaproject.io/armadaserver-e2e created\n"))

				as := installv1alpha1.ArmadaServer{}
				asKey := kclient.ObjectKey{Namespace: "default", Name: "armadaserver-e2e"}
				Eventually(func() error {
					return k8sClient.Get(ctx, asKey, &as)
				}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())

				secret := corev1.Secret{}
				secretKey := kclient.ObjectKey{Namespace: "default", Name: "armadaserver-e2e"}
				Eventually(func() error {
					return k8sClient.Get(ctx, secretKey, &secret)
				}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())
				Expect(secret.Data["armadaserver-e2e-config.yaml"]).NotTo(BeEmpty())

				deployment := appsv1.Deployment{}
				deploymentKey := kclient.ObjectKey{Namespace: "default", Name: "armadaserver-e2e"}
				Eventually(func() error {
					return k8sClient.Get(ctx, deploymentKey, &deployment)
				}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())
				Expect(deployment.Spec.Selector.MatchLabels["app"]).To(Equal("armadaserver-e2e"))

				service := corev1.Service{}
				serviceKey := kclient.ObjectKey{Namespace: "default", Name: "armadaserver-e2e"}
				Eventually(func() error {
					return k8sClient.Get(ctx, serviceKey, &service)
				}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())
			})
		})
	})

	// TODO Add second 'When' scenario - for updating
})
