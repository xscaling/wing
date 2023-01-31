package podresourcescaler

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	wingv1 "github.com/xscaling/wing/api/v1"
	"github.com/xscaling/wing/core/engine"
	"github.com/xscaling/wing/utils"
	"github.com/xscaling/wing/utils/metrics"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type scaler struct {
	kubernetesMetricsClient metrics.MetricsClient
	resource                corev1.ResourceName
	logger                  logr.Logger
	config                  Config
	pluginName              string
}

var _ engine.Scaler = &scaler{}

type Config struct {
	UtilizationToleration float64 `yaml:"utilizationToleration"`
}

func (c Config) Validate() error {
	if c.UtilizationToleration < DefaultUtilizationToleration || c.UtilizationToleration > 1 {
		return errors.New("pod resource toleration is in valid, requires [0.05, 1]")
	}
	return nil
}

const (
	DefaultUtilizationToleration = 0.05
)

func NewDefaultConfig() *Config {
	return &Config{
		UtilizationToleration: DefaultUtilizationToleration,
	}
}

type Settings struct {
	Utilization int `json:"utilization"`
}

func New(pluginName string, config Config, resource corev1.ResourceName, kubernetesMetricsClient metrics.MetricsClient) (*scaler, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &scaler{
		config:                  config,
		resource:                resource,
		kubernetesMetricsClient: kubernetesMetricsClient,
		logger:                  log.Log.WithValues("plugin", pluginName),
		pluginName:              pluginName,
	}, nil
}

func (s *scaler) Get(ctx engine.ScalerContext) (*engine.ScalerOutput, error) {
	settings := new(Settings)
	if err := ctx.LoadSettings(settings); err != nil {
		return nil, err
	}
	pods, err := ctx.InformerFactory.PodLister().Pods(ctx.Namespace).List(ctx.ScaledObjectSelector)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %v", err)
	}
	if len(pods) == 0 {
		s.logger.WithValues("namespace", ctx.Namespace, "scaleTargetRef", ctx.ScaleTargetRef.Name).Info("No pods found by selector for calculation, keep current replicas")
		return &engine.ScalerOutput{
			DesiredReplicas: ctx.CurrentReplicas,
		}, nil
	}

	resourceMetrics, _, err := s.kubernetesMetricsClient.GetResourceMetric(context.TODO(), s.resource, ctx.Namespace, ctx.ScaledObjectSelector, "")
	if err != nil {
		s.logger.Error(err, "Failed to get metrics")
		return nil, err
	}
	desiredReplicas, averageUtilization, _, err := tidyAndCalculateDesiredReplicas(s.config.UtilizationToleration, resourceMetrics, pods, s.resource, "", int32(settings.Utilization), ctx.CurrentReplicas)
	if err != nil {
		return nil, err
	}
	utils.SetTargetStatus(ctx.AutoscalerStatus, wingv1.TargetStatus{
		Target:          s.pluginName,
		Scaler:          s.pluginName,
		DesiredReplicas: desiredReplicas,
		Metric: wingv1.MetricTarget{
			Type:               wingv1.UtilizationMetricType,
			AverageUtilization: &averageUtilization,
		},
	})
	return &engine.ScalerOutput{
		DesiredReplicas: desiredReplicas,
	}, nil
}

func tidyAndCalculateDesiredReplicas(utilizationToleration float64, resourceMetrics metrics.PodMetricsInfo, podList []*corev1.Pod,
	resource corev1.ResourceName, container string, targetUtilization int32, currentReplicas int32) (replicaCount int32, utilization int32, rawAverageValue int64, err error) {

	readyPodCount, unreadyPods, missingPods, ignoredPods := groupPods(podList, resourceMetrics, resource, time.Second*10, time.Second*3)
	removeMetricsForPods(resourceMetrics, ignoredPods)
	removeMetricsForPods(resourceMetrics, unreadyPods)
	requests, err := calculatePodRequests(podList, container, resource)
	if err != nil {
		return 0, 0, 0, err
	}

	if len(resourceMetrics) == 0 {
		return 0, 0, 0, fmt.Errorf("did not receive metrics for any ready pods")
	}

	usageRatio, utilization, rawAverageValue, err := metrics.GetResourceUtilizationRatio(resourceMetrics, requests, targetUtilization)
	if err != nil {
		return 0, 0, 0, err
	}

	rebalanceIgnored := len(unreadyPods) > 0 && usageRatio > 1.0

	if !rebalanceIgnored && len(missingPods) == 0 {
		if math.Abs(1.0-usageRatio) <= utilizationToleration {
			// return the current replicas if the change would be too small
			return currentReplicas, utilization, rawAverageValue, nil
		}
		// if we don't have any unready or missing pods, we can calculate the new replica count now
		return int32(math.Ceil(usageRatio * float64(readyPodCount))), utilization, rawAverageValue, nil
	}

	if len(missingPods) > 0 {
		if usageRatio < 1.0 {
			// on a scale-down, treat missing pods as using 100% of the resource request
			for podName := range missingPods {
				resourceMetrics[podName] = metrics.PodMetric{Value: requests[podName]}
			}
		} else if usageRatio > 1.0 {
			// on a scale-up, treat missing pods as using 0% of the resource request
			for podName := range missingPods {
				resourceMetrics[podName] = metrics.PodMetric{Value: 0}
			}
		}
	}

	if rebalanceIgnored {
		// on a scale-up, treat unready pods as using 0% of the resource request
		for podName := range unreadyPods {
			resourceMetrics[podName] = metrics.PodMetric{Value: 0}
		}
	}

	// re-run the utilization calculation with our new numbers
	newUsageRatio, _, _, err := metrics.GetResourceUtilizationRatio(resourceMetrics, requests, targetUtilization)
	if err != nil {
		return 0, utilization, rawAverageValue, err
	}

	if math.Abs(1.0-newUsageRatio) <= utilizationToleration || (usageRatio < 1.0 && newUsageRatio > 1.0) || (usageRatio > 1.0 && newUsageRatio < 1.0) {
		// return the current replicas if the change would be too small,
		// or if the new usage ratio would cause a change in scale direction
		return currentReplicas, utilization, rawAverageValue, nil
	}

	newReplicas := int32(math.Ceil(newUsageRatio * float64(len(resourceMetrics))))
	if (newUsageRatio < 1.0 && newReplicas > currentReplicas) || (newUsageRatio > 1.0 && newReplicas < currentReplicas) {
		// return the current replicas if the change of metrics length would cause a change in scale direction
		return currentReplicas, utilization, rawAverageValue, nil
	}

	// return the result, where the number of replicas considered is
	// however many replicas factored into our calculation
	return newReplicas, utilization, rawAverageValue, nil
}
