package resource

import (
	"context"

	pln "github.com/Erichong/iscsi-operator/internal/planner"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func buildDaemonset(
	ctx context.Context,
	name string,
	pl *pln.Planner) *appsv1.DaemonSet {

	podSpec := buildTcmuRunnerPodSpec(pl)
	labels := map[string]string{
		"app":                          "tcmu-runner",
		"app.kubernetes.io/name":       "tcmu-runner",
		"app.kuvernetes.io/instance":   "tcmu-runner",
		"app.kubernetes.io/managed-by": "iscsi-operator",
	}

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: pl.Iscsigateway.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: podSpec,
			},
		},
	}
	return ds
}
