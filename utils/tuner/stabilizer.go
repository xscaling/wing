package tuner

import (
	"sync"
	"time"
)

const (
	DefaultStabilizationWindow    = time.Second * 30
	DefaultReplicaMemoryMaxSize   = 100
	DefaultReplicaMemoryRetention = time.Hour
)

var DefaultStabilizerPreference = StabilizerPreference{
	ScaleUpStabilizationSeconds:   int(DefaultStabilizationWindow.Seconds()),
	ScaleDownStabilizationSeconds: int(DefaultStabilizationWindow.Seconds()),
}

type Stabilizer struct {
	historicalRecommendation sync.Map
}

func NewStabilizer() *Stabilizer {
	return &Stabilizer{}
}

type StabilizerPreference struct {
	ScaleUpStabilizationSeconds   int `json:"scaleUpStabilizationSeconds" yaml:"scaleUpStabilizationSeconds"`
	ScaleDownStabilizationSeconds int `json:"scaleDownStabilizationSeconds" yaml:"scaleDownStabilizationSeconds"`
}

func (s *Stabilizer) GetName() string {
	return "stabilizer"
}

func (s *Stabilizer) GetRecommendation(keyForAutoscaler string,
	currentReplicas int32, desiredReplicas int32, preference interface{}) int32 {
	maxRecommendation := desiredReplicas
	stabilizationWindow := DefaultStabilizationWindow
	stabilizerPreference, ok := preference.(StabilizerPreference)
	if !ok {
		stabilizerPreference = DefaultStabilizerPreference
	}

	if desiredReplicas > currentReplicas {
		stabilizationWindow = time.Duration(stabilizerPreference.ScaleUpStabilizationSeconds) * time.Second
	} else {
		stabilizationWindow = time.Duration(stabilizerPreference.ScaleDownStabilizationSeconds) * time.Second
	}
	if stabilizationWindow == 0 {
		stabilizationWindow = DefaultStabilizationWindow
	}
	cutoff := time.Now().Add(-stabilizationWindow)

	rm, ok := s.historicalRecommendation.Load(keyForAutoscaler)
	if !ok {
		rm = NewSimpleReplicaMemory(DefaultReplicaMemoryMaxSize, DefaultReplicaMemoryRetention)
		s.historicalRecommendation.Store(keyForAutoscaler, rm)
	}

	events := rm.(ReplicaMemory).GetMemorySince(cutoff)

	for _, rec := range events {
		if rec.Replicas > maxRecommendation {
			maxRecommendation = rec.Replicas
		}
	}

	rm.(ReplicaMemory).Add(ReplicaSnapshot{
		Timestamp: time.Now(),
		Replicas:  desiredReplicas,
	})
	return maxRecommendation
}

// Stabilizer always accepts the recommendation.
func (s *Stabilizer) AcceptRecommendation(keyForAutoscaler string, currentReplicas int32, desiredReplicas int32) {
}
