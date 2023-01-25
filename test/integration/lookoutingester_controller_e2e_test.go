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

var lookoutIngesterYaml1 = `apiVersion: install.armadaproject.io/v1alpha1
kind: LookoutIngester
metadata:
  labels:
    app.kubernetes.io/name: lookoutingester
    app.kubernetes.io/instance: lookoutingester-sample
    app.kubernetes.io/part-of: armada-operator
    app.kubernetes.io/created-by: armada-operator
  name: lookoutingester-e2e-1
  namespace: default
spec:
  image:
    repository: test-lookoutingester
    tag: latest
  applicationConfig:
    postgres:
      maxOpenConns: 100
      maxIdleConns: 25
      connMaxLifetime: 30m
      connection:
        host: postgres
        port: 5432
        user: postgres
        password: psw
        dbname: postgres
        sslmode: disable
    metrics:
      port: 9000
    pulsar:
      enabled: true
      URL: "pulsar://pulsar:6650"
      jobsetEventsTopic: "events"
      receiveTimeout: 5s
      backoffTime: 1s
    paralellism: 1
    subscriptionName: "lookout-ingester"
    batchSize: 10000
    batchDuration: 500ms
    minJobSpecCompressionSize: 1024
    userAnnotationPrefix: "armadaproject.io/"
`

var lookoutIngesterYaml2 = `apiVersion: install.armadaproject.io/v1alpha1
kind: LookoutIngester
metadata:
  name: lookoutingester-e2e-2
  namespace: default
spec:
  image:
    repository: test-lookoutingester
    tag: latest
  applicationConfig:
    postgres:
      maxOpenConns: 100
      maxIdleConns: 25
      connMaxLifetime: 30m
      connection:
        host: postgres
        port: 5432
        user: postgres
        password: psw
        dbname: postgres
        sslmode: disable
    metrics:
      port: 9000
    pulsar:
      enabled: true
      URL: "pulsar://pulsar:6650"
      jobsetEventsTopic: "events"
      receiveTimeout: 5s
      backoffTime: 1s
    paralellism: 1
    subscriptionName: "lookout-ingester"
    batchSize: 10000
    batchDuration: 500ms
    minJobSpecCompressionSize: 1024
    userAnnotationPrefix: "armadaproject.io/"
`

var lookoutIngesterYaml2Updated = `apiVersion: install.armadaproject.io/v1alpha1
kind: LookoutIngester
metadata:
  labels:
    test: updated
  name: lookoutingester-e2e-2
  namespace: default
spec:
  image:
    repository: test-lookoutingester
    tag: latest
  applicationConfig:
    postgres:
      maxOpenConns: 100
      maxIdleConns: 25
      connMaxLifetime: 30m
      connection:
        host: postgres
        port: 5432
        user: postgres
        password: psw
        dbname: postgres
        sslmode: disable
    metrics:
      port: 9000
    pulsar:
      enabled: true
      URL: "pulsar://pulsar:6650"
      jobsetEventsTopic: "events"
      receiveTimeout: 5s
      backoffTime: 1s
    paralellism: 1
    subscriptionName: "lookout-ingester"
    batchSize: 10000
    batchDuration: 500ms
    minJobSpecCompressionSize: 1024
    userAnnotationPrefix: "armadaproject.io/"
`

var lookoutIngesterYaml3 = `apiVersion: install.armadaproject.io/v1alpha1
kind: LookoutIngester
metadata:
  name: lookoutingester-e2e-3
  namespace: default
spec:
  image:
    repository: test-lookoutingester
    tag: latest
  applicationConfig:
    postgres:
      maxOpenConns: 100
      maxIdleConns: 25
      connMaxLifetime: 30m
      connection:
        host: postgres
        port: 5432
        user: postgres
        password: psw
        dbname: postgres
        sslmode: disable
    metrics:
      port: 9000
    pulsar:
      enabled: true
      URL: "pulsar://pulsar:6650"
      jobsetEventsTopic: "events"
      receiveTimeout: 5s
      backoffTime: 1s
    paralellism: 1
    subscriptionName: "lookout-ingester"
    batchSize: 10000
    batchDuration: 500ms
    minJobSpecCompressionSize: 1024
    userAnnotationPrefix: "armadaproject.io/"
`

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
				f, err := CreateTempFile([]byte(lookoutIngesterYaml1))
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
				f1, err := CreateTempFile([]byte(lookoutIngesterYaml2))
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
				Expect(string(stdinBytes)).To(BeEquivalentTo("lookoutingester.install.armadaproject.io/lookoutingester-e2e-2 created\n"))

				lookoutIngester := installv1alpha1.LookoutIngester{}
				lookoutIngesterKey := kclient.ObjectKey{Namespace: "default", Name: "lookoutingester-e2e-2"}
				err = k8sClient.Get(ctx, lookoutIngesterKey, &lookoutIngester)
				Expect(err).NotTo(HaveOccurred())
				Expect("test").NotTo(BeKeyOf(lookoutIngester.Labels))

				f2, err := CreateTempFile([]byte(lookoutIngesterYaml2Updated))
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
				f, err := CreateTempFile([]byte(lookoutIngesterYaml3))
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
