package planner

import (
	api "github.com/Erichorng/iscsi-operator/api/v1alpha1"
	"github.com/Erichorng/iscsi-operator/internal/conf"
	"github.com/Erichorng/iscsi-operator/internal/iscsicc"
)

type InstanceConfiguration struct {
	Iscsigateway *api.Iscsigateway
	GlobalConfig *conf.OperatorConfig
}

type Planner struct {
	InstanceConfiguration
	ConfigState *iscsicc.IscsiContainerConfig
}

func New(
	ic InstanceConfiguration,
	state *iscsicc.IscsiContainerConfig) *Planner {
	return &Planner{
		InstanceConfiguration: ic,
		ConfigState:           state,
	}
}
