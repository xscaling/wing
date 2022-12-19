package prometheus

import (
	"errors"
	"fmt"
	"time"

	"github.com/xscaling/wing/core/engine"
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
	Toleration    float64       `yaml:"toleration"`
	Timeout       time.Duration `yaml:"timeout"`
	DefaultServer Server        `yaml:"defaultServer"`
}

func (c PluginConfig) Validate() error {
	if c.Toleration < 0 {
		return errors.New("toleration must be non-negative")
	}
	if c.DefaultServer.ServerAddress == nil {
		return errors.New("default server is required")
	}
	return nil
}

func setup(c engine.Controller) error {
	config := PluginConfig{}
	ok, err := c.GetPluginConfig(PluginName, &config)
	if !ok || err != nil {
		return fmt.Errorf("plugin config is required: ok %v err %v", ok, err)
	}
	prometheusScaler, err := New(config)
	if err != nil {
		return err
	}
	c.AddScaler(PluginName, prometheusScaler)
	return nil
}
