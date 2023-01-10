package integration

import (
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/armadaproject/armada-operator/controllers/utils"
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
				defer func() {
					Expect(f.Close()).ToNot(HaveOccurred())
				}()
				defer func() {
					Expect(os.Remove(f.Name())).ToNot(HaveOccurred())
				}()
				Expect(err).ToNot(HaveOccurred())

				k, err := testUser.Kubectl()
				Expect(err).ToNot(HaveOccurred())
				_, stderr, err := k.Run("apply", "-f", f.Name())
				if err != nil {
					stderrBytes, err := io.ReadAll(stderr)
					Expect(err).ToNot(HaveOccurred())
					Fail(string(stderrBytes))
				}
			})
		})
	})
})
