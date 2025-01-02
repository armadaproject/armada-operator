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

// LookoutIngesterReconciler reconciles a LookoutIngester object
type LookoutIngesterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=install.armadaproject.io,resources=lookoutingesters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=lookoutingesters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=secrets;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *LookoutIngesterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("namespace", req.Namespace, "name", req.Name)

	started := time.Now()

	logger.Info("Reconciling object")

	var lookoutIngester installv1alpha1.LookoutIngester
	if miss, err := common.GetObject(ctx, r.Client, &lookoutIngester, req.NamespacedName, logger); err != nil || miss {
		return ctrl.Result{}, err
	}

	finish, err := common.CheckAndHandleObjectDeletion(ctx, r.Client, &lookoutIngester, operatorFinalizer, nil, logger)
	if err != nil || finish {
		return ctrl.Result{}, err
	}

	commonConfig, err := builders.ParseCommonApplicationConfig(lookoutIngester.Spec.ApplicationConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	components, err := r.generateInstallComponents(&lookoutIngester, r.Scheme, commonConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	componentsCopy := components.DeepCopy()

	mutateFn := func() error {
		components.ReconcileComponents(componentsCopy)
		return nil
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.Secret, lookoutIngester.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.ServiceAccount, lookoutIngester.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.Deployment, lookoutIngester.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled object", "durationMillis", time.Since(started).Milliseconds())

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LookoutIngesterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&installv1alpha1.LookoutIngester{}).
		Complete(r)
}

func (r *LookoutIngesterReconciler) generateInstallComponents(
	lookoutIngester *installv1alpha1.LookoutIngester,
	scheme *runtime.Scheme,
	config *builders.CommonApplicationConfig,
) (*CommonComponents, error) {
	secret, err := builders.CreateSecret(lookoutIngester.Spec.ApplicationConfig, lookoutIngester.Name, lookoutIngester.Namespace, GetConfigFilename(lookoutIngester.Name))
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(lookoutIngester, secret, r.Scheme); err != nil {
		return nil, err
	}

	var serviceAccount *corev1.ServiceAccount
	serviceAccountName := lookoutIngester.Spec.CustomServiceAccount
	if serviceAccountName == "" {
		serviceAccount = builders.ServiceAccount(lookoutIngester.Name, lookoutIngester.Namespace, AllLabels(lookoutIngester.Name, lookoutIngester.Labels), lookoutIngester.Spec.ServiceAccount)
		if err = controllerutil.SetOwnerReference(lookoutIngester, serviceAccount, scheme); err != nil {
			return nil, errors.WithStack(err)
		}
		serviceAccountName = serviceAccount.Name
	}

	deployment, err := r.createDeployment(lookoutIngester, serviceAccountName, config)
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(lookoutIngester, deployment, r.Scheme); err != nil {
		return nil, err
	}

	profilingService, profilingIngress, err := newProfilingComponents(
		lookoutIngester,
		scheme,
		config,
		lookoutIngester.Spec.ProfilingIngressConfig,
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

// TODO: Flesh this out for lookoutingester
func (r *LookoutIngesterReconciler) createDeployment(
	lookoutIngester *installv1alpha1.LookoutIngester,
	serviceAccountName string,
	config *builders.CommonApplicationConfig,
) (*appsv1.Deployment, error) {
	var replicas int32 = 1

	env := createEnv(lookoutIngester.Spec.Environment)
	pulsarConfig, err := ExtractPulsarConfig(lookoutIngester.Spec.ApplicationConfig)
	if err != nil {
		return nil, err
	}
	volumes := createVolumes(lookoutIngester.Name, lookoutIngester.Spec.AdditionalVolumes)
	volumes = append(volumes, createPulsarVolumes(pulsarConfig)...)
	volumeMounts := createVolumeMounts(GetConfigFilename(lookoutIngester.Name), lookoutIngester.Spec.AdditionalVolumeMounts)
	volumeMounts = append(volumeMounts, createPulsarVolumeMounts(pulsarConfig)...)

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: lookoutIngester.Name, Namespace: lookoutIngester.Namespace, Labels: AllLabels(lookoutIngester.Name, lookoutIngester.Labels)},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: IdentityLabel(lookoutIngester.Name),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        lookoutIngester.Name,
					Namespace:   lookoutIngester.Namespace,
					Labels:      AllLabels(lookoutIngester.Name, lookoutIngester.Labels),
					Annotations: map[string]string{"checksum/config": GenerateChecksumConfig(lookoutIngester.Spec.ApplicationConfig.Raw)},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:            serviceAccountName,
					TerminationGracePeriodSeconds: lookoutIngester.Spec.TerminationGracePeriodSeconds,
					SecurityContext:               lookoutIngester.Spec.PodSecurityContext,
					Containers: []corev1.Container{{
						Name:            "lookoutingester",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Image:           ImageString(lookoutIngester.Spec.Image),
						Args:            []string{appConfigFlag, appConfigFilepath},
						Ports:           newContainerPortsMetrics(config),
						Env:             env,
						VolumeMounts:    volumeMounts,
						SecurityContext: lookoutIngester.Spec.SecurityContext,
					}},
					Tolerations: lookoutIngester.Spec.Tolerations,
					Volumes:     volumes,
				},
			},
		},
	}
	if lookoutIngester.Spec.Resources != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources = *lookoutIngester.Spec.Resources
		deployment.Spec.Template.Spec.Containers[0].Env = addGoMemLimit(deployment.Spec.Template.Spec.Containers[0].Env, *lookoutIngester.Spec.Resources)
	}

	return &deployment, nil
}
