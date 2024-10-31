package tuner

import (
	"math"
	"sync"
	"time"
)

type RuleType string

const (
	RuleTypeReplicaCount   RuleType = "ReplicaCount"
	RuleTypeReplicaPercent RuleType = "ReplicaPercent"
)

// MarshalJSON implements json.Marshaler
func (r RuleType) MarshalJSON() ([]byte, error) {
	return []byte(`"` + string(r) + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler
func (r *RuleType) UnmarshalJSON(data []byte) error {
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return nil
	}
	*r = RuleType(data[1 : len(data)-1])
	return nil
}

// MarshalYAML implements yaml.Marshaler
func (r RuleType) MarshalYAML() (interface{}, error) {
	return string(r), nil
}

// UnmarshalYAML implements yaml.Unmarshaler
func (r *RuleType) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	*r = RuleType(s)
	return nil
}

type RuleStrategy string

const (
	RuleStrategyMax RuleStrategy = "Max"
	RuleStrategyMin RuleStrategy = "Min"
)

// MarshalJSON implements json.Marshaler
func (r RuleStrategy) MarshalJSON() ([]byte, error) {
	return []byte(`"` + string(r) + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler
func (r *RuleStrategy) UnmarshalJSON(data []byte) error {
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return nil
	}
	*r = RuleStrategy(data[1 : len(data)-1])
	return nil
}

// MarshalYAML implements yaml.Marshaler
func (r RuleStrategy) MarshalYAML() (interface{}, error) {
	return string(r), nil
}

// UnmarshalYAML implements yaml.Unmarshaler
func (r *RuleStrategy) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	*r = RuleStrategy(s)
	return nil
}

type FluxPreference struct {
	ScaleUpRuleSet   *FluxRuleSet `json:"scaleUpRuleSet" yaml:"scaleUpRuleSet"`
	ScaleDownRuleSet *FluxRuleSet `json:"scaleDownRuleSet" yaml:"scaleDownRuleSet"`
}

var (
	DefaultScaleUpFluxRuleSet = &FluxRuleSet{
		Strategy: RuleStrategyMax,
		Rules: []FluxRule{
			{
				Type:          RuleTypeReplicaPercent,
				Value:         50,
				PeriodSeconds: 60,
			},
		},
	}
	DefaultScaleDownFluxRuleSet = &FluxRuleSet{
		Strategy: RuleStrategyMin,
		Rules: []FluxRule{
			{
				Type:          RuleTypeReplicaPercent,
				Value:         50,
				PeriodSeconds: 60,
			},
		},
	}
)

type FluxRuleSet struct {
	// The strategy to choose the rule to apply.
	Strategy RuleStrategy `json:"strategy" yaml:"strategy"`
	// Rules for various conditions.
	Rules []FluxRule `json:"rules" yaml:"rules"`
}

type FluxRule struct {
	// The type of rule to apply.
	Type RuleType `json:"type" yaml:"type"`
	// It should be positive.
	Value int32 `json:"value" yaml:"value"`
	// The period to apply the rule for.
	PeriodSeconds int `json:"periodSeconds" yaml:"periodSeconds"`
}

type FluxTuner struct {
	historicalScaleUpReplicaMemory   sync.Map
	historicalScaleDownReplicaMemory sync.Map
}

func NewFluxTuner() *FluxTuner {
	return &FluxTuner{
		historicalScaleUpReplicaMemory:   sync.Map{},
		historicalScaleDownReplicaMemory: sync.Map{},
	}
}

func (fc *FluxTuner) GetRecommendation(keyForAutoscaler string,
	currentReplicas int32, desiredReplicas int32, preference interface{}) int32 {
	fluxPreference, ok := preference.(FluxPreference)
	if !ok {
		fluxPreference = FluxPreference{}
	}

	// detect scale up or scale down
	var (
		ruleSet   *FluxRuleSet
		isScaleUp bool
	)
	if desiredReplicas > currentReplicas {
		ruleSet = fluxPreference.ScaleUpRuleSet
		if ruleSet == nil {
			ruleSet = DefaultScaleUpFluxRuleSet
		}
		isScaleUp = true
	} else {
		ruleSet = fluxPreference.ScaleDownRuleSet
		if ruleSet == nil {
			ruleSet = DefaultScaleDownFluxRuleSet
		}
	}

	// apply rules
	if isScaleUp {
		rm, ok := fc.historicalScaleUpReplicaMemory.Load(keyForAutoscaler)
		if !ok {
			rm = NewSimpleReplicaMemory(DefaultReplicaMemoryMaxSize, DefaultReplicaMemoryRetention)
			fc.historicalScaleUpReplicaMemory.Store(keyForAutoscaler, rm)
		}
		limit := fc.getScaleUpLimit(rm.(ReplicaMemory), currentReplicas, ruleSet)
		if desiredReplicas > limit {
			desiredReplicas = limit
		}
	} else {
		rm, ok := fc.historicalScaleDownReplicaMemory.Load(keyForAutoscaler)
		if !ok {
			rm = NewSimpleReplicaMemory(DefaultReplicaMemoryMaxSize, DefaultReplicaMemoryRetention)
			fc.historicalScaleDownReplicaMemory.Store(keyForAutoscaler, rm)
		}
		limit := fc.getScaleDownLimit(rm.(ReplicaMemory), currentReplicas, ruleSet)
		if desiredReplicas < limit {
			desiredReplicas = limit
		}
	}

	return desiredReplicas
}

func (fc *FluxTuner) AcceptRecommendation(keyForAutoscaler string, currentReplicas int32, desiredReplicas int32) {
	var rm ReplicaMemory
	if currentReplicas < desiredReplicas {
		recordedReplicaMemory, ok := fc.historicalScaleUpReplicaMemory.Load(keyForAutoscaler)
		if !ok {
			recordedReplicaMemory = NewSimpleReplicaMemory(DefaultReplicaMemoryMaxSize, DefaultReplicaMemoryRetention)
			fc.historicalScaleUpReplicaMemory.Store(keyForAutoscaler, recordedReplicaMemory)
		}
		rm = recordedReplicaMemory.(ReplicaMemory)
	} else {
		recordedReplicaMemory, ok := fc.historicalScaleDownReplicaMemory.Load(keyForAutoscaler)
		if !ok {
			recordedReplicaMemory = NewSimpleReplicaMemory(DefaultReplicaMemoryMaxSize, DefaultReplicaMemoryRetention)
			fc.historicalScaleDownReplicaMemory.Store(keyForAutoscaler, recordedReplicaMemory)
		}
		rm = recordedReplicaMemory.(ReplicaMemory)
	}
	rm.Add(ReplicaSnapshot{
		Timestamp: time.Now(),
		Replicas:  desiredReplicas,
	})
}

func (fc *FluxTuner) getScaleUpLimit(replicaMemory ReplicaMemory, currentReplicas int32, ruleSet *FluxRuleSet) int32 {
	limit := int32(math.MaxInt32)
	choosePolicy := min
	if ruleSet.Strategy == RuleStrategyMax {
		choosePolicy = max
		limit = int32(math.MinInt32)
	}
	for _, rule := range ruleSet.Rules {
		cutoff := time.Now().Add(-time.Duration(rule.PeriodSeconds) * time.Second)
		snapshotAfterCutoff := replicaMemory.GetFirstSnapshotAfter(cutoff)
		var replicasBase int32
		if snapshotAfterCutoff != nil {
			replicasBase = snapshotAfterCutoff.Replicas
		} else {
			replicasBase = currentReplicas
		}
		switch rule.Type {
		case RuleTypeReplicaCount:
			replicasBase += rule.Value
		case RuleTypeReplicaPercent:
			replicasBase += int32(math.Ceil(float64(replicasBase) * float64(rule.Value) / 100))
		}
		limit = choosePolicy(limit, replicasBase)
	}
	return limit
}

func (fc *FluxTuner) getScaleDownLimit(replicaMemory ReplicaMemory, currentReplicas int32, ruleSet *FluxRuleSet) int32 {
	limit := int32(math.MinInt32)
	choosePolicy := max
	if ruleSet.Strategy == RuleStrategyMin {
		choosePolicy = min
		limit = int32(math.MaxInt32)
	}
	for _, rule := range ruleSet.Rules {
		cutoff := time.Now().Add(-time.Duration(rule.PeriodSeconds) * time.Second)
		snapshotAfterCutoff := replicaMemory.GetFirstSnapshotAfter(cutoff)
		var replicasBase int32
		if snapshotAfterCutoff != nil {
			replicasBase = snapshotAfterCutoff.Replicas
		} else {
			replicasBase = currentReplicas
		}
		switch rule.Type {
		case RuleTypeReplicaCount:
			replicasBase -= rule.Value
		case RuleTypeReplicaPercent:
			replicasBase -= int32(math.Ceil(float64(replicasBase) * float64(rule.Value) / 100))
		}
		limit = choosePolicy(limit, replicasBase)
	}
	return limit
}
