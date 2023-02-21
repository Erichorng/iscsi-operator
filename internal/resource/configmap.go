package resource

import (
	"encoding/json"

	"github.com/Erichong/iscsi-operator/internal/iscsicc"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ConfigMapName is the name of the configmap volume.
	configMapName = "iscsi-container-config"

	// ConfigJSONKey is the name of the key our json is under.
	ConfigJSONKey = "config.json"
)

func newDefaultConfigMap(name, ns string) (*corev1.ConfigMap, error) {
	jb, err := json.MarshalIndent(iscsicc.New(), "", " ")
	if err != nil {
		return nil, err
	}
	data := map[string]string{ConfigJSONKey: string(jb)}
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Data: data,
	}
	return configMap, nil
}

func getContainerConfig(
	configMap *corev1.ConfigMap) (*iscsicc.IscsiContainerConfig, error) {
	cc := iscsicc.New()
	jstr, found := configMap.Data[ConfigJSONKey]
	if !found {
		return cc, nil
	}
	if err := json.Unmarshal([]byte(jstr), cc); err != nil {
		return nil, err
	}
	return cc, nil
}

func setContainerConfig(
	cm *corev1.ConfigMap, cc *iscsicc.IscsiContainerConfig) error {

	jb, err := json.MarshalIndent(cc, "", "  ")
	if err != nil {
		return err
	}
	cm.Data[ConfigJSONKey] = string(jb)
	return nil
}
