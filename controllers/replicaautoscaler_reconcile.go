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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	NotRequeue                = time.Duration(0)
	DefaultRequeueDelay       = RequeueDelayOnNormalState
	RequeueDelayOnErrorState  = time.Second * 30
	RequeueDelayOnNormalState = time.Second * 60
	RequeueDelayOnPanicState  = time.Second * 15

	DefaultScalingColdDown = time.Second * 30

	DefaultReplicator = "simple"
)

func (r *ReplicaAutoscalerReconciler) reconcile(logger logr.Logger,
	autoscaler *wingv1.ReplicaAutoscaler) (requeueDelay time.Duration) {
	if autoscaler.DeletionTimestamp != nil {
		logger.V(2).Info("Found terminating autoscaler turn finalizer")
		return r.finalizeAutoscaler(logger, autoscaler)
	}

	gvkr, scale, err := r.getScaleTarget(logger, autoscaler)
	if err != nil {
		logger.Error(err, "Unable to get scale target")
		autoscaler.Status.Conditions = wingv1.SetCondition(autoscaler.Status.Conditions, wingv1.Condition{
			Type:    wingv1.ConditionReady,
			Reason:  "FailedToGetScaleTarget",
			Message: fmt.Sprintf("Failed to get scale target: %v", err),
			Status:  metav1.ConditionFalse,
		})

		return NotRequeue
	}

	requeueDelay, err = r.updateExhaustedAutoscaler(logger, autoscaler, scale)
	if err != nil {
		logger.Error(err, "Failed to update exhausted autoscaler")
		return RequeueDelayOnErrorState
	}

	autoscaler.Status.ObservedGeneration = &autoscaler.Generation
	autoscaler.Status.CurrentReplicas = scale.Status.Replicas
	// TODO(@oif): Init various

	// A static replicas setting
	if autoscaler.Spec.MinReplicas == nil {
		logger.V(2).Info("Setting static replicas")
		if err = r.scaleReplicas(logger, autoscaler, gvkr,
			scale.DeepCopy(), autoscaler.Spec.MaxReplicas); err != nil {
			requeueDelay = RequeueDelayOnErrorState
		}
	} else {
		// Working on autoscaling flow
		requeueDelay = r.reconcileAutoscaling(logger, autoscaler, gvkr, scale)
	}

	autoscaler.Status.Conditions = wingv1.SetCondition(autoscaler.Status.Conditions, wingv1.Condition{
		Type:   wingv1.ConditionReady,
		Status: metav1.ConditionTrue,
	})
	return requeueDelay
}

func (r *ReplicaAutoscalerReconciler) getScaleTarget(logger logr.Logger,
	autoscaler *wingv1.ReplicaAutoscaler) (wingv1.GroupVersionKindResource, *autoscalingv1.Scale, error) {
	// Check is target ref is a scalable object
	if autoscaler.Spec.ScaleTargetRef.Name == "" || autoscaler.Spec.ScaleTargetRef.Kind == "" {
		logger.Info("autoscaler.Spec.ScaleTargetRef.Name or autoscaler.Spec.ScaleTargetRef.Kind missing")
		return wingv1.GroupVersionKindResource{}, nil, fmt.Errorf(
			"autoscaler.Spec.ScaleTargetRef.Name or autoscaler.Spec.ScaleTargetRef.Kind missing")
	}
	gvkr, err := utils.ParseGVKR(r.restMapper,
		autoscaler.Spec.ScaleTargetRef.APIVersion, autoscaler.Spec.ScaleTargetRef.Kind)
	if err != nil {
		logger.Error(err, "Failed to parse GVKR")
		return wingv1.GroupVersionKindResource{}, nil, err
	}
	scale, err := r.isTargetScalable(gvkr, autoscaler.Namespace, autoscaler.Spec.ScaleTargetRef.Name)
	if err != nil {
		logger.Error(err, "Target is unscalable", "target", gvkr.GVKString())
		return wingv1.GroupVersionKindResource{}, nil, err
	}
	return gvkr, scale, nil
}

func (r *ReplicaAutoscalerReconciler) isTargetScalable(gvkr wingv1.GroupVersionKindResource,
	namespace, name string) (*autoscalingv1.Scale, error) {
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

func (r *ReplicaAutoscalerReconciler) scaleReplicas(logger logr.Logger,
	autoscaler *wingv1.ReplicaAutoscaler,
	gvkr wingv1.GroupVersionKindResource, scale *autoscalingv1.Scale, desiredReplicas int32) error {
	autoscaler.Status.DesiredReplicas = desiredReplicas

	if scale.Spec.Replicas == desiredReplicas {
		logger.V(8).Info("Current replicas is expected, nothing todo")
		return nil
	}
	logger.V(2).Info("Scaling replicas",
		"currentReplicas", scale.Spec.Replicas, "desireReplicas", desiredReplicas)
	dryRunFlag := autoscaler.Annotations[wingv1.DryRunAnnotation]
	// WARNING(@oif): During wing alpha version, scaling action won't be performed by default.
	// This is to prevent any potential issues during alpha testing period.
	// This will be performed by default in next release.
	if dryRunFlag == "false" {
		logger.V(4).Info("Performing scaling action")
		scale.Spec.Replicas = desiredReplicas
		_, err := r.scaleClient.Scales(scale.Namespace).Update(
			context.TODO(), gvkr.GroupResource(), scale.DeepCopy(), metav1.UpdateOptions{})
		if err != nil {
			autoscaler.Status.Conditions = wingv1.SetCondition(autoscaler.Status.Conditions, wingv1.Condition{
				Type:    wingv1.ConditionReady,
				Status:  metav1.ConditionFalse,
				Reason:  "FailedToScale",
				Message: fmt.Sprintf("Failed to scale target: %s", err),
			})
			logger.Error(err, "Failed to scale target")
			return err
		}
	} else {
		logger.V(4).Info("Dry run scaling replicas",
			"currentReplicas", scale.Spec.Replicas, "desireReplicas", desiredReplicas)
	}

	now := metav1.NewTime(time.Now())
	autoscaler.Status.LastScaleTime = &now
	return nil
}

func (r *ReplicaAutoscalerReconciler) reconcileAutoscaling(logger logr.Logger,
	autoscaler *wingv1.ReplicaAutoscaler,
	gvkr wingv1.GroupVersionKindResource, scale *autoscalingv1.Scale) (requeueDelay time.Duration) {
	scaledObjectSelector, err := labels.Parse(scale.Status.Selector)
	if err != nil {
		logger.Error(err, "couldn't convert selector into a corresponding target selector object")
		return RequeueDelayOnErrorState
	}

	// Checking cold-down
	underPanicModeCurrently := utils.StillInPanicMode(autoscaler.Status, autoscaler.Spec.Strategy)
	if autoscaler.Status.LastScaleTime != nil &&
		time.Since(autoscaler.Status.LastScaleTime.Time) < DefaultScalingColdDown &&
		// Not in panic mode
		!underPanicModeCurrently {
		logger.V(8).Info("Still in scaling cold-down period")
		return DefaultRequeueDelay
	}

	now := time.Now()

	replicatorContext := engine.ReplicatorContext{
		Autoscaler:    autoscaler,
		Scale:         scale,
		ScalersOutput: make(map[string]engine.ScalerOutput),
	}

	var managedTargetStatus []string

	for _, target := range autoscaler.Spec.Targets {
		scalerStartAt := time.Now()
		scheduledTargetSettings, err := scheduling.GetScheduledSettingsRaw(now, target.Settings)
		if err != nil {
			logger.Error(err, "Failed to get scheduled target settings", "targetMetric", target.Metric)
			return RequeueDelayOnErrorState
		}
		logger.V(8).Info("Get scheduled target settings",
			"settings", string(scheduledTargetSettings), "metric", target.Metric)

		scaler, ok := r.Engine.GetScaler(target.Metric)
		if !ok {
			autoscaler.Status.Conditions = wingv1.SetCondition(autoscaler.Status.Conditions, wingv1.Condition{
				Type:    wingv1.ConditionReady,
				Status:  metav1.ConditionFalse,
				Reason:  "ScalerNotExists",
				Message: fmt.Sprintf("Scaler `%s` not exists for target", target.Metric),
			})
			return DefaultRequeueDelay
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
			return RequeueDelayOnErrorState
		}
		replicatorContext.ScalersOutput[target.Metric] = *scalerOutput
		managedTargetStatus = append(managedTargetStatus, scalerOutput.ManagedTargetStatus...)
		metricPluginElapsed.WithLabelValues(autoscaler.Namespace, autoscaler.Name, target.Metric, "scaler").Add(time.Since(scalerStartAt).Seconds())
	}

	// Purge unused scaler targetStatus
	utils.PurgeTargetStatus(managedTargetStatus, &autoscaler.Status)

	selectedReplicator := DefaultReplicator
	if autoscaler.Spec.Replicator != nil {
		selectedReplicator = *autoscaler.Spec.Replicator
	}
	replicatorStartAt := time.Now()
	replicator, ok := r.Engine.GetReplicator(selectedReplicator)
	metricPluginElapsed.WithLabelValues(autoscaler.Namespace, autoscaler.Name, selectedReplicator, "replicator").Add(time.Since(replicatorStartAt).Seconds())
	if !ok {
		autoscaler.Status.Conditions = wingv1.SetCondition(autoscaler.Status.Conditions, wingv1.Condition{
			Type:    wingv1.ConditionReady,
			Status:  metav1.ConditionFalse,
			Reason:  "ReplicatorNotExists",
			Message: fmt.Sprintf("Replicator `%s` not exists for target", selectedReplicator),
		})
		logger.Error(fmt.Errorf("replicator `%s` not registered", selectedReplicator), "Replicator not found")
		return NotRequeue
	}

	desiredReplicas, err := replicator.GetDesiredReplicas(replicatorContext)
	if err != nil {
		logger.Error(err, "Failed to get desired replicas from replicator", "replicator", selectedReplicator)
		return RequeueDelayOnErrorState
	}
	logger.V(4).Info("Replicator calculated desired replicas", "desiredReplicas", desiredReplicas)

	// Final normalize desired replicas
	var (
		scalingLimitedReason = ""

		maxReplicas = autoscaler.Spec.MaxReplicas
		minReplicas = wingv1.DefaultMinReplicas
	)
	if autoscaler.Spec.MinReplicas != nil {
		minReplicas = *autoscaler.Spec.MinReplicas
	}
	autoscaler.Status.Conditions = wingv1.SetCondition(autoscaler.Status.Conditions, wingv1.Condition{
		Type:   wingv1.ConditionReplicaPatched,
		Status: metav1.ConditionFalse,
		Reason: "No replica patch applied",
	})

	// Trying replica patch
	workingReplicaPatch, err := getWorkingReplicaPatch(autoscaler)
	if err != nil {
		logger.Error(err, "Failed to get working replica patch, fallback to default")
	} else if workingReplicaPatch == nil {
		logger.V(8).Info("No working replica patch, fallback to default")
	} else {
		// Apply replica patch
		maxReplicas = workingReplicaPatch.MaxReplicas
		minReplicas = workingReplicaPatch.MinReplicas
		autoscaler.Status.Conditions = wingv1.SetCondition(autoscaler.Status.Conditions, wingv1.Condition{
			Type:   wingv1.ConditionReplicaPatched,
			Status: metav1.ConditionTrue,
			Reason: fmt.Sprintf("Applied replica patch [%d, %d]", minReplicas, maxReplicas),
		})
	}

	if desiredReplicas > maxReplicas {
		desiredReplicas = maxReplicas
		scalingLimitedReason = "ReachMaxReplicas"
		logger.V(4).Info("Desired replicas exceed max replicas", "desiredReplicas", desiredReplicas, "maxReplicas", maxReplicas)
	}
	if desiredReplicas < minReplicas {
		desiredReplicas = minReplicas
		scalingLimitedReason = "ReachMinimalReplicas"
		logger.V(4).Info("Desired replicas below min replicas", "desiredReplicas", desiredReplicas, "minReplicas", minReplicas)
	}
	if scale.Spec.Replicas != desiredReplicas {
		if scale.Spec.Replicas < desiredReplicas {
			// ScaleUp
			r.EventRecorder.Eventf(autoscaler, wingv1.EventTypeNormal, wingv1.EventReasonScaling, "New replica %d; resource(s) are requiring scale-up", desiredReplicas)
		} else {
			// ScaleDown
			r.EventRecorder.Eventf(autoscaler, wingv1.EventTypeNormal, wingv1.EventReasonScaling, "New replica %d; all resources are below target trying to scale-down", desiredReplicas)
		}
		logger.Info("Decide to scale target replicas", "from", scale.Spec.Replicas, "to", desiredReplicas)
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
	if err := r.scaleReplicas(logger, autoscaler, gvkr, scale.DeepCopy(), desiredReplicas); err != nil {
		logger.Error(err, "Failed to scale replicas")
		return RequeueDelayOnErrorState
	}

	// Checking should enter panic mode or not(ReplicaPatch aware)
	shouldEnterPanicMode := utils.ShouldEnterPanicMode(desiredReplicas, scale.Spec.Replicas, autoscaler.Spec.Strategy)
	if shouldEnterPanicMode {
		if underPanicModeCurrently {
			logger.V(4).Info("Still in panic mode")
		} else {
			logger.Info("Enter panic mode")
			r.EventRecorder.Eventf(autoscaler, wingv1.EventTypeWarning, wingv1.EventReasonPanicMode,
				"Enter panic mode: %d -> %d(threshold %.2f with window %s).",
				scale.Spec.Replicas, desiredReplicas, autoscaler.Spec.Strategy.PanicThreshold.AsApproximateFloat64(),
				time.Duration(*autoscaler.Spec.Strategy.PanicWindowSeconds)*time.Second)
		}
		autoscaler.Status.Conditions = wingv1.SetCondition(autoscaler.Status.Conditions, wingv1.Condition{
			Type:   wingv1.ConditionPanicMode,
			Status: metav1.ConditionTrue,
		})
		return RequeueDelayOnPanicState
	}
	// out of Panic Mode period
	if !utils.StillInPanicMode(autoscaler.Status, autoscaler.Spec.Strategy) {
		// Just exit panic mode
		if wingv1.GetCondition(autoscaler.Status.Conditions, wingv1.ConditionPanicMode).Status == metav1.ConditionTrue {
			// Exit panic mode
			logger.Info("Exit panic mode")
			r.EventRecorder.Eventf(autoscaler, wingv1.EventTypeWarning, wingv1.EventReasonPanicMode,
				"Exit panic mode: %d -> %d.", scale.Spec.Replicas, desiredReplicas)
		}
		autoscaler.Status.Conditions = wingv1.SetCondition(autoscaler.Status.Conditions, wingv1.Condition{
			Type:   wingv1.ConditionPanicMode,
			Status: metav1.ConditionFalse,
		})
	}
	return DefaultRequeueDelay
}

func getWorkingReplicaPatch(autoscaler *wingv1.ReplicaAutoscaler) (*wingv1.ReplicaPatch, error) {
	replicaPatches, err := utils.GetReplicaPatches(*autoscaler)
	if err != nil {
		return nil, err
	}

	return scheduling.GetReplicaPatch(time.Now(), replicaPatches)
}

func (r *ReplicaAutoscalerReconciler) updateExhaustedAutoscaler(
	logger logr.Logger,
	autoscaler *wingv1.ReplicaAutoscaler,
	scale *autoscalingv1.Scale) (requeueDelay time.Duration, err error) {
	isExhaust := false
	exhaustedCondition := wingv1.Condition{
		Type:               wingv1.ConditionExhausted,
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
	}
	if exhaust := autoscaler.Spec.Exhaust; exhaust != nil && exhaust.Type == wingv1.ExhaustOnPending {
		scaledObjectSelector, err := labels.Parse(scale.Status.Selector)
		if err != nil {
			logger.Error(err, "couldn't convert selector into a corresponding target selector object")
			return RequeueDelayOnErrorState, err
		}
		pods, err := r.Engine.InformerFactory.PodLister().Pods(autoscaler.Namespace).List(scaledObjectSelector)
		if err != nil {
			return RequeueDelayOnErrorState, err
		}
		pendingCount := 0
		oldestPendingBornAt := time.Now()
		for _, pod := range pods {
			if pod.Status.Phase != corev1.PodPending {
				continue
			}
			pendingCount++
			if createdAt := pod.CreationTimestamp.Time; createdAt.Before(oldestPendingBornAt) {
				oldestPendingBornAt = createdAt
			}
		}
		numberThreshold, err := intstr.GetScaledValueFromIntOrPercent(
			&exhaust.Pending.Threshold, int(autoscaler.Status.CurrentReplicas), true)
		if err != nil {
			return RequeueDelayOnErrorState, err
		}
		isExhaust = pendingCount > numberThreshold &&
			time.Since(oldestPendingBornAt) > time.Duration(exhaust.Pending.TimeoutSeconds)*time.Second
		if isExhaust {
			exhaustedCondition.Status = metav1.ConditionTrue
			exhaustedCondition.Reason = "ExhaustedOnPending"
			exhaustedCondition.Message = fmt.Sprintf(
				"Pending pods count is over threshold `%d` and oldest pending pod waiting over timeout `%d` second(s)",
				numberThreshold, exhaust.Pending.TimeoutSeconds,
			)
		}
	}

	autoscaler.Status.Conditions = wingv1.SetCondition(autoscaler.Status.Conditions, exhaustedCondition)
	return NotRequeue, nil
}
