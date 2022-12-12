package podresourcescaler

import (
	"testing"
	"time"

	"github.com/xscaling/wing/utils/metrics"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func withPodStatus(podPhase corev1.PodPhase, startTime metav1.Time, status corev1.ConditionStatus) func(*corev1.Pod) {
	return func(pod *corev1.Pod) {
		pod.Status = corev1.PodStatus{
			Phase:     podPhase,
			StartTime: &startTime,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: status,
				},
			},
		}
	}
}

func makeTestPod(name string, resources corev1.ResourceRequirements, podPatches ...func(*corev1.Pod)) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Resources: resources,
				},
			},
		},
	}
	for _, patch := range podPatches {
		patch(pod)
	}
	return pod
}

func TestTidyAndCalculateDesiredReplicas(t *testing.T) {
	for index, testCase := range []struct {
		utilizationToleration float64
		resourceMetrics       metrics.PodMetricsInfo
		podList               []*corev1.Pod
		resource              corev1.ResourceName
		targetUtilization     int32
		currentReplicas       int32
		// expected
		hasError                bool
		expectedDesiredReplica  int32
		expectedUtilization     int32
		expectedRawAverageValue int64
	}{
		{
			// single pod with full loaded
			utilizationToleration: 0.05,
			resourceMetrics: map[string]metrics.PodMetric{
				"pod1": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     100,
				},
			},
			podList: []*corev1.Pod{
				makeTestPod("pod1", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("100m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
			},
			resource:          corev1.ResourceCPU,
			targetUtilization: 80,
			currentReplicas:   1,

			hasError:                false,
			expectedDesiredReplica:  2,
			expectedUtilization:     100,
			expectedRawAverageValue: 100,
		},
		{
			// missing metric of pod: only calculate resources of found one
			utilizationToleration: 0.05,
			resourceMetrics: map[string]metrics.PodMetric{
				"pod1": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     100,
				},
			},
			podList: []*corev1.Pod{
				makeTestPod("pod1", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("100m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
				makeTestPod("pod2", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("100m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
			},
			resource:          corev1.ResourceCPU,
			targetUtilization: 80,
			currentReplicas:   2,

			hasError:                false,
			expectedDesiredReplica:  2,
			expectedUtilization:     100,
			expectedRawAverageValue: 100,
		},
		{
			// stable state
			utilizationToleration: 0.05,
			resourceMetrics: map[string]metrics.PodMetric{
				"pod1": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     50,
				},
				"pod2": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     80,
				},
			},
			podList: []*corev1.Pod{
				makeTestPod("pod1", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("100m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
				makeTestPod("pod2", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("100m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
			},
			resource:          corev1.ResourceCPU,
			targetUtilization: 80,
			currentReplicas:   2,

			hasError:                false,
			expectedDesiredReplica:  2,
			expectedUtilization:     65,
			expectedRawAverageValue: 65,
		},
		{
			// wanna scale up with same resource requirements
			utilizationToleration: 0.05,
			resourceMetrics: map[string]metrics.PodMetric{
				"pod1": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     100,
				},
				"pod2": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     130,
				},
			},
			podList: []*corev1.Pod{
				makeTestPod("pod1", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("80m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
				makeTestPod("pod2", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("80m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
			},
			resource:          corev1.ResourceCPU,
			targetUtilization: 80,
			currentReplicas:   2,

			hasError:                false,
			expectedDesiredReplica:  4,
			expectedUtilization:     143,
			expectedRawAverageValue: 115,
		},
		{
			// wanna scale up with different resource requirements(maybe rolling state)
			utilizationToleration: 0.05,
			resourceMetrics: map[string]metrics.PodMetric{
				"pod1": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     160,
				},
				"pod2": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     130,
				},
			},
			podList: []*corev1.Pod{
				// Different version
				makeTestPod("pod1", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("200m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
				makeTestPod("pod2", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("100m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
			},
			resource:          corev1.ResourceCPU,
			targetUtilization: 80,
			currentReplicas:   2,

			hasError:                false,
			expectedDesiredReplica:  3,
			expectedUtilization:     96,
			expectedRawAverageValue: 145,
		},
		{
			// won't wanna scale up due to huge utilization toleration
			utilizationToleration: 0.2,
			resourceMetrics: map[string]metrics.PodMetric{
				"pod1": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     160,
				},
				"pod2": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     130,
				},
			},
			podList: []*corev1.Pod{
				// Different version
				makeTestPod("pod1", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("200m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
				makeTestPod("pod2", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("100m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
			},
			resource:          corev1.ResourceCPU,
			targetUtilization: 80,
			currentReplicas:   2,

			hasError:                false,
			expectedDesiredReplica:  2,
			expectedUtilization:     96,
			expectedRawAverageValue: 145,
		},
		{
			// with unready pod
			utilizationToleration: 0.05,
			resourceMetrics: map[string]metrics.PodMetric{
				"pod1": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     160,
				},
				"pod2": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     130,
				},
			},
			podList: []*corev1.Pod{
				// Different version
				makeTestPod("pod1", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("100m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
				makeTestPod("pod2", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("200m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionFalse)),
			},
			resource:          corev1.ResourceCPU,
			targetUtilization: 80,
			currentReplicas:   2,

			hasError:                false,
			expectedDesiredReplica:  2,
			expectedUtilization:     160,
			expectedRawAverageValue: 160,
		},
		// Scale down parts
		{
			utilizationToleration: 0.05,
			resourceMetrics: map[string]metrics.PodMetric{
				"pod1": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     5,
				},
				"pod2": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     5,
				},
			},
			podList: []*corev1.Pod{
				makeTestPod("pod1", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("200m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
				makeTestPod("pod2", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("200m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
			},
			resource:          corev1.ResourceCPU,
			targetUtilization: 80,
			currentReplicas:   2,

			hasError:                false,
			expectedDesiredReplica:  1,
			expectedUtilization:     2,
			expectedRawAverageValue: 5,
		},
		{
			// won't scale-down due to average
			utilizationToleration: 0.1,
			resourceMetrics: map[string]metrics.PodMetric{
				"pod1": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     140,
				},
				"pod2": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     160,
				},
			},
			podList: []*corev1.Pod{
				makeTestPod("pod1", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("200m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
				makeTestPod("pod2", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("200m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
			},
			resource:          corev1.ResourceCPU,
			targetUtilization: 80,
			currentReplicas:   2,

			hasError:                false,
			expectedDesiredReplica:  2,
			expectedUtilization:     75,
			expectedRawAverageValue: 150,
		},
		{
			// won't scale-down due to toleration
			utilizationToleration: 0.1,
			resourceMetrics: map[string]metrics.PodMetric{
				"pod1": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     130,
				},
				"pod2": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     160,
				},
			},
			podList: []*corev1.Pod{
				makeTestPod("pod1", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("200m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
				makeTestPod("pod2", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("200m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
			},
			resource:          corev1.ResourceCPU,
			targetUtilization: 80,
			currentReplicas:   2,

			hasError:                false,
			expectedDesiredReplica:  2,
			expectedUtilization:     72,
			expectedRawAverageValue: 145,
		},
		{
			// on scale down
			utilizationToleration: 0.1,
			resourceMetrics: map[string]metrics.PodMetric{
				// missing pod1 metric, will be regard as 100% usage of request
				"pod2": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     120,
				},
			},
			podList: []*corev1.Pod{
				makeTestPod("pod1", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("200m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
				makeTestPod("pod2", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("200m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
			},
			resource:          corev1.ResourceCPU,
			targetUtilization: 80,
			currentReplicas:   2,

			hasError:                false,
			expectedDesiredReplica:  2,
			expectedUtilization:     60,
			expectedRawAverageValue: 120,
		},
		{
			// on scale up
			utilizationToleration: 0.1,
			resourceMetrics: map[string]metrics.PodMetric{
				// missing pod1 metric, will be regard as 0% usage of request
				"pod2": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     160,
				},
			},
			podList: []*corev1.Pod{
				makeTestPod("pod1", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("200m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
				makeTestPod("pod2", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("200m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
			},
			resource:          corev1.ResourceCPU,
			targetUtilization: 40,
			currentReplicas:   2,

			hasError:                false,
			expectedDesiredReplica:  2,
			expectedUtilization:     80,
			expectedRawAverageValue: 160,
		},
		// failed cases
		{
			// not ready pods
			utilizationToleration: 0.1,
			resourceMetrics: map[string]metrics.PodMetric{
				"pod1": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     130,
				},
				"pod2": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     160,
				},
			},
			podList: []*corev1.Pod{
				makeTestPod("pod1", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("200m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionFalse)),
				makeTestPod("pod2", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("200m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionFalse)),
			},
			resource:          corev1.ResourceCPU,
			targetUtilization: 80,
			currentReplicas:   2,

			hasError: true,
		},
		{
			// without metrics and pods
			utilizationToleration: 0.05,
			resourceMetrics:       map[string]metrics.PodMetric{},
			podList:               []*corev1.Pod{},
			resource:              corev1.ResourceCPU,
			targetUtilization:     60,
			currentReplicas:       3,
			hasError:              true,
		},
		{
			// missing request
			utilizationToleration: 0.1,
			resourceMetrics: map[string]metrics.PodMetric{
				"pod1": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     130,
				},
				"pod2": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     160,
				},
			},
			podList: []*corev1.Pod{
				makeTestPod("pod1", corev1.ResourceRequirements{}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
				makeTestPod("pod2", corev1.ResourceRequirements{}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
			},
			resource:          corev1.ResourceCPU,
			targetUtilization: 80,
			currentReplicas:   2,

			hasError: true,
		},
		{
			// resource mismatched known pod
			utilizationToleration: 0.1,
			resourceMetrics: map[string]metrics.PodMetric{
				"pod2": {
					Timestamp: time.Now(),
					Window:    time.Minute,
					Value:     160,
				},
			},
			podList: []*corev1.Pod{
				makeTestPod("pod1", corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("200m"),
					},
				}, withPodStatus(corev1.PodRunning, metav1.Now(), corev1.ConditionTrue)),
			},
			resource:          corev1.ResourceCPU,
			targetUtilization: 80,
			currentReplicas:   2,

			hasError: true,
		},
	} {
		desiredReplica, utilization, rawAverageValue, err := tidyAndCalculateDesiredReplicas(
			testCase.utilizationToleration, testCase.resourceMetrics, testCase.podList,
			testCase.resource, "", testCase.targetUtilization, testCase.currentReplicas)
		if testCase.hasError {
			assert.Error(t, err, "case %d", index)
			continue
		}
		if !assert.NoError(t, err) {
			continue
		}
		assert.Equal(t, testCase.expectedDesiredReplica, desiredReplica, "case %d", index)
		assert.Equal(t, testCase.expectedUtilization, utilization, "case %d", index)
		assert.Equal(t, testCase.expectedRawAverageValue, rawAverageValue, "case %d", index)
	}
}
