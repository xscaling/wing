package simple

import (
	"strings"
	"time"

	wingv1 "github.com/xscaling/wing/api/v1"
	"github.com/xscaling/wing/core/engine"

	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/record"
)

const (
	DefaultDownscaleStabilizationWindow = time.Second * 30
)

type Config struct {
	DownscaleStabilizationWindow time.Duration `yaml:"downscaleStabilizationWindow"`
}

func NewDefaultConfig() *Config {
	return &Config{
		DownscaleStabilizationWindow: DefaultDownscaleStabilizationWindow,
	}
}

type timestampedRecommendation struct {
	timestamp time.Time
	replicas  int32
}

type replicator struct {
	config Config
	logger logr.Logger

	eventRecorder            record.EventRecorder
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
	cutoff := time.Now().Add(-r.config.DownscaleStabilizationWindow)
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

	keyForAutoscaler := getUniqueKeyForAutoscaler(ctx.Autoscaler)
	if r.historicalRecommendation[keyForAutoscaler] == nil {
		r.historicalRecommendation[keyForAutoscaler] = []timestampedRecommendation{{
			timestamp: time.Now(),
			// Used expected replicas to avoid status pollution
			replicas: ctx.Scale.Spec.Replicas,
		}}
	}

	var (
		desiredReplicas       int32
		triggerScaleUpScalers []string
	)
	for scaler, scalerOutput := range ctx.ScalersOutput {
		logger.V(8).Info("Got scaler desired replicas",
			"scaler", scaler, "selectedDesiredReplicas", desiredReplicas, "desiredReplicas", scalerOutput.DesiredReplicas)
		if scalerOutput.DesiredReplicas > desiredReplicas {
			desiredReplicas = scalerOutput.DesiredReplicas
			logger.V(8).Info("Using scaler replicas", "replicas", desiredReplicas, "scaler", scaler)
		}
		if scalerOutput.DesiredReplicas > ctx.Scale.Spec.Replicas {
			// Want scale up
			triggerScaleUpScalers = append(triggerScaleUpScalers, scaler)
		}
	}

	if desiredReplicas < *ctx.Autoscaler.Spec.MinReplicas {
		desiredReplicas = *ctx.Autoscaler.Spec.MinReplicas
	} else if desiredReplicas > ctx.Autoscaler.Spec.MaxReplicas {
		desiredReplicas = ctx.Autoscaler.Spec.MaxReplicas
	} else {
		stabilizedReplicas := r.stabilizeRecommendation(keyForAutoscaler, desiredReplicas)
		if stabilizedReplicas != desiredReplicas {
			logger.V(2).Info("Stabilized desire replicas",
				"normalizedDesiredReplicas", desiredReplicas, "stabilizedReplicas", stabilizedReplicas)
			desiredReplicas = stabilizedReplicas
		}
	}

	if ctx.Scale.Spec.Replicas != desiredReplicas {
		if ctx.Scale.Spec.Replicas > desiredReplicas {
			// ScaleUp
			r.eventRecorder.Eventf(ctx.Autoscaler, wingv1.EventTypeNormal, wingv1.EventReasonScaling, "New replica %d; %s are requiring scale-up", desiredReplicas, strings.Join(triggerScaleUpScalers, ","))
		} else {
			// ScaleDown
			r.eventRecorder.Eventf(ctx.Autoscaler, wingv1.EventTypeNormal, wingv1.EventReasonScaling, "New replica %d; all resources are below target trying to scale-down", desiredReplicas)
		}
		logger.Info("Decide to scale target replicas", "from", ctx.Scale.Spec.Replicas, "to", desiredReplicas)
	}
	return desiredReplicas, nil
}
