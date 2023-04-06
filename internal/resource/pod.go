package resource

import (
	pln "github.com/Erichorng/iscsi-operator/internal/planner"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func buildTcmuRunnerPodSpec(pl *pln.Planner) corev1.PodSpec {
	var (
		volumes    = newVolKeeper()
		containers []corev1.Container
	)

	cephVol := cephVolumeAndMount(pl)
	volumes.add(cephVol)

	devVol := devVolumeAndMount(pl)
	volumes.add(devVol)

	libVol := libVolumeAndMount(pl)
	volumes.add(libVol)

	containers = append(containers, buildTcmuCtr(pl, volumes))

	podSpec := corev1.PodSpec{}
	podSpec.Volumes = getVolumes(volumes.all())

	podSpec.Containers = containers

	return podSpec
}

func buildClusteredPodSpec(
	pl *pln.Planner, sharedPVCName string) corev1.PodSpec {
	var (
		volumes        = newVolKeeper()
		initContainers []corev1.Container
		containers     []corev1.Container
	)
	cephVol := cephVolumeAndMount(pl)
	volumes.add(cephVol)

	configVol := configVolumeAndMount(pl)
	volumes.add(configVol)

	stateVol := iscsiStateVolumeAndMount(pl)
	volumes.add(stateVol)

	devVol := devVolumeAndMount(pl)
	volumes.add(devVol)

	libVol := libVolumeAndMount(pl)
	volumes.add(libVol)

	podEnv := defaultPodEnv(pl)

	initContainers = append(initContainers, buildInitCtr(pl, podEnv, volumes))

	initContainers = append(initContainers, buildIscsiSetNodeCtr(pl, podEnv, volumes))

	containers = append(containers, buildIscsiCtrs(pl, podEnv, volumes)...)

	podSpec := corev1.PodSpec{}
	podSpec.Volumes = getVolumes(volumes.all())
	podSpec.InitContainers = initContainers
	podSpec.Containers = containers
	return podSpec
}

func defaultPodEnv(planner *pln.Planner) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{
			Name:  "ISCSI_CONTAINER_ID",
			Value: planner.InstanceName(),
		},
		{
			Name:  "ISCSI_CONFIG",
			Value: planner.ContainerConfig(),
		},
	}

	return env
}

func buildInitCtr(
	pl *pln.Planner,
	env []corev1.EnvVar,
	vols *volKeeper) corev1.Container {

	mounts := getMounts(vols.all())
	return corev1.Container{
		Image:           pl.GlobalConfig.IscsiContainerImage,
		ImagePullPolicy: imagePullPolicy(pl),
		Name:            "init",
		Args:            pl.Args().Initializer("init"),
		Env:             env,
		VolumeMounts:    mounts,
	}
}

func buildIscsiSetNodeCtr(
	pl *pln.Planner,
	env []corev1.EnvVar,
	vols *volKeeper) corev1.Container {

	mounts := getMounts(vols.all())
	return corev1.Container{
		Image:           pl.GlobalConfig.IscsiContainerImage,
		ImagePullPolicy: imagePullPolicy(pl),
		Name:            "Iscsi-set-node",
		Args:            pl.Args().SetNode(),
		Env:             env,
		VolumeMounts:    mounts,
	}
}

func buildIscsiCtrs(
	pl *pln.Planner,
	env []corev1.EnvVar,
	vols *volKeeper) []corev1.Container {

	ctrs := []corev1.Container{}
	ctrs = append(ctrs, buildIscsiCtr(pl, env, vols))
	ctrs = append(ctrs, buildUpdateConfigWatchCtr(pl, env, vols))
	return ctrs
}

func buildTcmuCtr(
	pl *pln.Planner,
	vols *volKeeper) corev1.Container {

	mounts := getMounts(vols.all())
	var t bool = true
	return corev1.Container{
		Image:           pl.GlobalConfig.TcmuRunnerImage,
		ImagePullPolicy: imagePullPolicy(pl),
		Name:            pl.GlobalConfig.TcmuRunnerName,
		Command: []string{
			"/usr/sbin/init",
		},
		SecurityContext: &corev1.SecurityContext{
			Privileged: &t,
		},
		//Args: ,
		//Env: ,
		VolumeMounts: mounts,
	}

}

func buildIscsiCtr(
	pl *pln.Planner,
	env []corev1.EnvVar,
	vols *volKeeper) corev1.Container {

	mounts := getMounts(vols.all())
	iscsiport := pl.GlobalConfig.IscsiPort
	var t bool = true
	return corev1.Container{
		Image:           pl.GlobalConfig.IscsiContainerImage,
		ImagePullPolicy: imagePullPolicy(pl),
		Name:            pl.GlobalConfig.IscsiContainerName,
		Command: []string{
			"systemctl start tcmu-runner",
			"systemctl start rbd-target-gw",
			"systemctl start rbd-target-api",
		},
		SecurityContext: &corev1.SecurityContext{
			Privileged: &t,
		},
		Args:         pl.Args().Run("iscsi-daemon"), // no use right now
		Env:          env,
		VolumeMounts: mounts,
		// liveness probe, readiness Probe
		// TODO
		// use 5001? 3260

		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(iscsiport),
				},
			},
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(iscsiport),
				},
			},
		},
	}

}

func buildUpdateConfigWatchCtr(
	planner *pln.Planner,
	env []corev1.EnvVar,
	vols *volKeeper) corev1.Container {
	// ---
	mounts := getMounts(vols.all())
	return corev1.Container{
		Image:        planner.GlobalConfig.IscsiContainerImage,
		Name:         "watch-update-config",
		Args:         planner.Args().UpdateConfigWatch(),
		Env:          env,
		VolumeMounts: mounts,
	}
}

func imagePullPolicy(pl *pln.Planner) corev1.PullPolicy {
	pullPolicy := corev1.PullPolicy(pl.GlobalConfig.ImagePullPolicy)
	switch {
	case pullPolicy == corev1.PullAlways:
	case pullPolicy == corev1.PullNever:
	case pullPolicy == corev1.PullIfNotPresent:
	default:
		pullPolicy = corev1.PullIfNotPresent
	}
	return pullPolicy
}
