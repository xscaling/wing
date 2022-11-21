/*
Copyright 2022 xScaling.

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

// ReplicaAutoscalerSpec defines the desired state of ReplicaAutoscaler
type ReplicaAutoscalerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of ReplicaAutoscaler. Edit replicaautoscaler_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// ReplicaAutoscalerStatus defines the observed state of ReplicaAutoscaler
type ReplicaAutoscalerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ReplicaAutoscaler is the Schema for the replicaautoscalers API
type ReplicaAutoscaler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReplicaAutoscalerSpec   `json:"spec,omitempty"`
	Status ReplicaAutoscalerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ReplicaAutoscalerList contains a list of ReplicaAutoscaler
type ReplicaAutoscalerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ReplicaAutoscaler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ReplicaAutoscaler{}, &ReplicaAutoscalerList{})
}
