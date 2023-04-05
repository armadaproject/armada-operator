package e2e_test

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	testEnv       *envtest.Environment
	cfg           *rest.Config
	k8sClient     client.Client
	testUser      *envtest.AuthenticatedUser
	ctx           context.Context
	runtimeScheme *runtime.Scheme
	cancel        context.CancelFunc
	crdPath       = filepath.Join("..", "..", "config", "crd", "bases")
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2e Suite")
}

var _ = BeforeSuite(func() {

	ctx, cancel = context.WithCancel(context.TODO())

	testEnv = &envtest.Environment{
		AttachControlPlaneOutput: false,
	}

	var err error

	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	user := envtest.User{Name: "armada-operator-test-user", Groups: []string{"system:masters"}}
	testUser, err = testEnv.AddUser(user, cfg)
	Expect(err).NotTo(HaveOccurred())

	runtimeScheme = scheme.Scheme

	k8sClient, err := client.New(cfg, client.Options{Scheme: runtimeScheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "armada",
		},
	}

	Expect(k8sClient.Create(ctx, ns)).To(Succeed())
})

var _ = Describe("E2e", func() {

	It("Install Bases CRDs", func() {
		k, err := testUser.Kubectl()
		Expect(err).NotTo(HaveOccurred())

		stdout, stderr, err := k.Run("apply", "-n", "armada", "-f", crdPath)
		if err != nil {
			stderrBytes, err := io.ReadAll(stderr)
			Expect(err).ToNot(HaveOccurred())
			Fail(string(stderrBytes))
		}

		stdoutBytes, err := io.ReadAll(stdout)
		Expect(err).ToNot(HaveOccurred())
		fmt.Println(string(stdoutBytes)) // just for debuging
	})

	It("Deploy Armada Components", func() {
		k, err := testUser.Kubectl()
		Expect(err).NotTo(HaveOccurred())

		stdout, stderr, err := k.Run("apply", "-n", "armada", "-f", filepath.Join("..", "..", "config", "samples", "deploy_armada.yaml"))

		if err != nil {
			stderrBytes, err := io.ReadAll(stderr)
			Expect(err).ToNot(HaveOccurred())
			Fail(string(stderrBytes))
		}

		stdoutBytes, err := io.ReadAll(stdout)
		Expect(err).ToNot(HaveOccurred())
		fmt.Println(string(stdoutBytes)) // just for debuging
	})

	It("Validate CRDs Exists", func() {

		k, err := testUser.Kubectl()
		Expect(err).NotTo(HaveOccurred())

		stdout, stderr, err := k.Run("get", "customresourcedefinitions.apiextensions.k8s.io")

		if err != nil {
			stderrBytes, err := io.ReadAll(stderr)
			Expect(err).ToNot(HaveOccurred())
			Fail(string(stderrBytes))
		}

		stdoutBytes, err := io.ReadAll(stdout)
		Expect(err).ToNot(HaveOccurred())

		crds := []string{
			"armadaservers.install.armadaproject.io",
			"binoculars.install.armadaproject.io",
			"eventingesters.install.armadaproject.io",
			"executors.install.armadaproject.io",
			"lookoutingesters.install.armadaproject.io",
			"lookouts.install.armadaproject.io",
			"queues.core.armadaproject.io",
		}
		for _, crd := range crds {
			fmt.Println(crd) // just for debuging
			Expect(string(stdoutBytes)).To(ContainSubstring(crd))
		}
	})

	It("Validate CRDs Can be Deleted", func() {
		k, err := testUser.Kubectl()
		Expect(err).NotTo(HaveOccurred())

		stdout, stderr, err := k.Run("delete", "-n", "armada", "-f", crdPath)
		if err != nil {
			stderrBytes, err := io.ReadAll(stderr)
			Expect(err).ToNot(HaveOccurred())
			Fail(string(stderrBytes))
		}

		stdoutBytes, err := io.ReadAll(stdout)
		Expect(err).ToNot(HaveOccurred())
		fmt.Println(string(stdoutBytes)) // just for debuging
	})

})
var _ = AfterSuite(func() {
	// TODO: Add teardown code here.
	By("tearing down test environment")

	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())

})
