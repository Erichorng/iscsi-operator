package planner

import (
	"path"
)

func (pl *Planner) ConfigMountPath() string {
	return "/etc/container-config"
}

func (pl *Planner) IscsiStateDir() string {
	return "/var/lib/iscsi"
}

func (pl *Planner) CephMountPath() string {
	return "/etc/ceph"
}

func (pl *Planner) DevMountPath() string {
	return "/dev"
}

func (pl *Planner) LibMountPath() string {
	return "/lib/modules"
}

func (pl *Planner) ContainerConfig() string {
	return path.Join(pl.ConfigMountPath(), "config.json")
}
