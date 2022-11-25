package cpu

import (
	"github.com/xscaling/wing/core/engine"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

func init() {
	engine.RegisterPlugin("cpu", engine.Plugin{
		Endpoint:  engine.PluginEndpointScaler,
		SetupFunc: setup,
	})
}

func setup(c engine.Controller) error {
	s := &scaler{
		kubernetesMetricsClient: c.GetKubernetesMetricsClient(),
		logger:                  log.Log.WithName("cpu"),
	}

	c.AddScaler("cpu", s)
	return nil
}
