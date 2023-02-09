package utils

import (
	"time"

	wingv1 "github.com/xscaling/wing/api/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func IsPanicModeConfigured(strategy *wingv1.ReplicaAutoscalerStrategy) bool {
	// Panic Mode will be enabled when threshold and window is set.
	return strategy != nil && strategy.PanicThreshold != nil && strategy.PanicWindowSeconds != nil
}

func ShouldEnterPanicMode(desiredReplicas, currentReplicas int32, strategy *wingv1.ReplicaAutoscalerStrategy) bool {
	// Panic Mode will be enabled when threshold and window is set.
	if !IsPanicModeConfigured(strategy) ||
		// Scale down or stay zero replicas won't enter panic mode
		desiredReplicas == 0 {
		return false
	}

	// For better bootstrap performance, we will enter panic mode when scale from zero to any positive replicas.
	if currentReplicas == 0 && desiredReplicas > 0 {
		return true
	}
	percentage := float64(desiredReplicas) / float64(currentReplicas)
	return percentage >= strategy.PanicThreshold.AsApproximateFloat64()
}

func StillInPanicMode(status wingv1.ReplicaAutoscalerStatus, strategy *wingv1.ReplicaAutoscalerStrategy) bool {
	if !IsPanicModeConfigured(strategy) {
		return false
	}
	condition := wingv1.GetCondition(status.Conditions, wingv1.ConditionPanicMode)
	return condition.Type == wingv1.ConditionPanicMode && condition.Status == metav1.ConditionTrue && time.Since(condition.LastTransitionTime.Time) < time.Duration(*strategy.PanicWindowSeconds)*time.Second
}
