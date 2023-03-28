// Package conf defines the operator's configuration parameters.
package conf

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var DefaultOperatorConfig = OperatorConfig{
	IscsiContainerImage: "docker.com/ruohwai/iscsi:v17.2.2",
	IscsiContainerName:  "iscsi",
	TcmuRunnerImage:     "docker.com/ruohwai/iscsi:v17.2.2.tcmu",
	TcmuRunnerName:      "tcmu-runner",
	ImagePullPolicy:     "IfNotPresent",
	PoolName:            "rbd",
	Hostname:            "iqn.0000.default:client",
	User:                "IscsiUser",
	Password:            "1234",
	StatePVCSize:        "1G",
	ApiPort:             5001,
	IscsiPort:           3260,
}

type OperatorConfig struct {
	IscsiContainerImage string
	IscsiContainerName  string
	TcmuRunnerImage     string
	TcmuRunnerName      string
	ImagePullPolicy     string
	User                string
	Password            string
	Hostname            string
	PoolName            string
	StatePVCSize        string
	ApiPort             int
	IscsiPort           int
}

func (oc *OperatorConfig) Validate() error {
	if oc.IscsiContainerImage == "" {
		return fmt.Errorf(
			"IscsiContainerImage value [%s] imvalid", oc.IscsiContainerImage)
	}
	return nil
}

type Source struct {
	v    *viper.Viper
	fset *pflag.FlagSet
}

func NewSource() *Source {
	d := DefaultOperatorConfig
	v := viper.New()
	v.SetDefault("iscsi-container-image", d.IscsiContainerImage)
	v.SetDefault("tcmu-runner-image", d.TcmuRunnerImage)
	v.SetDefault("iscsi-container-name", d.IscsiContainerName)
	v.SetDefault("tcmu-runner-name", d.TcmuRunnerName)
	v.SetDefault("iscsi-pool-name", d.PoolName)
	v.SetDefault("iscsi-username", d.User)
	v.SetDefault("iscsi-password", d.Password)
	v.SetDefault("iscsi-host", d.Hostname)
	v.SetDefault("state-pvc-size", d.StatePVCSize)
	v.SetDefault("image-pull-policy", d.ImagePullPolicy)
	v.SetDefault("api-port", d.ApiPort)
	v.SetDefault("iscsi-port", d.IscsiPort)
	return &Source{v: v}
}

func (s *Source) Flags() *pflag.FlagSet {
	if s.fset != nil {
		return s.fset
	}
	s.fset = pflag.NewFlagSet("conf", pflag.ExitOnError)
	for _, k := range s.v.AllKeys() {
		s.fset.String(k, "", fmt.Sprintf("Specify the %q configuration parameter", k))
	}
	return s.fset
}

func (s *Source) Read() (*OperatorConfig, error) {
	v := s.v

	v.AddConfigPath("/etc/iscsi-operator")
	v.AddConfigPath(".")
	v.SetConfigName("iscsi-operator")
	err := v.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}
	v.SetEnvPrefix("ISCSI_OP")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	// use cli flags if available
	if s.fset != nil {
		err = v.BindPFlags(s.fset)
		if err != nil {
			return nil, err
		}
	}
	c := &OperatorConfig{}
	if err := v.Unmarshal(c); err != nil {
		return nil, err
	}
	return c, nil
}
