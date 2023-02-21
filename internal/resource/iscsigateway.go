package resource

import (
	"context"
	"fmt"

	iscsigateway "github.com/Erichong/iscsi-operator/api/v1alpha1"
	"github.com/Erichong/iscsi-operator/internal/conf"
	pln "github.com/Erichong/iscsi-operator/internal/planner"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	rtclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const gatewayfinalizer = "gatewayFinalizer"

type IscsiGatewayManager struct {
	client rtclient.Client
	scheme *runtime.Scheme
	logger logr.Logger
	cfg    *conf.OperatorConfig
}

func NewIscsiGatewayManager(
	client rtclient.Client,
	scheme *runtime.Scheme,
	logger logr.Logger,
) *IscsiGatewayManager {
	return &IscsiGatewayManager{
		client: client,
		scheme: scheme,
		logger: logger,
		cfg:    conf.Get(),
	}
}

func (m *IscsiGatewayManager) Process(
	ctx context.Context,
	nsname types.NamespacedName) Result {
	instance := &iscsigateway.Iscsigateway{}
	err := m.client.Get(ctx, nsname, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return Done
		}
		m.logger.Error(
			err,
			"Failed to get IscsiGateway",
			"IscsiGateway.Namespace", nsname.Namespace,
			"IscsiGateway.Name", nsname.Name,
		)
		return Result{err: err}
	}

	if instance.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(instance, gatewayfinalizer) {
			return m.Finalize(ctx, instance)
		}
	}
	return m.Update(ctx, instance)
}

func (m *IscsiGatewayManager) Update(
	ctx context.Context,
	instance *iscsigateway.Iscsigateway) Result {
	m.logger.Info(
		"Updating state for IscsiGateway",
		"IscsiGateway.Namespace", instance.Namespace,
		"IscsiGateway.Name", instance.Name,
		"IscsiGateway.UID", instance.UID,
	)
	changed, err := m.addFinalizer(ctx, instance)
	if err != nil {
		return Result{err: err}
	}
	if changed {
		m.logger.Info(
			"Add finalizer to IscsiGateway",
			"IscsiGateway.Namespace", instance.Namespace,
			"IscsiGateway.Name", instance.Name,
		)
		return Requeue
	}

	var planner *pln.Planner
	if p, result := m.updateConfigMap(ctx, instance); !result.Yield() {
		planner = p
	} else {
		return result
	}

	// ceph config

}

func (m *IscsiGatewayManager) Finalize(
	ctx context.Context,
	instance *iscsigateway.Iscsigateway) Result {

	// some handle
	// TODO

	m.logger.Info("Remove finalizer")
	controllerutil.RemoveFinalizer(instance, gatewayfinalizer)
	err := m.client.Update(ctx, instance)
	if err != nil {
		return Result{err: err}
	}
	return Done
}

func (m *IscsiGatewayManager) addFinalizer(
	ctx context.Context,
	instance *iscsigateway.Iscsigateway) (bool, error) {

	if controllerutil.ContainsFinalizer(instance, gatewayfinalizer) {
		return true, m.client.Update(ctx, instance)
	}
	return false, nil
}

func (m *IscsiGatewayManager) updateConfigMap(
	ctx context.Context,
	ig *iscsigateway.Iscsigateway) (*pln.Planner, Result) {
	// destNamespace := i.Namespace
	configMap, created, err := m.getOrCreateConfigMap(ctx, ig)
	if err != nil {
		return nil, Result{err: err}
	}
	if created {
		m.logger.Info("Created ConfigMap")
		return nil, Requeue
	}
	// is this step already finish in getorcreate?
	changed, err := m.claimOwnership(ctx, ig, configMap)
	if err != nil {
		return nil, Result{err: err}
	} else if changed {
		m.logger.Info("Update configMap ownership")
	}

	planner, changed, err := m.updateConfiguration(ctx, configMap, ig)
	if err != nil {
		return nil, Result{err: err}
	}
	if changed {
		m.logger.Info("Updated configMap")
		return nil, Requeue
	}
	return planner, Done
}

// obj is an object that can be any api resource
func (m *IscsiGatewayManager) claimOwnership(
	ctx context.Context,
	ig *iscsigateway.Iscsigateway,
	obj rtclient.Object) (bool, error) {
	gvk, err := apiutil.GVKForObject(ig, m.scheme)
	if err != nil {
		return false, err
	}
	refs := obj.GetOwnerReferences()
	for _, ref := range refs {
		refgv, err := schema.ParseGroupVersion(ref.APIVersion)
		if err != nil {
			return false, err
		}
		if gvk.Group == refgv.Group && gvk.Kind == ref.Kind && ig.GetName() == ref.Name {
			//found it. return false to indicate no changes
			return false, nil
		}
	}

	oref := metav1.OwnerReference{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		UID:        ig.GetUID(),
		Name:       ig.GetName(),
	}
	refs = append(refs, oref)
	obj.SetOwnerReferences(refs)
	return true, m.client.Update(ctx, obj)
}

func (m *IscsiGatewayManager) updateConfiguration(
	ctx context.Context,
	configMap *corev1.ConfigMap,
	ig *iscsigateway.Iscsigateway) (*pln.Planner, bool, error) {
	//extract config from map
	cc, err := getContainerConfig(configMap)
	if err != nil {
		m.logger.Error(err, "Unable to reade iscsi container config")
		return nil, false, err
	}
	isDeleting := ig.GetDeletionTimestamp() != nil
	if isDeleting {
		err := fmt.Errorf(
			"updateConfiguration called for deleted iscsi gateway: %s",
			ig.Name)
		return nil, false, err
	}
	gatewayInstance, err := m.getGatewayInstance(ctx, ig)
	if err != nil {
		return nil, false, err
	}

	// extract config from map
	var changed bool
	planner := pln.New(gatewayInstance, cc)
	changed, err = planner.Update()
	if err != nil {
		m.logger.Error(err, "unable to update iscsi container config")
		return nil, false, err
	}
	if !changed {
		return planner, false, nil
	}
	err = setContainerConfig(configMap, planner.ConfigState) // we have already update the configstate in the last step
	if err != nil {
		m.logger.Error(
			err,
			"unable to set container config in ConfigMap",
			"ConfigMap.Namespace", configMap.Namespace,
			"ConfigMap.Name", configMap.Name,
		)
		return nil, false, err
	}
	err = m.client.Update(ctx, configMap)
	if err != nil {
		m.logger.Error(
			err,
			"failed to update ConfigMap",
			"ConfigMap.Namespace", configMap.Namespace,
			"ConfigMap.Name", configMap.Name,
		)
		return nil, false, err
	}
	return planner, true, nil
}
