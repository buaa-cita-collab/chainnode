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

// ChainConfigSpec defines the desired state of ChainConfig
type ChainConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// This field is deleted, use ChainNode.ObjectMeta.Name as chainname instead
	// ChainName       string   `json:"chain_name,omitempty"`

	// init_sys_config.toml: validators
	Authorities []string `json:"authorities,omitempty"`

	// init_sys_config.toml: admin
	SuperAdmin string `json:"super_admin,omitempty"`

	// init_sys_config.toml: block_interval
	// Not allowed to change, any change will ignored
	// Can be empty, default to 3
	BlockInterval string `json:"block_interval,omitempty"`

	// genesis.toml: timestamp
	// Not allowed to change, any change will ignored
	// Can be empty, default to unix_now*1000
	Timestamp string `json:"timestamp,omitempty"`

	// genesis.toml: prevhash
	// Not allowed to change, any change will ignored
	// Can be empty, default to  "0x00000000000000000
	// 00000000000000000000000000000000000000000000000"
	PrevHash string `json:"prevhash,omitempty"`

	// network-config.toml: enable_tls
	// Not allowed to change, any change will ignored
	// Can be empty, default to true
	// This is a string,
	EnableTLS string `json:"enable_tls,omitempty"`

	// network-config.toml: peers
	// Can be changed
	Nodes []string `json:"nodes,omitempty"`

	// Not allowed to change, any change will ignored
	NetworkImage string `json:"network_image,omitempty"`

	// Not allowed to change, any change will ignored
	ConsensusImage string `json:"consensus_image,omitempty"`

	// Not allowed to change, any change will ignored
	ExecutorImage string `json:"executor_image,omitempty"`

	// Not allowed to change, any change will ignored
	StorageImage string `json:"storage_image,omitempty"`

	// Not allowed to change, any change will ignored
	ControllerImage string `json:"controller_image,omitempty"`

	// Not allowed to change, any change will ignored
	KmsImage string `json:"kms_image,omitempty"`
}

// ChainConfigStatus defines the observed state of ChainConfig
type ChainConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ChainNode reconcile will only be called when
	// ChainConfig is ready
	Ready bool `json:"ready,omitempty"`

	// Backing up some fields to make sure it is can not be changed once set
	Authorities     []string `json:"authorities,omitempty"`
	SuperAdmin      string   `json:"super_admin,omitempty"`
	BlockInterval   string   `json:"block_interval,omitempty"`
	Timestamp       string   `json:"timestamp,omitempty"`
	PrevHash        string   `json:"prevhash,omitempty"`
	EnableTLS       string   `json:"enable_tls,omitempty"`
	NetworkImage    string   `json:"network_image,omitempty"`
	ConsensusImage  string   `json:"consensus_image,omitempty"`
	ExecutorImage   string   `json:"executor_image,omitempty"`
	StorageImage    string   `json:"storage_image,omitempty"`
	ControllerImage string   `json:"controller_image,omitempty"`
	KmsImage        string   `json:"kms_image,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ChainConfig is the Schema for the chainconfigs API
type ChainConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChainConfigSpec   `json:"spec,omitempty"`
	Status ChainConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ChainConfigList contains a list of ChainConfig
type ChainConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ChainConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ChainConfig{}, &ChainConfigList{})
}
