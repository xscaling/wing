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
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// ReplicaAutoscalerSpec defines the desired state of ReplicaAutoscaler
type ReplicaAutoscalerSpec struct {
	// Replicator specified which replicator used for aggregating scalers output and
	// make final scaling decision
	// +optional
	Replicator *string `json:"replicator,omitempty"`
	// ScaleTargetRef points to the target resource to scale, and is used to the pods for which metrics
	// should be collected, as well as to actually change the replica count.
	ScaleTargetRef autoscalingv2.CrossVersionObjectReference `json:"scaleTargetRef"`
	// minReplicas is the lower limit for the number of replicas to which the autoscaler can scale down.
	// If `minReplicas` is nil then the replicas will be set as `maxReplicas` without autoscaling.
	// +optional
	MinReplicas *int32 `json:"minReplicas,omitempty"`
	// maxReplicas is the upper limit for the number of replicas to which the autoscaler can scale up.
	// It cannot be less that minReplicas(if it has been set).
	MaxReplicas int32 `json:"maxReplicas"`

	// Targets contain various scaling metrics and thresholds used for calculating the final desired replicas.
	// +kubebuilder:validation:Optional
	// +optional
	Targets []ReplicaAutoscalerTarget `json:"targets,omitempty"`

	// TODO(@oif): Advance scaling strategy
}

// ReplicaAutoscalerTarget defines metric provider and target threshold
type ReplicaAutoscalerTarget struct {
	// metric indicates which metric provider should present utilization stat.
	Metric string `json:"metric"`
	// metricType represents whether the metric type is Utilization, Value, or AverageValue
	MetricType autoscalingv2.MetricTargetType `json:"metricType,omitempty"`

	Settings TargetSettings `json:"settings"`
}

type TargetSettings struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	Default *runtime.RawExtension `json:"default"`

	// +kubebuilder:validation:Optional
	// +optional
	Schedules []ScheduleTargetSettings `json:"schedules,omitempty"`
}

type ScheduleTargetSettings struct {
	Timezone string `json:"timezone"`
	Start    string `json:"start"`
	End      string `json:"end"`
	// +kubebuilder:pruning:PreserveUnknownFields
	Settings *runtime.RawExtension `json:"settings"`
}

// ReplicaAutoscalerStatus defines the observed state of ReplicaAutoscaler
type ReplicaAutoscalerStatus struct {
	// observedGeneration is the most recent generation observed by this autoscaler.
	// +optional
	ObservedGeneration *int64 `json:"observedGeneration,omitempty"`

	// lastScaleTime is the last time the ReplicaAutoscaler scaled,
	// used by the autoscaler to control how often the replicas is changed.
	// +optional
	LastScaleTime *metav1.Time `json:"lastScaleTime,omitempty"`

	// currentReplicas is current replicas of object managed by this autoscaler,
	// as last seen by the autoscaler.
	// +optional
	CurrentReplicas int32 `json:"currentReplicas,omitempty"`

	// desiredReplicas is the desired replicas of object managed by this autoscaler,
	// as last calculated by the autoscaler.
	DesiredReplicas int32 `json:"desiredReplicas"`

	// targets indicates state of targets used by this autoscaler
	// +listType=atomic
	// +patchMergeKey=target
	// +patchStrategy=replace
	// +optional
	Targets []TargetStatus `json:"targets,omitempty" patchStrategy:"replace" patchMergeKey:"target"`

	// conditions is the set of conditions required for this autoscaler to scale its target,
	// and indicates whether or not those conditions are met.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []ReplicaAutoscalerCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" listType:"map"`
}

type ReplicaAutoscalerCondition struct {
	Type string `json:"type"`
}

// TargetStatus represents the running status of scaling target
type TargetStatus struct {
	// Target indicates the source of status
	Target string `json:"target"`
	// Metric holds key values of scaler which used for calculate desired replicas
	Metric autoscalingv2.MetricTarget ` json:"metric"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Reference",type=string,JSONPath=`.spec.scaleTargetRef.name`
//+kubebuilder:printcolumn:name="Min",type=string,JSONPath=`.spec.minReplicas`
//+kubebuilder:printcolumn:name="Max",type=string,JSONPath=`.spec.maxReplicas`
//+kubebuilder:printcolumn:name="Replicas",type=string,JSONPath=`.status.currentReplicas`
//+kubebuilder:printcolumn:name="Targets",type=string,JSONPath=`.status.targets[*].target`
//+kubebuilder:printcolumn:name="LastScaleTime",type=string,JSONPath=`.status.lastScaleTime`

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
