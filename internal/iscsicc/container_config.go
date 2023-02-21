package iscsicc

import (
	api "github.com/Erichong/iscsi-operator/api/v1alpha1"
)

type Key string

const (
	Globals = Key("globals")
	Storage = Key("storage")

	Glob_Host = "hostname"
	Glob_User = "username"
	Glob_PWD  = "password"
	Glob_LUN  = "lun"
)

type IscsiContainerConfig struct {
	TargetName string               `json:"targetname,omitempty"`
	Storage    PoolConfig           `json:"storage,omitempty"`
	Hosts      HostConfig           `json:"hosts,omitempty"`
	Globals    map[Key]GlobalConfig `json:"globals,omitempty"`
}

/* type HostConfig struct {
	Host map[Key]HostInformation `json:"host,omitempty"`
} */

type HostInfo struct {
	User     string   `json:"user,omitempty"`
	Password string   `json:"password,omitempty"`
	Lun      []string `json:"lun,omitempty"`
}

type GlobalConfig struct {
	Options IscsiOptions `json:"options,omitempty"`
}

type GlobalOptions struct {
	DefaultHostName string
	DefaultUser     string
	DefaultPassword string
}

type IscsiOptions map[string]string

// (diskname, disksize)
type DiskConfig map[string]string

// (poolname, diskConfig)
type PoolConfig map[string]DiskConfig

type HostConfig map[string]HostInfo

func New() *IscsiContainerConfig {
	return &IscsiContainerConfig{
		TargetName: "",
		Storage:    PoolConfig{},
		Hosts:      HostConfig{},
		Globals:    map[Key]GlobalConfig{},
	}
}

func NewGlobalOptions() GlobalOptions {
	return GlobalOptions{}
}

func NewGlobals(globalopt GlobalOptions) GlobalConfig {
	return GlobalConfig{
		Options: IscsiOptions{
			Glob_Host: globalopt.DefaultHostName,
			Glob_User: globalopt.DefaultUser,
			Glob_PWD:  globalopt.DefaultPassword,
		},
	}
}

func NewEmptyDisk() DiskConfig {
	return DiskConfig{}
}

func NewHostInfo(user, pwd string, luns []string) HostInfo {
	return HostInfo{
		User:     user,
		Password: pwd,
		Lun:      luns,
	}
}

func GetLuns(luns []api.IscsiLunSpec) []string {
	l := []string{}
	for i := 0; i < len(luns); i++ {
		poolName := luns[i].PoolName
		diskName := luns[i].DiskName
		lun := poolName + "/" + diskName
		l = append(l, lun)
	}
	return l
}

func NewPools() PoolConfig {
	return PoolConfig{}
}
