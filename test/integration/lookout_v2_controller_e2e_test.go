package integration

import (
	"io"
	"os"
	"time"

	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Armada Operator", func() {
	When("User applies LookoutV2 YAML using kubectl", func() {
		It("Kubernetes should create LookoutV2 Kubernetes resources", func() {
			By("Calling the LookoutV2 Controller Reconcile function", func() {
				f, err := os.Open("./resources/lookout-v2-1.yaml")
				Expect(err).ToNot(HaveOccurred())
				defer f.Close()

				k, err := testUser.Kubectl()
				Expect(err).ToNot(HaveOccurred())
				stdin, stderr, err := k.Run("create", "-f", f.Name())
				if err != nil {
					stderrBytes, err := io.ReadAll(stderr)
					Expect(err).ToNot(HaveOccurred())
					Fail(string(stderrBytes))
				}
				stdinBytes, err := io.ReadAll(stdin)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(stdinBytes)).To(Equal("lookoutv2.install.armadaproject.io/lookout-v2-e2e-1 created\n"))

				time.Sleep(2 * time.Second)

				lookoutV2 := installv1alpha1.LookoutV2{}
				lookoutV2Key := kclient.ObjectKey{Namespace: "default", Name: "lookout-v2-e2e-1"}
				err = k8sClient.Get(ctx, lookoutV2Key, &lookoutV2)
				Expect(err).NotTo(HaveOccurred())

				secret := corev1.Secret{}
				secretKey := kclient.ObjectKey{Namespace: "default", Name: "lookout-v2-e2e-1"}
				err = k8sClient.Get(ctx, secretKey, &secret)
				Expect(err).NotTo(HaveOccurred())
				Expect(secret.Data["lookout-v2-e2e-1-config.yaml"]).NotTo(BeEmpty())

				deployment := appsv1.Deployment{}
				deploymentKey := kclient.ObjectKey{Namespace: "default", Name: "lookout-v2-e2e-1"}
				err = k8sClient.Get(ctx, deploymentKey, &deployment)
				Expect(err).NotTo(HaveOccurred())
				Expect(deployment.Spec.Selector.MatchLabels["app"]).To(Equal("lookout-v2-e2e-1"))

				service := corev1.Service{}
				serviceKey := kclient.ObjectKey{Namespace: "default", Name: "lookout-v2-e2e-1"}
				err = k8sClient.Get(ctx, serviceKey, &service)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	When("User applies an existing LookoutV2 YAML with updated values using kubectl", func() {
		It("Kubernetes should update the LookoutV2 Kubernetes resources", func() {
			By("Calling the LookoutV2 Controller Reconcile function", func() {
				f1, err := os.Open("./resources/lookout-v2-2.yaml")
				Expect(err).ToNot(HaveOccurred())
				defer f1.Close()

				k, err := testUser.Kubectl()
				Expect(err).ToNot(HaveOccurred())
				stdin, stderr, err := k.Run("create", "-f", f1.Name())
				if err != nil {
					stderrBytes, err := io.ReadAll(stderr)
					Expect(err).ToNot(HaveOccurred())
					Fail(string(stderrBytes))
				}
				stdinBytes, err := io.ReadAll(stdin)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(stdinBytes)).To(BeEquivalentTo("lookoutv2.install.armadaproject.io/lookout-v2-e2e-2 created\n"))

				lookoutV2 := installv1alpha1.LookoutV2{}
				lookoutV2Key := kclient.ObjectKey{Namespace: "default", Name: "lookout-v2-e2e-2"}
				err = k8sClient.Get(ctx, lookoutV2Key, &lookoutV2)
				Expect(err).NotTo(HaveOccurred())
				Expect("test").NotTo(BeKeyOf(lookoutV2.Labels))

				f2, err := os.Open("./resources/lookout-v2-2-updated.yaml")
				Expect(err).ToNot(HaveOccurred())
				defer f2.Close()

				Expect(err).ToNot(HaveOccurred())
				stdin, stderr, err = k.Run("apply", "-f", f2.Name())
				if err != nil {
					stderrBytes, err := io.ReadAll(stderr)
					Expect(err).ToNot(HaveOccurred())
					Fail(string(stderrBytes))
				}
				stdinBytes, err = io.ReadAll(stdin)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(stdinBytes)).To(Equal("lookoutv2.install.armadaproject.io/lookout-v2-e2e-2 configured\n"))

				time.Sleep(2 * time.Second)

				lookoutV2 = installv1alpha1.LookoutV2{}
				err = k8sClient.Get(ctx, lookoutV2Key, &lookoutV2)
				Expect(err).NotTo(HaveOccurred())
				Expect(lookoutV2.Labels["test"]).To(BeEquivalentTo("updated"))
			})
		})
	})
})
