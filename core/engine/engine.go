package engine

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/xscaling/wing/utils"
	"github.com/xscaling/wing/utils/metrics"

	"k8s.io/apimachinery/pkg/util/wait"
	cacheddiscovery "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/record"
	resourceclient "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
	"k8s.io/metrics/pkg/client/custom_metrics"
	"k8s.io/metrics/pkg/client/external_metrics"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type engineProvisioner struct {
	kubeConfig    *rest.Config
	scalers       map[string]Scaler
	replicators   map[string]Replicator
	metricsClient metrics.MetricsClient
	pluginConfigs map[string]utils.YamlRawMessage
	eventRecorder record.EventRecorder
}

func newEngineProvisioner(kubeConfig *rest.Config, RESTMapper *restmapper.DeferredDiscoveryRESTMapper,
	pluginConfigs map[string]utils.YamlRawMessage, eventRecorder record.EventRecorder) *engineProvisioner {
	ep := &engineProvisioner{
		kubeConfig:    kubeConfig,
		scalers:       make(map[string]Scaler),
		replicators:   make(map[string]Replicator),
		pluginConfigs: pluginConfigs,
		eventRecorder: eventRecorder,
	}
	clientSet := utils.ClientOrDie(*ep.kubeConfig, "wing-engine")
	apiVersionsGetter := custom_metrics.NewAvailableAPIsGetter(clientSet.Discovery())
	// invalidate the discovery information roughly once per resync interval our API
	// information is *at most* two resync intervals old.
	go custom_metrics.PeriodicallyInvalidate(
		apiVersionsGetter,
		time.Minute,
		context.TODO().Done())

	ep.metricsClient = metrics.NewRESTMetricsClient(
		resourceclient.NewForConfigOrDie(kubeConfig),
		custom_metrics.NewForConfig(kubeConfig, RESTMapper, apiVersionsGetter),
		external_metrics.NewForConfigOrDie(kubeConfig),
	)
	return ep
}

func (p *engineProvisioner) AddReplicator(name string, replicator Replicator) {
	log.Log.Info("Adding replicator to engine", "name", name)
	p.replicators[name] = replicator
}

func (p *engineProvisioner) AddScaler(name string, scaler Scaler) {
	log.Log.Info("Adding scaler to engine", "name", name)
	p.scalers[name] = scaler
}

func (p *engineProvisioner) GetPluginConfig(name string, configReceiver interface{}) (ok bool, err error) {
	rawConfig, ok := p.pluginConfigs[name]
	if !ok {
		return false, nil
	}
	typeOfConfigReceiver := reflect.TypeOf(configReceiver)
	if typeOfConfigReceiver == nil || typeOfConfigReceiver.Kind() != reflect.Pointer {
		return true, errors.New("plugin config receiver must be a non-nil pointer")
	}
	return true, rawConfig.Unmarshal(configReceiver)
}

func (p *engineProvisioner) GetScaler(name string) (Scaler, bool) {
	scaler, ok := p.scalers[name]
	return scaler, ok
}

func (p *engineProvisioner) GetReplicator(name string) (Replicator, bool) {
	replicator, ok := p.replicators[name]
	return replicator, ok
}

func (p *engineProvisioner) GetKubernetesMetricsClient() metrics.MetricsClient {
	return p.metricsClient
}

func (p *engineProvisioner) GetEventRecorder() record.EventRecorder {
	return p.eventRecorder
}

type Engine struct {
	*engineProvisioner
	*InformerFactory
}

func New(kubeConfig *rest.Config, pluginConfigs map[string]utils.YamlRawMessage, eventRecorder record.EventRecorder) (*Engine, error) {
	// Use a discovery client capable of being refreshed.
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(
		cacheddiscovery.NewMemCacheClient(
			utils.DiscoveryClientOrDie(*kubeConfig, "wing-controller-discovery")))
	go wait.Forever(func() {
		restMapper.Reset()
	}, 30*time.Second)

	e := &Engine{
		engineProvisioner: newEngineProvisioner(kubeConfig, restMapper, pluginConfigs, eventRecorder),
		InformerFactory:   NewInformerFactory(utils.ClientOrDie(*kubeConfig, "wing-engine")),
	}
	e.InformerFactory.Run(make(<-chan struct{}))
	if err := e.loadPlugins(); err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Engine) loadPlugins() error {
	for _, pluginName := range Replicators {
		logger := log.Log.WithValues("Replicator", pluginName)
		logger.Info("Loading plugin")
		plugin, ok := GetPlugin(PluginEndpointReplicator, pluginName)
		if !ok {
			return fmt.Errorf("%s plugin %s not exists", PluginEndpointReplicator, pluginName)
		}
		if err := plugin.SetupFunc(e.engineProvisioner); err != nil {
			return fmt.Errorf("%s plugin %s failed to setup: %v", PluginEndpointReplicator, pluginName, err)
		}
	}

	for _, pluginName := range Scalers {
		logger := log.Log.WithValues("Scaler", pluginName)
		logger.Info("Loading plugin")
		plugin, ok := GetPlugin(PluginEndpointScaler, pluginName)
		if !ok {
			return fmt.Errorf("%s plugin %s not exists", PluginEndpointScaler, pluginName)
		}
		if err := plugin.SetupFunc(e.engineProvisioner); err != nil {
			return fmt.Errorf("%s plugin %s failed to setup: %v", PluginEndpointScaler, pluginName, err)
		}
	}
	return nil
}
