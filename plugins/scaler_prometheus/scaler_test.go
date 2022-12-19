package prometheus

import (
	"errors"
	"fmt"
	"testing"
	"time"

	wingv1 "github.com/xscaling/wing/api/v1"
	"github.com/xscaling/wing/core/engine"

	"github.com/stretchr/testify/require"
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
