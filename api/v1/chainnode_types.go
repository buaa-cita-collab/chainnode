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
	ConfigName string `json:"configname,omitempty"`
	// better to use string instead of int
	NodeID         string `json:"nodeid,omitempty"`
	ChainConfig    string `json:"chainconfig,omitempty"`
	KmsPassword    string `json:"kms_password,omitempty"`
	NetworkKey     string `json:"network_key,omitempty"`
	IsStdOut       string `json:"is_std_out,omitempty"`
	LogLevel       string `json:"log_level,omitempty"`
	ServicePort    string `json:"service_port,omitempty"`
	ServiceEipName string `json:"service_eipName,omitempty"`
}

// ChainNodeStatus defines the observed state of ChainNode
type ChainNodeStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ChainName string `json:"chainname,omitempty"`
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
