package simple

import (
	wingv1 "github.com/xscaling/wing/api/v1"
	"github.com/xscaling/wing/core/engine"
	"github.com/xscaling/wing/utils/tuner"

	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/record"
)

type Config struct {
}

func NewDefaultConfig() *Config {
	return &Config{}
}

func (c Config) Validate() error {
	return nil
}

type replicator struct {
	config Config
	logger logr.Logger

	eventRecorder record.EventRecorder
	stabilizer    tuner.Tuner
	flux          tuner.Tuner
}

func getUniqueKeyForAutoscaler(autoscaler *wingv1.ReplicaAutoscaler) string {
	return autoscaler.Name + "/" + autoscaler.Namespace
}

func (r *replicator) GetName() string {
	return "simple"
}

func (r *replicator) GetDesiredReplicas(ctx engine.ReplicatorContext) (int32, error) {
	logger := r.logger.WithValues("namespace", ctx.Autoscaler.Namespace, "replicaAutoscaler", ctx.Autoscaler.Name)

	keyForAutoscaler := getUniqueKeyForAutoscaler(ctx.Autoscaler)

	var (
		desiredReplicas int32
	)
	for scaler, scalerOutput := range ctx.ScalersOutput {
		logger.V(8).Info("Got scaler desired replicas",
			"scaler", scaler, "selectedDesiredReplicas", desiredReplicas, "desiredReplicas", scalerOutput.DesiredReplicas)
		if scalerOutput.DesiredReplicas > desiredReplicas {
			desiredReplicas = scalerOutput.DesiredReplicas
			logger.V(8).Info("Using scaler replicas", "replicas", desiredReplicas, "scaler", scaler)
		}
	}

	stabilizedReplicas := r.stabilizer.GetRecommendation(keyForAutoscaler,
		ctx.Autoscaler.Status.CurrentReplicas, desiredReplicas, tuner.StabilizerPreference{})
	if stabilizedReplicas != desiredReplicas {
		logger.V(2).Info("Stabilized desire replicas",
			"normalizedDesiredReplicas", desiredReplicas, "stabilizedReplicas", stabilizedReplicas)
		desiredReplicas = stabilizedReplicas
	}
	return desiredReplicas, nil
}
