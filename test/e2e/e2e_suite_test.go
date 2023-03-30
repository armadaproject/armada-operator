package e2e_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2e Suite")
}

var _ = BeforeSuite(func() {
	// TODO: Add setup code here.

	// Create a kind cluster
})
var _ = Describe("E2e", func() {

	It("Should Create CRD", func() {
	})

	It("Should Delete CRD", func() {
	})

})
var _ = AfterSuite(func() {
	// TODO: Add teardown code here.
})
