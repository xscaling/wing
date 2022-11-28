package simple

import (
	"github.com/xscaling/wing/core/engine"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

func init() {
	engine.RegisterPlugin("simple", engine.Plugin{
		Endpoint:  engine.PluginEndpointReplicator,
		SetupFunc: setup,
	})
}

func setup(c engine.Controller) error {
	r := &replicator{
		logger:                   log.Log.WithName("simple"),
		historicalRecommendation: make(map[string][]timestampedRecommendation),
	}

	c.AddReplicator("simple", r)
	return nil
}
