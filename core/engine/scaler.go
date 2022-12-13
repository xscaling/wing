package engine

import (
	"encoding/json"
	"fmt"

	wingv1 "github.com/xscaling/wing/api/v1"

	"k8s.io/apimachinery/pkg/labels"
)

type ScalerOutput struct {
	DesiredReplicas int32
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
