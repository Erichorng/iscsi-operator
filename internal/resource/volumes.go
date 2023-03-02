package resource

import (
	"fmt"

	pln "github.com/Erichong/iscsi-operator/internal/planner"
	corev1 "k8s.io/api/core/v1"
)

const (
	stateVolName = "iscsi-state-dir"
	cephVolName  = "iscsi-ceph-config-dir"
	devVolName   = "dev-vol-dir"
	libVolName   = "lib-vol-dir"
)

type volMountTag uint

type volMount struct {
	volume corev1.Volume
	mount  corev1.VolumeMount
	tag    volMountTag
}

type volKeeper struct {
	vols []volMount
}

func getVolumes(vols []volMount) []corev1.Volume {
	v := make([]corev1.Volume, len(vols))
	for i := range vols {
		v[i] = vols[i].volume
	}
	return v
}

func getMounts(vols []volMount) []corev1.VolumeMount {
	m := make([]corev1.VolumeMount, len(vols))
	for i := range vols {
		m[i] = vols[i].mount
	}
	return m
}

func newVolKeeper() *volKeeper {
	return &volKeeper{vols: []volMount{}}
}

func configVolumeAndMount(pl *pln.Planner) volMount {
	var vmnt volMount
	configMapSrc := &corev1.ConfigMapVolumeSource{}
	configMapSrc.Name = pl.InstanceName() //the same as the containerconfig name
	vmnt.volume = corev1.Volume{
		Name: configMapName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: configMapSrc,
		},
	}
	vmnt.mount = corev1.VolumeMount{
		MountPath: pl.ConfigMountPath(),
		Name:      configMapName,
	}
	vmnt.tag = volMountTag(0x0) //dont' use the tag
	return vmnt
}

func iscsiStateVolumeAndMount(pl *pln.Planner) volMount {
	var vmnt volMount
	vmnt.volume = corev1.Volume{
		Name: stateVolName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumDefault,
			},
		},
	}
	vmnt.mount = corev1.VolumeMount{
		MountPath: pl.IscsiStateDir(),
		Name:      stateVolName,
	}
	vmnt.tag = volMountTag(0x0)
	return vmnt
}

func cephVolumeAndMount(pl *pln.Planner) volMount {
	var vmnt volMount
	configMapSrc := &corev1.ConfigMapVolumeSource{}
	configMapSrc.Name = pl.CephConfigName() // ceph config name
	vmnt.volume = corev1.Volume{
		Name: cephVolName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: configMapSrc,
		},
	}
	vmnt.mount = corev1.VolumeMount{
		MountPath: pl.CephMountPath(),
		Name:      cephVolName,
	}
	vmnt.tag = volMountTag(0x0)
	return vmnt
}

func devVolumeAndMount(pl *pln.Planner) volMount {
	var vmnt volMount
	hostpathtype := corev1.HostPathDirectory
	vmnt.volume = corev1.Volume{
		Name: devVolName,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: pl.DevMountPath(),
				Type: &hostpathtype,
			},
		},
	}

	vmnt.mount = corev1.VolumeMount{
		MountPath: pl.DevMountPath(),
		Name:      devVolName,
	}
	vmnt.tag = volMountTag(0x0)
	return vmnt
}

func libVolumeAndMount(pl *pln.Planner) volMount {
	var vmnt volMount
	hostpathtype := corev1.HostPathDirectory
	vmnt.volume = corev1.Volume{
		Name: libVolName,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: pl.LibMountPath(),
				Type: &hostpathtype,
			},
		},
	}
	vmnt.mount = corev1.VolumeMount{
		MountPath: pl.LibMountPath(),
		Name:      libVolName,
	}
	vmnt.tag = volMountTag(0x0)
	return vmnt
}

func (vk *volKeeper) add(v volMount) *volKeeper {
	vk.vols = append(vk.vols, v)
	return vk
}

// validate the volume/mounts are in a good state, returning an
// error describing the issue or nil on no error.
func (vk *volKeeper) validate() error {
	vnames := map[string]bool{}
	for _, vmnt := range vk.vols {
		if vmnt.volume.Name != vmnt.mount.Name {
			return fmt.Errorf(
				"volume/mount name mismatch: %s != %s",
				vmnt.volume.Name,
				vmnt.mount.Name)
		}
		if vnames[vmnt.volume.Name] {
			return fmt.Errorf(
				"duplicate volume name found: %s",
				vmnt.volume.Name)
		}
		vnames[vmnt.volume.Name] = true
	}
	return nil
}

// mustValidate panics if the volKeeper or the contained volume/mounts
// are in a bad state. See validate for details. Returns the current
// volkeeper for chaining.
func (vk *volKeeper) mustValidate() *volKeeper {
	// call check to validate certain programmer level invariants
	// are good. Ideally, this is exercised by a unit test.
	if err := vk.validate(); err != nil {
		panic(err)
	}
	return vk
}

// all volumes tracked by the volKeeper.
func (vk *volKeeper) all() []volMount {
	return vk.mustValidate().vols
}
