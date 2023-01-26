package integration

import (
	"io"
	"net/http"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("LookoutIngester Controller", func() {
	// BeforeEach(func() {
	// 	Expect(k8sClient.Create(ctx, &namespaceObject)).ToNot(HaveOccurred())
	// })
	// AfterEach(func() {
	// 	Expect(k8sClient.Delete(ctx, &namespaceObject)).ToNot(HaveOccurred())
	// })
	When("User applies a new LookoutIngester YAML using kubectl", func() {
		It("Kubernetes should create the LookoutIngester Kubernetes resources", func() {
			By("Calling the LookoutIngester Controller Reconcile function", func() {
				const namespace = "default"
				f, err := os.Open("./resources/lookoutIngester1.yaml")
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
				Expect(string(stdinBytes)).To(Equal("lookoutingester.install.armadaproject.io/lookoutingester-e2e-1 created\n"))

				time.Sleep(2 * time.Second)

				lookoutIngester := installv1alpha1.LookoutIngester{}
				lookoutIngesterKey := kclient.ObjectKey{Namespace: namespace, Name: "lookoutingester-e2e-1"}
				err = k8sClient.Get(ctx, lookoutIngesterKey, &lookoutIngester)
				Expect(err).NotTo(HaveOccurred())

				secret := corev1.Secret{}
				secretKey := kclient.ObjectKey{Namespace: namespace, Name: "lookoutingester-e2e-1"}
				err = k8sClient.Get(ctx, secretKey, &secret)
				Expect(err).NotTo(HaveOccurred())
				Expect(secret.Data["lookoutingester-e2e-1-config.yaml"]).NotTo(BeEmpty())

				deployment := appsv1.Deployment{}
				deploymentKey := kclient.ObjectKey{Namespace: namespace, Name: "lookoutingester-e2e-1"}
				err = k8sClient.Get(ctx, deploymentKey, &deployment)
				Expect(err).NotTo(HaveOccurred())
				Expect(deployment.Spec.Selector.MatchLabels["app"]).To(Equal("lookoutingester-e2e-1"))

				_, stderr, err = k.Run("delete", "-f", f.Name())
				if err != nil {
					stderrBytes, err := io.ReadAll(stderr)
					Expect(err).ToNot(HaveOccurred())
					Fail(string(stderrBytes))
				}
			})
		})
	})

	When("User applies an existing LookoutIngester YAML with updated values using kubectl", func() {
		It("Kubernetes should update the LookoutIngester Kubernetes resources", func() {
			By("Calling the LookoutIngester Controller Reconcile function", func() {
				f1, err := os.Open("./resources/lookoutIngester2.yaml")
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
				Expect(string(stdinBytes)).To(BeEquivalentTo("lookoutingester.install.armadaproject.io/lookoutingester-e2e-2 created\n"))

				lookoutIngester := installv1alpha1.LookoutIngester{}
				lookoutIngesterKey := kclient.ObjectKey{Namespace: "default", Name: "lookoutingester-e2e-2"}
				err = k8sClient.Get(ctx, lookoutIngesterKey, &lookoutIngester)
				Expect(err).NotTo(HaveOccurred())
				Expect("test").NotTo(BeKeyOf(lookoutIngester.Labels))

				f2, err := os.Open("./resources/lookoutIngester2-updated.yaml")
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
				Expect(string(stdinBytes)).To(Equal("lookoutingester.install.armadaproject.io/lookoutingester-e2e-2 configured\n"))

				time.Sleep(2 * time.Second)

				lookoutIngester = installv1alpha1.LookoutIngester{}
				err = k8sClient.Get(ctx, lookoutIngesterKey, &lookoutIngester)
				Expect(err).NotTo(HaveOccurred())
				Expect(lookoutIngester.Labels["test"]).To(BeEquivalentTo("updated"))

				_, stderr, err = k.Run("delete", "-f", f2.Name())
				if err != nil {
					stderrBytes, err := io.ReadAll(stderr)
					Expect(err).ToNot(HaveOccurred())
					Fail(string(stderrBytes))
				}
			})
		})
	})

	When("User deletes an existing LookoutIngester YAML using kubectl", func() {
		It("Kubernetes should delete the LookoutIngester Kubernetes resources", func() {
			By("Calling the LookoutIngester Controller Reconcile function", func() {
				f, err := os.Open("./resources/lookoutIngester3.yaml")
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
				Expect(string(stdinBytes)).To(BeEquivalentTo("lookoutingester.install.armadaproject.io/lookoutingester-e2e-3 created\n"))

				time.Sleep(1 * time.Second)

				stdin, stderr, err = k.Run("delete", "-f", f.Name())
				if err != nil {
					stderrBytes, err := io.ReadAll(stderr)
					Expect(err).ToNot(HaveOccurred())
					Fail(string(stderrBytes))
				}
				stdinBytes, err = io.ReadAll(stdin)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(stdinBytes)).To(Equal("lookoutingester.install.armadaproject.io \"lookoutingester-e2e-3\" deleted\n"))

				time.Sleep(2 * time.Second)

				lookoutIngester := installv1alpha1.LookoutIngester{}
				lookoutIngesterKey := kclient.ObjectKey{Namespace: "default", Name: "lookoutingester-e2e-3"}
				err = k8sClient.Get(ctx, lookoutIngesterKey, &lookoutIngester)
				Expect(err).To(BeAssignableToTypeOf(&errors.StatusError{}))
				notFoundErr := err.(*errors.StatusError)
				Expect(notFoundErr.ErrStatus.Code).To(BeEquivalentTo(http.StatusNotFound))

				secret := corev1.Secret{}
				secretKey := kclient.ObjectKey{Namespace: "lookoutingester", Name: "lookoutingester-e2e-3"}
				err = k8sClient.Get(ctx, secretKey, &secret)
				Expect(err).To(BeAssignableToTypeOf(&errors.StatusError{}))
				notFoundErr = err.(*errors.StatusError)
				Expect(notFoundErr.ErrStatus.Code).To(BeEquivalentTo(http.StatusNotFound))

				deployment := appsv1.Deployment{}
				deploymentKey := kclient.ObjectKey{Namespace: "lookoutingester", Name: "lookoutingester-e2e-3"}
				err = k8sClient.Get(ctx, deploymentKey, &deployment)
				Expect(err).To(BeAssignableToTypeOf(&errors.StatusError{}))
				notFoundErr = err.(*errors.StatusError)
				Expect(notFoundErr.ErrStatus.Code).To(BeEquivalentTo(http.StatusNotFound))
			})
		})
	})
})
