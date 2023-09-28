package integration

import (
	"io"
	"net/http"
	"os"
	"time"

	"github.com/armadaproject/armada-operator/test/util"

	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	installv1alpha1 "github.com/armadaproject/armada-operator/api/install/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Executor Controller", func() {
	When("User applies a new Executor YAML using kubectl", func() {
		It("Kubernetes should create the Executor Kubernetes resources", func() {
			By("Calling the Executor Controller Reconcile function", func() {
				f, err := os.Open("./resources/executor1.yaml")
				Expect(err).ToNot(HaveOccurred())
				defer f.Close()

				k, err := testUser.Kubectl()
				Expect(err).ToNot(HaveOccurred())
				stdin, stderr, err := k.Run("apply", "-f", f.Name())
				if err != nil {
					stderrBytes, err := io.ReadAll(stderr)
					Expect(err).ToNot(HaveOccurred())
					Fail(string(stderrBytes))
				}
				stdinBytes, err := io.ReadAll(stdin)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(stdinBytes)).To(Equal("executor.install.armadaproject.io/executor-e2e-1 created\n"))

				executor := installv1alpha1.Executor{}
				executorKey := kclient.ObjectKey{Namespace: "default", Name: "executor-e2e-1"}
				Eventually(func() error {
					return k8sClient.Get(ctx, executorKey, &executor)
				}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())

				secret := corev1.Secret{}
				secretKey := kclient.ObjectKey{Namespace: "default", Name: "executor-e2e-1"}
				Eventually(func() error {
					return k8sClient.Get(ctx, secretKey, &secret)
				}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())
				Expect(secret.Data["executor-e2e-1-config.yaml"]).NotTo(BeEmpty())

				deployment := appsv1.Deployment{}
				deploymentKey := kclient.ObjectKey{Namespace: "default", Name: "executor-e2e-1"}
				Eventually(func() error {
					return k8sClient.Get(ctx, deploymentKey, &deployment)
				}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())
				Expect(deployment.Spec.Selector.MatchLabels["app"]).To(Equal("executor-e2e-1"))

				service := corev1.Service{}
				serviceKey := kclient.ObjectKey{Namespace: "default", Name: "executor-e2e-1"}
				Eventually(func() error {
					return k8sClient.Get(ctx, serviceKey, &service)
				}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())

				clusterRole := rbacv1.ClusterRole{}
				clusterRoleKey := kclient.ObjectKey{Namespace: "", Name: "executor-e2e-1"}
				Eventually(func() error {
					return k8sClient.Get(ctx, clusterRoleKey, &clusterRole)
				}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())
				clusterRoleBinding := rbacv1.ClusterRoleBinding{}
				clusterRoleBindingKey := kclient.ObjectKey{Namespace: "", Name: "executor-e2e-1"}
				Eventually(func() error {
					return k8sClient.Get(ctx, clusterRoleBindingKey, &clusterRoleBinding)
				}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())
			})
		})
	})

	When("User applies an existing Executor YAML with updated values using kubectl", func() {
		It("Kubernetes should update the Executor Kubernetes resources", func() {
			By("Calling the Executor Controller Reconcile function", func() {
				f1, err := os.Open("./resources/executor2.yaml")
				Expect(err).ToNot(HaveOccurred())
				defer f1.Close()

				k, err := testUser.Kubectl()
				Expect(err).ToNot(HaveOccurred())
				stdin, stderr, err := k.Run("apply", "-f", f1.Name())
				if err != nil {
					stderrBytes, err := io.ReadAll(stderr)
					Expect(err).ToNot(HaveOccurred())
					Fail(string(stderrBytes))
				}
				stdinBytes, err := io.ReadAll(stdin)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(stdinBytes)).To(BeEquivalentTo("executor.install.armadaproject.io/executor-e2e-2 created\n"))

				time.Sleep(1 * time.Second)

				executor := installv1alpha1.Executor{}
				executorKey := kclient.ObjectKey{Namespace: "default", Name: "executor-e2e-2"}
				err = k8sClient.Get(ctx, executorKey, &executor)
				Expect(err).NotTo(HaveOccurred())
				Expect("test").NotTo(BeKeyOf(executor.Labels))

				deployment := appsv1.Deployment{}
				deploymentKey := kclient.ObjectKey{Namespace: "default", Name: "executor-e2e-2"}
				Eventually(func() string {
					err = k8sClient.Get(ctx, deploymentKey, &deployment)
					return deployment.Spec.Template.Spec.Containers[0].Image
				}, defaultTimeout, defaultPollInterval).Should(Equal("test-executor:0.3.33"))

				f2, err := os.Open("./resources/executor2-updated.yaml")
				Expect(err).ToNot(HaveOccurred())
				defer f2.Close()

				Expect(err).ToNot(HaveOccurred())
				stdin, stderr, err = k.Run("apply", "-f", f2.Name())
				if err != nil {
					stderrBytes, err := io.ReadAll(stderr)
					Expect(err).ToNot(HaveOccurred())
					Fail(string(stderrBytes))
				}
				stdinBytes, err = io.ReadAll(stdin)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(stdinBytes)).To(Equal("executor.install.armadaproject.io/executor-e2e-2 configured\n"))

				executor = installv1alpha1.Executor{}
				Eventually(func() error {
					return k8sClient.Get(ctx, executorKey, &executor)
				}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())
				Expect(executor.Labels["test"]).To(BeEquivalentTo("updated"))

				Eventually(func() string {
					err = k8sClient.Get(ctx, deploymentKey, &deployment)
					return deployment.Spec.Template.Spec.Containers[0].Image
				}, defaultTimeout, defaultPollInterval).Should(Equal("test-executor:0.3.34"))
			})
		})
	})

	When("User deletes an existing Executor YAML using kubectl", func() {
		It("Kubernetes should delete the Executor Kubernetes resources", func() {
			By("Calling the Executor Controller Reconcile function", func() {
				f, err := os.Open("./resources/executor3.yaml")
				Expect(err).ToNot(HaveOccurred())
				defer f.Close()

				k, err := testUser.Kubectl()
				Expect(err).ToNot(HaveOccurred())
				stdin, stderr, err := k.Run("apply", "-f", f.Name())
				if err != nil {
					stderrBytes, err := io.ReadAll(stderr)
					Expect(err).ToNot(HaveOccurred())
					Fail(string(stderrBytes))
				}
				stdinBytes, err := io.ReadAll(stdin)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(stdinBytes)).To(BeEquivalentTo("executor.install.armadaproject.io/executor-e2e-3 created\n"))

				oldExecutor := installv1alpha1.Executor{}
				executorKey := kclient.ObjectKey{Namespace: "default", Name: "executor-e2e-3"}
				Eventually(func() error {
					return k8sClient.Get(ctx, executorKey, &oldExecutor)
				}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())

				stdin, stderr, err = k.Run("delete", "-f", f.Name())
				if err != nil {
					stderrBytes, err := io.ReadAll(stderr)
					Expect(err).ToNot(HaveOccurred())
					Fail(string(stderrBytes))
				}
				stdinBytes, err = io.ReadAll(stdin)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(stdinBytes)).To(Equal("executor.install.armadaproject.io \"executor-e2e-3\" deleted\n"))

				// executor
				deletedExecutor := installv1alpha1.Executor{}
				Eventually(func() error {
					return k8sClient.Get(ctx, executorKey, &deletedExecutor)
				}, defaultTimeout, defaultPollInterval).Should(BeAssignableToTypeOf(&k8serrors.StatusError{}))
				Eventually(func() int32 {
					err = k8sClient.Get(ctx, executorKey, &deletedExecutor)
					notFoundErr := err.(*k8serrors.StatusError)
					return notFoundErr.ErrStatus.Code
				}, defaultTimeout, defaultPollInterval).Should(BeEquivalentTo(http.StatusNotFound))

				// secret
				secret := corev1.Secret{}
				secretKey := kclient.ObjectKey{Namespace: "default", Name: "executor-e2e-3"}
				Eventually(func() error {
					return k8sClient.Get(ctx, secretKey, &secret)
				}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())
				Expect(util.HasOwnerReference(&oldExecutor, &secret, runtimeScheme)).To(BeTrue())

				// deployment
				deployment := appsv1.Deployment{}
				deploymentKey := kclient.ObjectKey{Namespace: "default", Name: "executor-e2e-3"}
				Eventually(func() error {
					return k8sClient.Get(ctx, deploymentKey, &deployment)
				}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())
				Expect(util.HasOwnerReference(&oldExecutor, &deployment, runtimeScheme)).To(BeTrue())

				// service account
				serviceAccount := corev1.ServiceAccount{}
				serviceAccountKey := kclient.ObjectKey{Namespace: "default", Name: "executor-e2e-3"}
				Eventually(func() error {
					return k8sClient.Get(ctx, serviceAccountKey, &serviceAccount)
				}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())
				Expect(util.HasOwnerReference(&oldExecutor, &serviceAccount, runtimeScheme)).To(BeTrue())

				// service
				service := corev1.Service{}
				serviceKey := kclient.ObjectKey{Namespace: "default", Name: "executor-e2e-3"}
				Eventually(func() error {
					return k8sClient.Get(ctx, serviceKey, &service)
				}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())
				Expect(util.HasOwnerReference(&oldExecutor, &service, runtimeScheme)).To(BeTrue())

				// clusterrole
				clusterRole := rbacv1.ClusterRole{}
				clusterRoleKey := kclient.ObjectKey{Namespace: "", Name: "executor-e2e-3"}
				Eventually(func() error {
					return k8sClient.Get(ctx, clusterRoleKey, &clusterRole)
				}, defaultTimeout, defaultPollInterval).Should(BeAssignableToTypeOf(&k8serrors.StatusError{}))
				Eventually(func() int32 {
					err = k8sClient.Get(ctx, clusterRoleKey, &clusterRole)
					notFoundErr := err.(*k8serrors.StatusError)
					return notFoundErr.ErrStatus.Code
				}, defaultTimeout, defaultPollInterval).Should(BeEquivalentTo(http.StatusNotFound))

				// clusterrolebinding
				clusterRoleBinding := rbacv1.ClusterRoleBinding{}
				clusterRoleBindingKey := kclient.ObjectKey{Namespace: "", Name: "executor-e2e-3"}
				Eventually(func() error {
					return k8sClient.Get(ctx, clusterRoleBindingKey, &clusterRoleBinding)
				}, defaultTimeout, defaultPollInterval).Should(BeAssignableToTypeOf(&k8serrors.StatusError{}))
				Eventually(func() int32 {
					err = k8sClient.Get(ctx, clusterRoleBindingKey, &clusterRoleBinding)
					notFoundErr := err.(*k8serrors.StatusError)
					return notFoundErr.ErrStatus.Code
				}, defaultTimeout, defaultPollInterval).Should(BeEquivalentTo(http.StatusNotFound))
			})
		})
	})
})
