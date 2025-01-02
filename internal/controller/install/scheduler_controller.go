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
	"fmt"
	"time"

	"github.com/armadaproject/armada-operator/internal/controller/common"

	"k8s.io/utils/ptr"

	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/pkg/errors"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	installv1alpha1 "github.com/armadaproject/armada-operator/api/install/v1alpha1"
	"github.com/armadaproject/armada-operator/internal/controller/builders"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"
)

// SchedulerReconciler reconciles a Scheduler object
type SchedulerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=install.armadaproject.io,resources=schedulers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=schedulers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=install.armadaproject.io,resources=schedulers/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=groups;users,verbs=impersonate
//+kubebuilder:rbac:groups=core,resources=secrets;services;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="batch",resources=jobs;cronjobs,verbs=get;list;watch;create;delete;deletecollection;patch;update
//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules;servicemonitors,verbs=get;list;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings;users,verbs=get;list;watch;create;update;patch;delete;impersonate

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *SchedulerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("namespace", req.Namespace, "name", req.Name)
	started := time.Now()
	logger.Info("Reconciling object")

	var scheduler installv1alpha1.Scheduler
	if miss, err := common.GetObject(ctx, r.Client, &scheduler, req.NamespacedName, logger); err != nil || miss {
		return ctrl.Result{}, err
	}

	finish, err := common.CheckAndHandleObjectDeletion(ctx, r.Client, &scheduler, operatorFinalizer, nil, logger)
	if err != nil || finish {
		return ctrl.Result{}, err
	}

	commonConfig, err := builders.ParseCommonApplicationConfig(scheduler.Spec.ApplicationConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	var components *CommonComponents
	components, err = generateSchedulerInstallComponents(&scheduler, r.Scheme, commonConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	componentsCopy := components.DeepCopy()

	mutateFn := func() error {
		components.ReconcileComponents(componentsCopy)
		return nil
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.ServiceAccount, scheduler.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.Secret, scheduler.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if len(components.Jobs) > 0 {
		if err := upsertObjectIfNeeded(ctx, r.Client, components.Jobs[0], scheduler.Kind, mutateFn, logger); err != nil {
			return ctrl.Result{}, err
		}

		if err := waitForJob(ctx, r.Client, components.Jobs[0], jobPollInterval, jobTimeout); err != nil {
			return ctrl.Result{}, err
		}
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.Deployment, scheduler.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.Service, scheduler.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.IngressGrpc, scheduler.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.ServiceMonitor, scheduler.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if err := upsertObjectIfNeeded(ctx, r.Client, components.PrometheusRule, scheduler.Kind, mutateFn, logger); err != nil {
		return ctrl.Result{}, err
	}

	if scheduler.Spec.Pruner != nil && scheduler.Spec.Pruner.Enabled {
		if err := upsertObjectIfNeeded(ctx, r.Client, components.CronJob, scheduler.Kind, mutateFn, logger); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if err := deleteObjectIfNeeded(ctx, r.Client, components.CronJob, scheduler.Kind, logger); err != nil {
			return ctrl.Result{}, err
		}
	}

	logger.Info("Successfully reconciled Scheduler object", "durationMillis", time.Since(started).Milliseconds())

	return ctrl.Result{}, nil
}

type SchedulerConfig struct {
	Postgres PostgresConfig
}

func generateSchedulerInstallComponents(
	scheduler *installv1alpha1.Scheduler,
	scheme *runtime.Scheme,
	config *builders.CommonApplicationConfig,
) (*CommonComponents, error) {
	secret, err := builders.CreateSecret(scheduler.Spec.ApplicationConfig, scheduler.Name, scheduler.Namespace, GetConfigFilename(scheduler.Name))
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(scheduler, secret, scheme); err != nil {
		return nil, err
	}

	var serviceAccount *corev1.ServiceAccount
	serviceAccountName := scheduler.Spec.CustomServiceAccount
	if serviceAccountName == "" {
		serviceAccount = builders.ServiceAccount(scheduler.Name, scheduler.Namespace, AllLabels(scheduler.Name, scheduler.Labels), scheduler.Spec.ServiceAccount)
		if err = controllerutil.SetOwnerReference(scheduler, serviceAccount, scheme); err != nil {
			return nil, errors.WithStack(err)
		}
		serviceAccountName = serviceAccount.Name
	}

	deployment, err := newSchedulerDeployment(scheduler, serviceAccountName, config)
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(scheduler, deployment, scheme); err != nil {
		return nil, err
	}

	service := builders.Service(
		scheduler.Name,
		scheduler.Namespace,
		AllLabels(scheduler.Name, scheduler.Labels),
		IdentityLabel(scheduler.Name),
		config,
		builders.ServiceEnableApplicationPortsOnly,
	)
	if err := controllerutil.SetOwnerReference(scheduler, service, scheme); err != nil {
		return nil, err
	}

	profilingService, profilingIngress, err := newProfilingComponents(
		scheduler,
		scheme,
		config,
		scheduler.Spec.ProfilingIngressConfig,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var serviceMonitor *monitoringv1.ServiceMonitor
	var prometheusRule *monitoringv1.PrometheusRule
	if scheduler.Spec.Prometheus != nil && scheduler.Spec.Prometheus.Enabled {
		serviceMonitor = newSchedulerServiceMonitor(scheduler)
		if err := controllerutil.SetOwnerReference(scheduler, serviceMonitor, scheme); err != nil {
			return nil, err
		}
		prometheusRule = newSchedulerPrometheusRule(scheduler)
		if err := controllerutil.SetOwnerReference(scheduler, prometheusRule, scheme); err != nil {
			return nil, err
		}
	}

	job, err := newSchedulerMigrationJob(scheduler, serviceAccountName)
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(scheduler, job, scheme); err != nil {
		return nil, err
	}

	cronJob, err := newSchedulerCronJob(scheduler, serviceAccountName)
	if err != nil {
		return nil, err
	}
	if err := controllerutil.SetOwnerReference(scheduler, cronJob, scheme); err != nil {
		return nil, err
	}

	ingressGRPC, err := newSchedulerIngressGRPC(scheduler, config)
	if err != nil {
		return nil, err
	}
	if ingressGRPC != nil {
		if err := controllerutil.SetOwnerReference(scheduler, ingressGRPC, scheme); err != nil {
			return nil, err
		}
	}

	return &CommonComponents{
		Deployment:       deployment,
		Service:          service,
		ServiceProfiling: profilingService,
		ServiceAccount:   serviceAccount,
		Secret:           secret,
		IngressGrpc:      ingressGRPC,
		IngressProfiling: profilingIngress,
		Jobs:             []*batchv1.Job{job},
		ServiceMonitor:   serviceMonitor,
		PrometheusRule:   prometheusRule,
		CronJob:          cronJob,
	}, nil
}

// newSchedulerServiceMonitor will return a ServiceMonitor for this
func newSchedulerServiceMonitor(scheduler *installv1alpha1.Scheduler) *monitoringv1.ServiceMonitor {
	return &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			Kind: "ServiceMonitor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      scheduler.Name,
			Namespace: scheduler.Namespace,
			Labels:    AllLabels(scheduler.Name, scheduler.Spec.Labels, scheduler.Spec.Prometheus.Labels),
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{
				{Port: "metrics", Interval: "15s"},
			},
		},
	}
}

// Function to build the deployment object for Scheduler.
// This should be changing from CRD to CRD.  Not sure if generalize this helps much
func newSchedulerDeployment(
	scheduler *installv1alpha1.Scheduler,
	serviceAccountName string,
	config *builders.CommonApplicationConfig,
) (*appsv1.Deployment, error) {
	env := createEnv(scheduler.Spec.Environment)
	pulsarConfig, err := ExtractPulsarConfig(scheduler.Spec.ApplicationConfig)
	if err != nil {
		return nil, err
	}
	volumes := createVolumes(scheduler.Name, scheduler.Spec.AdditionalVolumes)
	volumes = append(volumes, createPulsarVolumes(pulsarConfig)...)
	volumeMounts := createVolumeMounts(GetConfigFilename(scheduler.Name), scheduler.Spec.AdditionalVolumeMounts)
	volumeMounts = append(volumeMounts, createPulsarVolumeMounts(pulsarConfig)...)
	readinessProbe, livenessProbe := CreateProbesWithScheme(GetServerScheme(config.GRPC.TLS))

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: scheduler.Name, Namespace: scheduler.Namespace, Labels: AllLabels(scheduler.Name, scheduler.Labels)},
		Spec: appsv1.DeploymentSpec{
			Replicas: scheduler.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: IdentityLabel(scheduler.Name),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        scheduler.Name,
					Namespace:   scheduler.Namespace,
					Labels:      AllLabels(scheduler.Name, scheduler.Labels),
					Annotations: map[string]string{"checksum/config": GenerateChecksumConfig(scheduler.Spec.ApplicationConfig.Raw)},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:            serviceAccountName,
					TerminationGracePeriodSeconds: scheduler.DeletionGracePeriodSeconds,
					SecurityContext:               scheduler.Spec.PodSecurityContext,
					Affinity:                      defaultAffinity(scheduler.Name, 100),
					Containers: []corev1.Container{{
						Name:            "scheduler",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Image:           ImageString(scheduler.Spec.Image),
						Args:            []string{"run", appConfigFlag, appConfigFilepath},
						Ports:           newContainerPortsAll(config),
						Env:             env,
						VolumeMounts:    volumeMounts,
						SecurityContext: scheduler.Spec.SecurityContext,
						ReadinessProbe:  readinessProbe,
						LivenessProbe:   livenessProbe,
					}},
					Volumes: volumes,
				},
			},
		},
	}

	if scheduler.Spec.Resources != nil {
		deployment.Spec.Template.Spec.Containers[0].Resources = *scheduler.Spec.Resources
		deployment.Spec.Template.Spec.Containers[0].Env = addGoMemLimit(deployment.Spec.Template.Spec.Containers[0].Env, *scheduler.Spec.Resources)
	}

	return &deployment, nil
}

func newSchedulerIngressGRPC(scheduler *installv1alpha1.Scheduler, config *builders.CommonApplicationConfig) (*networking.Ingress, error) {
	if len(scheduler.Spec.HostNames) == 0 {
		// if no hostnames, no ingress can be configured
		return nil, nil
	}

	name := scheduler.Name + "-grpc"
	labels := AllLabels(scheduler.Name, scheduler.Spec.Labels, scheduler.Spec.Ingress.Labels)
	baseAnnotations := map[string]string{
		"nginx.ingress.kubernetes.io/ssl-redirect": "true",
	}
	annotations := buildIngressAnnotations(scheduler.Spec.Ingress, baseAnnotations, BackendProtocolGRPC, config.GRPC.Enabled)

	secretName := scheduler.Name + "-service-tls"
	serviceName := scheduler.Name
	servicePort := config.GRPCPort
	path := "/"
	ingress, err := builders.Ingress(name, scheduler.Namespace, labels, annotations, scheduler.Spec.HostNames, serviceName, secretName, path, servicePort)
	return ingress, errors.WithStack(err)
}

// newSchedulerMigrationJob returns a batch Job or an error if the app config is not correct
func newSchedulerMigrationJob(scheduler *installv1alpha1.Scheduler, serviceAccountName string) (*batchv1.Job, error) {
	var terminationGracePeriodSeconds int64
	if scheduler.Spec.TerminationGracePeriodSeconds != nil {
		terminationGracePeriodSeconds = *scheduler.Spec.TerminationGracePeriodSeconds
	}
	parallelism := int32(1)
	completions := int32(1)
	backoffLimit := int32(0)
	env := scheduler.Spec.Environment
	volumes := createVolumes(scheduler.Name, scheduler.Spec.AdditionalVolumes)
	volumeMounts := createVolumeMounts(GetConfigFilename(scheduler.Name), scheduler.Spec.AdditionalVolumeMounts)

	schedulerConfig, err := extractSchedulerConfig(scheduler.Spec.ApplicationConfig)
	if err != nil {
		return nil, err
	}

	job := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        scheduler.Name + "-migration",
			Namespace:   scheduler.Namespace,
			Labels:      AllLabels(scheduler.Name, scheduler.Labels),
			Annotations: map[string]string{"checksum/config": GenerateChecksumConfig(scheduler.Spec.ApplicationConfig.Raw)},
		},
		Spec: batchv1.JobSpec{
			Parallelism:  &parallelism,
			Completions:  &completions,
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      scheduler.Name + "-migration",
					Namespace: scheduler.Namespace,
					Labels:    AllLabels(scheduler.Name, scheduler.Labels),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:            serviceAccountName,
					RestartPolicy:                 "Never",
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					SecurityContext:               scheduler.Spec.PodSecurityContext,
					InitContainers: []corev1.Container{{
						Name:  "scheduler-migration-db-wait",
						Image: "postgres:15.2-alpine",
						Command: []string{
							"/bin/sh",
							"-c",
							`echo "Waiting for Postgres..."
                                                         while ! nc -z $PGHOST $PGPORT; do
                                                           sleep 1
                                                         done
                                                         echo "Postgres started!"
							 echo "Creating DB $PGDB if needed..."
							 psql -v ON_ERROR_STOP=1 --username "$PGUSER" -c "CREATE DATABASE $PGDB"
							 psql -v ON_ERROR_STOP=1 --username "$PGUSER" -c "GRANT ALL PRIVILEGES ON DATABASE $PGDB TO $PGUSER"
							 echo "DB $PGDB created"`,
						},
						Env: []corev1.EnvVar{
							{
								Name:  "PGHOST",
								Value: schedulerConfig.Postgres.Connection.Host,
							},
							{
								Name:  "PGPORT",
								Value: schedulerConfig.Postgres.Connection.Port,
							},
							{
								Name:  "PGUSER",
								Value: schedulerConfig.Postgres.Connection.User,
							},
							{
								Name:  "PGPASSWORD",
								Value: schedulerConfig.Postgres.Connection.Password,
							},
							{
								Name:  "PGDB",
								Value: schedulerConfig.Postgres.Connection.Dbname,
							},
						},
					}},
					Containers: []corev1.Container{{
						Name:            "scheduler-migration",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Image:           ImageString(scheduler.Spec.Image),
						Args: []string{
							"migrateDatabase",
							appConfigFlag,
							appConfigFilepath,
						},
						Env:             env,
						VolumeMounts:    volumeMounts,
						SecurityContext: scheduler.Spec.SecurityContext,
					}},
					Tolerations: scheduler.Spec.Tolerations,
					Volumes:     volumes,
				},
			},
		},
	}

	return &job, nil
}

// newSchedulerCronJob returns a batch CronJob or an error if the app config is not correct
func newSchedulerCronJob(scheduler *installv1alpha1.Scheduler, serviceAccountName string) (*batchv1.CronJob, error) {
	terminationGracePeriodSeconds := int64(0)
	if scheduler.Spec.TerminationGracePeriodSeconds != nil {
		terminationGracePeriodSeconds = *scheduler.Spec.TerminationGracePeriodSeconds
	}
	parallelism := int32(1)
	completions := int32(1)
	backoffLimit := int32(0)
	env := scheduler.Spec.Environment
	volumes := createVolumes(scheduler.Name, scheduler.Spec.AdditionalVolumes)
	volumeMounts := createVolumeMounts(GetConfigFilename(scheduler.Name), scheduler.Spec.AdditionalVolumeMounts)

	appConfig, err := builders.ConvertRawExtensionToYaml(scheduler.Spec.ApplicationConfig)
	if err != nil {
		return nil, err
	}
	var schedulerConfig SchedulerConfig
	err = yaml.Unmarshal([]byte(appConfig), &schedulerConfig)
	if err != nil {
		return nil, err
	}

	prunerArgs := []string{
		"pruneDatabase",
		appConfigFlag,
		appConfigFilepath,
	}
	var prunerResources corev1.ResourceRequirements
	if pruner := scheduler.Spec.Pruner; pruner != nil {
		if pruner.Args.Timeout != "" {
			prunerArgs = append(prunerArgs, "--timeout", scheduler.Spec.Pruner.Args.Timeout)
		}
		if pruner.Args.Batchsize > 0 {
			prunerArgs = append(prunerArgs, "--batchsize", fmt.Sprintf("%v", scheduler.Spec.Pruner.Args.Batchsize))
		}
		if pruner.Args.ExpireAfter != "" {
			prunerArgs = append(prunerArgs, "--expireAfter", scheduler.Spec.Pruner.Args.ExpireAfter)
		}
		if pruner.Resources != nil {
			prunerResources = *scheduler.Spec.Pruner.Resources
		}
	}

	name := scheduler.Name + "-db-pruner"

	job := batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   scheduler.Namespace,
			Labels:      AllLabels(name, scheduler.Labels),
			Annotations: map[string]string{"checksum/config": GenerateChecksumConfig(scheduler.Spec.ApplicationConfig.Raw)},
		},
		Spec: batchv1.CronJobSpec{
			Schedule:          scheduler.Spec.Pruner.Schedule,
			ConcurrencyPolicy: batchv1.ForbidConcurrent,
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: scheduler.Namespace,
					Labels:    AllLabels(name, scheduler.Labels),
				},
				Spec: batchv1.JobSpec{
					Parallelism:  &parallelism,
					Completions:  &completions,
					BackoffLimit: &backoffLimit,
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:      name,
							Namespace: scheduler.Namespace,
							Labels:    AllLabels(name, scheduler.Labels),
						},
						Spec: corev1.PodSpec{
							ServiceAccountName:            serviceAccountName,
							RestartPolicy:                 "Never",
							TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
							SecurityContext:               scheduler.Spec.PodSecurityContext,
							InitContainers: []corev1.Container{{
								Name:  "scheduler-db-pruner-db-wait",
								Image: defaultAlpineImage(),
								Command: []string{
									"/bin/sh",
									"-c",
									`echo "Waiting for Postgres..."
                                                         while ! nc -z $PGHOST $PGPORT; do
                                                           sleep 1
                                                         done
                                                         echo "Postgres started!"`,
								},
								Env: []corev1.EnvVar{
									{
										Name:  "PGHOST",
										Value: schedulerConfig.Postgres.Connection.Host,
									},
									{
										Name:  "PGPORT",
										Value: schedulerConfig.Postgres.Connection.Port,
									},
								},
							}},
							Containers: []corev1.Container{{
								Name:            name,
								ImagePullPolicy: corev1.PullIfNotPresent,
								Image:           ImageString(scheduler.Spec.Image),
								Args:            prunerArgs,
								Env:             env,
								VolumeMounts:    volumeMounts,
								SecurityContext: scheduler.Spec.SecurityContext,
								Resources:       prunerResources,
							}},
							Tolerations: scheduler.Spec.Tolerations,
							Volumes:     volumes,
						},
					},
				},
			},
		},
	}

	return &job, nil
}

// newSchedulerPrometheusRule creates a PrometheusRule for monitoring Armada scheduler.
func newSchedulerPrometheusRule(scheduler *installv1alpha1.Scheduler) *monitoringv1.PrometheusRule {
	rules := []monitoringv1.Rule{
		{
			Record: "node:armada_scheduler_failed_jobs",
			Expr:   intstr.IntOrString{StrVal: `sum by (node) (armada_scheduler_job_state_counter_by_node{state="failed"})`},
		},
		{
			Record: "cluster_category_subCategory:armada_scheduler_failed_jobs",
			Expr:   intstr.IntOrString{StrVal: `sum by (cluster, category, subCategory) (armada_scheduler_error_classification_by_node)`},
		},
		{
			Record: "queue_category_subCategory:armada_scheduler_failed_jobs",
			Expr:   intstr.IntOrString{StrVal: `sum by (queue, category, subCategory) (armada_scheduler_job_error_classification_by_queue)`},
		},
		{
			Record: "node:armada_scheduler_succeeded_jobs",
			Expr:   intstr.IntOrString{StrVal: `sum by (node) (armada_scheduler_job_state_counter_by_node{state="succeeded"})`},
		},
		{
			Record: "cluster_category_subCategory:armada_scheduler_succeeded_jobs",
			Expr:   intstr.IntOrString{StrVal: `sum by (cluster, category, subCategory) (armada_scheduler_job_state_counter_by_node{state="succeeded"})`},
		},
		{
			Record: "queue_category_subCategory:armada_scheduler_succeeded_jobs",
			Expr:   intstr.IntOrString{StrVal: `sum by (queue) (armada_scheduler_job_state_counter_by_queue{state="succeeded"})`},
		},
		{
			Record: "node:armada_scheduler_failed_jobs:increase1m",
			Expr:   intstr.IntOrString{StrVal: `increase(node:armada_scheduler_job_state_counter_by_queue{state="failed"}[1m:])`},
		},
		{
			Record: "node:armada_scheduler_failed_jobs:increase10m",
			Expr:   intstr.IntOrString{StrVal: `increase(node:armada_scheduler_job_state_counter_by_queue{state="failed"}[10m:])`},
		},
		{
			Record: "node:armada_scheduler_failed_jobs:increase1h",
			Expr:   intstr.IntOrString{StrVal: `increase(node:armada_scheduler_job_state_counter_by_queue{state="failed"}[1h:])`},
		},
		{
			Record: "cluster_category_subCategory:armada_scheduler_failed_jobs:increase1m",
			Expr:   intstr.IntOrString{StrVal: `increase(cluster_category_subCategory:armada_scheduler_failed_jobs[1m:])`},
		},
		{
			Record: "cluster_category_subCategory:armada_scheduler_failed_jobs:increase10m",
			Expr:   intstr.IntOrString{StrVal: `increase(cluster_category_subCategory:armada_scheduler_failed_jobs[10m:])`},
		},
		{
			Record: "cluster_category_subCategory:armada_scheduler_failed_jobs:increase1h",
			Expr:   intstr.IntOrString{StrVal: `increase(cluster_category_subCategory:armada_scheduler_failed_jobs[1h:])`},
		},
		{
			Record: "queue_category_subCategory:armada_scheduler_failed_jobs:increase1m",
			Expr:   intstr.IntOrString{StrVal: `increase(queue_category_subCategory:armada_scheduler_failed_jobs[1m:])`},
		},
		{
			Record: "queue_category_subCategory:armada_scheduler_failed_jobs:increase10m",
			Expr:   intstr.IntOrString{StrVal: `increase(queue_category_subCategory:armada_scheduler_failed_jobs[10m:])`},
		},
		{
			Record: "queue_category_subCategory:armada_scheduler_failed_jobs:increase1h",
			Expr:   intstr.IntOrString{StrVal: `increase(queue_category_subCategory:armada_scheduler_failed_jobs[1h:])`},
		},
		{
			Record: "node:armada_scheduler_succeeded_jobs:increase1m",
			Expr:   intstr.IntOrString{StrVal: `increase(node:armada_scheduler_succeeded_jobs[1m:])`},
		},
		{
			Record: "node:armada_scheduler_succeeded_jobs:increase10m",
			Expr:   intstr.IntOrString{StrVal: `increase(node:armada_scheduler_succeeded_jobs[10m:])`},
		},
		{
			Record: "node:armada_scheduler_succeeded_jobs:increase1h",
			Expr:   intstr.IntOrString{StrVal: `increase(node:armada_scheduler_succeeded_jobs[1h:])`},
		},
		{
			Record: "cluster_category_subCategory:armada_scheduler_succeeded_jobs:increase1m",
			Expr:   intstr.IntOrString{StrVal: `increase(cluster_category_subCategory:armada_scheduler_succeeded_jobs[1m:])`},
		},
		{
			Record: "cluster_category_subCategory:armada_scheduler_succeeded_jobs:increase10m",
			Expr:   intstr.IntOrString{StrVal: `increase(cluster_category_subCategory:armada_scheduler_succeeded_jobs[10m:])`},
		},
		{
			Record: "cluster_category_subCategory:armada_scheduler_succeeded_jobs:increase1h",
			Expr:   intstr.IntOrString{StrVal: `increase(cluster_category_subCategory:armada_scheduler_succeeded_jobs[1h:])`},
		},
		{
			Record: "queue_category_subCategory:armada_scheduler_succeeded_jobs:increase1m",
			Expr:   intstr.IntOrString{StrVal: `increase(queue_category_subCategory:armada_scheduler_succeeded_jobs[1m:])`},
		},
		{
			Record: "queue_category_subCategory:armada_scheduler_succeeded_jobs:increase10m",
			Expr:   intstr.IntOrString{StrVal: `increase(queue_category_subCategory:armada_scheduler_succeeded_jobs[10m:])`},
		},
		{
			Record: "queue_category_subCategory:armada_scheduler_succeeded_jobs:increase1h",
			Expr:   intstr.IntOrString{StrVal: `increase(queue_category_subCategory:armada_scheduler_succeeded_jobs[1h:])`},
		},
		{
			Record: "node:armada_scheduler_failed_rate_jobs:increase1m",
			Expr:   intstr.IntOrString{StrVal: `sum by(node) (node:armada_scheduler_failed_jobs:increase1m) / on(node) group_left() ((sum by(node) (node:armada_scheduler_failed_jobs:increase1m)) + (sum by(node) (node:armada_scheduler_succeeded_jobs:increase1m)))`},
		},
		{
			Record: "node:armada_scheduler_failed_rate_jobs:increase10m",
			Expr:   intstr.IntOrString{StrVal: `sum by(node) (node:armada_scheduler_failed_jobs:increase10m) / on(node) group_left() ((sum by(node) (node:armada_scheduler_failed_jobs:increase10m)) + (sum by(node) (node:armada_scheduler_succeeded_jobs:increase10m)))`},
		},
		{
			Record: "node:armada_scheduler_failed_rate_jobs:increase1h",
			Expr:   intstr.IntOrString{StrVal: `sum by(node) (node:armada_scheduler_failed_jobs:increase1h) / on(node) group_left() ((sum by(node) (node:armada_scheduler_failed_jobs:increase1h)) + (sum by(node) (node:armada_scheduler_succeeded_jobs:increase1h)))`},
		},
		{
			Record: "cluster_category_subCategory:armada_scheduler_failed_rate_jobs:increase1m",
			Expr:   intstr.IntOrString{StrVal: `sum by(cluster, category, subCategory) (cluster_category_subCategory:armada_scheduler_failed_jobs:increase1m) / on(cluster) group_left() ((sum by(cluster) (cluster_category_subCategory:armada_scheduler_failed_jobs:increase1m)) + (sum by(cluster) (cluster_category_subCategory:armada_scheduler_succeeded_jobs:increase1m)))`},
		},
		{
			Record: "cluster_category_subCategory:armada_scheduler_failed_rate_jobs:increase10m",
			Expr:   intstr.IntOrString{StrVal: `sum by(cluster, category, subCategory) (cluster_category_subCategory:armada_scheduler_failed_jobs:increase10m) / on(cluster) group_left() ((sum by(cluster) (cluster_category_subCategory:armada_scheduler_failed_jobs:increase10m)) + (sum by(cluster) (cluster_category_subCategory:armada_scheduler_succeeded_jobs:increase10m)))`},
		},
		{
			Record: "cluster_category_subCategory:armada_scheduler_failed_rate_jobs:increase1h",
			Expr:   intstr.IntOrString{StrVal: `sum by(cluster, category, subCategory) (cluster_category_subCategory:armada_scheduler_failed_jobs:increase1h) / on(cluster) group_left() ((sum by(cluster) (cluster_category_subCategory:armada_scheduler_failed_jobs:increase1h)) + (sum by(cluster) (cluster_category_subCategory:armada_scheduler_succeeded_jobs:increase1h)))`},
		},
		{
			Record: "queue_category_subCategory:armada_scheduler_failed_rate_jobs:increase1m",
			Expr:   intstr.IntOrString{StrVal: `sum by(queue, category, subCategory) (queue_category_subCategory:armada_scheduler_failed_jobs:increase1m) / on(queue) group_left() ((sum by(queue) (queue_category_subCategory:armada_scheduler_failed_jobs:increase1m)) + (sum by(queue) (queue_category_subCategory:armada_scheduler_succeeded_jobs:increase1m)))`},
		},
		{
			Record: "queue_category_subCategory:armada_scheduler_failed_rate_jobs:increase10m",
			Expr:   intstr.IntOrString{StrVal: `sum by(queue, category, subCategory) (queue_category_subCategory:armada_scheduler_failed_jobs:increase10m) / on(queue) group_left() ((sum by(queue) (queue_category_subCategory:armada_scheduler_failed_jobs:increase10m)) + (sum by(queue) (queue_category_subCategory:armada_scheduler_succeeded_jobs:increase10m)))`},
		},
		{
			Record: "queue_category_subCategory:armada_scheduler_failed_rate_jobs:increase1h",
			Expr:   intstr.IntOrString{StrVal: `sum by(queue, category, subCategory) (queue_category_subCategory:armada_scheduler_failed_jobs:increase1h) / on(queue) group_left() ((sum by(queue) (queue_category_subCategory:armada_scheduler_failed_jobs:increase1h)) + (sum by(queue) (queue_category_subCategory:armada_scheduler_succeeded_jobs:increase1h)))`},
		},
	}

	objectMetaName := "armada-" + scheduler.Name + "-metrics"
	scrapeInterval := &metav1.Duration{Duration: defaultPrometheusInterval}
	if interval := scheduler.Spec.Prometheus.ScrapeInterval; interval != nil {
		scrapeInterval = &metav1.Duration{Duration: interval.Duration}
	}
	durationString := duration.ShortHumanDuration(scrapeInterval.Duration)
	return &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scheduler.Name,
			Namespace: scheduler.Namespace,
			Labels:    AllLabels(scheduler.Name, scheduler.Spec.Labels, scheduler.Spec.Prometheus.Labels),
		},
		Spec: monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{{
				Name:     objectMetaName,
				Interval: ptr.To(monitoringv1.Duration(durationString)),
				Rules:    rules,
			}},
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *SchedulerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&installv1alpha1.Scheduler{}).
		Complete(r)
}

// extractSchedulerConfig will unmarshal the appconfig and return the SchedulerConfig portion
func extractSchedulerConfig(config runtime.RawExtension) (SchedulerConfig, error) {
	appConfig, err := builders.ConvertRawExtensionToYaml(config)
	if err != nil {
		return SchedulerConfig{}, err
	}
	var schedulerConfig SchedulerConfig
	err = yaml.Unmarshal([]byte(appConfig), &schedulerConfig)
	if err != nil {
		return SchedulerConfig{}, err
	}
	return schedulerConfig, err
}
