package planner

func (pl *Planner) Scale() int32 {
	if pl.Iscsigateway.Spec.Scale == 0 {
		return 1
	}
	return int32(pl.Iscsigateway.Spec.Scale)
}

func (pl *Planner) IsClustered() bool {
	return pl.Scale() != 1
}

func (pl *Planner) InstanceName() string {
	return pl.Iscsigateway.Name
}

func (pl *Planner) CephConfigName() string {
	return pl.Iscsigateway.Spec.CephConfig
}

func (pl *Planner) GetApiPort() int {
	if pl.GlobalConfig.ApiPort != 0 {
		return pl.GlobalConfig.ApiPort
	}
	return pl.GlobalConfig.ApiPort
}
