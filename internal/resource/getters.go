package resource

import (
	"context"

	iscsigateway "github.com/Erichong/iscsi-operator/api/v1alpha1"
	pln "github.com/Erichong/iscsi-operator/internal/planner"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	kresource "k8s.io/apimachinery/pkg/api/resource"
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

func (m *IscsiGatewayManager) getOrCreateStatefulSet(
	ctx context.Context,
	pl *pln.Planner,
	ns string) (*appsv1.StatefulSet, bool, error) {
	found, err := m.getExistingStatefulSet(ctx, pl, ns)
	if err != nil {
		return nil, false, err
	}
	if found != nil {
		return found, false, nil
	}

	// if not found, create a new stateful set
	ss := buildStatefulSet(
		pl,
		ns,
		sharedStatePVCName(pl),
	)

	err = controllerutil.SetControllerReference(
		pl.Iscsigateway, ss, m.scheme)
	if err != nil {
		m.logger.Error(
			err,
			"Failed to set controller reference",
			"Iscsigateway.Namespace", pl.Iscsigateway.Namespace,
			"Iscsigateway.Name", pl.Iscsigateway.Name,
			"StatefulSet.Namespace", ss.Namespace,
			"StatefulSet.Name", ss.Name,
		)
		return ss, false, err
	}
	m.logger.Info(
		"Creating a new StatefulSet",
		"Iscsigateway.Namespace", pl.Iscsigateway.Namespace,
		"Iscsigateway.Name", pl.Iscsigateway.Name,
		"StatefulSet.Namespace", ss.Namespace,
		"StatefulSet.Name", ss.Name,
	)
	err = m.client.Create(ctx, ss)
	if err != nil {
		m.logger.Error(
			err,
			"Failed to create new StatefulSet",
			"Iscsigateway.Namespace", pl.Iscsigateway.Namespace,
			"Iscsigateway.Name", pl.Iscsigateway.Name,
			"StatefulSet.Namespace", ss.Namespace,
			"StatefulSet.Name", ss.Name,
		)
		return ss, false, err
	}
	return ss, true, err
}

func (m *IscsiGatewayManager) getOrCreateStatePVC(
	ctx context.Context,
	planner *pln.Planner,
	ns string) (*corev1.PersistentVolumeClaim, bool, error) {

	name := sharedStatePVCName(planner)
	squant, err := kresource.ParseQuantity(planner.GlobalConfig.StatePVCSize)
	if err != nil {
		return nil, false, err
	}
	spec := &corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{
			corev1.ReadWriteMany,
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: squant,
			},
		},
	}
	pvc, cr, err := m.getOrCreateGenericPVC(
		ctx, planner.Iscsigateway, spec, name, ns)
	if err != nil {
		m.logger.Error(err, "Error establishing shared state PVC")
	}
	return pvc, cr, err

}

func (m *IscsiGatewayManager) getExistingStatefulSet(
	ctx context.Context,
	pl *pln.Planner,
	ns string) (*appsv1.StatefulSet, error) {

	found := &appsv1.StatefulSet{}
	ssKey := types.NamespacedName{
		Namespace: ns,
		Name:      pl.InstanceName(),
	}
	err := m.client.Get(ctx, ssKey, found)
	if err == nil {
		return found, nil
	}

	if !errors.IsNotFound(err) {
		m.logger.Error(
			err,
			"Failed to get StatefulSet",
			"Iscsigateway.Namespace", ns,
			"Iscsigateway.Name", pl.Iscsigateway.Name,
			"StatefulSet.Namespace", ssKey.Namespace,
			"StatefulSet.Name", ssKey.Name,
		)
		return nil, err
	}
	return nil, nil

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

func (m *IscsiGatewayManager) getOrCreateGenericPVC(
	ctx context.Context,
	ig *iscsigateway.Iscsigateway,
	spec *corev1.PersistentVolumeClaimSpec,
	name, ns string) (*corev1.PersistentVolumeClaim, bool, error) {

	pvc, err := m.getExistingPVC(ctx, name, ns)
	if err != nil {
		return nil, false, err
	}
	if pvc != nil {
		return pvc, false, nil
	}
	// create new pvc
	pvc = &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: *spec,
	}
	err = controllerutil.SetControllerReference(
		ig, pvc, m.scheme)
	if err != nil {
		m.logger.Error(
			err,
			"Failed to set controller reference",
			"IscsiGateway.Namespace", ig.Namespace,
			"IScsiGateway.Name", ig.Name,
			"PersistentVolumeClaim.Namespace", pvc.Namespace,
			"PersistentVolumeClaim.Name", pvc.Name,
		)
		return pvc, false, err
	}
	m.logger.Info(
		"Creating a new PVC",
		"IscsiGateway.Namespace", ig.Namespace,
		"IScsiGateway.Name", ig.Name,
		"PersistentVolumeClaim.Namespace", pvc.Namespace,
		"PersistentVolumeClaim.Name", pvc.Name,
	)
	err = m.client.Create(ctx, pvc)
	if err != nil {
		m.logger.Error(
			err,
			"Failed to create new PVC",
			"IscsiGateway.Namespace", ig.Namespace,
			"IScsiGateway.Name", ig.Name,
			"PersistentVolumeClaim.Namespace", pvc.Namespace,
			"PersistentVolumeClaim.Name", pvc.Name,
		)
		return pvc, false, err
	}
	return pvc, true, nil
}

func (m *IscsiGatewayManager) getExistingPVC(
	ctx context.Context,
	name, ns string) (*corev1.PersistentVolumeClaim, error) {

	pvc := &corev1.PersistentVolumeClaim{}
	pvcKey := types.NamespacedName{
		Namespace: ns,
		Name:      name,
	}
	err := m.client.Get(ctx, pvcKey, pvc)
	if err == nil {
		return pvc, nil
	}
	if !errors.IsNotFound(err) {
		m.logger.Error(
			err,
			"Failed to get PVC",
			"PersistentVolumeClaim.Namespace", pvcKey.Namespace,
			"PersistentVolumeClaim.Name", pvcKey.Name,
		)
		return nil, err
	}
	return nil, nil
}
