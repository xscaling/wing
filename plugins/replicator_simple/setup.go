package simple

import (
	"fmt"

	"github.com/xscaling/wing/core/engine"
	"github.com/xscaling/wing/utils/tuner"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

func init() {
	engine.RegisterPlugin(PluginName, engine.Plugin{
		Endpoint:  engine.PluginEndpointReplicator,
		SetupFunc: setup,
	})
}

func setup(c engine.Controller) error {
	config := NewDefaultConfig()
	ok, err := c.GetPluginConfig(PluginName, config)
	if !ok || err != nil {
		return fmt.Errorf("plugin config is required: ok %v err %v", ok, err)
	}

	c.AddReplicator(PluginName, NewReplicator(*config))
	return nil
}

func NewReplicator(conf Config) *replicator {
	r := &replicator{
		config: conf,
		logger: log.Log.WithName(PluginName),
	}

	if !conf.DisableTuner {
		r.flux = tuner.NewFluxTuner(conf.Flux)
	}
	return r
}
