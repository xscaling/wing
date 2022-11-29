package simple

import (
	"fmt"

	"github.com/xscaling/wing/core/engine"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	PluginName = "simple"
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

	r := &replicator{
		config:                   *config,
		logger:                   log.Log.WithName(PluginName),
		historicalRecommendation: make(map[string][]timestampedRecommendation),
	}

	c.AddReplicator("simple", r)
	return nil
}
