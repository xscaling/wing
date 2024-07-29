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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// MetricTargetType specifies the type of metric being targeted, and should be either
// "Value", "AverageValue", or "Utilization"
type MetricTargetType string

const (
	// UtilizationMetricType declares a MetricTarget is an AverageUtilization value
	UtilizationMetricType MetricTargetType = "Utilization"
	// ValueMetricType declares a MetricTarget is a raw value
	ValueMetricType MetricTargetType = "Value"
	// AverageValueMetricType declares a MetricTarget is an
	AverageValueMetricType MetricTargetType = "AverageValue"
)

// MetricTarget defines the target value, average value, or average utilization of a specific metric
type MetricTarget struct {
	// type represents whether the metric type is Utilization, Value, or AverageValue
	Type MetricTargetType `json:"type"`
	// value is the target value of the metric (as a quantity).
	// +optional
	Value *resource.Quantity `json:"value,omitempty"`
	// averageValue is the target value of the average of the
	// metric across all relevant pods (as a quantity)
	// +optional
	AverageValue *resource.Quantity `json:"averageValue,omitempty"`
	// averageUtilization is the target value of the average of the
	// resource metric across all relevant pods, represented as a percentage of
	// the requested value of the resource for the pods.
	// Currently only valid for Resource metric source type
	// +optional
	AverageUtilization *int32 `json:"averageUtilization,omitempty"`
}

// ReplicaAutoscalerSpec defines the desired state of ReplicaAutoscaler
type ReplicaAutoscalerSpec struct {
	// Replicator specified which replicator used for aggregating scalers output and
	// make final scaling decision
	// +optional
	Replicator *string `json:"replicator,omitempty"`

	// ReplicatorSettings is the configuration of replicator
	// +optional
	ReplicatorSettings *runtime.RawExtension `json:"replicatorSettings,omitempty"`

	// ScaleTargetRef points to the target resource to scale, and is used to the pods for which metrics
	// should be collected, as well as to actually change the replica count.
	ScaleTargetRef CrossVersionObjectReference `json:"scaleTargetRef"`
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

	// Strategy decides how to make scaling decision
	// +kubebuilder:validation:Optional
	// +optional
	Strategy *ReplicaAutoscalerStrategy `json:"strategy,omitempty"`

	// +optional
	Exhaust *Exhaust `json:"exhaust,omitempty" yaml:"exhaust,omitempty"`
}

type ReplicaAutoscalerStrategy struct {
	// Panic Mode
	// Panic Windows in seconds indicates how long the panic mode will last after startup.
	PanicWindowSeconds *int32 `json:"panicWindowSeconds,omitempty"`
	// Panic Threshold indicates the threshold of replicas to trigger panic mode.
	// Value: 1.1 - 10.0 e.g 1.1 means the desired replicas is 110% of the current replicas.
	PanicThreshold *resource.Quantity `json:"panicThreshold,omitempty"`
}

// ReplicaAutoscalerTarget defines metric provider and target threshold
type ReplicaAutoscalerTarget struct {
	// metric indicates which metric provider should present utilization stat.
	Metric string `json:"metric"`
	// metricType represents whether the metric type is Utilization, Value, or AverageValue
	MetricType MetricTargetType `json:"metricType,omitempty"`

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

// Exhaust is the settings for exhaust checking
type Exhaust struct {
	// Type of exhaust mode, only `Pending` is currently supported.
	Type ExhaustType `json:"type,omitempty" yaml:"type,omitempty"`

	// Pending is the details for exhaust check config.
	// If oldest pending pod life is not shorter than timeout,
	// and percentage or number of pending pod(s) is not smaller than threshold,
	// then the exhaust mode will be triggered.
	Pending *ExhaustPending `json:"pending,omitempty" yaml:"pending,omitempty"`
}

type ExhaustType string

const (
	ExhaustOnPending ExhaustType = "Pending"
)

type ExhaustPending struct {
	Threshold      intstr.IntOrString `json:"threshold" yaml:"threshold"`
	TimeoutSeconds int32              `json:"timeout" yaml:"timeout"`
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
	// +optional
	Conditions Conditions `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" listType:"map"`
}

// TargetStatus represents the running status of scaling target
type TargetStatus struct {
	// Target indicates the source of status
	Target string `json:"target"`
	// Scaler indicates which scaler used for calculating desired replicas
	Scaler string `json:"scaler"`
	// Target desired replicas calculated by giving settings
	DesiredReplicas int32 `json:"desireReplicas"`
	// Metric holds key values of scaler which used for calculate desired replicas
	Metric MetricTarget ` json:"metric"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=ra;wra
//+kubebuilder:printcolumn:name="Reference",type=string,JSONPath=`.spec.scaleTargetRef.name`
//+kubebuilder:printcolumn:name="Min",type=string,JSONPath=`.spec.minReplicas`
//+kubebuilder:printcolumn:name="Max",type=string,JSONPath=`.spec.maxReplicas`
//+kubebuilder:printcolumn:name="Replicas",type=string,JSONPath=`.status.currentReplicas`
//+kubebuilder:printcolumn:name="Scalers",type=string,JSONPath=`.status.targets[*].scaler`
//+kubebuilder:printcolumn:name="LastScaleTime",type=string,JSONPath=`.status.lastScaleTime`
//+kubebuilder:printcolumn:name="ReplicaPatched",type=string,JSONPath=`.status.conditions[?(@.type=="ReplicaPatched")].status`
//+kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
//+kubebuilder:printcolumn:name="PanicMode",type="string",JSONPath=".status.conditions[?(@.type==\"PanicMode\")].status"

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

const (
	ReplicaPatchesAnnotation = "wing.xscaling.dev/replica-patches"
)

// Advanced feature: Replica Patch, dynamic controls the scaling range of the replica autoscaler.
// implemented in ReplicaAutoscaler annotation `wing.xscaling.dev/replica-patches`.
// As stored in annotation, it's a json string of []ReplicaPatch and it's mutable for controller rather than spec.
// WARNING: If it's a static replica autoscaler, this patch will be ignored.
type ReplicaPatch struct {
	// Specified the working timezone of the patch.
	Timezone string `json:"timezone"`
	// Start and End could be a cron expression or a time string.
	// But can't be mixed.
	Start string `json:"start"`
	End   string `json:"end"`
	// When using specified time range, retention seconds is required.
	// It's the time duration of the patch will be hold for after end time, then will be purge.
	// Zero means will be deleted once found out of the time range.
	RetentionSeconds *int64 `json:"retentionSeconds,omitempty"`
	// MinReplicas is the lower limit for the number of replicas to which the autoscaler can scale down.
	MinReplicas int32 `json:"minReplicas"`
	// MaxReplicas is the upper limit for the number of replicas to which the autoscaler can scale up.
	MaxReplicas int32 `json:"maxReplicas"`
}

type ReplicaPatches []ReplicaPatch

func init() {
	SchemeBuilder.Register(&ReplicaAutoscaler{}, &ReplicaAutoscalerList{})
}
