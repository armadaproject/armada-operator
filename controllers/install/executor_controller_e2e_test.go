package install

import (
	"io"
	"os"
	"time"

	"github.com/armadaproject/armada-operator/controllers/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var executorYaml = `apiVersion: install.armadaproject.io/v1alpha1
kind: Executor
metadata:
  labels:
    app.kubernetes.io/name: executor
    app.kubernetes.io/instance: executor-sample
    app.kubernetes.io/part-of: armada-operator
    app.kubernetes.io/created-by: armada-operator
  name: executor-e2e
spec:
  image:
    repository: test-executor
    tag: latest
  applicationConfig:
    server: example.com:443
    forceNoTls: true
    toleratedTaints:
      - key: armada.io/batch
        operator: in
`

var _ = Describe("Armada Operator", func() {
	When("User applies Executor YAML using kubectl", func() {
		It("Kubernetes should create Executor Kubernetes resources", func() {
			By("Calling the Executor Controller Reconcile function", func() {
				f, err := utils.CreateTempFile([]byte(executorYaml))
				Expect(err).ToNot(HaveOccurred())
				defer f.Close()
				defer os.Remove(f.Name())

				k, err := testUser.Kubectl()
				Expect(err).ToNot(HaveOccurred())
				_, stderr, err := k.Run("apply", "-f", f.Name())
				if err != nil {
					stderrBytes, err := io.ReadAll(stderr)
					Expect(err).ToNot(HaveOccurred())
					Fail(string(stderrBytes))
				}

				time.Sleep(5 * time.Second)

				secret := corev1.Secret{}
				secretKey := kclient.ObjectKey{Namespace: "default", Name: "executor-e2e"}
				err = k8sClient.Get(ctx, secretKey, &secret)
				Expect(err).NotTo(HaveOccurred())
				Expect(secret.Data["armada-config.yaml"]).NotTo(BeEmpty())

				deployment := appsv1.Deployment{}
				deploymentKey := kclient.ObjectKey{Namespace: "default", Name: "executor-e2e"}
				err = k8sClient.Get(ctx, deploymentKey, &deployment)
				Expect(err).NotTo(HaveOccurred())
				Expect(deployment.Spec.Selector.MatchLabels["app"]).To(Equal("executor-e2e"))

				service := corev1.Service{}
				serviceKey := kclient.ObjectKey{Namespace: "default", Name: "executor-e2e"}
				err = k8sClient.Get(ctx, serviceKey, &service)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
