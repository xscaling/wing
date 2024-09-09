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
	Toleration     float64       `yaml:"toleration"`
	DefaultTimeout time.Duration `yaml:"defaultTimeout"`
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

	timeout := s.DefaultTimeout
	if t := settings.Timeout; t != nil {
		timeout = *t
	}
	metricValue, err := settings.request(timeout)
	if err != nil {
		return
	}
	var (
		desiredReplicas int32
		averageValue    float64
	)
	if metricValue == 0 {
		// Ability to scale to zero
		desiredReplicas = 0
		averageValue = 0
	} else if ctx.CurrentReplicas == 0 {
		// Scale from zero
		desiredReplicas = int32(math.Ceil(metricValue / settings.Value))
		averageValue = metricValue / float64(desiredReplicas)
	} else {
		averageValue = metricValue / float64(ctx.CurrentReplicas)
		scaleRatio := averageValue / settings.Value
		if math.Abs(100.0-scaleRatio*100) >= s.Toleration*100 {
			desiredReplicas = int32(math.Ceil(scaleRatio * float64(ctx.CurrentReplicas)))
		}
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
