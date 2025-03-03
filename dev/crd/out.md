# API Reference

## Packages
- [core.armadaproject.io/v1alpha1](#corearmadaprojectiov1alpha1)
- [install.armadaproject.io/v1alpha1](#installarmadaprojectiov1alpha1)


## core.armadaproject.io/v1alpha1

Package v1alpha1 contains API Schema definitions for the core v1alpha1 API group

### Resource Types
- [Queue](#queue)
- [QueueList](#queuelist)



#### Queue



Queue is the Schema for the queues API



_Appears in:_
- [QueueList](#queuelist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `core.armadaproject.io/v1alpha1` | | |
| `kind` _string_ | `Queue` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[QueueSpec](#queuespec)_ |  |  |  |
| `status` _[QueueStatus](#queuestatus)_ |  |  |  |


#### QueueList



QueueList contains a list of Queue





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `core.armadaproject.io/v1alpha1` | | |
| `kind` _string_ | `QueueList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Queue](#queue) array_ |  |  |  |


#### QueueSpec



QueueSpec defines the desired state of Queue



_Appears in:_
- [Queue](#queue)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `foo` _string_ | Foo is an example field of Queue. Edit queue_types.go to remove/update |  |  |


#### QueueStatus



QueueStatus defines the observed state of Queue



_Appears in:_
- [Queue](#queue)




## install.armadaproject.io/v1alpha1

Package v1alpha1 contains API Schema definitions for the install v1alpha1 API group

### Resource Types
- [ArmadaServer](#armadaserver)
- [ArmadaServerList](#armadaserverlist)
- [Binoculars](#binoculars)
- [BinocularsList](#binocularslist)
- [EventIngester](#eventingester)
- [EventIngesterList](#eventingesterlist)
- [Executor](#executor)
- [ExecutorList](#executorlist)
- [Lookout](#lookout)
- [LookoutIngester](#lookoutingester)
- [LookoutIngesterList](#lookoutingesterlist)
- [LookoutList](#lookoutlist)
- [Scheduler](#scheduler)
- [SchedulerIngester](#scheduleringester)
- [SchedulerIngesterList](#scheduleringesterlist)
- [SchedulerList](#schedulerlist)



#### AdditionalClusterRoleBinding







_Appears in:_
- [ExecutorSpec](#executorspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `nameSuffix` _string_ |  |  |  |
| `clusterRoleName` _string_ |  |  |  |


#### ArmadaServer



ArmadaServer is the Schema for the Armada Server API



_Appears in:_
- [ArmadaServerList](#armadaserverlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `install.armadaproject.io/v1alpha1` | | |
| `kind` _string_ | `ArmadaServer` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ArmadaServerSpec](#armadaserverspec)_ |  |  |  |
| `status` _[ArmadaServerStatus](#armadaserverstatus)_ |  |  |  |




#### ArmadaServerList



ArmadaServerList contains a list of ArmadaServer





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `install.armadaproject.io/v1alpha1` | | |
| `kind` _string_ | `ArmadaServerList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[ArmadaServer](#armadaserver) array_ |  |  |  |


#### ArmadaServerSpec



ArmadaServerSpec defines the desired state of ArmadaServer



_Appears in:_
- [ArmadaServer](#armadaserver)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `CommonSpecBase` _[CommonSpecBase](#commonspecbase)_ |  |  |  |
| `replicas` _integer_ | Replicas is the number of replicated instances for ArmadaServer |  |  |
| `nodeSelector` _object (keys:string, values:string)_ | NodeSelector restricts the ArmadaServer pod to run on nodes matching the configured selectors |  |  |
| `ingress` _[IngressConfig](#ingressconfig)_ | Ingress defines configuration for the Ingress resource |  |  |
| `profilingIngressConfig` _[IngressConfig](#ingressconfig)_ | ProfilingIngressConfig defines configuration for the profiling Ingress resource |  |  |
| `hostNames` _string array_ | An array of host names to build ingress rules for |  |  |
| `clusterIssuer` _string_ | Who is issuing certificates for CA |  |  |
| `pulsarInit` _boolean_ | Run Pulsar Init Jobs On Startup |  |  |
| `securityContext` _[SecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#securitycontext-v1-core)_ | SecurityContext defines the security options the container should be run with |  |  |
| `podSecurityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#podsecuritycontext-v1-core)_ | PodSecurityContext defines the security options the pod should be run with |  |  |


#### ArmadaServerStatus



ArmadaServerStatus defines the observed state of ArmadaServer



_Appears in:_
- [ArmadaServer](#armadaserver)





#### Binoculars



Binoculars is the Schema for the binoculars API



_Appears in:_
- [BinocularsList](#binocularslist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `install.armadaproject.io/v1alpha1` | | |
| `kind` _string_ | `Binoculars` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[BinocularsSpec](#binocularsspec)_ |  |  |  |
| `status` _[BinocularsStatus](#binocularsstatus)_ |  |  |  |




#### BinocularsList



BinocularsList contains a list of Binoculars





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `install.armadaproject.io/v1alpha1` | | |
| `kind` _string_ | `BinocularsList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Binoculars](#binoculars) array_ |  |  |  |


#### BinocularsSpec



BinocularsSpec defines the desired state of Binoculars



_Appears in:_
- [Binoculars](#binoculars)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `CommonSpecBase` _[CommonSpecBase](#commonspecbase)_ |  |  |  |
| `replicas` _integer_ | Replicas is the number of replicated instances for Binoculars |  |  |
| `nodeSelector` _object (keys:string, values:string)_ | NodeSelector restricts the pod to run on nodes matching the configured selectors |  |  |
| `ingress` _[IngressConfig](#ingressconfig)_ | Ingress for this component. Used to inject labels/annotations into ingress |  |  |
| `profilingIngressConfig` _[IngressConfig](#ingressconfig)_ | ProfilingIngressConfig defines configuration for the profiling Ingress resource |  |  |
| `hostNames` _string array_ | An array of host names to build ingress rules for |  |  |
| `clusterIssuer` _string_ | Who is issuing certificates for CA |  |  |
| `securityContext` _[SecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#securitycontext-v1-core)_ | SecurityContext defines the security options the container should be run with |  |  |
| `podSecurityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#podsecuritycontext-v1-core)_ | PodSecurityContext defines the security options the pod should be run with |  |  |


#### BinocularsStatus



BinocularsStatus defines the observed state of binoculars



_Appears in:_
- [Binoculars](#binoculars)



#### CommonSpecBase



CommonSpecBase is the common configuration for all services.
NOTE(Clif): You must label this with `json:""` when using it as an embedded
struct in order for controller-gen to use the promoted fields as expected.



_Appears in:_
- [ArmadaServerSpec](#armadaserverspec)
- [BinocularsSpec](#binocularsspec)
- [EventIngesterSpec](#eventingesterspec)
- [ExecutorSpec](#executorspec)
- [LookoutIngesterSpec](#lookoutingesterspec)
- [LookoutSpec](#lookoutspec)
- [SchedulerIngesterSpec](#scheduleringesterspec)
- [SchedulerSpec](#schedulerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `labels` _object (keys:string, values:string)_ | Labels is the map of labels which wil be added to all objects |  |  |
| `image` _[Image](#image)_ | Image is the configuration block for the image repository and tag |  |  |
| `applicationConfig` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#rawextension-runtime-pkg)_ | ApplicationConfig is the internal configuration of the application which will be created as a Kubernetes Secret and mounted in the Kubernetes Deployment object |  | Schemaless: \{\} <br /> |
| `prometheus` _[PrometheusConfig](#prometheusconfig)_ | PrometheusConfig is the configuration block for Prometheus monitoring |  |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#resourcerequirements-v1-core)_ | Resources is the configuration block for setting resource requirements for this service |  |  |
| `tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#toleration-v1-core) array_ | Tolerations is the configuration block for specifying which taints this pod can tolerate |  |  |
| `terminationGracePeriodSeconds` _integer_ | TerminationGracePeriodSeconds specifies how many seconds should Kubernetes wait for the application to shut down gracefully before sending a KILL signal |  |  |
| `customServiceAccount` _string_ | if CustomServiceAccount is specified, then that service account is referenced in the Deployment (overrides service account defined in spec.serviceAccount field) |  |  |
| `serviceAccount` _[ServiceAccountConfig](#serviceaccountconfig)_ | if ServiceAccount configuration is defined, it creates a new service account and references it in the deployment |  |  |
| `environment` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#envvar-v1-core) array_ | Extra environment variables that get added to deployment |  |  |
| `additionalVolumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#volume-v1-core) array_ | Additional volumes that are mounted into deployments |  |  |
| `additionalVolumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#volumemount-v1-core) array_ | Additional volume mounts that are added as volumes |  |  |


#### EventIngester



EventIngester is the Schema for the eventingesters API



_Appears in:_
- [EventIngesterList](#eventingesterlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `install.armadaproject.io/v1alpha1` | | |
| `kind` _string_ | `EventIngester` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[EventIngesterSpec](#eventingesterspec)_ |  |  |  |
| `status` _[EventIngesterStatus](#eventingesterstatus)_ |  |  |  |




#### EventIngesterList



EventIngesterList contains a list of EventIngester





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `install.armadaproject.io/v1alpha1` | | |
| `kind` _string_ | `EventIngesterList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[EventIngester](#eventingester) array_ |  |  |  |


#### EventIngesterSpec



EventIngesterSpec defines the desired state of EventIngester



_Appears in:_
- [EventIngester](#eventingester)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `CommonSpecBase` _[CommonSpecBase](#commonspecbase)_ |  |  |  |
| `replicas` _integer_ | Replicas is the number of replicated instances for EventIngester |  |  |
| `nodeSelector` _object (keys:string, values:string)_ | NodeSelector restricts the Executor pod to run on nodes matching the configured selectors |  |  |
| `securityContext` _[SecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#securitycontext-v1-core)_ | SecurityContext defines the security options the container should be run with |  |  |
| `podSecurityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#podsecuritycontext-v1-core)_ | PodSecurityContext defines the security options the pod should be run with |  |  |
| `profilingIngressConfig` _[IngressConfig](#ingressconfig)_ | ProfilingIngressConfig defines configuration for the profiling Ingress resource |  |  |


#### EventIngesterStatus



EventIngesterStatus defines the observed state of EventIngester



_Appears in:_
- [EventIngester](#eventingester)





#### Executor



Executor is the Schema for the executors API



_Appears in:_
- [ExecutorList](#executorlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `install.armadaproject.io/v1alpha1` | | |
| `kind` _string_ | `Executor` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ExecutorSpec](#executorspec)_ |  |  |  |
| `status` _[ExecutorStatus](#executorstatus)_ |  |  |  |




#### ExecutorList



ExecutorList contains a list of Executor





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `install.armadaproject.io/v1alpha1` | | |
| `kind` _string_ | `ExecutorList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Executor](#executor) array_ |  |  |  |


#### ExecutorSpec



ExecutorSpec defines the desired state of Executor



_Appears in:_
- [Executor](#executor)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `CommonSpecBase` _[CommonSpecBase](#commonspecbase)_ |  |  |  |
| `replicas` _integer_ | Replicas is the number of replicated instances for Executor |  |  |
| `nodeSelector` _object (keys:string, values:string)_ | NodeSelector restricts the Executor pod to run on nodes matching the configured selectors |  |  |
| `additionalClusterRoleBindings` _[AdditionalClusterRoleBinding](#additionalclusterrolebinding) array_ | Additional ClusterRoleBindings which will be created |  |  |
| `priorityClasses` _[PriorityClass](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#priorityclass-v1-scheduling) array_ | List of PriorityClasses which will be created |  |  |
| `securityContext` _[SecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#securitycontext-v1-core)_ | SecurityContext defines the security options the container should be run with |  |  |
| `podSecurityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#podsecuritycontext-v1-core)_ | PodSecurityContext defines the security options the pod should be run with |  |  |
| `profilingIngressConfig` _[IngressConfig](#ingressconfig)_ | ProfilingIngressConfig defines configuration for the profiling Ingress resource |  |  |


#### ExecutorStatus



ExecutorStatus defines the observed state of Executor



_Appears in:_
- [Executor](#executor)





#### Image







_Appears in:_
- [CommonSpecBase](#commonspecbase)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `repository` _string_ |  |  | MinLength: 1 <br />Pattern: `^([a-z0-9]+(?:[._-][a-z0-9]+)*/*)+$` <br /> |
| `tag` _string_ |  |  | MinLength: 1 <br />Pattern: `^[a-zA-Z0-9_.-]*$` <br /> |


#### IngressConfig







_Appears in:_
- [ArmadaServerSpec](#armadaserverspec)
- [BinocularsSpec](#binocularsspec)
- [EventIngesterSpec](#eventingesterspec)
- [ExecutorSpec](#executorspec)
- [LookoutIngesterSpec](#lookoutingesterspec)
- [LookoutSpec](#lookoutspec)
- [SchedulerIngesterSpec](#scheduleringesterspec)
- [SchedulerSpec](#schedulerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `labels` _object (keys:string, values:string)_ | Labels is the map of labels which wil be added to all objects |  |  |
| `annotations` _object (keys:string, values:string)_ | Annotations is a map of annotations which will be added to all ingress rules |  |  |
| `ingressClass` _string_ | The type of ingress that is used |  |  |
| `hostNames` _string array_ | An array of host names to build ingress rules for |  |  |
| `clusterIssuer` _string_ | Who is issuing certificates for CA |  |  |


#### Lookout



Lookout is the Schema for the lookout API



_Appears in:_
- [LookoutList](#lookoutlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `install.armadaproject.io/v1alpha1` | | |
| `kind` _string_ | `Lookout` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[LookoutSpec](#lookoutspec)_ |  |  |  |
| `status` _[LookoutStatus](#lookoutstatus)_ |  |  |  |




#### LookoutIngester



LookoutIngester is the Schema for the lookoutingesters API



_Appears in:_
- [LookoutIngesterList](#lookoutingesterlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `install.armadaproject.io/v1alpha1` | | |
| `kind` _string_ | `LookoutIngester` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[LookoutIngesterSpec](#lookoutingesterspec)_ |  |  |  |
| `status` _[LookoutIngesterStatus](#lookoutingesterstatus)_ |  |  |  |




#### LookoutIngesterList



LookoutIngesterList contains a list of LookoutIngester





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `install.armadaproject.io/v1alpha1` | | |
| `kind` _string_ | `LookoutIngesterList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[LookoutIngester](#lookoutingester) array_ |  |  |  |


#### LookoutIngesterSpec



LookoutIngesterSpec defines the desired state of LookoutIngester



_Appears in:_
- [LookoutIngester](#lookoutingester)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `CommonSpecBase` _[CommonSpecBase](#commonspecbase)_ |  |  |  |
| `replicas` _integer_ | Replicas is the number of replicated instances for LookoutIngester |  |  |
| `securityContext` _[SecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#securitycontext-v1-core)_ | SecurityContext defines the security options the container should be run with |  |  |
| `podSecurityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#podsecuritycontext-v1-core)_ | PodSecurityContext defines the security options the pod should be run with |  |  |
| `profilingIngressConfig` _[IngressConfig](#ingressconfig)_ | ProfilingIngressConfig defines configuration for the profiling Ingress resource |  |  |


#### LookoutIngesterStatus



LookoutIngesterStatus defines the observed state of LookoutIngester



_Appears in:_
- [LookoutIngester](#lookoutingester)





#### LookoutList



LookoutList contains a list of Lookout





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `install.armadaproject.io/v1alpha1` | | |
| `kind` _string_ | `LookoutList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Lookout](#lookout) array_ |  |  |  |


#### LookoutSpec



LookoutSpec defines the desired state of Lookout



_Appears in:_
- [Lookout](#lookout)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `CommonSpecBase` _[CommonSpecBase](#commonspecbase)_ |  |  |  |
| `replicas` _integer_ | Replicas is the number of replicated instances for Lookout |  |  |
| `nodeSelector` _object (keys:string, values:string)_ | NodeSelector restricts the Lookout pod to run on nodes matching the configured selectors |  |  |
| `ingress` _[IngressConfig](#ingressconfig)_ | Ingress defines labels and annotations for the Ingress controller of Lookout |  |  |
| `hostNames` _string array_ | An array of host names to build ingress rules for |  |  |
| `clusterIssuer` _string_ | Who is issuing certificates for CA |  |  |
| `migrate` _boolean_ | Migrate toggles whether to run migrations when installed |  |  |
| `dbPruningEnabled` _boolean_ | DbPruningEnabled when true a pruning CronJob is created |  |  |
| `dbPruningSchedule` _string_ | DbPruningSchedule schedule to use for db pruning CronJob |  |  |
| `securityContext` _[SecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#securitycontext-v1-core)_ | SecurityContext defines the security options the container should be run with |  |  |
| `podSecurityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#podsecuritycontext-v1-core)_ | PodSecurityContext defines the security options the pod should be run with |  |  |
| `profilingIngressConfig` _[IngressConfig](#ingressconfig)_ | ProfilingIngressConfig defines configuration for the profiling Ingress resource |  |  |


#### LookoutStatus



LookoutStatus defines the observed state of lookout



_Appears in:_
- [Lookout](#lookout)



#### PrometheusConfig







_Appears in:_
- [CommonSpecBase](#commonspecbase)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ | Enabled toggles should PrometheusRule and ServiceMonitor be created |  |  |
| `labels` _object (keys:string, values:string)_ | Labels field enables adding additional labels to PrometheusRule and ServiceMonitor |  |  |
| `scrapeInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#duration-v1-meta)_ | ScrapeInterval defines the interval at which Prometheus should scrape metrics |  | Format: duration <br />Type: string <br /> |


#### PrunerArgs



PrunerArgs represent command-line args to the pruner cron job



_Appears in:_
- [PrunerConfig](#prunerconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `timeout` _string_ |  |  |  |
| `batchsize` _integer_ |  |  |  |
| `expireAfter` _string_ |  |  |  |


#### PrunerConfig



PrunerConfig definees the pruner cronjob settings



_Appears in:_
- [SchedulerSpec](#schedulerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `enabled` _boolean_ |  |  |  |
| `schedule` _string_ |  |  |  |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#resourcerequirements-v1-core)_ |  |  |  |
| `args` _[PrunerArgs](#prunerargs)_ |  |  |  |


#### Scheduler



Scheduler is the Schema for the scheduler API



_Appears in:_
- [SchedulerList](#schedulerlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `install.armadaproject.io/v1alpha1` | | |
| `kind` _string_ | `Scheduler` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[SchedulerSpec](#schedulerspec)_ |  |  |  |
| `status` _[SchedulerStatus](#schedulerstatus)_ |  |  |  |




#### SchedulerIngester



SchedulerIngester is the Schema for the scheduleringesters API



_Appears in:_
- [SchedulerIngesterList](#scheduleringesterlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `install.armadaproject.io/v1alpha1` | | |
| `kind` _string_ | `SchedulerIngester` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[SchedulerIngesterSpec](#scheduleringesterspec)_ |  |  |  |
| `status` _[SchedulerIngesterStatus](#scheduleringesterstatus)_ |  |  |  |




#### SchedulerIngesterList



SchedulerIngesterList contains a list of SchedulerIngester





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `install.armadaproject.io/v1alpha1` | | |
| `kind` _string_ | `SchedulerIngesterList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[SchedulerIngester](#scheduleringester) array_ |  |  |  |


#### SchedulerIngesterSpec



SchedulerIngesterSpec defines the desired state of SchedulerIngester



_Appears in:_
- [SchedulerIngester](#scheduleringester)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `CommonSpecBase` _[CommonSpecBase](#commonspecbase)_ |  |  |  |
| `replicas` _integer_ | Replicas is the number of replicated instances for SchedulerIngester |  |  |
| `securityContext` _[SecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#securitycontext-v1-core)_ | SecurityContext defines the security options the container should be run with |  |  |
| `podSecurityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#podsecuritycontext-v1-core)_ | PodSecurityContext defines the security options the pod should be run with |  |  |
| `profilingIngressConfig` _[IngressConfig](#ingressconfig)_ | ProfilingIngressConfig defines configuration for the profiling Ingress resource |  |  |


#### SchedulerIngesterStatus



SchedulerIngesterStatus defines the observed state of SchedulerIngester



_Appears in:_
- [SchedulerIngester](#scheduleringester)





#### SchedulerList



SchedulerList contains a list of Scheduler





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `install.armadaproject.io/v1alpha1` | | |
| `kind` _string_ | `SchedulerList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Scheduler](#scheduler) array_ |  |  |  |


#### SchedulerSpec



SchedulerSpec defines the desired state of Scheduler



_Appears in:_
- [Scheduler](#scheduler)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `CommonSpecBase` _[CommonSpecBase](#commonspecbase)_ |  |  |  |
| `replicas` _integer_ | Replicas is the number of replicated instances for Scheduler |  |  |
| `ingress` _[IngressConfig](#ingressconfig)_ | Ingress defines labels and annotations for the Ingress controller of Scheduler |  |  |
| `profilingIngressConfig` _[IngressConfig](#ingressconfig)_ | ProfilingIngressConfig defines configuration for the profiling Ingress resource |  |  |
| `hostNames` _string array_ | An array of host names to build ingress rules for |  |  |
| `clusterIssuer` _string_ | Who is issuing certificates for CA |  |  |
| `migrate` _boolean_ | Migrate toggles whether to run migrations when installed |  |  |
| `pruner` _[PrunerConfig](#prunerconfig)_ | Pruning config for cron job |  |  |
| `securityContext` _[SecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#securitycontext-v1-core)_ | SecurityContext defines the security options the container should be run with |  |  |
| `podSecurityContext` _[PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#podsecuritycontext-v1-core)_ | PodSecurityContext defines the security options the pod should be run with |  |  |


#### SchedulerStatus



SchedulerStatus defines the observed state of scheduler



_Appears in:_
- [Scheduler](#scheduler)



#### ServiceAccountConfig







_Appears in:_
- [CommonSpecBase](#commonspecbase)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `secrets` _[ObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#objectreference-v1-core) array_ |  |  |  |
| `imagePullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.29/#localobjectreference-v1-core) array_ |  |  |  |
| `automountServiceAccountToken` _boolean_ |  |  |  |


