package integration

import (
	"io"
	"net/http"
	"os"

	"k8s.io/apimachinery/pkg/api/errors"

	installv1alpha1 "github.com/armadaproject/armada-operator/api/install/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("EventIngester Controller", func() {
	When("User applies a new EventIngester YAML using kubectl", func() {
		It("Kubernetes should create the EventIngester Kubernetes resources", func() {
			By("Calling the EventIngester Controller Reconcile function", func() {
				f, err := os.Open("./resources/eventIngester1.yaml")
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
				Expect(string(stdinBytes)).To(Equal("eventingester.install.armadaproject.io/eventingester-e2e-1 created\n"))

				eventIngester := installv1alpha1.EventIngester{}
				eventIngesterKey := kclient.ObjectKey{Namespace: "default", Name: "eventingester-e2e-1"}
				Eventually(func() error {
					return k8sClient.Get(ctx, eventIngesterKey, &eventIngester)
				}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())

				secret := corev1.Secret{}
				secretKey := kclient.ObjectKey{Namespace: "default", Name: "eventingester-e2e-1"}
				Eventually(func() error {
					return k8sClient.Get(ctx, secretKey, &secret)
				}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())
				Expect(secret.Data["eventingester-e2e-1-config.yaml"]).NotTo(BeEmpty())

				deployment := appsv1.Deployment{}
				deploymentKey := kclient.ObjectKey{Namespace: "default", Name: "eventingester-e2e-1"}
				Eventually(func() error {
					return k8sClient.Get(ctx, deploymentKey, &deployment)
				}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())
				Expect(deployment.Spec.Selector.MatchLabels["app"]).To(Equal("eventingester-e2e-1"))

				_, stderr, err = k.Run("delete", "-f", f.Name())
				if err != nil {
					stderrBytes, err := io.ReadAll(stderr)
					Expect(err).ToNot(HaveOccurred())
					Fail(string(stderrBytes))
				}
			})
		})
	})

	When("User applies an existing EventIngester YAML with updated values using kubectl", func() {
		It("Kubernetes should update the EventIngester Kubernetes resources", func() {
			By("Calling the EventIngester Controller Reconcile function", func() {
				f, err := os.Open("./resources/eventIngester2.yaml")
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
				Expect(string(stdinBytes)).To(BeEquivalentTo("eventingester.install.armadaproject.io/eventingester-e2e-2 created\n"))

				eventIngester := installv1alpha1.EventIngester{}
				eventIngesterKey := kclient.ObjectKey{Namespace: "default", Name: "eventingester-e2e-2"}
				err = k8sClient.Get(ctx, eventIngesterKey, &eventIngester)
				Expect(err).NotTo(HaveOccurred())
				Expect("test").NotTo(BeKeyOf(eventIngester.Labels))

				f2, err := os.Open("./resources/eventIngester2-updated.yaml")
				Expect(err).ToNot(HaveOccurred())
				defer f.Close()

				Expect(err).ToNot(HaveOccurred())
				stdin, stderr, err = k.Run("apply", "-f", f2.Name())
				if err != nil {
					stderrBytes, err := io.ReadAll(stderr)
					Expect(err).ToNot(HaveOccurred())
					Fail(string(stderrBytes))
				}
				stdinBytes, err = io.ReadAll(stdin)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(stdinBytes)).To(Equal("eventingester.install.armadaproject.io/eventingester-e2e-2 configured\n"))

				eventIngester = installv1alpha1.EventIngester{}
				Eventually(func() error {
					return k8sClient.Get(ctx, eventIngesterKey, &eventIngester)
				}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())
				Expect(eventIngester.Labels["test"]).To(BeEquivalentTo("updated"))

				_, stderr, err = k.Run("delete", "-f", f2.Name())
				if err != nil {
					stderrBytes, err := io.ReadAll(stderr)
					Expect(err).ToNot(HaveOccurred())
					Fail(string(stderrBytes))
				}
			})
		})
	})

	When("User deletes an existing EventIngester YAML using kubectl", func() {
		It("Kubernetes should delete the EventIngester Kubernetes resources", func() {
			By("Calling the EventIngester Controller Reconcile function", func() {
				f, err := os.Open("./resources/eventIngester3.yaml")
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
				Expect(string(stdinBytes)).To(BeEquivalentTo("eventingester.install.armadaproject.io/eventingester-e2e-3 created\n"))

				stdin, stderr, err = k.Run("delete", "-f", f.Name())
				if err != nil {
					stderrBytes, err := io.ReadAll(stderr)
					Expect(err).ToNot(HaveOccurred())
					Fail(string(stderrBytes))
				}
				stdinBytes, err = io.ReadAll(stdin)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(stdinBytes)).To(Equal("eventingester.install.armadaproject.io \"eventingester-e2e-3\" deleted\n"))

				eventIngester := installv1alpha1.EventIngester{}
				eventIngesterKey := kclient.ObjectKey{Namespace: "default", Name: "eventingester-e2e-3"}
				err = k8sClient.Get(ctx, eventIngesterKey, &eventIngester)
				Expect(err).To(BeAssignableToTypeOf(&errors.StatusError{}))
				notFoundErr := err.(*errors.StatusError)
				Expect(notFoundErr.ErrStatus.Code).To(BeEquivalentTo(http.StatusNotFound))

				secret := corev1.Secret{}
				secretKey := kclient.ObjectKey{Namespace: "eventingester", Name: "eventingester-e2e-3"}
				err = k8sClient.Get(ctx, secretKey, &secret)
				Expect(err).To(BeAssignableToTypeOf(&errors.StatusError{}))
				notFoundErr = err.(*errors.StatusError)
				Expect(notFoundErr.ErrStatus.Code).To(BeEquivalentTo(http.StatusNotFound))

				deployment := appsv1.Deployment{}
				deploymentKey := kclient.ObjectKey{Namespace: "eventingester", Name: "eventingester-e2e-3"}
				err = k8sClient.Get(ctx, deploymentKey, &deployment)
				Expect(err).To(BeAssignableToTypeOf(&errors.StatusError{}))
				notFoundErr = err.(*errors.StatusError)
				Expect(notFoundErr.ErrStatus.Code).To(BeEquivalentTo(http.StatusNotFound))
			})
		})
	})
})
