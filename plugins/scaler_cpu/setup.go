package cpu

import (
	"fmt"

	"github.com/xscaling/wing/core/engine"
	"github.com/xscaling/wing/utils/podresourcescaler"

	corev1 "k8s.io/api/core/v1"
)

const (
	PluginName = "cpu"
)

func init() {
	engine.RegisterPlugin(PluginName, engine.Plugin{
		Endpoint:  engine.PluginEndpointScaler,
		SetupFunc: setup,
	})
}

type PluginConfig struct {
	podresourcescaler.Config `yaml:",inline"`
}

func setup(c engine.Controller) error {
	config := PluginConfig{
		Config: *podresourcescaler.NewDefaultConfig(),
	}
	ok, err := c.GetPluginConfig(PluginName, &config)
	if !ok || err != nil {
		return fmt.Errorf("plugin config is required: ok %v err %v", ok, err)
	}
	podResourceScaler, err := podresourcescaler.New(PluginName, config.Config,
		corev1.ResourceCPU, c.GetKubernetesMetricsClient())
	if err != nil {
		return err
	}
	c.AddScaler(PluginName, podResourceScaler)
	return nil
}
