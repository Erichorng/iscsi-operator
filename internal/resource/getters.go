package resource

import (
	"context"

	iscsigateway "github.com/Erichong/iscsi-operator/api/v1alpha1"
	pln "github.com/Erichong/iscsi-operator/internal/planner"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (m *IscsiGatewayManager) getOrCreateConfigMap(
	ctx context.Context,
	ig *iscsigateway.Iscsigateway) (*corev1.ConfigMap, bool, error) {

	planner := pln.New(
		pln.InstanceConfiguration{
			Iscsigateway: ig,
			GlobalConfig: m.cfg,
		},
		nil)
	found := &corev1.ConfigMap{}
	// name of the configMap defaults to smbshare's name
	cmNsname := types.NamespacedName{
		Name:      planner.Iscsigateway.Name,
		Namespace: ig.Namespace,
	}
	err := m.client.Get(ctx, cmNsname, found)
	if err == nil {
		return found, false, nil
	}
	if !errors.IsNotFound(err) {
		m.logger.Error(
			err,
			"Failed to get configMap",
			"IscsiGateway.Name", planner.Iscsigateway.Name,
			"IscsiGateway.Namespace", planner.Iscsigateway.Namespace,
			"ConfigMap.Name", cmNsname.Name, cmNsname.Name,
			"ConfigMap.Namespace", cmNsname.Name, cmNsname.Namespace,
		)
		return nil, false, err
	}

	configMap, err := newDefaultConfigMap(cmNsname.Name, cmNsname.Namespace)
	if err != nil {
		m.logger.Error(
			err,
			"Failed to create default ConfigMap",
			"IscsiGateway.Name", planner.Iscsigateway.Name,
			"IscsiGateway.Namespace", planner.Iscsigateway.Namespace,
			"ConfigMap.Name", cmNsname.Name, cmNsname.Name,
			"ConfigMap.Namespace", cmNsname.Name, cmNsname.Namespace,
		)
		return configMap, false, err
	}
	// set the isscigateway instance as the owner and controller
	err = controllerutil.SetControllerReference(
		planner.Iscsigateway, configMap, m.scheme)
	if err != nil {
		m.logger.Error(
			err,
			"Failed to set controller reference",
			"IscsiGateway.Name", ig.Name,
			"IscsiGateway.Namespace", ig.Namespace,
			"ConfigMap.Name", configMap.Name,
			"ConfigMap.Namespace", configMap.Namespace,
		)
		return configMap, false, err
	}

	err = m.client.Create(ctx, configMap)
	if err != nil {
		m.logger.Error(
			err,
			"Failed to create new ConfigMap",
			"IscsiGateway.Name", ig.Name,
			"IscsiGateway.Namespace", ig.Namespace,
			"ConfigMap.Name", configMap.Name,
			"ConfigMap.Namespace", configMap.Namespace,
		)
		return configMap, false, err
	}
	return configMap, true, nil
}

func (m *IscsiGatewayManager) getGatewayInstance(
	ctx context.Context,
	ig *iscsigateway.Iscsigateway) (pln.InstanceConfiguration, error) {
	gatewayInstance := pln.InstanceConfiguration{
		Iscsigateway: ig,
		GlobalConfig: m.cfg,
	}
	return gatewayInstance, nil
}
