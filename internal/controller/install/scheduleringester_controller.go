/*
Copyright 2023.

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

package install

import (
	"context"
	"time"

	"github.com/armadaproject/armada-operator/internal/controller/common"

	"github.com/pkg/errors"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	installv1alpha1 "github.com/armadaproject/armada-operator/api/install/v1alpha1"
	"github.com/armadaproject/armada-operator/internal/controller/builders"
)

// SchedulerIngesterReconciler reconciles a SchedulerIngester object
type SchedulerIngesterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=install.armadaproject.io,resources=scheduleringesters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=scheduleringesters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=secrets;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *SchedulerIngesterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("namespace", req.Namespace, "name", req.Name)

	started := time.Now()

	logger.Info("Reconciling object")

	var schedulerIngester installv1alpha1.SchedulerIngester
	if miss, err := common.GetObject(ctx, r.Client, &schedulerIngester, req.NamespacedName, logger); err != nil || miss {
		return ctrl.Result{}, err
	}

	finish, err := common.CheckAndHandleObjectDeletion(ctx, r.Client, &schedulerIngester, operatorFinalizer, nil, logger)
	if err != nil || finish {
		return ctrl.Result{}, err
	}

	commonConfig, err := builders.ParseCommonApplicationConfig(schedulerIngester.Spec.ApplicationConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	components, err := r.generateSchedulerIngesterComponents(&schedulerIngester, r.Scheme, commonConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	componentsCopy := components.DeepCopy()

	mutateFn := func() error {
		components.ReconcileComponents(componentsCopy)
		return nil
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.ServiceAccount, schedulerIngester.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.Secret, schedulerIngester.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.Deployment, schedulerIngester.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled object", "durationMillis", time.Since(started).Milliseconds())

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SchedulerIngesterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&installv1alpha1.SchedulerIngester{}).
		Complete(r)
}

func (r *SchedulerIngesterReconciler) generateSchedulerIngesterComponents(
	schedulerIngester *installv1alpha1.SchedulerIngester,
	scheme *runtime.Scheme,
	config *builders.CommonApplicationConfig,
) (*CommonComponents, error) {
	secret, err := builders.CreateSecret(schedulerIngester.Spec.ApplicationConfig, schedulerIngester.Name, schedulerIngester.Namespace, GetConfigFilename(schedulerIngester.Name))
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(schedulerIngester, secret, scheme); err != nil {
		return nil, err
	}

	var serviceAccount *corev1.ServiceAccount
	serviceAccountName := schedulerIngester.Spec.CustomServiceAccount
	if serviceAccountName == "" {
		serviceAccount = builders.ServiceAccount(schedulerIngester.Name, schedulerIngester.Namespace, AllLabels(schedulerIngester.Name, schedulerIngester.Labels), schedulerIngester.Spec.ServiceAccount)
		if err = controllerutil.SetOwnerReference(schedulerIngester, serviceAccount, scheme); err != nil {
			return nil, errors.WithStack(err)
		}
		serviceAccountName = serviceAccount.Name
	}

	deployment, err := r.createDeployment(schedulerIngester, serviceAccountName, config)
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(schedulerIngester, deployment, scheme); err != nil {
		return nil, err
	}

	profilingService, profilingIngress, err := newProfilingComponents(
		schedulerIngester,
		scheme,
		config,
		schedulerIngester.Spec.ProfilingIngressConfig,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &CommonComponents{
		Deployment:       deployment,
		ServiceAccount:   serviceAccount,
		Secret:           secret,
		ServiceProfiling: profilingService,
		IngressProfiling: profilingIngress,
	}, nil
}

func (r *SchedulerIngesterReconciler) createDeployment(
	schedulerIngester *installv1alpha1.SchedulerIngester,
	serviceAccountName string,
	config *builders.CommonApplicationConfig,
) (*appsv1.Deployment, error) {
	env := createEnv(schedulerIngester.Spec.Environment)
	pulsarConfig, err := ExtractPulsarConfig(schedulerIngester.Spec.ApplicationConfig)
	if err != nil {
		return nil, err
	}
	volumes := createVolumes(schedulerIngester.Name, schedulerIngester.Spec.AdditionalVolumes)
	volumes = append(volumes, createPulsarVolumes(pulsarConfig)...)
	volumeMounts := createVolumeMounts(GetConfigFilename(schedulerIngester.Name), schedulerIngester.Spec.AdditionalVolumeMounts)
	volumeMounts = append(volumeMounts, createPulsarVolumeMounts(pulsarConfig)...)

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: schedulerIngester.Name, Namespace: schedulerIngester.Namespace, Labels: AllLabels(schedulerIngester.Name, schedulerIngester.Labels)},
		Spec: appsv1.DeploymentSpec{
			Replicas: schedulerIngester.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: IdentityLabel(schedulerIngester.Name),
			},
			Strategy: defaultDeploymentStrategy(1),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        schedulerIngester.Name,
					Namespace:   schedulerIngester.Namespace,
					Labels:      AllLabels(schedulerIngester.Name, schedulerIngester.Labels),
					Annotations: map[string]string{"checksum/config": GenerateChecksumConfig(schedulerIngester.Spec.ApplicationConfig.Raw)},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:            serviceAccountName,
					TerminationGracePeriodSeconds: schedulerIngester.Spec.TerminationGracePeriodSeconds,
					SecurityContext:               schedulerIngester.Spec.PodSecurityContext,
					Affinity:                      defaultAffinity(schedulerIngester.Name, 100),
					Containers: []corev1.Container{{
						Name:            "scheduleringester",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Image:           ImageString(schedulerIngester.Spec.Image),
						Args:            []string{appConfigFlag, appConfigFilepath},
						Ports:           newContainerPortsMetrics(config),
						Env:             env,
						VolumeMounts:    volumeMounts,
						SecurityContext: schedulerIngester.Spec.SecurityContext,
					}},
					Tolerations: schedulerIngester.Spec.Tolerations,
					Volumes:     volumes,
				},
			},
		},
	}
	if schedulerIngester.Spec.Resources != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources = *schedulerIngester.Spec.Resources
		deployment.Spec.Template.Spec.Containers[0].Env = addGoMemLimit(deployment.Spec.Template.Spec.Containers[0].Env, *schedulerIngester.Spec.Resources)
	}

	return &deployment, nil

}
