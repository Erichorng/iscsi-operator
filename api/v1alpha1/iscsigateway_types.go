/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IscsigatewaySpec defines the desired state of Iscsigateway
type IscsigatewaySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// GatewayName is an optional string that lets you define an ISCSI gateway
	// name. If unset, the name will be defived automatically.
	// +optional
	TargetName string             `json:"targetname"`
	Storage    []IscsiStorageSpec `json:"storage"`
	Hosts      []IscsiHostSpec    `json:"hosts"`
	Scale      int                `json:"scale"`
}

type IscsiStorageSpec struct {
	PoolName string          `json:"poolname"`
	Disks    []IscsiDiskSpec `json:"disks"`
}

type IscsiDiskSpec struct {
	DiskName string `json:"diskname"`
	DiskSize string `json:"disksize"`
}

type IscsiHostSpec struct {
	HostName string `json:"hostName"`

	Username string         `json:"userName"`
	Password string         `json:"password"`
	Luns     []IscsiLunSpec `json:"luns"`
}

type IscsiLunSpec struct {
	PoolName string `json:"poolname"`
	DiskName string `json:"diskname"`
}

// IscsigatewayStatus defines the observed state of Iscsigateway
type IscsigatewayStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ServerGroup string `json:"serverGroup"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Iscsigateway is the Schema for the iscsigateways API
type Iscsigateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IscsigatewaySpec   `json:"spec,omitempty"`
	Status IscsigatewayStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IscsigatewayList contains a list of Iscsigateway
type IscsigatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Iscsigateway `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Iscsigateway{}, &IscsigatewayList{})
}
