package simple

import (
	"github.com/go-logr/logr"
	"github.com/xscaling/wing/core/engine"
)

type replicator struct {
	logger logr.Logger
}

func (r *replicator) GetDesiredReplicas(ctx engine.ReplicatorContext) (int32, error) {
	desiredReplicas := ctx.Autoscaler.Status.CurrentReplicas
	for scaler, scalerOutput := range ctx.ScalersOutput {
		if scalerOutput.DesiredReplicas > desiredReplicas {
			desiredReplicas = scalerOutput.DesiredReplicas
			r.logger.Info("Choosing scaler replicas", "replicas", desiredReplicas, "scaler", scaler)
		}
	}
	if desiredReplicas < *ctx.Autoscaler.Spec.MinReplicas {
		desiredReplicas = *ctx.Autoscaler.Spec.MinReplicas
	} else if desiredReplicas > ctx.Autoscaler.Spec.MaxReplicas {
		desiredReplicas = ctx.Autoscaler.Spec.MaxReplicas
	}

	r.logger.Info("Decide to scale target replicas", "from", ctx.Autoscaler.Status.CurrentReplicas, "to", desiredReplicas)
	return desiredReplicas, nil
}
