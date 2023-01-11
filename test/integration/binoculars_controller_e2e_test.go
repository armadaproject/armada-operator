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

var BinocularsYaml1 = `apiVersion: install.armadaproject.io/v1alpha1
kind: Binoculars
metadata:
  labels:
    app.kubernetes.io/name: binoculars
    app.kubernetes.io/instance: binoculars-sample
    app.kubernetes.io/part-of: armada-operator
    app.kubernetes.io/created-by: armada-operator
  name: executor-e2e-1
spec:
  image:
    repository: test-binoculars
    tag: latest
  applicationConfig:
    server: example.com:443
    forceNoTls: true
    toleratedTaints:
      - key: armada.io/batch
        operator: in
`

var binocularsYaml2 = `apiVersion: install.armadaproject.io/v1alpha1
kind: Binoculars
metadata:
  labels:
    app.kubernetes.io/name: bincoulars
    app.kubernetes.io/instance: binoculars-sample
    app.kubernetes.io/part-of: armada-operator
    app.kubernetes.io/created-by: armada-operator
  name: executor-e2e-2
spec:
  image:
    repository: test-binoculars
    tag: latest
  applicationConfig:
    server: example.com:443
    forceNoTls: true
    toleratedTaints:
      - key: armada.io/batch
        operator: in
`

var BinocularsYaml2Updated = `apiVersion: install.armadaproject.io/v1alpha1
kind: Binoculars
metadata:
  labels:
    app.kubernetes.io/name: binoculars
    app.kubernetes.io/instance: bincoulars-sample
    app.kubernetes.io/part-of: armada-operator
    app.kubernetes.io/created-by: armada-operator
    test: updated
  name: binocoulars-e2e-2
spec:
  image:
    repository: test-binoculars
    tag: latest
  applicationConfig:
    server: example.com:443
    forceNoTls: true
    toleratedTaints:
      - key: armada.io/batch
        operator: in
`

var _ = Describe("Binoculars Controller", func() {
	When("User applies a new Binoculars YAML using kubectl", func() {
		It("Kubernetes should create the Binoculars Kubernetes resources", func() {
			By("Calling the Binoculars Controller Reconcile function", func() {
				f, err := CreateTempFile([]byte(executorYaml1))
				Expect(err).ToNot(HaveOccurred())
				defer f.Close()
				defer os.Remove(f.Name())

				k, err := testUser.Kubectl()
				Expect(err).ToNot(HaveOccurred())
				stdin, stderr, err := k.Run("apply", "-f", f.Name())
				if err != nil {
					stderrBytes, err := io.ReadAll(stderr)
					Expect(err).ToNot(HaveOccurred())
					Fail(string(stderrBytes))
				}
				stdinBytes, err := io.ReadAll(stdin)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(stdinBytes)).To(Equal("executor.install.armadaproject.io/executor-e2e-1 created\n"))

				time.Sleep(2 * time.Second)

				executor := installv1alpha1.Executor{}
				executorKey := kclient.ObjectKey{Namespace: "default", Name: "executor-e2e-1"}
				err = k8sClient.Get(ctx, executorKey, &executor)
				Expect(err).NotTo(HaveOccurred())

				secret := corev1.Secret{}
				secretKey := kclient.ObjectKey{Namespace: "default", Name: "executor-e2e-1"}
				err = k8sClient.Get(ctx, secretKey, &secret)
				Expect(err).NotTo(HaveOccurred())
				Expect(secret.Data["armada-config.yaml"]).NotTo(BeEmpty())

				deployment := appsv1.Deployment{}
				deploymentKey := kclient.ObjectKey{Namespace: "default", Name: "executor-e2e-1"}
				err = k8sClient.Get(ctx, deploymentKey, &deployment)
				Expect(err).NotTo(HaveOccurred())
				Expect(deployment.Spec.Selector.MatchLabels["app"]).To(Equal("executor-e2e-1"))

				service := corev1.Service{}
				serviceKey := kclient.ObjectKey{Namespace: "default", Name: "executor-e2e-1"}
				err = k8sClient.Get(ctx, serviceKey, &service)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	When("User applies an existing Executor YAML with updated values using kubectl", func() {
		It("Kubernetes should update the Executor Kubernetes resources", func() {
			By("Calling the Executor Controller Reconcile function", func() {
				f1, err := CreateTempFile([]byte(executorYaml2))
				Expect(err).ToNot(HaveOccurred())
				defer f1.Close()
				defer os.Remove(f1.Name())

				k, err := testUser.Kubectl()
				Expect(err).ToNot(HaveOccurred())
				stdin, stderr, err := k.Run("apply", "-f", f1.Name())
				if err != nil {
					stderrBytes, err := io.ReadAll(stderr)
					Expect(err).ToNot(HaveOccurred())
					Fail(string(stderrBytes))
				}
				stdinBytes, err := io.ReadAll(stdin)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(stdinBytes)).To(BeEquivalentTo("executor.install.armadaproject.io/executor-e2e-2 created\n"))

				executor := installv1alpha1.Executor{}
				executorKey := kclient.ObjectKey{Namespace: "default", Name: "executor-e2e-2"}
				err = k8sClient.Get(ctx, executorKey, &executor)
				Expect(err).NotTo(HaveOccurred())
				Expect("test").NotTo(BeKeyOf(executor.Labels))

				f2, err := CreateTempFile([]byte(executorYaml2Updated))
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
				Expect(string(stdinBytes)).To(Equal("executor.install.armadaproject.io/executor-e2e-2 configured\n"))

				time.Sleep(2 * time.Second)

				executor = installv1alpha1.Executor{}
				err = k8sClient.Get(ctx, executorKey, &executor)
				Expect(err).NotTo(HaveOccurred())
				Expect(executor.Labels["test"]).To(BeEquivalentTo("updated"))
			})
		})
	})
})
