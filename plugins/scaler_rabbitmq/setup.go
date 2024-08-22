package rabbitmq

import (
	"errors"
	"fmt"
	"math"
	"time"

	wingv1 "github.com/xscaling/wing/api/v1"
	"github.com/xscaling/wing/core/engine"
	"github.com/xscaling/wing/utils"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	PluginName = "rabbitmq"
)

func init() {
	engine.RegisterPlugin(PluginName, engine.Plugin{
		Endpoint:  engine.PluginEndpointScaler,
		SetupFunc: setup,
	})
}

type ScalerConfig struct {
	Toleration float64       `yaml:"toleration"`
	Timeout    time.Duration `yaml:"timeout"`
}

func (c ScalerConfig) Validate() error {
	if c.Toleration < 0 {
		return errors.New("toleration must be non-negative")
	}
	return nil
}

type scaler struct {
	ScalerConfig
}

var _ engine.Scaler = &scaler{}

func setup(c engine.Controller) error {
	config := ScalerConfig{}
	ok, err := c.GetPluginConfig(PluginName, &config)
	if !ok || err != nil {
		return fmt.Errorf("plugin config is required: ok %v err %v", ok, err)
	}
	c.AddScaler(PluginName, &scaler{config})
	return nil
}

func (s *scaler) Get(ctx engine.ScalerContext) (so *engine.ScalerOutput, err error) {
	settings := new(Settings)
	err = ctx.LoadSettings(settings)
	if err != nil {
		return
	}
	err = settings.Validate()
	if err != nil {
		return
	}
	metricValue, err := settings.request(s.Timeout)
	if err != nil {
		return
	}
	desiredReplicas := int32(0)
	averageValue := metricValue / float64(ctx.CurrentReplicas)
	scaleRatio := averageValue / settings.Value
	if math.Abs(100.0-scaleRatio*100) >= s.Toleration*100 {
		desiredReplicas = int32(math.Ceil(scaleRatio * float64(ctx.CurrentReplicas)))
	}
	utils.SetTargetStatus(ctx.AutoscalerStatus, wingv1.TargetStatus{
		Target:          settings.GetStatusMetricName(),
		Scaler:          PluginName,
		DesiredReplicas: desiredReplicas,
		Metric: wingv1.MetricTarget{
			Type:         wingv1.AverageValueMetricType,
			AverageValue: resource.NewMilliQuantity(int64(averageValue*1000), resource.DecimalSI),
		},
	})
	so = &engine.ScalerOutput{
		DesiredReplicas:     desiredReplicas,
		ManagedTargetStatus: []string{settings.GetStatusMetricName()},
	}
	return
}
