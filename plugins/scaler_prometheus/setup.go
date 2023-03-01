package prometheus

import (
	"fmt"

	"github.com/xscaling/wing/core/engine"
	"github.com/xscaling/wing/utils/scalerprovider/prometheus"
)

const (
	PluginName = "prometheus"
)

func init() {
	engine.RegisterPlugin(PluginName, engine.Plugin{
		Endpoint:  engine.PluginEndpointScaler,
		SetupFunc: setup,
	})
}

type PluginConfig struct {
	prometheus.ScalerConfig `yaml:",inline"`
}

func setup(c engine.Controller) error {
	config := PluginConfig{
		ScalerConfig: *prometheus.NewDefaultConfig(),
	}
	ok, err := c.GetPluginConfig(PluginName, &config)
	if !ok || err != nil {
		return fmt.Errorf("plugin config is required: ok %v err %v", ok, err)
	}
	prometheusScaler, err := prometheus.New(PluginName, config.ScalerConfig)
	if err != nil {
		return err
	}
	c.AddScaler(PluginName, prometheusScaler)
	return nil
}
