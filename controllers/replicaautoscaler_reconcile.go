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
	"time"

	wingv1 "github.com/xscaling/wing/api/v1"
	"github.com/xscaling/wing/utils"
	autoscalingv1 "k8s.io/api/autoscaling/v1"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

	// TODO(@oif): Init various

	// A static replicas setting
	if autoscaler.Spec.MinReplicas == nil {
		logger.V(2).Info("Setting static replicas")
		return true, r.scaleReplicas(logger, autoscaler, gvkr, scale)
	}

	// Working on autoscaling flow
	return r.reconcileAutoscaling(logger, autoscaler)
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

func (r *ReplicaAutoscalerReconciler) scaleReplicas(logger logr.Logger, autoscaler *wingv1.ReplicaAutoscaler, gvkr wingv1.GroupVersionKindResource, scale *autoscalingv1.Scale) error {
	if scale.Spec.Replicas == autoscaler.Spec.MaxReplicas {
		logger.V(4).Info("Nothing to scale")
		return nil
	}
	logger.V(2).Info("Scaling replicas", "currentReplicas", scale.Spec.Replicas, "desireReplicas", autoscaler.Spec.MaxReplicas)
	scale.Spec.Replicas = autoscaler.Spec.MaxReplicas
	_, err := r.scaleClient.Scales(autoscaler.Namespace).Update(context.TODO(), gvkr.GroupResource(), scale, metav1.UpdateOptions{})
	if err != nil {
		logger.Error(err, "Failed to scale replicas")
		return err
	}
	return nil
}

const (
	DefaultScalingColdDown = time.Minute * 3
)

func (r *ReplicaAutoscalerReconciler) reconcileAutoscaling(logger logr.Logger, autoscaler *wingv1.ReplicaAutoscaler) (requeue bool, err error) {
	// Checking cold-down
	if time.Since(autoscaler.Status.LastScaleTime.Time) < DefaultScalingColdDown {
		logger.V(2).Info("Still in scaling cold-down period")
		return
	}
	// Collect metrics from providers

	// Getting desired replicas from scaler

	// Normalize replicas with autoscaler boundary
	return false, nil
}
