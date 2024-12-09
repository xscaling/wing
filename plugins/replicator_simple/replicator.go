package simple

import (
	"github.com/xscaling/wing/utils"

	wingv1 "github.com/xscaling/wing/api/v1"
	"github.com/xscaling/wing/core/engine"
	"github.com/xscaling/wing/utils/tuner"

	"github.com/go-logr/logr"
)

const (
	PluginName = "simple"
)

type Config struct {
	DisableTuner bool              `json:"disableTuner" yaml:"disableTuner"`
	Flux         tuner.FluxOptions `json:"flux" yaml:"flux"`
}

func NewDefaultConfig() *Config {
	return &Config{
		Flux: tuner.NewDefaultFluxOptions(),
	}
}

func (c Config) Validate() error {
	return nil
}

type Settings struct {
	FluxPreference *tuner.FluxPreference `json:"flux,omitempty" yaml:"flux,omitempty"`
}

func (s *Settings) Validate() error {
	return nil
}

type replicator struct {
	config Config
	logger logr.Logger

	// Tuners
	flux tuner.Tuner
}

func getUniqueKeyForAutoscaler(autoscaler *wingv1.ReplicaAutoscaler) string {
	return autoscaler.Name + "/" + autoscaler.Namespace
}

func (r *replicator) GetName() string {
	return PluginName
}

func (r *replicator) GetDesiredReplicas(ctx engine.ReplicatorContext) (int32, error) {
	logger := r.logger.WithValues("namespace", ctx.Autoscaler.Namespace, "replicaAutoscaler", ctx.Autoscaler.Name)

	keyForAutoscaler := getUniqueKeyForAutoscaler(ctx.Autoscaler)
	var settings Settings
	err := utils.ExtractRawExtension(ctx.Autoscaler.Spec.ReplicatorSettings, &settings)
	if err != nil {
		logger.Error(err, "failed to unmarshal JSON into replicator setting")
		return ctx.Autoscaler.Status.CurrentReplicas, err
	}

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

	if r.flux != nil {
		fluxReplicas := r.flux.GetRecommendation(keyForAutoscaler,
			ctx.Autoscaler.Status.CurrentReplicas, desiredReplicas, settings.FluxPreference)
		if fluxReplicas != desiredReplicas {
			logger.V(2).Info("Fluxed desire replicas",
				"normalizedDesiredReplicas", desiredReplicas, "fluxReplicas", fluxReplicas)
			desiredReplicas = fluxReplicas
		}
		r.flux.AcceptRecommendation(keyForAutoscaler, ctx.Autoscaler.Status.CurrentReplicas, desiredReplicas)
	}

	return desiredReplicas, nil
}
