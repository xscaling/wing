package engine

import (
	"encoding/json"
	"fmt"

	wingv1 "github.com/xscaling/wing/api/v1"

	"k8s.io/apimachinery/pkg/labels"
)

type ScalerOutput struct {
	Settings            interface{}
	DesiredReplicas     int32
	ManagedTargetStatus []string

	replicatedChecker func(r Replicator) bool
}

func (o *ScalerOutput) ReplicatedBy(r Replicator) bool {
	if o.replicatedChecker == nil {
		return true
	}
	return o.replicatedChecker(r)
}

func (o *ScalerOutput) SetReplicatorLimited(checker func(Replicator) bool) {
	o.replicatedChecker = checker
}

type Scaler interface {
	Get(ctx ScalerContext) (*ScalerOutput, error)
}

type ScalerContext struct {
	*InformerFactory
	RawSettings          []byte
	ScaleTargetRef       wingv1.CrossVersionObjectReference
	Namespace            string
	ScaledObjectSelector labels.Selector
	CurrentReplicas      int32
	AutoscalerStatus     *wingv1.ReplicaAutoscalerStatus
}

func (c ScalerContext) LoadSettings(receiver interface{}) error {
	err := json.Unmarshal(c.RawSettings, receiver)
	if err != nil {
		return fmt.Errorf("invalid settings `%s`: %w", c.RawSettings, err)
	}
	return nil
}
