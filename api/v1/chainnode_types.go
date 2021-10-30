/*
Copyright 2021 buaa-cita.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ChainNodeSpec defines the desired state of ChainNode
type ChainNodeSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Better to use string instead of int

	// Namespace of the ChainConfig used
	// Do not change this field
	ConfigNamespace string `json:"config_namespace,omitempty"`

	// Name of the ChainConfig used
	// Do not change this field
	ConfigName string `json:"config_name,omitempty"`

	// Can be changed
	// Can be empty
	LogLevel string `json:"log_level,omitempty"`

	// Network secret
	NetworkKey string `json:"network_key,omitempty"`

	// Node Address
	// Not allowed to change, any change will ignored
	NodeAddress string `json:"node_address,omitempty"`

	// Node Key
	// Not allowed to change, any change will ignored
	NodeKey string `json:"node_key,omitempty"`

	// Kms password, change policy not set yet
	KmsPassword string `json:"kms_pwd,omitempty"`

	// Weither or not follow when chainConfig has updated
	// Can be updated
	// Choose from AutoUpdate, NoUpdate
	// Default to AutoUpdate
	// Can be empty
	UpdatePoilcy string `json:"update_policy,omitempty"`

	// The cluster this chainnode belongs to
	Cluster string `json:"cluster,omitempty"`

	// SelfPort string `json:"self_port"`
	// SelfHost string `json:"self_host"`
}

// ChainNodeStatus defines the observed state of ChainNode
type ChainNodeStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Back up some fields to detect change
	Nodes    []NodeID `json:"nodes,omitempty"`
	LogLevel string   `json:"log_level,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ChainNode is the Schema for the chainnodes API
type ChainNode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChainNodeSpec   `json:"spec,omitempty"`
	Status ChainNodeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ChainNodeList contains a list of ChainNode
type ChainNodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ChainNode `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ChainNode{}, &ChainNodeList{})
}
