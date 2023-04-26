package resource

import (
	"context"
	"fmt"

	iscsigateway "github.com/Erichorng/iscsi-operator/api/v1alpha1"
	"github.com/Erichorng/iscsi-operator/internal/conf"
	pln "github.com/Erichorng/iscsi-operator/internal/planner"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	rtclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const gatewayfinalizer = "gatewayFinalizer"
const tcmuDaemonSet = "tcmu-runner"

type IscsiGatewayManager struct {
	client   rtclient.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
	logger   Logger
	cfg      *conf.OperatorConfig
}

func NewIscsiGatewayManager(
	client rtclient.Client,
	scheme *runtime.Scheme,
	logger logr.Logger,
	recorder record.EventRecorder,
) *IscsiGatewayManager {
	return &IscsiGatewayManager{
		client:   client,
		scheme:   scheme,
		recorder: recorder,
		logger:   logger,
		cfg:      conf.Get(),
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

	// check cephconfig
	err := m.checkCephConfig(ctx, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			m.logger.Error(err,
				"Can't find cephConfigMap. Please create configMap contains ceph.conf, keyring and iscsigateway.cfg")
			return Result{err: err}
		}
		m.logger.Error(err, "Failed to get cephConfigMap")
		return Result{err: err}
	}

	changed, err := m.addFinalizer(ctx, instance, gatewayfinalizer)
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

	if result := m.checkPool(ctx, instance); result.Yield() {
		return result
	}

	var planner *pln.Planner
	if p, result := m.updateConfigMap(ctx, instance); !result.Yield() {
		planner = p
	} else {
		return result
	}

	// make sure tcmu-runner daemon set is running
	if result := m.updateTcmuRunner(ctx, planner); !result.Yield() {
		m.logger.Info("Successfully update tcmu-runner",
			"daemonset.Name", tcmuDaemonSet,
			"daemonset.Namespace", instance.Namespace)
	} else {
		return result
	}

	if planner.Scale() == 1 {
		// TODO
		m.logger.Info("Please set the scale to a number greater then 1. Do not allow single node service",
			"iscsigateway.Name", instance.Name,
			"iscsigateway.Namespace", instance.Namespace)
		return Done

	} else {
		if result := m.updateClusterState(ctx, planner); result.Yield() {
			return result
		}
	}

	// Update iscsi service

	m.logger.Info("Done updating iscsi gateway resources",
		"iscsigateway.Name", instance.Name,
		"iscsigateway.Namespace", instance.Namespace)
	return Done

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

func (m *IscsiGatewayManager) checkCephConfig(
	ctx context.Context,
	ig *iscsigateway.Iscsigateway) error {

	found := &corev1.ConfigMap{}
	cmNsname := types.NamespacedName{
		Namespace: ig.Namespace,
		Name:      ig.Spec.CephConfig,
	}
	err := m.client.Get(ctx, cmNsname, found)
	if err == nil {
		return nil
	}
	return err
}

func (m *IscsiGatewayManager) addFinalizer(
	ctx context.Context,
	o rtclient.Object,
	f string) (bool, error) {

	if changed := controllerutil.AddFinalizer(o, f); changed {
		err := m.client.Update(ctx, o)
		return true, err
	}
	return false, nil
}

func (m *IscsiGatewayManager) updateTcmuRunner(
	ctx context.Context, pl *pln.Planner) Result {

	// check if tcmu-runner exist
	daemonset, created, err := m.getOrCreateTcmuRunner(ctx, tcmuDaemonSet, pl)

	if err != nil {
		return Result{err: err}
	}
	if created {
		m.logger.Info("Created tcmu-runner Daemonset")
		return Requeue
	}

	changed, err := m.claimOwnership(ctx, pl.Iscsigateway, daemonset)
	if err != nil {
		return Result{err: err}
	} else if changed {
		m.logger.Info("Update tcmu-runner ownership")
	}
	return Done
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
		m.logger.Info("Created container ConfigMap",
			"configmap.Name", configMap.Name,
			"configmap.Namespace", configMap.Namespace,
			"iscsigateway.Name", ig.Name,
			"iscsigateway.Namespace", ig.Namespace)
		return nil, Requeue
	}
	// is this step already finish in getOrCreate?
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
		m.logger.Info("Updated configMap",
			"configmap.Name", configMap.Name,
			"configmap.Namespace", configMap.Namespace,
			"iscsigateway.Name", ig.Name,
			"iscsigateway.Namespace", ig.Namespace)
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

func (m *IscsiGatewayManager) updateClusterState(
	ctx context.Context,
	planner *pln.Planner) Result {

	_, created, err := m.getOrCreateStatePVC(
		ctx, planner, planner.Iscsigateway.Namespace)
	if err != nil {
		return Result{err: err}
	}
	if created {
		m.logger.Info("Created shared state PVC")
		return Requeue
	}

	statefulset, created, err := m.getOrCreateStatefulSet(
		ctx, planner, planner.Iscsigateway.Namespace)
	if err != nil {
		return Result{err: err}
	}
	if created {
		m.logger.Info("Created StatefulSet")
		m.recorder.Eventf(planner.Iscsigateway,
			EventNormal,
			ReasonCreatedStatefulSet,
			"Create stateful set %s for IscsiGateway", statefulset.Name)
		return Requeue
	}

	changed, err := m.claimOwnership(ctx, planner.Iscsigateway, statefulset)
	if err != nil {
		return Result{err: err}
	} else if changed {
		m.logger.Info("Updated statefulSet ownership")
		return Requeue
	}
	resized, err := m.updateStatefulSetSize(
		ctx, statefulset,
		int32(planner.Scale()))
	if err != nil {
		return Result{err: err}
	} else if resized {
		m.logger.Info("Resized statefulSet")
		return Requeue
	}
	return Done
}

func (m *IscsiGatewayManager) updateStatefulSetSize(
	ctx context.Context,
	ss *appsv1.StatefulSet,
	size int32) (bool, error) {

	if *ss.Spec.Replicas < size {
		ss.Spec.Replicas = &size
		err := m.client.Update(ctx, ss)
		if err != nil {
			m.logger.Error(
				err,
				"Failed to update StatefulSet",
				"StatefulSet.Namespace", ss.Namespace,
				"StatefulSet.Name", ss.Name)
			return false, err
		}
		return true, nil
	}
	return false, nil

}

func sharedStatePVCName(planner *pln.Planner) string {
	return planner.InstanceName() + "-state"
}

func (m *IscsiGatewayManager) checkPool(
	ctx context.Context,
	ig *iscsigateway.Iscsigateway) Result {

	for i := 0; i < len(ig.Spec.Storage); i++ {
		poolname := ig.Spec.Storage[i].PoolName
		poolspec := ig.Spec.Storage[i].CephpoolSpec
		ns := m.cfg.RookNamespace
		pool, created, err := m.getOrCreatePool(ctx, poolname, ns, poolspec, ig)
		if err != nil {
			return Result{err: err}
		}
		if created {
			m.logger.Info("Create CephBlockPool",
				"CephBlockPool.Name", pool.Name,
				"CpehBlockPool.Namespace", pool.Namespace,
				"iscsigateway.Name", ig.Name,
				"iscsigateway.Namespace", ig.Namespace,
			)
			return Requeue
		}
		if pool != nil {
			// add finalizer
			/* changed, err := m.addFinalizer(ctx, pool, ig.Name)
			if err != nil {
				m.logger.Error(err,
					"failed to add finalizer",
					"pool.Name", pool.Name,
					"pool.Namespace", pool.Namespace,
					"iscsigateway.Name", ig.Name,
					"iscsigateway.Namespace", ig.Namespace,
				)
				return Result{err: err}
			}
			if changed {
				m.logger.Info("add finalizer",
					"pool.Name", pool.Name,
					"pool.Namespace", pool.Namespace,
					"iscsigateway.Name", ig.Name,
					"iscsigateway.Namespace", ig.Namespace,
				)
				return Requeue
			} */

		} else {
			// pool already bean used. change pool name.
			err := fmt.Errorf("pool already been used. please change a name")
			m.logger.Error(err,
				"This pool is already been used. please change a name.",
				"pool.Name", poolname,
				"pool.Namespace", ns,
				"iscsigateway.Name", ig.Name,
				"iscsigateway.Namespace", ig.Namespace,
			)
			return Result{err: err}
		}
	}

	// remove finalizer from deleted pool
	configMap, err := m.getExistingConfigmap(ctx, ig.Name, ig.Namespace)

	if err != nil {
		if !errors.IsNotFound(err) {
			m.logger.Error(
				err,
				"Failed to get configMap",
				"configMap.Namespace", ig.Namespace,
				"configMap.Name", ig.Name,
			)
			return Result{err: err}
		}
		return Done
	}
	cc, err := getContainerConfig(configMap)
	if err != nil {
		m.logger.Error(err, "Unable to read iscsi container config")
		return Result{err: err}
	}

	new_pool_list := make([]string, len(ig.Spec.Storage))
	for i := 0; i < len(ig.Spec.Storage); i++ {
		new_pool_list[i] = ig.Spec.Storage[i].PoolName
	}
	for k := range cc.Storage {
		if !pln.Exist(k, new_pool_list) {
			// remove finalizer in pool
			pool, err := m.getExistingPool(ctx, k, m.cfg.RookNamespace)
			if err != nil {
				m.logger.Error(err,
					"Failed to get exist pool which may exist",
					"CephBlockPool.Name", k,
					"CephBlockPool.Namespace", m.cfg.RookNamespace)
				return Result{err: err}
			}
			if pool != nil {
				// remove finalizer
				/* changed := controllerutil.RemoveFinalizer(pool, ig.Name)
				if changed {
					m.logger.Info("delete finalizer from pool",
						"finalizer", ig.Name,
						"pool.Name", k)
					return Requeue
				} */
				// delete pool
				err2 := m.client.Delete(ctx, pool)
				if err2 != nil {
					m.logger.Error(err2,
						"Failed to delete pool.",
						"Pool.Name", pool.Name,
						"Pool.Namespace", pool.Namespace,
					)
				} else {
					m.logger.Info("Successfully delete pool",
						"Pool.Name", pool.Name,
						"Pool.Namespace", pool.Namespace,
					)
				}
			}
		}
	}

	return Done
}
