package install

import (
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/armadaproject/armada-operator/controllers/utils"
)

var lookoutYaml = `apiVersion: install.armadaproject.io/v1alpha1
kind: Lookout
metadata:
  labels:
    app.kubernetes.io/name: lookout
    app.kubernetes.io/instance: lookout-sample
    app.kubernetes.io/part-of: armada-operator
    app.kubernetes.io/created-by: armada-operator
  name: lookout-e2e
spec:
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

var _ = Describe("Armada Operator", func() {
	When("User applies Lookout YAML using kubectl", func() {
		It("Kubernetes should create Lookout Kubernetes resources", func() {
			By("Calling the Lookout Controller Reconcile function", func() {
				f, err := utils.CreateTempFile([]byte(lookoutYaml))
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
