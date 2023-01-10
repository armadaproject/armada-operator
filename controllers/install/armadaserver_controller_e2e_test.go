package install

import (
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/armadaproject/armada-operator/controllers/utils"
)

var armadaserverYaml = `apiVersion: install.armadaproject.io/v1alpha1
kind: ArmadaServer
metadata:
  labels:
    app.kubernetes.io/name: armadaserver
    app.kubernetes.io/instance: armadaserver-sample
    app.kubernetes.io/part-of: armada-operator
    app.kubernetes.io/created-by: armada-operator
  name: armadaserver-e2e
spec:
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
	When("User applies ArmadaServer YAML using kubectl", func() {
		It("Kubernetes should create ArmadaServer Kubernetes resources", func() {
			By("Calling the ArmadaServer Controller Reconcile function", func() {
				f, err := utils.CreateTempFile([]byte(armadaserverYaml))
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
