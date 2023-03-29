package resource

import (
	"strings"

	"github.com/Erichorng/iscsi-operator/internal/conf"
	pln "github.com/Erichorng/iscsi-operator/internal/planner"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func buildStatefulSet(
	pl *pln.Planner,
	ns,
	statePVCName string) *appsv1.StatefulSet {
	labels := labelsForIscsiServer(pl.InstanceName())
	size := pl.Scale()
	podSpec := buildClusteredPodSpec(pl, statePVCName)

	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pl.InstanceName(),
			Namespace: ns,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &size,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotationsForIscsiPod(pl.GlobalConfig),
				},
				Spec: podSpec,
			},
		},
	}
	return statefulSet

}

func labelsForIscsiServer(name string) map[string]string {
	return map[string]string{
		"app":                          "iscsi",
		"app.kubernetes.io/name":       "iscsi",
		"app.kuvernetes.io/instance":   labelValue("iscsi", name),
		"app.kubernetes.io/managed-by": "iscsi-operator",
	}
}

func labelValue(s ...string) string {
	out := strings.Join(s, "-")
	if len(out) > 63 {
		out = out[:63]
	}
	return out
}

func annotationsForIscsiPod(cfg *conf.OperatorConfig) map[string]string {
	name := cfg.IscsiContainerName
	annotations := map[string]string{
		"kubectl.kubernetes.io/default-logs-container": name,
		"kubectl.kubernetes.io/default-container":      name,
	}
	return annotations
}
