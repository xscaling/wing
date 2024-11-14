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

func NewReplicatorContext(autoscaler *wingv1.ReplicaAutoscaler, scale *autoscalingv1.Scale) ReplicatorContext {
	return ReplicatorContext{
		Autoscaler:    autoscaler,
		Scale:         scale,
		ScalersOutput: make(map[string]ScalerOutput),
	}
}
