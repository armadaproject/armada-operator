package install

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	installv1alpha1 "github.com/armadaproject/armada-operator/api/install/v1alpha1"
	operatorconfig "github.com/armadaproject/armada-operator/internal/config"
	"github.com/armadaproject/armada-operator/internal/controller/builders"
)

// spec.nodeSelector and spec.tolerations are declared on every component's CRD
// (CommonSpecBase), so they must reach the pod template of every workload the
// operator creates.
func TestNodeSelectorPropagates(t *testing.T) {
	t.Parallel()

	nodeSelector := map[string]string{"kubernetes.io/hostname": "control-plane"}
	tolerations := []corev1.Toleration{{Key: "dedicated", Operator: corev1.TolerationOpExists}}
	common := installv1alpha1.CommonSpecBase{
		Image:             installv1alpha1.Image{Repository: "testrepo", Tag: "1.0.0"},
		ApplicationConfig: runtime.RawExtension{Raw: []byte(`{}`)},
		NodeSelector:      nodeSelector,
		Tolerations:       tolerations,
	}
	meta := metav1.ObjectMeta{Namespace: "default", Name: "component"}
	config := &builders.CommonApplicationConfig{HTTPPort: 8080, GRPCPort: 5051, MetricsPort: 9000}

	tests := []struct {
		name    string
		podSpec func() (*corev1.PodSpec, error)
	}{
		{
			name: "Scheduler Deployment",
			podSpec: func() (*corev1.PodSpec, error) {
				scheduler := &installv1alpha1.Scheduler{ObjectMeta: meta, Spec: installv1alpha1.SchedulerSpec{
					CommonSpecBase: common,
					Replicas:       ptr.To[int32](1),
				}}
				deployment, err := newSchedulerDeployment(scheduler, "sa", config)
				if err != nil {
					return nil, err
				}
				return &deployment.Spec.Template.Spec, nil
			},
		},
		{
			name: "Scheduler pruner CronJob",
			podSpec: func() (*corev1.PodSpec, error) {
				scheduler := &installv1alpha1.Scheduler{ObjectMeta: meta, Spec: installv1alpha1.SchedulerSpec{
					CommonSpecBase: common,
					Pruner:         &installv1alpha1.PrunerConfig{Enabled: true},
				}}
				cronJob, err := newSchedulerCronJob(scheduler, "sa", operatorconfig.Images{})
				if err != nil {
					return nil, err
				}
				return &cronJob.Spec.JobTemplate.Spec.Template.Spec, nil
			},
		},
		{
			name: "SchedulerIngester Deployment",
			podSpec: func() (*corev1.PodSpec, error) {
				ingester := &installv1alpha1.SchedulerIngester{ObjectMeta: meta, Spec: installv1alpha1.SchedulerIngesterSpec{
					CommonSpecBase: common,
					Replicas:       ptr.To[int32](1),
				}}
				deployment, err := (&SchedulerIngesterReconciler{}).createDeployment(ingester, "sa", config)
				if err != nil {
					return nil, err
				}
				return &deployment.Spec.Template.Spec, nil
			},
		},
		{
			name: "LookoutIngester Deployment",
			podSpec: func() (*corev1.PodSpec, error) {
				ingester := &installv1alpha1.LookoutIngester{ObjectMeta: meta, Spec: installv1alpha1.LookoutIngesterSpec{
					CommonSpecBase: common,
					Replicas:       ptr.To[int32](1),
				}}
				deployment, err := (&LookoutIngesterReconciler{}).createDeployment(ingester, "sa", config)
				if err != nil {
					return nil, err
				}
				return &deployment.Spec.Template.Spec, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			podSpec, err := tt.podSpec()
			require.NoError(t, err)
			assert.Equal(t, nodeSelector, podSpec.NodeSelector)
			assert.Equal(t, tolerations, podSpec.Tolerations)
		})
	}
}
