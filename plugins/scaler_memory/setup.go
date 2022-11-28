package memory

import (
	"github.com/xscaling/wing/core/engine"
	"github.com/xscaling/wing/utils/podresourcescaler"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func init() {
	engine.RegisterPlugin("memory", engine.Plugin{
		Endpoint:  engine.PluginEndpointScaler,
		SetupFunc: setup,
	})
}

func setup(c engine.Controller) error {
	c.AddScaler("memory", podresourcescaler.New(
		corev1.ResourceMemory, c.GetKubernetesMetricsClient(), log.Log.WithName("scaler_memory")))
	return nil
}
