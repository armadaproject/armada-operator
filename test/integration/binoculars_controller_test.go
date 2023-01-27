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

var _ = Describe("Binoculars Controller", func() {
	When("User applies a new Binoculars YAML using kubectl", func() {
		It("Kubernetes should create the Binoculars Kubernetes resources", func() {
			By("Calling the Binoculars Controller Reconcile function", func() {
				f, err := os.Open("./resources/binoculars1.yaml")
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
				Expect(string(stdinBytes)).To(Equal("binoculars.install.armadaproject.io/binoculars-e2e-1 created\n"))

				time.Sleep(2 * time.Second)

				binoculars := installv1alpha1.Binoculars{}
				binocularsKey := kclient.ObjectKey{Namespace: "default", Name: "binoculars-e2e-1"}
				err = k8sClient.Get(ctx, binocularsKey, &binoculars)
				Expect(err).NotTo(HaveOccurred())

				secret := corev1.Secret{}
				secretKey := kclient.ObjectKey{Namespace: "default", Name: "binoculars-e2e-1"}
				err = k8sClient.Get(ctx, secretKey, &secret)
				Expect(err).NotTo(HaveOccurred())
				Expect(secret.Data["binoculars-e2e-1-config.yaml"]).NotTo(BeEmpty())

				deployment := appsv1.Deployment{}
				deploymentKey := kclient.ObjectKey{Namespace: "default", Name: "binoculars-e2e-1"}
				err = k8sClient.Get(ctx, deploymentKey, &deployment)
				Expect(err).NotTo(HaveOccurred())
				Expect(deployment.Spec.Selector.MatchLabels["app"]).To(Equal("binoculars-e2e-1"))

				service := corev1.Service{}
				serviceKey := kclient.ObjectKey{Namespace: "default", Name: "binoculars-e2e-1"}
				err = k8sClient.Get(ctx, serviceKey, &service)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	When("User applies an existing Binoculars YAML with updated values using kubectl", func() {
		It("Kubernetes should update the Binoculars Kubernetes resources", func() {
			By("Calling the Binoculars Controller Reconcile function", func() {
				f, err := os.Open("./resources/binoculars2.yaml")
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
				Expect(string(stdinBytes)).To(BeEquivalentTo("binoculars.install.armadaproject.io/binoculars-e2e-2 created\n"))

				binoculars := installv1alpha1.Binoculars{}
				binocularsKey := kclient.ObjectKey{Namespace: "default", Name: "binoculars-e2e-2"}
				err = k8sClient.Get(ctx, binocularsKey, &binoculars)
				Expect(err).NotTo(HaveOccurred())
				Expect("test").NotTo(BeKeyOf(binoculars.Labels))

				f2, err := os.Open("./resources/binoculars2-updated.yaml")
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
				Expect(string(stdinBytes)).To(Equal("binoculars.install.armadaproject.io/binoculars-e2e-2 configured\n"))

				time.Sleep(2 * time.Second)

				binoculars = installv1alpha1.Binoculars{}
				err = k8sClient.Get(ctx, binocularsKey, &binoculars)
				Expect(err).NotTo(HaveOccurred())
				Expect(binoculars.Labels["test"]).To(BeEquivalentTo("updated"))
			})
		})
	})
})
