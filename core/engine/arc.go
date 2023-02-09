package engine

import (
	"fmt"

	"github.com/xscaling/wing/utils/metrics"

	"k8s.io/client-go/tools/record"
)

const (
	PluginEndpointScaler     = "Scaler"
	PluginEndpointReplicator = "Replicator"
)

type Controller interface {
	GetPluginConfig(name string, configReceiver interface{}) (ok bool, err error)
	AddReplicator(name string, replicator Replicator)
	AddScaler(name string, scaler Scaler)
	GetKubernetesMetricsClient() metrics.MetricsClient
	GetEventRecorder() record.EventRecorder
}

type PluginSetupFunc func(c Controller) error

type Plugin struct {
	Endpoint  string
	SetupFunc PluginSetupFunc
}

var (
	registeredPlugins = make(map[string]map[string]Plugin)
)

func RegisterPlugin(name string, plugin Plugin) {
	if name == "" {
		panic("plugin must have a name")
	}
	if _, ok := registeredPlugins[plugin.Endpoint]; !ok {
		registeredPlugins[plugin.Endpoint] = make(map[string]Plugin)
	}
	if _, duplicated := registeredPlugins[plugin.Endpoint][name]; duplicated {
		panic(fmt.Sprintf("endpoint `%s` already has a plugin registered named `%s`", plugin.Endpoint, name))
	}
	registeredPlugins[plugin.Endpoint][name] = plugin
}

// return map<endpoint>plugins' name
func ListPlugins() map[string][]string {
	plugins := make(map[string][]string)
	for endpoint, pluginsMap := range registeredPlugins {
		for pluginName := range pluginsMap {
			plugins[endpoint] = append(plugins[endpoint], pluginName)
		}
	}
	return plugins
}

func GetPlugin(endpoint, name string) (Plugin, bool) {
	if _, ok := registeredPlugins[endpoint]; !ok {
		return Plugin{}, false
	}
	plugin, ok := registeredPlugins[endpoint][name]
	return plugin, ok
}
