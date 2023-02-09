package utils

import (
	"testing"

	wingv1 "github.com/xscaling/wing/api/v1"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/pointer"
)

func TestShouldEnterPanicMode(t *testing.T) {
	for _, testCase := range []struct {
		currentReplicas    int32
		desiredReplicas    int32
		panicThreshold     *resource.Quantity
		panicWindowSeconds *int32

		shouldEnter bool
	}{
		// without strategy and stay zero replicas
		{0, 0, nil, nil, false},
		// without strategy and scale up
		{0, 1, nil, nil, false},
		// without strategy and scale down
		{1, 0, nil, nil, false},
		// without part of strategy and scale up
		{1, 100, resource.NewMilliQuantity(1200, resource.DecimalSI), nil, false},
		{1, 100, nil, pointer.Int32(30), false},

		// Scale from zero always enter panic mode
		{0, 1, resource.NewMilliQuantity(1200, resource.DecimalSI), pointer.Int32(30), true},

		// Stay current replicas
		{1, 1, resource.NewMilliQuantity(1200, resource.DecimalSI), pointer.Int32(30), false},
		// Scale up under threshold
		{10, 11, resource.NewMilliQuantity(1200, resource.DecimalSI), pointer.Int32(30), false},

		// Scale up equals threshold
		{10, 12, resource.NewMilliQuantity(1200, resource.DecimalSI), pointer.Int32(30), true},
		// Scale up over threshold
		{10, 13, resource.NewMilliQuantity(1200, resource.DecimalSI), pointer.Int32(30), true},

		// Scale down whatever
		{10, 9, resource.NewMilliQuantity(1200, resource.DecimalSI), pointer.Int32(30), false},
		{10, 2, resource.NewMilliQuantity(1200, resource.DecimalSI), pointer.Int32(30), false},
		{10, 0, resource.NewMilliQuantity(1200, resource.DecimalSI), pointer.Int32(30), false},
	} {
		shouldEnter := ShouldEnterPanicMode(testCase.desiredReplicas, testCase.currentReplicas, &wingv1.ReplicaAutoscalerStrategy{
			PanicThreshold:     testCase.panicThreshold,
			PanicWindowSeconds: testCase.panicWindowSeconds,
		})
		assert.Equal(t, testCase.shouldEnter, shouldEnter,
			"shouldEnterPanicMode(%d, %d, %v, %v)", testCase.currentReplicas, testCase.desiredReplicas, testCase.panicThreshold, testCase.panicWindowSeconds)
	}
}
