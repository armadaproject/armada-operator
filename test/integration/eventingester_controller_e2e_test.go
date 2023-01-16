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

var eventIngesterYaml1 = `apiVersion: install.armadaproject.io/v1alpha1
kind: EventIngester
metadata:
  labels:
    app.kubernetes.io/name: eventingester
    app.kubernetes.io/instance: eventingester-sample
    app.kubernetes.io/part-of: armada-operator
    app.kubernetes.io/created-by: armada-operator
  name: eventingester-e2e-1
  namespace: default
spec:
  image:
    repository: test-eventingester
    tag: latest
  applicationConfig:
    server: example.com:443
    forceNoTls: true
    toleratedTaints:
      - key: armada.io/batch
        operator: in
`

var eventIngesterYaml2 = `apiVersion: install.armadaproject.io/v1alpha1
kind: EventIngester
metadata:
  name: eventingester-e2e-2
  namespace: default
spec:
  image:
    repository: test-eventingester
    tag: latest
  applicationConfig:
    server: example.com:443
    forceNoTls: true
    toleratedTaints:
      - key: armada.io/batch
        operator: in
`

var eventIngesterYaml2Updated = `apiVersion: install.armadaproject.io/v1alpha1
kind: EventIngester
metadata:
  labels:
    test: updated
  name: eventingester-e2e-2
  namespace: default
spec:
  image:
    repository: test-eventingester
    tag: latest
  applicationConfig:
    server: example.com:443
    forceNoTls: true
    toleratedTaints:
      - key: armada.io/batch
        operator: in
`

var eventIngesterYaml3 = `apiVersion: install.armadaproject.io/v1alpha1
kind: EventIngester
metadata:
  name: eventingester-e2e-3
  namespace: default
spec:
  image:
    repository: test-eventingester
    tag: latest
  applicationConfig:
    server: example.com:443
    forceNoTls: true
    toleratedTaints:
      - key: armada.io/batch
        operator: in
`

var _ = Describe("EventIngester Controller", func() {
	// BeforeEach(func() {
	// 	Expect(k8sClient.Create(ctx, &namespaceObject)).ToNot(HaveOccurred())
	// })
	// AfterEach(func() {
	// 	Expect(k8sClient.Delete(ctx, &namespaceObject)).ToNot(HaveOccurred())
	// })
	When("User applies a new EventIngester YAML using kubectl", func() {
		It("Kubernetes should create the EventIngester Kubernetes resources", func() {
			By("Calling the EventIngester Controller Reconcile function", func() {
				const namespace = "default"
				f, err := CreateTempFile([]byte(eventIngesterYaml1))
				Expect(err).ToNot(HaveOccurred())
				defer f.Close()
				defer os.Remove(f.Name())

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

				time.Sleep(2 * time.Second)

				eventIngester := installv1alpha1.EventIngester{}
				eventIngesterKey := kclient.ObjectKey{Namespace: namespace, Name: "eventingester-e2e-1"}
				err = k8sClient.Get(ctx, eventIngesterKey, &eventIngester)
				Expect(err).NotTo(HaveOccurred())

				secret := corev1.Secret{}
				secretKey := kclient.ObjectKey{Namespace: namespace, Name: "eventingester-e2e-1"}
				err = k8sClient.Get(ctx, secretKey, &secret)
				Expect(err).NotTo(HaveOccurred())
				Expect(secret.Data["armada-config.yaml"]).NotTo(BeEmpty())

				deployment := appsv1.Deployment{}
				deploymentKey := kclient.ObjectKey{Namespace: namespace, Name: "eventingester-e2e-1"}
				err = k8sClient.Get(ctx, deploymentKey, &deployment)
				Expect(err).NotTo(HaveOccurred())
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
				f1, err := CreateTempFile([]byte(eventIngesterYaml2))
				Expect(err).ToNot(HaveOccurred())
				defer f1.Close()
				defer os.Remove(f1.Name())

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
				Expect(string(stdinBytes)).To(BeEquivalentTo("eventingester.install.armadaproject.io/eventingester-e2e-2 created\n"))

				eventIngester := installv1alpha1.EventIngester{}
				eventIngesterKey := kclient.ObjectKey{Namespace: "default", Name: "eventingester-e2e-2"}
				err = k8sClient.Get(ctx, eventIngesterKey, &eventIngester)
				Expect(err).NotTo(HaveOccurred())
				Expect("test").NotTo(BeKeyOf(eventIngester.Labels))

				f2, err := CreateTempFile([]byte(eventIngesterYaml2Updated))
				Expect(err).ToNot(HaveOccurred())
				defer f2.Close()
				defer os.Remove(f2.Name())

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

				time.Sleep(2 * time.Second)

				eventIngester = installv1alpha1.EventIngester{}
				err = k8sClient.Get(ctx, eventIngesterKey, &eventIngester)
				Expect(err).NotTo(HaveOccurred())
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
				f, err := CreateTempFile([]byte(eventIngesterYaml3))
				Expect(err).ToNot(HaveOccurred())
				defer f.Close()
				defer os.Remove(f.Name())

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

				time.Sleep(1 * time.Second)

				stdin, stderr, err = k.Run("delete", "-f", f.Name())
				if err != nil {
					stderrBytes, err := io.ReadAll(stderr)
					Expect(err).ToNot(HaveOccurred())
					Fail(string(stderrBytes))
				}
				stdinBytes, err = io.ReadAll(stdin)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(stdinBytes)).To(Equal("eventingester.install.armadaproject.io \"eventingester-e2e-3\" deleted\n"))

				time.Sleep(2 * time.Second)

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
