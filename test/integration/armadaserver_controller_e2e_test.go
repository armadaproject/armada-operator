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

const armadaserverYaml1 = `apiVersion: install.armadaproject.io/v1alpha1
kind: ArmadaServer
metadata:
  labels:
    app.kubernetes.io/name: armadaserver
    app.kubernetes.io/instance: armadaserver-sample
    app.kubernetes.io/part-of: armada-operator
    app.kubernetes.io/created-by: armada-operator
  name: armadaserver-e2e-1
  namespace: default
spec:
  ingress:
    ingressClass: nginx
  clusterIssuer: test
  image:
    repository: test-armadaserver
    tag: latest
  applicationConfig:
    server: example.com:443
    forceNoTls: true
    toleratedTaints:
      - key: armada.io/batch
        operator: in
`

var _ = Describe("Armada Operator", func() {
	When("User applies a new ArmadaServer YAML using kubectl", func() {
		It("Kubernetes should create ArmadaServer Kubernetes resources", func() {
			By("Calling the ArmadaServer Controller Reconcile function", func() {
				f, err := CreateTempFile([]byte(armadaserverYaml1))
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
				Expect(string(stdinBytes)).To(Equal("armadaserver.install.armadaproject.io/armadaserver-e2e-1 created\n"))

				time.Sleep(2 * time.Second)

				as := installv1alpha1.ArmadaServer{}
				asKey := kclient.ObjectKey{Namespace: "default", Name: "armadaserver-e2e-1"}
				err = k8sClient.Get(ctx, asKey, &as)
				Expect(err).NotTo(HaveOccurred())

				secret := corev1.Secret{}
				secretKey := kclient.ObjectKey{Namespace: "default", Name: "armadaserver-e2e-1"}
				err = k8sClient.Get(ctx, secretKey, &secret)
				Expect(err).NotTo(HaveOccurred())
				Expect(secret.Data["armadaserver-e2e-1-config.yaml"]).NotTo(BeEmpty())

				deployment := appsv1.Deployment{}
				deploymentKey := kclient.ObjectKey{Namespace: "default", Name: "armadaserver-e2e-1"}
				err = k8sClient.Get(ctx, deploymentKey, &deployment)
				Expect(err).NotTo(HaveOccurred())
				Expect(deployment.Spec.Selector.MatchLabels["app"]).To(Equal("armadaserver-e2e-1"))

				service := corev1.Service{}
				serviceKey := kclient.ObjectKey{Namespace: "default", Name: "armadaserver-e2e-1"}
				err = k8sClient.Get(ctx, serviceKey, &service)
				Expect(err).NotTo(HaveOccurred())

			})
		})
	})

	// TODO Add second 'When' scenario - for updating
})
