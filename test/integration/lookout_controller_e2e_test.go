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

var lookoutYaml1 = `apiVersion: install.armadaproject.io/v1alpha1
kind: Lookout
metadata:
  labels:
    app.kubernetes.io/name: lookout
    app.kubernetes.io/instance: lookout-sample
    app.kubernetes.io/part-of: armada-operator
    app.kubernetes.io/created-by: armada-operator
  name: lookout-e2e-1
  namespace: default
spec:
  replicas: 2
  ingress:
    ingressClass: nginx
  clusterIssuer: test
  image:
    repository: test-lookout
    tag: latest
  applicationConfig:
    server: example.com:443
    forceNoTls: true
    toleratedTaints:
      - key: armada.io/batch
        operator: in
`

var lookoutYaml2 = `apiVersion: install.armadaproject.io/v1alpha1
kind: Lookout
metadata:
  labels:
    app.kubernetes.io/name: lookout
    app.kubernetes.io/instance: lookout-sample
    app.kubernetes.io/part-of: armada-operator
    app.kubernetes.io/created-by: armada-operator
  name: lookout-e2e-2
  namespace: default
spec:
  replicas: 2
  ingress:
    ingressClass: nginx
  clusterIssuer: test
  image:
    repository: test-lookout
    tag: latest
  applicationConfig:
    server: example.com:443
    forceNoTls: true
    toleratedTaints:
      - key: armada.io/batch
        operator: in
`

var LookoutYaml2Updated = `apiVersion: install.armadaproject.io/v1alpha1
kind: Lookout
metadata:
  labels:
    app.kubernetes.io/name: lookout
    app.kubernetes.io/instance: lookout-sample
    app.kubernetes.io/part-of: armada-operator
    app.kubernetes.io/created-by: armada-operator
    test: updated
  name: lookout-e2e-2
  namespace: default
spec:
  replicas: 2
  ingress:
    ingressClass: nginx
  clusterIssuer: test
  image:
    repository: test-lookout
    tag: latest
  applicationConfig:
    server: example.com:443
    forceNoTls: true
    toleratedTaints:
      - key: armada.io/batch
        operator: in
`

var _ = Describe("Lookout Controller", func() {
	When("User applies a new Lookout YAML using kubectl", func() {
		It("Kubernetes should create the Lookout Kubernetes resources", func() {
			By("Calling the Lookout Controller Reconcile function", func() {
				f, err := CreateTempFile([]byte(lookoutYaml1))
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
				Expect(string(stdinBytes)).To(Equal("lookout.install.armadaproject.io/lookout-e2e-1 created\n"))

				time.Sleep(2 * time.Second)

				lookout := installv1alpha1.Lookout{}
				lookoutKey := kclient.ObjectKey{Namespace: "default", Name: "lookout-e2e-1"}
				err = k8sClient.Get(ctx, lookoutKey, &lookout)
				Expect(err).NotTo(HaveOccurred())

				secret := corev1.Secret{}
				secretKey := kclient.ObjectKey{Namespace: "default", Name: "lookout-e2e-1"}
				err = k8sClient.Get(ctx, secretKey, &secret)
				Expect(err).NotTo(HaveOccurred())
				Expect(secret.Data["lookout-e2e-1-config.yaml"]).NotTo(BeEmpty())

				deployment := appsv1.Deployment{}
				deploymentKey := kclient.ObjectKey{Namespace: "default", Name: "lookout-e2e-1"}
				err = k8sClient.Get(ctx, deploymentKey, &deployment)
				Expect(err).NotTo(HaveOccurred())
				Expect(deployment.Spec.Selector.MatchLabels["app"]).To(Equal("lookout-e2e-1"))

				service := corev1.Service{}
				serviceKey := kclient.ObjectKey{Namespace: "default", Name: "lookout-e2e-1"}
				err = k8sClient.Get(ctx, serviceKey, &service)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	When("User applies an existing Lookout YAML with updated values using kubectl", func() {
		It("Kubernetes should update the Lookout Kubernetes resources", func() {
			By("Calling the Lookout Controller Reconcile function", func() {
				f1, err := CreateTempFile([]byte(lookoutYaml2))
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
				Expect(string(stdinBytes)).To(BeEquivalentTo("lookout.install.armadaproject.io/lookout-e2e-2 created\n"))

				lookout := installv1alpha1.Lookout{}
				lookoutKey := kclient.ObjectKey{Namespace: "default", Name: "lookout-e2e-2"}
				err = k8sClient.Get(ctx, lookoutKey, &lookout)
				Expect(err).NotTo(HaveOccurred())
				Expect("test").NotTo(BeKeyOf(lookout.Labels))

				f2, err := CreateTempFile([]byte(LookoutYaml2Updated))
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
				Expect(string(stdinBytes)).To(Equal("lookout.install.armadaproject.io/lookout-e2e-2 configured\n"))

				time.Sleep(2 * time.Second)

				lookout = installv1alpha1.Lookout{}
				err = k8sClient.Get(ctx, lookoutKey, &lookout)
				Expect(err).NotTo(HaveOccurred())
				Expect(lookout.Labels["test"]).To(BeEquivalentTo("updated"))
			})
		})
	})
})
