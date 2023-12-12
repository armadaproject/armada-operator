/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package integration

import (
	"context"
	"crypto/tls"
	"os"
	"path/filepath"
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/armadaproject/armada-operator/internal/controller/install"

	ctrl "sigs.k8s.io/controller-runtime"

	installv1alpha1 "github.com/armadaproject/armada-operator/api/install/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg           *rest.Config
	k8sClient     client.Client // You'll be using this client in your tests.
	testEnv       *envtest.Environment
	testUser      *envtest.AuthenticatedUser
	ctx           context.Context
	cancel        context.CancelFunc
	runtimeScheme *runtime.Scheme
)

const (
	defaultTimeout      = "50s"
	defaultPollInterval = "50ms"
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	kubebuilderAssets := os.Getenv("KUBEBUILDER_ASSETS")
	logf.Log.Info("Kubebuilder assets path", "path", kubebuilderAssets)

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "..", "config", "webhook", "manifests.yaml")},
		},
		ErrorIfCRDPathMissing:    true,
		AttachControlPlaneOutput: true,
		CRDInstallOptions: envtest.CRDInstallOptions{
			CleanUpAfterUse: true,
		},
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	user := envtest.User{Name: "armada-operator-test-user", Groups: []string{"system:masters"}}
	testUser, err = testEnv.AddUser(user, cfg)
	Expect(err).NotTo(HaveOccurred())

	runtimeScheme = scheme.Scheme

	Expect(installv1alpha1.AddToScheme(runtimeScheme)).NotTo(HaveOccurred())
	Expect(monitoringv1.AddToScheme(runtimeScheme)).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme
	k8sClient, err = client.New(cfg, client.Options{Scheme: runtimeScheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	webhookOpts := webhook.Options{
		CertDir: testEnv.WebhookInstallOptions.LocalServingCertDir,
		Host:    testEnv.WebhookInstallOptions.LocalServingHost,
		Port:    testEnv.WebhookInstallOptions.LocalServingPort,
		TLSOpts: []func(*tls.Config){func(config *tls.Config) {}},
	}
	mgrOpts := ctrl.Options{
		Scheme:        runtimeScheme,
		WebhookServer: webhook.NewServer(webhookOpts),
	}
	k8sManager, err := ctrl.NewManager(cfg, mgrOpts)
	Expect(err).ToNot(HaveOccurred())

	err = (&install.BinocularsReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&install.EventIngesterReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&install.ExecutorReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&install.LookoutReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&install.ArmadaServerReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&install.LookoutIngesterReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// Webhooks
	Expect((&installv1alpha1.ArmadaServer{}).SetupWebhookWithManager(k8sManager)).ToNot(HaveOccurred())
	Expect((&installv1alpha1.Executor{}).SetupWebhookWithManager(k8sManager)).ToNot(HaveOccurred())
	Expect((&installv1alpha1.EventIngester{}).SetupWebhookWithManager(k8sManager)).ToNot(HaveOccurred())
	Expect((&installv1alpha1.Binoculars{}).SetupWebhookWithManager(k8sManager)).ToNot(HaveOccurred())
	Expect((&installv1alpha1.Lookout{}).SetupWebhookWithManager(k8sManager)).ToNot(HaveOccurred())
	Expect((&installv1alpha1.LookoutIngester{}).SetupWebhookWithManager(k8sManager)).ToNot(HaveOccurred())
	Expect((&installv1alpha1.SchedulerIngester{}).SetupWebhookWithManager(k8sManager)).ToNot(HaveOccurred())
	Expect((&installv1alpha1.Scheduler{}).SetupWebhookWithManager(k8sManager)).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		Expect(k8sManager.Start(ctx)).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
