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

package controllers

import (
	"context"
	"fmt"
	"time"

	wingv1 "github.com/xscaling/wing/api/v1"
	"github.com/xscaling/wing/core/engine"
	"github.com/xscaling/wing/core/scheduling"
	"github.com/xscaling/wing/utils"

	"github.com/go-logr/logr"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaultScalingColdDown = time.Second * 15

	DefaultReplicator = "simple"
)

// TODO(@oif): Status reporting and event recording
func (r *ReplicaAutoscalerReconciler) reconcile(logger logr.Logger, autoscaler *wingv1.ReplicaAutoscaler) (requeue bool, err error) {
	if autoscaler.DeletionTimestamp != nil {
		logger.V(2).Info("Found terminating autoscaler turn finalizer")
		return r.finalizeAutoscaler(logger, autoscaler)
	}
	// Check is target ref is a scalable object
	if autoscaler.Spec.ScaleTargetRef.Name == "" || autoscaler.Spec.ScaleTargetRef.Kind == "" {
		logger.Info("autoscaler.Spec.ScaleTargetRef.Name or autoscaler.Spec.ScaleTargetRef.Kind missing")
		return false, nil
	}
	gvkr, err := utils.ParseGVKR(r.restMapper, autoscaler.Spec.ScaleTargetRef.APIVersion, autoscaler.Spec.ScaleTargetRef.Kind)
	if err != nil {
		return false, err
	}
	scale, err := r.isTargetScalable(gvkr, autoscaler.Namespace, autoscaler.Spec.ScaleTargetRef.Name)
	if err != nil {
		logger.Error(err, "Target(%s) is unscalable", gvkr.GVKString())
		return false, nil
	}
	observingAutoscaler := autoscaler.DeepCopy()

	autoscaler.Status.ObservedGeneration = &autoscaler.Generation
	autoscaler.Status.CurrentReplicas = scale.Status.Replicas
	// TODO(@oif): Init various

	// A static replicas setting
	if autoscaler.Spec.MinReplicas == nil {
		logger.V(2).Info("Setting static replicas")
		err = r.scaleReplicas(logger, autoscaler, gvkr, scale, autoscaler.Spec.MaxReplicas)
	} else {
		// Working on autoscaling flow
		requeue, err = r.reconcileAutoscaling(logger, autoscaler, gvkr, scale)
	}
	if err != nil {
		autoscaler.Status.Conditions = wingv1.SetCondition(autoscaler.Status.Conditions, wingv1.Condition{
			Type:    wingv1.ConditionReady,
			Status:  metav1.ConditionFalse,
			Reason:  "FailedToScale",
			Message: fmt.Sprintf("Failed to scale target: %s", err),
		})
		logger.Error(err, "Failed to scale target")
		return requeue, err
	}

	autoscaler.Status.Conditions = wingv1.SetCondition(autoscaler.Status.Conditions, wingv1.Condition{
		Type:   wingv1.ConditionReady,
		Status: metav1.ConditionTrue,
	})

	isEqual := utils.DeepEqual(autoscaler.Status, observingAutoscaler.Status)
	if !isEqual {
		logger.V(4).Info("Updating ReplicaAutoscaler status")
		patch := runtimeclient.MergeFrom(observingAutoscaler.DeepCopy())
		observingAutoscaler.Status = autoscaler.Status
		err = r.Client.Status().Patch(context.TODO(), observingAutoscaler, patch)
		if err != nil {
			logger.Error(err, "Failed to update autoscaler status")
			return true, err
		}
	}
	return true, nil
}

func (r *ReplicaAutoscalerReconciler) isTargetScalable(gvkr wingv1.GroupVersionKindResource, namespace, name string) (*autoscalingv1.Scale, error) {
	targetGR := gvkr.GroupResource()

	scale, err := r.scaleClient.Scales(namespace).Get(context.TODO(), targetGR, name, metav1.GetOptions{})
	if err != nil {
		// maybe scale target not exists or maybe not scalable, check target ref existence
		// whatever this target is regarded as unscalable anyway.
		err = r.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, utils.GetGVKRUnstructured(gvkr))
		if errors.IsNotFound(err) {
			// Target not exists
			return nil, ErrRefTargetIsNotExists
		} else if err == nil {
			// Found target, so it is not scalable
			return nil, ErrRefTargetIsNotScalable
		}
		return nil, err
	}

	knownScalable, _ := utils.GetGroupResourceKnownScalable(targetGR.String())
	if !knownScalable {
		// Set new known group resource as scalable
		utils.SetGroupResourceKnownScalable(targetGR.String(), true)
	}

	return scale, nil
}

func (r *ReplicaAutoscalerReconciler) scaleReplicas(logger logr.Logger, autoscaler *wingv1.ReplicaAutoscaler, gvkr wingv1.GroupVersionKindResource, scale *autoscalingv1.Scale, desiredReplicas int32) error {
	autoscaler.Status.DesiredReplicas = desiredReplicas

	if scale.Spec.Replicas == desiredReplicas {
		logger.V(8).Info("Current replicas is expected, nothing todo")
		return nil
	}
	logger.V(2).Info("Scaling replicas", "currentReplicas", scale.Spec.Replicas, "desireReplicas", desiredReplicas)
	scale.Spec.Replicas = desiredReplicas
	_, err := r.scaleClient.Scales(scale.Namespace).Update(context.TODO(), gvkr.GroupResource(), scale, metav1.UpdateOptions{})
	if err != nil {
		logger.Error(err, "Failed to scale replicas")
		return err
	}

	now := metav1.NewTime(time.Now())
	autoscaler.Status.LastScaleTime = &now
	return nil
}

func (r *ReplicaAutoscalerReconciler) reconcileAutoscaling(logger logr.Logger, autoscaler *wingv1.ReplicaAutoscaler, gvkr wingv1.GroupVersionKindResource, scale *autoscalingv1.Scale) (requeue bool, err error) {
	scaledObjectSelector, err := labels.Parse(scale.Status.Selector)
	if err != nil {
		logger.Error(err, "couldn't convert selector into a corresponding target selector object")
		return true, err
	}

	// Checking cold-down
	if autoscaler.Status.LastScaleTime != nil && time.Since(autoscaler.Status.LastScaleTime.Time) < DefaultScalingColdDown {
		logger.V(8).Info("Still in scaling cold-down period")
		return true, nil
	}

	now := time.Now()

	replicatorContext := engine.ReplicatorContext{
		Autoscaler:    autoscaler,
		Scale:         scale,
		ScalersOutput: make(map[string]engine.ScalerOutput),
	}

	for _, target := range autoscaler.Spec.Targets {
		scheduledTargetSettings, err := scheduling.GetScheduledSettingsRaw(now, target.Settings)
		if err != nil {
			logger.Error(err, "Failed to get scheduled target settings", "targetMetric", target.Metric)
			return true, err
		}
		logger.V(8).Info("Get scheduled target settings", "settings", string(scheduledTargetSettings), "metric", target.Metric)

		scaler, ok := r.Engine.GetScaler(target.Metric)
		if !ok {
			return false, fmt.Errorf("scaler `%s` not exists for target", target.Metric)
		}
		// Getting desired replicas from scaler
		scalerOutput, err := scaler.Get(engine.ScalerContext{
			InformerFactory:      r.Engine.InformerFactory,
			RawSettings:          scheduledTargetSettings,
			Namespace:            autoscaler.Namespace,
			ScaledObjectSelector: scaledObjectSelector,
			CurrentReplicas:      scale.Spec.Replicas,
			AutoscalerStatus:     &autoscaler.Status,
		})
		if err != nil {
			logger.Error(err, "Failed to get result from scaler", "scaler", target.Metric)
			return true, err
		}
		replicatorContext.ScalersOutput[target.Metric] = *scalerOutput
	}

	selectedReplicator := DefaultReplicator
	if autoscaler.Spec.Replicator != nil {
		selectedReplicator = *autoscaler.Spec.Replicator
	}

	replicator, ok := r.Engine.GetReplicator(selectedReplicator)
	if !ok {
		err = fmt.Errorf("replicator `%s` not registered", selectedReplicator)
		logger.Error(err, "Replicator not found")
		return false, err
	}

	desireReplicas, err := replicator.GetDesiredReplicas(replicatorContext)
	if err != nil {
		return false, fmt.Errorf("failed to get desired replicas from `%s`: %v", selectedReplicator, err)
	}
	logger.V(4).Info("Replicator calculated desired replicas", "desiredReplicas", desireReplicas)

	// Final normalize desired replicas
	var (
		scalingLimitedReason = ""
	)
	if desireReplicas > autoscaler.Spec.MaxReplicas {
		desireReplicas = autoscaler.Spec.MaxReplicas
		scalingLimitedReason = "ReachMaxReplicas"
	} else if desireReplicas < *autoscaler.Spec.MinReplicas {
		desireReplicas = *autoscaler.Spec.MinReplicas
		scalingLimitedReason = "ReachMinimalReplicas"
	}

	if scalingLimitedReason != "" {
		autoscaler.Status.Conditions = wingv1.SetCondition(autoscaler.Status.Conditions, wingv1.Condition{
			Type:   wingv1.ConditionScaleLimited,
			Status: metav1.ConditionTrue,
			Reason: scalingLimitedReason,
		})
	} else {
		autoscaler.Status.Conditions = wingv1.SetCondition(autoscaler.Status.Conditions, wingv1.Condition{
			Type:   wingv1.ConditionScaleLimited,
			Status: metav1.ConditionFalse,
		})
	}

	return true, r.scaleReplicas(logger, autoscaler, gvkr, scale, desireReplicas)
}
