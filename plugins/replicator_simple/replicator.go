package simple

import (
	"time"

	wingv1 "github.com/xscaling/wing/api/v1"
	"github.com/xscaling/wing/core/engine"

	"github.com/go-logr/logr"
)

const (
	// TODO(@oif): Make options configurable
	downscaleStabilizationWindow = time.Second * 30
)

type timestampedRecommendation struct {
	timestamp time.Time
	replicas  int32
}

type replicator struct {
	logger logr.Logger

	historicalRecommendation map[string][]timestampedRecommendation
}

func getUniqueKeyForAutoscaler(autoscaler *wingv1.ReplicaAutoscaler) string {
	return autoscaler.Name + "/" + autoscaler.Namespace
}

// stabilizeRecommendation:
// - replaces old recommendation with the newest recommendation,
// - returns max of recommendations that are not older than downscaleStabilizationWindow.
func (r *replicator) stabilizeRecommendation(key string, normalizedDesiredReplicas int32) int32 {
	maxRecommendation := normalizedDesiredReplicas
	foundOldSample := false
	oldSampleIndex := 0
	cutoff := time.Now().Add(-downscaleStabilizationWindow)
	for i, rec := range r.historicalRecommendation[key] {
		if rec.timestamp.Before(cutoff) {
			foundOldSample = true
			oldSampleIndex = i
		} else if rec.replicas > maxRecommendation {
			maxRecommendation = rec.replicas
		}
	}
	if foundOldSample {
		r.historicalRecommendation[key][oldSampleIndex] = timestampedRecommendation{
			timestamp: time.Now(),
			replicas:  normalizedDesiredReplicas,
		}
	} else {
		r.historicalRecommendation[key] = append(r.historicalRecommendation[key], timestampedRecommendation{
			timestamp: time.Now(),
			replicas:  normalizedDesiredReplicas,
		})
	}
	return maxRecommendation
}

func (r *replicator) GetDesiredReplicas(ctx engine.ReplicatorContext) (int32, error) {
	logger := r.logger.WithValues("namespace", ctx.Autoscaler.Namespace, "replicaAutoscaler", ctx.Autoscaler.Name)

	desiredReplicas := ctx.Autoscaler.Status.CurrentReplicas
	for scaler, scalerOutput := range ctx.ScalersOutput {
		if scalerOutput.DesiredReplicas > desiredReplicas {
			desiredReplicas = scalerOutput.DesiredReplicas
			logger.V(8).Info("Using scaler replicas", "replicas", desiredReplicas, "scaler", scaler)
		}
	}
	if desiredReplicas < *ctx.Autoscaler.Spec.MinReplicas {
		desiredReplicas = *ctx.Autoscaler.Spec.MinReplicas
	} else if desiredReplicas > ctx.Autoscaler.Spec.MaxReplicas {
		desiredReplicas = ctx.Autoscaler.Spec.MaxReplicas
	} else {
		stabilizedReplicas := r.stabilizeRecommendation(getUniqueKeyForAutoscaler(ctx.Autoscaler), desiredReplicas)
		if stabilizedReplicas != desiredReplicas {
			logger.V(2).Info("Stabilized desire replicas", "normalizedDesiredReplicas", desiredReplicas, "stabilizedReplicas", stabilizedReplicas)
			desiredReplicas = stabilizedReplicas
		}
	}

	if ctx.Scale.Spec.Replicas != desiredReplicas {
		logger.Info("Decide to scale target replicas", "from", ctx.Scale.Spec.Replicas, "to", desiredReplicas)
	}
	return desiredReplicas, nil
}
