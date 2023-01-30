package prometheus

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	wingv1 "github.com/xscaling/wing/api/v1"
	"github.com/xscaling/wing/core/engine"
	"github.com/xscaling/wing/utils"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/pointer"
)

type fakeQueryClient struct {
	metricValue float64
	err         error
}

func (f *fakeQueryClient) Query(server Server, query string, when time.Time) (float64, error) {
	return f.metricValue, f.err
}

func TestScaler(t *testing.T) {
	// without default server
	_, err := New(PluginConfig{
		Timeout: 10 & time.Second,
	})
	require.Error(t, err)
	// invalid toleration
	_, err = New(PluginConfig{
		Toleration: -0.1,
		Timeout:    10 & time.Second,
		DefaultServer: Server{
			ServerAddress: pointer.String("https://prometheus.example.com"),
		},
	})
	require.Error(t, err)

	testScaler, err := New(PluginConfig{
		Toleration: 0.1,
		Timeout:    10 & time.Second,
		DefaultServer: Server{
			ServerAddress: pointer.String("https://prometheus.example.com"),
		},
	})
	require.NoError(t, err)
	fakeError := errors.New("testing")
	for index, testCase := range []struct {
		query           string
		currentReplicas int32
		threshold       float64
		// fake query client
		metricValue   float64
		hasQueryError bool

		expectedError    bool
		expectedReplicas int32
	}{
		// empty query then got error
		{
			query:           "",
			currentReplicas: 1,
			threshold:       1,
			expectedError:   true,
		},
		// query error then got error
		{
			query:           "up",
			currentReplicas: 1,
			threshold:       1,
			hasQueryError:   true,
			expectedError:   true,
		},
		// query returns zero value
		{
			query:            "up",
			currentReplicas:  1,
			threshold:        1,
			metricValue:      0,
			expectedReplicas: 0,
		},
		// scale up: 2 replicas with 4 metric value and target threshold 1 then got 4
		{
			query:            "up",
			currentReplicas:  2,
			threshold:        1,
			metricValue:      4,
			expectedReplicas: 4,
		},
		// scale down: 4 replicas with 4 metric value and target threshold 2 then got 2
		{
			query:            "up",
			currentReplicas:  4,
			threshold:        2,
			metricValue:      4,
			expectedReplicas: 2,
		},
		// stable: 2 replicas with 2 metric value and target threshold 1 then got 2
		{
			query:            "up",
			currentReplicas:  2,
			threshold:        1,
			metricValue:      2,
			expectedReplicas: 2,
		},
		// under toleration
		{
			query:            "up",
			currentReplicas:  2,
			threshold:        2,
			metricValue:      4.3,
			expectedReplicas: 2,
		},
		// edge scale up
		{
			query:            "up",
			currentReplicas:  2,
			threshold:        2,
			metricValue:      4.4,
			expectedReplicas: 3,
		},
		// edge scale down
		{
			query:            "up",
			currentReplicas:  4,
			threshold:        2,
			metricValue:      5.7,
			expectedReplicas: 3,
		},
		// edge bottom
		{
			query:            "up",
			currentReplicas:  4,
			threshold:        2,
			metricValue:      0,
			expectedReplicas: 0,
		},
	} {
		// preparing fake query client
		queryClient := &fakeQueryClient{
			metricValue: testCase.metricValue,
		}
		if testCase.hasQueryError {
			queryClient.err = fakeError
		}

		// inject fake client
		testScaler.queryClient = queryClient

		output, err := testScaler.Get(engine.ScalerContext{
			RawSettings:      []byte(fmt.Sprintf(`{"query":"%s","threshold":%f}`, testCase.query, testCase.threshold)),
			CurrentReplicas:  testCase.currentReplicas,
			AutoscalerStatus: &wingv1.ReplicaAutoscalerStatus{},
		})
		require.Equal(t, testCase.expectedError, err != nil, "[%d] error mismatch, got: %v", index, err)
		if testCase.expectedError {
			continue
		}
		require.Equal(t, testCase.expectedReplicas, output.DesiredReplicas, "[%d] desired replicas mismatch", index)
	}
}

func TestFailoverSettingsAreMutuallyExclusive(t *testing.T) {
	x := Settings{
		FailAsZero:      pointer.Bool(true),
		FailAsLastValue: pointer.Bool(true),
	}
	err := x.Validate()
	require.Error(t, err)
}

func TestScalerFailAsLastValue(t *testing.T) {
	testScaler, err := New(PluginConfig{
		Toleration: 0.1,
		Timeout:    10 & time.Second,
		DefaultServer: Server{
			ServerAddress: pointer.String("https://prometheus.example.com"),
		},
	})
	require.NoError(t, err)
	fakeQueryClient := &fakeQueryClient{
		metricValue: 100,
	}
	testScaler.queryClient = fakeQueryClient

	status := &wingv1.ReplicaAutoscalerStatus{}

	settings := Settings{
		Query:           "up",
		Threshold:       10,
		FailAsLastValue: pointer.Bool(true),
	}
	parseRawSettings := func() []byte {
		payload, err := json.Marshal(settings)
		if err != nil {
			panic(err)
		}
		return payload
	}

	ctx := engine.ScalerContext{
		RawSettings:      parseRawSettings(),
		CurrentReplicas:  10,
		AutoscalerStatus: status,
	}
	output, err := testScaler.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, int32(10), output.DesiredReplicas)

	// Check status target
	targetStatus, ok := utils.GetTargetStatus(status, makeTargetStatusName(settings.Query))
	require.True(t, ok)

	require.NotNil(t, targetStatus.Metric.AverageValue)
	// Average value = 10
	require.Equal(t, resource.NewMilliQuantity(10000, resource.DecimalSI), targetStatus.Metric.AverageValue)

	// Set fetch error
	fakeQueryClient.err = errors.New("fake error, whatever it's null data, multiple date, fetch error, etc.")
	// Re-calculate with same current replicas without error due to failover
	output, err = testScaler.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, int32(10), output.DesiredReplicas)

	// Lost replicas while failover, should not scale down due to last value(average value) same as before
	ctx.CurrentReplicas = 9
	output, err = testScaler.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, int32(9), output.DesiredReplicas)
	ctx.CurrentReplicas = 10

	// Change query to make last value out-of-date
	settings.Query = "up2"
	ctx.RawSettings = parseRawSettings()
	// Should got error here due to missing last value
	_, err = testScaler.Get(ctx)
	require.NotNil(t, err)

	// Disable failover
	settings.FailAsLastValue = pointer.Bool(false)
	ctx.RawSettings = parseRawSettings()
	_, err = testScaler.Get(ctx)
	require.NotNil(t, err)
}

func TestScalerFailAsZero(t *testing.T) {
	testScaler, err := New(PluginConfig{
		Toleration: 0.1,
		Timeout:    10 & time.Second,
		DefaultServer: Server{
			ServerAddress: pointer.String("https://prometheus.example.com"),
		},
	})
	require.NoError(t, err)
	fakeQueryClient := &fakeQueryClient{
		metricValue: 100,
	}
	testScaler.queryClient = fakeQueryClient

	status := &wingv1.ReplicaAutoscalerStatus{}

	settings := Settings{
		Query:      "up",
		Threshold:  10,
		FailAsZero: pointer.Bool(true),
	}
	parseRawSettings := func() []byte {
		payload, err := json.Marshal(settings)
		if err != nil {
			panic(err)
		}
		return payload
	}

	ctx := engine.ScalerContext{
		RawSettings:      parseRawSettings(),
		CurrentReplicas:  10,
		AutoscalerStatus: status,
	}
	output, err := testScaler.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, int32(10), output.DesiredReplicas)

	// Check status target
	targetStatus, ok := utils.GetTargetStatus(status, makeTargetStatusName(settings.Query))
	require.True(t, ok)

	require.NotNil(t, targetStatus.Metric.AverageValue)
	// Average value = 10
	require.Equal(t, resource.NewMilliQuantity(10000, resource.DecimalSI), targetStatus.Metric.AverageValue)

	// Set fetch error
	fakeQueryClient.err = errors.New("fake error, whatever it's null data, multiple date, fetch error, etc.")
	// Re-calculate will got zero desired replicas without error due to failover
	output, err = testScaler.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, int32(0), output.DesiredReplicas)

	// Disable failover
	settings.FailAsZero = pointer.Bool(false)
	ctx.RawSettings = parseRawSettings()
	_, err = testScaler.Get(ctx)
	require.NotNil(t, err)
}
