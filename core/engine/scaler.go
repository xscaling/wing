package engine

import (
	"encoding/json"
	"fmt"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
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
	ScaleTargetRef       autoscalingv2.CrossVersionObjectReference
	Namespace            string
	ScaledObjectSelector labels.Selector
	CurrentReplicas      int32
}

func (c ScalerContext) LoadSettings(receiver interface{}) error {
	err := json.Unmarshal(c.RawSettings, receiver)
	if err != nil {
		return fmt.Errorf("invalid settings `%s`: %w", c.RawSettings, err)
	}
	return nil
}
