package integration

import (
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Armada Operator", func() {
	When("User applies Lookout YAML using kubectl", func() {
		It("Kubernetes should create Lookout Kubernetes resources", func() {
			By("Calling the Lookout Controller Reconcile function", func() {
				f, err := os.Open("./resources/lookout1.yaml")
				Expect(err).ToNot(HaveOccurred())
				defer f.Close()
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
