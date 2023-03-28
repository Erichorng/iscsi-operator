package planner

import (
	"strconv"
	"time"

	"github.com/Erichong/iscsi-operator/internal/iscsicc"
)

func sameStringSlice(x, y []string) bool {
	if len(x) != len(y) {
		return false
	}
	// create a map of string -> int
	diff := make(map[string]int, len(x))
	for _, _x := range x {
		// 0 value for int is 0, so just increment a counter for the string
		diff[_x]++
	}
	for _, _y := range y {
		// If the string _y is not in diff bail out early
		if _, ok := diff[_y]; !ok {
			return false
		}
		diff[_y] -= 1
		if diff[_y] == 0 {
			delete(diff, _y)
		}
	}
	return len(diff) == 0
}

func exist(ss string, l []string) bool {
	for _, s := range l {
		if s == ss {
			return true
		}
	}
	return false
}

func (pl *Planner) targetName() string {
	year := strconv.Itoa(time.Now().Year())
	day := strconv.Itoa(time.Now().Day())
	prefix := "iqn." + year + "-" + day + ".com.redhat.iscsi-gw:"
	if pl.Iscsigateway.Spec.TargetName == "" {
		return prefix + pl.Iscsigateway.Name // Or default name?
	} else if checkValidTargetName(pl.Iscsigateway.Spec.TargetName) {
		return pl.Iscsigateway.Spec.TargetName
	} else {
		return prefix + pl.Iscsigateway.Name // Or default name?
	}
}

func (pl *Planner) Update() (changed bool, err error) {
	// set target name
	targetName := pl.targetName()
	pl.ConfigState.TargetName = targetName

	// set global config
	// if the global section in the container config not found,
	// create it by the default settings of operator config
	globals, found := pl.ConfigState.Globals[iscsicc.Globals]
	if !found {
		globalOptions := iscsicc.NewGlobalOptions()
		globalOptions.DefaultHostName = pl.GlobalConfig.Hostname
		globalOptions.DefaultPassword = pl.GlobalConfig.Password
		globalOptions.DefaultUser = pl.GlobalConfig.User
		globals = iscsicc.NewGlobals(globalOptions)
		pl.ConfigState.Globals[iscsicc.Globals] = globals
		changed = true
	}

	// if we change the setting of the operator config and rebuild it,
	// we will change the container config's global section.
	if globals.Options[iscsicc.Glob_Host] != pl.GlobalConfig.Hostname {
		globals.Options[iscsicc.Glob_Host] = pl.GlobalConfig.Hostname
		pl.ConfigState.Globals[iscsicc.Globals] = globals
		changed = true
	}
	if globals.Options[iscsicc.Glob_User] != pl.GlobalConfig.User {
		globals.Options[iscsicc.Glob_User] = pl.GlobalConfig.User
		pl.ConfigState.Globals[iscsicc.Globals] = globals
		changed = true
	}
	if globals.Options[iscsicc.Glob_PWD] != pl.GlobalConfig.Password {
		globals.Options[iscsicc.Glob_PWD] = pl.GlobalConfig.Password
		pl.ConfigState.Globals[iscsicc.Globals] = globals
		changed = true
	}

	// Storage section

	for i := 0; i < len(pl.Iscsigateway.Spec.Storage); i++ {
		goalPoolName := pl.Iscsigateway.Spec.Storage[i].PoolName
		_, found := pl.ConfigState.Storage[goalPoolName]
		// if we have new pool, add it to container config with all it's disk config
		if !found {
			// add pool with no disk
			pl.ConfigState.Storage[goalPoolName] = iscsicc.NewEmptyDisk()
			// add disks
			disks := pl.Iscsigateway.Spec.Storage[i].Disks
			for d := 0; d < len(disks); d++ {
				diskName := disks[d].DiskName
				diskSize := disks[d].DiskSize
				pl.ConfigState.Storage[goalPoolName][diskName] = diskSize
			}
			changed = true
			continue
		}

		//if it is an old pool, see if we have new disk or disk resize
		for j := 0; j < len(pl.Iscsigateway.Spec.Storage[i].Disks); j++ {
			goalDiskName := pl.Iscsigateway.Spec.Storage[i].Disks[j].DiskName
			goalDiskSize := pl.Iscsigateway.Spec.Storage[i].Disks[j].DiskSize
			_, found := pl.ConfigState.Storage[goalPoolName][goalDiskName]

			if !found {
				pl.ConfigState.Storage[goalPoolName][goalDiskName] = goalDiskSize
				changed = true
				continue
			}
			// if the disk exist but the size have changed

			if pl.ConfigState.Storage[goalPoolName][goalDiskName] != goalDiskSize {
				pl.ConfigState.Storage[goalPoolName][goalDiskName] = goalDiskSize
				changed = true
			}
		}
		// if disk removed
		allDisk := pl.ConfigState.Storage[goalPoolName]
		new_disk_list := make([]string, len(pl.Iscsigateway.Spec.Storage[i].Disks))
		for idx, w := range pl.Iscsigateway.Spec.Storage[i].Disks {
			new_disk_list[idx] = w.DiskName
		}
		for k := range allDisk {
			if !exist(k, new_disk_list) {
				delete(pl.ConfigState.Storage[goalPoolName], k)
				changed = true
			}
		}
	}

	//check if there are pools removed
	new_pool_list := make([]string, len(pl.Iscsigateway.Spec.Storage))
	for i := 0; i < len(pl.Iscsigateway.Spec.Storage); i++ {
		new_pool_list[i] = pl.Iscsigateway.Spec.Storage[i].PoolName
	}
	for k := range pl.ConfigState.Storage {
		if !exist(k, new_pool_list) {
			// remove the pool from configstate
			delete(pl.ConfigState.Storage, k)
			changed = true
		}
	}

	// host section

	// if new host
	for i := 0; i < len(pl.Iscsigateway.Spec.Hosts); i++ {
		goalHostname := pl.Iscsigateway.Spec.Hosts[i].HostName
		goalUser := pl.Iscsigateway.Spec.Hosts[i].Username
		goalPwd := pl.Iscsigateway.Spec.Hosts[i].Password
		goallun := iscsicc.GetLuns(pl.Iscsigateway.Spec.Hosts[i].Luns)

		_, found := pl.ConfigState.Hosts[goalHostname]
		if !found {
			pl.ConfigState.Hosts[goalHostname] = iscsicc.NewHostInfo(goalUser, goalPwd, goallun)
			changed = true
		}

		// if found but lun change (can user, pwd change? seems like yes)

		if host, found := pl.ConfigState.Hosts[goalHostname]; found {
			if host.User != goalUser {
				host.User = goalUser
				//pl.ConfigState.Hosts[goalHostname] = host
				changed = true
			}
			if host.Password != goalPwd {
				host.Password = goalPwd
				//pl.ConfigState.Hosts[goalHostname] = host
				changed = true
			}
			if !sameStringSlice(host.Lun, goallun) {
				host.Lun = goallun
				//pl.ConfigState.Hosts[goalHostname] = host
				changed = true
			}
			pl.ConfigState.Hosts[goalHostname] = host
		}
	}

	new_host_list := make([]string, len(pl.Iscsigateway.Spec.Hosts))
	for k := range pl.ConfigState.Hosts {
		if !exist(k, new_host_list) {
			delete(pl.ConfigState.Hosts, k)
			changed = true
		}
	}

	return
}

func checkValidTargetName(string) bool {
	// TODO
	return true
}
