package integration

import (
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	installv1alpha1 "github.com/armadaproject/armada-operator/api/install/v1alpha1"
)

var _ = Describe("Armada Operator", func() {
	When("User applies a new ArmadaServer YAML using kubectl", func() {
		It("Kubernetes should create ArmadaServer Kubernetes resources", func() {
			By("Calling the ArmadaServer Controller Reconcile function", func() {
				f, err := os.Open("./resources/server1-pulsar-jobs.yaml")
				Expect(err).ToNot(HaveOccurred())
				defer f.Close()
				Expect(err).ToNot(HaveOccurred())
				defer f.Close()

				k, err := testUser.Kubectl()
				Expect(err).ToNot(HaveOccurred())
				stdout, stderr, err := k.Run("create", "-f", f.Name())
				if err != nil {
					stderrBytes, err := io.ReadAll(stderr)
					Expect(err).ToNot(HaveOccurred())
					Fail(string(stderrBytes))
				}
				stdoutBytes, err := io.ReadAll(stdout)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(stdoutBytes)).To(Equal("armadaserver.install.armadaproject.io/armadaserver-e2e-pulsar created\n"))

				as := installv1alpha1.ArmadaServer{}
				asKey := kclient.ObjectKey{Namespace: "default", Name: "armadaserver-e2e-pulsar"}
				Eventually(func() error {
					return k8sClient.Get(ctx, asKey, &as)
				}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())

				secret := corev1.Secret{}
				secretKey := kclient.ObjectKey{Namespace: "default", Name: "armadaserver-e2e-pulsar"}
				Eventually(func() error {
					return k8sClient.Get(ctx, secretKey, &secret)
				}, "15s", "50ms").ShouldNot(HaveOccurred())
				Expect(secret.Data["armadaserver-e2e-pulsar-config.yaml"]).NotTo(BeEmpty())

				for _, jobName := range []string{"wait-for-pulsar", "init-pulsar"} {
					// set migrate Job to complete -- there is no JobController in this environment,
					// so we are mocking the Job's completion
					job := batchv1.Job{}
					jobKey := kclient.ObjectKey{Namespace: "default", Name: jobName}
					Eventually(func() error {
						return k8sClient.Get(ctx, jobKey, &job)
					}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())

					job.Status = batchv1.JobStatus{
						Conditions: []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}},
					}
					err = k8sClient.Status().Update(ctx, &job)
					Expect(err).NotTo(HaveOccurred())

					err = k8sClient.Get(ctx, jobKey, &job)
					Expect(err).NotTo(HaveOccurred())
					Expect(job.Status.Conditions[0].Type).To(Equal(batchv1.JobComplete))
				}

				deployment := appsv1.Deployment{}
				deploymentKey := kclient.ObjectKey{Namespace: "default", Name: "armadaserver-e2e-pulsar"}
				Eventually(func() error {
					return k8sClient.Get(ctx, deploymentKey, &deployment)
				}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())
				Expect(deployment.Spec.Selector.MatchLabels["app"]).To(Equal("armadaserver-e2e-pulsar"))

				service := corev1.Service{}
				serviceKey := kclient.ObjectKey{Namespace: "default", Name: "armadaserver-e2e-pulsar"}
				Eventually(func() error {
					return k8sClient.Get(ctx, serviceKey, &service)
				}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())
			})
		})
	})

	It("Kubernetes should create ArmadaServer Kubernetes resources", func() {
		By("Calling the ArmadaServer Controller Reconcile function", func() {
			f, err := os.Open("./resources/server1-no-pulsar-jobs.yaml")
			Expect(err).ToNot(HaveOccurred())
			defer f.Close()
			Expect(err).ToNot(HaveOccurred())
			defer f.Close()

			k, err := testUser.Kubectl()
			Expect(err).ToNot(HaveOccurred())
			stdout, stderr, err := k.Run("create", "-f", f.Name())
			if err != nil {
				stderrBytes, err := io.ReadAll(stderr)
				Expect(err).ToNot(HaveOccurred())
				Fail(string(stderrBytes))
			}
			stdoutBytes, err := io.ReadAll(stdout)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(stdoutBytes)).To(Equal("armadaserver.install.armadaproject.io/armadaserver-e2e-no-pulsar created\n"))

			as := installv1alpha1.ArmadaServer{}
			asKey := kclient.ObjectKey{Namespace: "default", Name: "armadaserver-e2e-no-pulsar"}
			Eventually(func() error {
				return k8sClient.Get(ctx, asKey, &as)
			}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())

			secret := corev1.Secret{}
			secretKey := kclient.ObjectKey{Namespace: "default", Name: "armadaserver-e2e-no-pulsar"}
			Eventually(func() error {
				return k8sClient.Get(ctx, secretKey, &secret)
			}, "15s", "50ms").ShouldNot(HaveOccurred())
			Expect(secret.Data["armadaserver-e2e-no-pulsar-config.yaml"]).NotTo(BeEmpty())

			deployment := appsv1.Deployment{}
			deploymentKey := kclient.ObjectKey{Namespace: "default", Name: "armadaserver-e2e-no-pulsar"}
			Eventually(func() error {
				return k8sClient.Get(ctx, deploymentKey, &deployment)
			}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())
			Expect(deployment.Spec.Selector.MatchLabels["app"]).To(Equal("armadaserver-e2e-no-pulsar"))

			service := corev1.Service{}
			serviceKey := kclient.ObjectKey{Namespace: "default", Name: "armadaserver-e2e-no-pulsar"}
			Eventually(func() error {
				return k8sClient.Get(ctx, serviceKey, &service)
			}, defaultTimeout, defaultPollInterval).ShouldNot(HaveOccurred())
		})
	})
})
