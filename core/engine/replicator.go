package engine

import (
	wingv1 "github.com/xscaling/wing/api/v1"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
)

type Replicator interface {
	GetName() string
	GetDesiredReplicas(ctx ReplicatorContext) (int32, error)
}

type ReplicatorContext struct {
	Autoscaler    *wingv1.ReplicaAutoscaler
	Scale         *autoscalingv1.Scale
	ScalersOutput map[string]ScalerOutput
}

func (rc ReplicatorContext) GetScalerOutput(r Replicator) map[string]ScalerOutput {
	outputForReplicator := make(map[string]ScalerOutput)
	for scaler, scalerOutput := range rc.ScalersOutput {
		if !scalerOutput.ReplicatedBy(r) {
			continue
		}
		outputForReplicator[scaler] = scalerOutput
	}
	return outputForReplicator
}
