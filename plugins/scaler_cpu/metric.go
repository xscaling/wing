package cpu

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/xscaling/wing/core/engine"
	"github.com/xscaling/wing/utils/metrics"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

// Only supports for pods

type scaler struct {
	kubernetesMetricsClient metrics.MetricsClient
	logger                  logr.Logger
}

type Settings struct {
	Utilization int `json:"utilization"`
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
		return nil, errors.New("no pods found by selector for calculation")
	}

	resourceMetrics, _, err := s.kubernetesMetricsClient.GetResourceMetric(context.TODO(), corev1.ResourceCPU, ctx.Namespace, ctx.ScaledObjectSelector, "")
	if err != nil {
		s.logger.Error(err, "Failed to get metrics")
		return nil, err
	}
	desiredReplicas, _, _, err := tidyAndCalculateDesiredReplicas(resourceMetrics, pods, corev1.ResourceCPU, "", int32(settings.Utilization), ctx.CurrentReplicas)
	if err != nil {
		return nil, err
	}
	return &engine.ScalerOutput{
		DesiredReplicas: desiredReplicas,
	}, nil
}

const (
	utilizationToleration = 0.05
)

func tidyAndCalculateDesiredReplicas(resourceMetrics metrics.PodMetricsInfo, podList []*corev1.Pod,
	resource corev1.ResourceName, container string, targetUtilization int32, currentReplicas int32) (replicaCount int32, utilization int32, rawUtilization int64, err error) {

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

	usageRatio, utilization, rawUtilization, err := metrics.GetResourceUtilizationRatio(resourceMetrics, requests, targetUtilization)
	if err != nil {
		return 0, 0, 0, err
	}

	rebalanceIgnored := len(unreadyPods) > 0 && usageRatio > 1.0

	if !rebalanceIgnored && len(missingPods) == 0 {
		if math.Abs(1.0-usageRatio) <= utilizationToleration {
			// return the current replicas if the change would be too small
			return currentReplicas, utilization, rawUtilization, nil
		}

		// if we don't have any unready or missing pods, we can calculate the new replica count now
		return int32(math.Ceil(usageRatio * float64(readyPodCount))), utilization, rawUtilization, nil
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
		return 0, utilization, rawUtilization, err
	}

	if math.Abs(1.0-newUsageRatio) <= utilizationToleration || (usageRatio < 1.0 && newUsageRatio > 1.0) || (usageRatio > 1.0 && newUsageRatio < 1.0) {
		// return the current replicas if the change would be too small,
		// or if the new usage ratio would cause a change in scale direction
		return currentReplicas, utilization, rawUtilization, nil
	}

	newReplicas := int32(math.Ceil(newUsageRatio * float64(len(resourceMetrics))))
	if (newUsageRatio < 1.0 && newReplicas > currentReplicas) || (newUsageRatio > 1.0 && newReplicas < currentReplicas) {
		// return the current replicas if the change of metrics length would cause a change in scale direction
		return currentReplicas, utilization, rawUtilization, nil
	}

	// return the result, where the number of replicas considered is
	// however many replicas factored into our calculation
	return newReplicas, utilization, rawUtilization, nil
}
