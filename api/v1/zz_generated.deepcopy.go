//go:build !ignore_autogenerated
// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Condition) DeepCopyInto(out *Condition) {
	*out = *in
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Condition.
func (in *Condition) DeepCopy() *Condition {
	if in == nil {
		return nil
	}
	out := new(Condition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in Conditions) DeepCopyInto(out *Conditions) {
	{
		in := &in
		*out = make(Conditions, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Conditions.
func (in Conditions) DeepCopy() Conditions {
	if in == nil {
		return nil
	}
	out := new(Conditions)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CrossVersionObjectReference) DeepCopyInto(out *CrossVersionObjectReference) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CrossVersionObjectReference.
func (in *CrossVersionObjectReference) DeepCopy() *CrossVersionObjectReference {
	if in == nil {
		return nil
	}
	out := new(CrossVersionObjectReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GroupVersionKindResource) DeepCopyInto(out *GroupVersionKindResource) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GroupVersionKindResource.
func (in *GroupVersionKindResource) DeepCopy() *GroupVersionKindResource {
	if in == nil {
		return nil
	}
	out := new(GroupVersionKindResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MetricTarget) DeepCopyInto(out *MetricTarget) {
	*out = *in
	if in.Value != nil {
		in, out := &in.Value, &out.Value
		x := (*in).DeepCopy()
		*out = &x
	}
	if in.AverageValue != nil {
		in, out := &in.AverageValue, &out.AverageValue
		x := (*in).DeepCopy()
		*out = &x
	}
	if in.AverageUtilization != nil {
		in, out := &in.AverageUtilization, &out.AverageUtilization
		*out = new(int32)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetricTarget.
func (in *MetricTarget) DeepCopy() *MetricTarget {
	if in == nil {
		return nil
	}
	out := new(MetricTarget)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplicaAutoscaler) DeepCopyInto(out *ReplicaAutoscaler) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplicaAutoscaler.
func (in *ReplicaAutoscaler) DeepCopy() *ReplicaAutoscaler {
	if in == nil {
		return nil
	}
	out := new(ReplicaAutoscaler)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ReplicaAutoscaler) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplicaAutoscalerList) DeepCopyInto(out *ReplicaAutoscalerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ReplicaAutoscaler, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplicaAutoscalerList.
func (in *ReplicaAutoscalerList) DeepCopy() *ReplicaAutoscalerList {
	if in == nil {
		return nil
	}
	out := new(ReplicaAutoscalerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ReplicaAutoscalerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplicaAutoscalerSpec) DeepCopyInto(out *ReplicaAutoscalerSpec) {
	*out = *in
	if in.Replicator != nil {
		in, out := &in.Replicator, &out.Replicator
		*out = new(string)
		**out = **in
	}
	out.ScaleTargetRef = in.ScaleTargetRef
	if in.MinReplicas != nil {
		in, out := &in.MinReplicas, &out.MinReplicas
		*out = new(int32)
		**out = **in
	}
	if in.Targets != nil {
		in, out := &in.Targets, &out.Targets
		*out = make([]ReplicaAutoscalerTarget, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Strategy != nil {
		in, out := &in.Strategy, &out.Strategy
		*out = new(ReplicaAutoscalerStrategy)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplicaAutoscalerSpec.
func (in *ReplicaAutoscalerSpec) DeepCopy() *ReplicaAutoscalerSpec {
	if in == nil {
		return nil
	}
	out := new(ReplicaAutoscalerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplicaAutoscalerStatus) DeepCopyInto(out *ReplicaAutoscalerStatus) {
	*out = *in
	if in.ObservedGeneration != nil {
		in, out := &in.ObservedGeneration, &out.ObservedGeneration
		*out = new(int64)
		**out = **in
	}
	if in.LastScaleTime != nil {
		in, out := &in.LastScaleTime, &out.LastScaleTime
		*out = (*in).DeepCopy()
	}
	if in.Targets != nil {
		in, out := &in.Targets, &out.Targets
		*out = make([]TargetStatus, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(Conditions, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplicaAutoscalerStatus.
func (in *ReplicaAutoscalerStatus) DeepCopy() *ReplicaAutoscalerStatus {
	if in == nil {
		return nil
	}
	out := new(ReplicaAutoscalerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplicaAutoscalerStrategy) DeepCopyInto(out *ReplicaAutoscalerStrategy) {
	*out = *in
	if in.PanicWindowSeconds != nil {
		in, out := &in.PanicWindowSeconds, &out.PanicWindowSeconds
		*out = new(int32)
		**out = **in
	}
	if in.PanicThreshold != nil {
		in, out := &in.PanicThreshold, &out.PanicThreshold
		x := (*in).DeepCopy()
		*out = &x
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplicaAutoscalerStrategy.
func (in *ReplicaAutoscalerStrategy) DeepCopy() *ReplicaAutoscalerStrategy {
	if in == nil {
		return nil
	}
	out := new(ReplicaAutoscalerStrategy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplicaAutoscalerTarget) DeepCopyInto(out *ReplicaAutoscalerTarget) {
	*out = *in
	in.Settings.DeepCopyInto(&out.Settings)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplicaAutoscalerTarget.
func (in *ReplicaAutoscalerTarget) DeepCopy() *ReplicaAutoscalerTarget {
	if in == nil {
		return nil
	}
	out := new(ReplicaAutoscalerTarget)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ReplicaPatch) DeepCopyInto(out *ReplicaPatch) {
	*out = *in
	if in.RetentionSeconds != nil {
		in, out := &in.RetentionSeconds, &out.RetentionSeconds
		*out = new(int64)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplicaPatch.
func (in *ReplicaPatch) DeepCopy() *ReplicaPatch {
	if in == nil {
		return nil
	}
	out := new(ReplicaPatch)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in ReplicaPatches) DeepCopyInto(out *ReplicaPatches) {
	{
		in := &in
		*out = make(ReplicaPatches, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ReplicaPatches.
func (in ReplicaPatches) DeepCopy() ReplicaPatches {
	if in == nil {
		return nil
	}
	out := new(ReplicaPatches)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ScheduleTargetSettings) DeepCopyInto(out *ScheduleTargetSettings) {
	*out = *in
	if in.Settings != nil {
		in, out := &in.Settings, &out.Settings
		*out = new(runtime.RawExtension)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ScheduleTargetSettings.
func (in *ScheduleTargetSettings) DeepCopy() *ScheduleTargetSettings {
	if in == nil {
		return nil
	}
	out := new(ScheduleTargetSettings)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TargetSettings) DeepCopyInto(out *TargetSettings) {
	*out = *in
	if in.Default != nil {
		in, out := &in.Default, &out.Default
		*out = new(runtime.RawExtension)
		(*in).DeepCopyInto(*out)
	}
	if in.Schedules != nil {
		in, out := &in.Schedules, &out.Schedules
		*out = make([]ScheduleTargetSettings, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetSettings.
func (in *TargetSettings) DeepCopy() *TargetSettings {
	if in == nil {
		return nil
	}
	out := new(TargetSettings)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TargetStatus) DeepCopyInto(out *TargetStatus) {
	*out = *in
	in.Metric.DeepCopyInto(&out.Metric)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetStatus.
func (in *TargetStatus) DeepCopy() *TargetStatus {
	if in == nil {
		return nil
	}
	out := new(TargetStatus)
	in.DeepCopyInto(out)
	return out
}
