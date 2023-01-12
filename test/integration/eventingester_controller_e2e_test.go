package integration

import (
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var eventYaml = `apiVersion: install.armadaproject.io/v1alpha1
kind: EventIngester
metadata:
  labels:
    app.kubernetes.io/name: eventingester
    app.kubernetes.io/instance: eventingester-sample
    app.kubernetes.io/part-of: armada-operator
    app.kubernetes.io/created-by: armada-operator
  name: eventingester-e2e
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

var _ = Describe("Armada Operator", func() {
	When("User applies EventIngester YAML using kubectl", func() {
		It("Kubernetes should create EventIngester Kubernetes resources", func() {
			By("Calling the EventIngester Controller Reconcile function", func() {
				f, err := CreateTempFile([]byte(eventYaml))
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
