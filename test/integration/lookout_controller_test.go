package integration

import (
	"io"
	"os"
	"time"

	installv1alpha1 "github.com/armadaproject/armada-operator/apis/install/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Armada Operator", func() {
	When("User applies Lookout YAML using kubectl", func() {
		It("Kubernetes should create Lookout Kubernetes resources", func() {
			By("Calling the Lookout Controller Reconcile function", func() {
				f, err := os.Open("./resources/lookout1.yaml")
				Expect(err).ToNot(HaveOccurred())
				defer f.Close()

				k, err := testUser.Kubectl()
				Expect(err).ToNot(HaveOccurred())
				stdin, stderr, err := k.Run("create", "-f", f.Name())
				if err != nil {
					stderrBytes, err := io.ReadAll(stderr)
					Expect(err).ToNot(HaveOccurred())
					Fail(string(stderrBytes))
				}
				stdinBytes, err := io.ReadAll(stdin)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(stdinBytes)).To(Equal("lookout.install.armadaproject.io/lookout-e2e-1 created\n"))

				time.Sleep(2 * time.Second)

				lookout := installv1alpha1.Lookout{}
				lookoutKey := kclient.ObjectKey{Namespace: "default", Name: "lookout-e2e-1"}
				err = k8sClient.Get(ctx, lookoutKey, &lookout)
				Expect(err).NotTo(HaveOccurred())

				secret := corev1.Secret{}
				secretKey := kclient.ObjectKey{Namespace: "default", Name: "lookout-e2e-1"}
				err = k8sClient.Get(ctx, secretKey, &secret)
				Expect(err).NotTo(HaveOccurred())
				Expect(secret.Data["lookout-e2e-1-config.yaml"]).NotTo(BeEmpty())

				// set migrate Job to complete -- there is no JobController in this environment,
				// so we are mocking the Job's completion
				job := batchv1.Job{}
				jobKey := kclient.ObjectKey{Namespace: "default", Name: "lookout-e2e-1-migration"}
				err = k8sClient.Get(ctx, jobKey, &job)
				Expect(err).NotTo(HaveOccurred())

				job.Status = batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{{
						Type:   batchv1.JobComplete,
						Status: corev1.ConditionTrue,
					}},
				}
				err = k8sClient.Status().Update(ctx, &job)
				Expect(err).NotTo(HaveOccurred())

				jobKey = kclient.ObjectKey{Namespace: "default", Name: "lookout-e2e-1-migration"}
				err = k8sClient.Get(ctx, jobKey, &job)
				Expect(err).NotTo(HaveOccurred())
				Expect(job.Status.Conditions[0].Type).To(Equal(batchv1.JobComplete))

				time.Sleep(4 * time.Second)

				deployment := appsv1.Deployment{}
				deploymentKey := kclient.ObjectKey{Namespace: "default", Name: "lookout-e2e-1"}
				err = k8sClient.Get(ctx, deploymentKey, &deployment)
				Expect(err).NotTo(HaveOccurred())
				Expect(deployment.Spec.Selector.MatchLabels["app"]).To(Equal("lookout-e2e-1"))

				service := corev1.Service{}
				serviceKey := kclient.ObjectKey{Namespace: "default", Name: "lookout-e2e-1"}
				err = k8sClient.Get(ctx, serviceKey, &service)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	When("User applies an existing Lookout YAML with updated values using kubectl", func() {
		It("Kubernetes should update the Lookout Kubernetes resources", func() {
			By("Calling the Lookout Controller Reconcile function", func() {
				f1, err := os.Open("./resources/lookout2.yaml")
				Expect(err).ToNot(HaveOccurred())
				defer f1.Close()

				k, err := testUser.Kubectl()
				Expect(err).ToNot(HaveOccurred())
				stdin, stderr, err := k.Run("create", "-f", f1.Name())
				if err != nil {
					stderrBytes, err := io.ReadAll(stderr)
					Expect(err).ToNot(HaveOccurred())
					Fail(string(stderrBytes))
				}
				stdinBytes, err := io.ReadAll(stdin)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(stdinBytes)).To(BeEquivalentTo("lookout.install.armadaproject.io/lookout-e2e-2 created\n"))

				lookout := installv1alpha1.Lookout{}
				lookoutKey := kclient.ObjectKey{Namespace: "default", Name: "lookout-e2e-2"}
				err = k8sClient.Get(ctx, lookoutKey, &lookout)
				Expect(err).NotTo(HaveOccurred())
				Expect("test").NotTo(BeKeyOf(lookout.Labels))

				f2, err := os.Open("./resources/lookout2-updated.yaml")
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
				Expect(string(stdinBytes)).To(Equal("lookout.install.armadaproject.io/lookout-e2e-2 configured\n"))

				time.Sleep(2 * time.Second)

				lookout = installv1alpha1.Lookout{}
				err = k8sClient.Get(ctx, lookoutKey, &lookout)
				Expect(err).NotTo(HaveOccurred())
				Expect(lookout.Labels["test"]).To(BeEquivalentTo("updated"))
			})
		})
	})
})
