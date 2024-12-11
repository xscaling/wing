package tuner

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/log"
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

// FluxOptions used for initialize flux tuner
type FluxOptions struct {
	DefaultPreference FluxPreference `json:"defaultPreference" yaml:"defaultPreference"`
	// Used for initialize replica memory, default is 2000
	ReplicaMemoryMaxSize int `json:"replicaMemoryMaxSize" yaml:"replicaMemoryMaxSize"`
	// default is 1 hour
	ReplicaMemoryRetention time.Duration `json:"replicaMemoryRetention" yaml:"replicaMemoryRetention"`

	// Used for getting the snapshot for cutoff time with jitter toleration
	MemoryCutoffJitterToleration time.Duration `json:"memoryCutoffJitterToleration" yaml:"memoryCutoffJitterToleration"`

	// Stabilization windows to prevent rapid fluctuations
	ScaleUpStabilizationWindow   time.Duration `json:"scaleUpStabilizationWindow" yaml:"scaleUpStabilizationWindow"`
	ScaleDownStabilizationWindow time.Duration `json:"scaleDownStabilizationWindow" yaml:"scaleDownStabilizationWindow"`
}

func NewDefaultFluxOptions() FluxOptions {
	return FluxOptions{
		DefaultPreference: FluxPreference{
			ScaleUpRuleSet:   DefaultScaleUpFluxRuleSet,
			ScaleDownRuleSet: DefaultScaleDownFluxRuleSet,
		},
		ReplicaMemoryMaxSize:   DefaultReplicaMemoryMaxSize,
		ReplicaMemoryRetention: DefaultReplicaMemoryRetention,
	}
}

func (o FluxOptions) ApplyDefaults() FluxOptions {
	if o.ReplicaMemoryMaxSize == 0 {
		o.ReplicaMemoryMaxSize = DefaultReplicaMemoryMaxSize
	}
	if o.ReplicaMemoryRetention == 0 {
		o.ReplicaMemoryRetention = DefaultReplicaMemoryRetention
	}
	if o.DefaultPreference.ScaleUpRuleSet == nil || len(o.DefaultPreference.ScaleUpRuleSet.Rules) == 0 {
		o.DefaultPreference.ScaleUpRuleSet = DefaultScaleUpFluxRuleSet
	}
	if o.DefaultPreference.ScaleDownRuleSet == nil || len(o.DefaultPreference.ScaleDownRuleSet.Rules) == 0 {
		o.DefaultPreference.ScaleDownRuleSet = DefaultScaleDownFluxRuleSet
	}
	if o.MemoryCutoffJitterToleration <= 0 {
		o.MemoryCutoffJitterToleration = DefaultMemoryCutoffJitterToleration
	}
	if o.ScaleUpStabilizationWindow <= 0 {
		o.ScaleUpStabilizationWindow = 3 * time.Minute
	}
	if o.ScaleDownStabilizationWindow <= 0 {
		o.ScaleDownStabilizationWindow = 5 * time.Minute
	}
	return o
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
				Value:         intstr.FromInt(50),
				PeriodSeconds: 60,
			},
		},
	}
	DefaultScaleDownFluxRuleSet = &FluxRuleSet{
		Strategy: RuleStrategyMin,
		Rules: []FluxRule{
			{
				Type:          RuleTypeReplicaPercent,
				Value:         intstr.FromInt(50),
				PeriodSeconds: 60,
			},
		},
	}
)

const (
	DefaultMemoryCutoffJitterToleration = time.Second * 10
)

type FluxRuleSet struct {
	// Stabilization windows to prevent rapid fluctuations
	StabilizationWindowSeconds *int32 `json:"stabilizationWindowSeconds" yaml:"stabilizationWindowSeconds"`
	// The strategy to choose the rule to apply.
	Strategy RuleStrategy `json:"strategy" yaml:"strategy"`
	// Rules for various conditions.
	Rules []FluxRule `json:"rules" yaml:"rules"`
}

type FluxRule struct {
	// The type of rule to apply.
	Type RuleType `json:"type" yaml:"type"`
	// It should be positive.
	Value intstr.IntOrString `json:"value" yaml:"value"`
	// The period to apply the rule for.
	PeriodSeconds int `json:"periodSeconds" yaml:"periodSeconds"`
}

type FluxTuner struct {
	options                          FluxOptions
	historicalScaleUpReplicaMemory   *sync.Map
	historicalScaleDownReplicaMemory *sync.Map
}

func NewFluxTuner(options FluxOptions) *FluxTuner {
	return &FluxTuner{
		// Apply default configs for rest of the fields
		options:                          options.ApplyDefaults(),
		historicalScaleUpReplicaMemory:   &sync.Map{},
		historicalScaleDownReplicaMemory: &sync.Map{},
	}
}

func (f *FluxTuner) GetName() string {
	return "flux"
}

func (f *FluxTuner) loadPreference(preference interface{}) FluxPreference {
	bytes, err := json.Marshal(preference)
	if err != nil {
		return f.options.DefaultPreference
	}
	var fluxPreference FluxPreference
	err = json.Unmarshal(bytes, &fluxPreference)
	if err != nil {
		return f.options.DefaultPreference
	}
	if fluxPreference.ScaleUpRuleSet == nil || len(fluxPreference.ScaleUpRuleSet.Rules) == 0 {
		fluxPreference.ScaleUpRuleSet = f.options.DefaultPreference.ScaleUpRuleSet
	}
	if fluxPreference.ScaleDownRuleSet == nil || len(fluxPreference.ScaleDownRuleSet.Rules) == 0 {
		fluxPreference.ScaleDownRuleSet = f.options.DefaultPreference.ScaleDownRuleSet
	}
	return fluxPreference
}

func (f *FluxTuner) GetRecommendation(keyForAutoscaler string,
	currentReplicas int32, desiredReplicas int32, preference interface{}) int32 {
	logger := log.FromContext(context.TODO()).WithValues(
		"tuner", f.GetName(),
		"keyForAutoscaler", keyForAutoscaler,
		"currentReplicas", currentReplicas,
		"desiredReplicas", desiredReplicas,
	)

	fluxPreference := f.loadPreference(preference)

	// apply rules
	if desiredReplicas > currentReplicas {
		rm, ok := f.historicalScaleUpReplicaMemory.Load(keyForAutoscaler)
		if !ok {
			logger.V(4).Info("Initialize scale up replica memory")
			rm = f.newReplicaMemory()
			f.historicalScaleUpReplicaMemory.Store(keyForAutoscaler, rm)
		}

		// 应用 stabilization window
		stabilizationWindow := f.options.ScaleUpStabilizationWindow
		if w := fluxPreference.ScaleUpRuleSet.StabilizationWindowSeconds; w != nil {
			stabilizationWindow = time.Second * time.Duration(*w)
		}
		cutoff := time.Now().Add(-stabilizationWindow)
		snapshots := rm.(ReplicaMemory).GetMemorySince(cutoff, f.options.MemoryCutoffJitterToleration)
		if len(snapshots) > 0 {
			// 在窗口期内取最小值以避免过度伸缩
			stableReplicas := snapshots[0].Replicas
			for _, snapshot := range snapshots {
				stableReplicas = min(stableReplicas, snapshot.Replicas)
			}
			desiredReplicas = min(desiredReplicas, stableReplicas)
		}

		limit := f.getScaleUpLimit(logger, rm.(ReplicaMemory), currentReplicas, fluxPreference.ScaleUpRuleSet)
		if limit != nil && desiredReplicas > *limit {
			logger.V(2).Info("Scale up limit reached", "limit", limit)
			desiredReplicas = *limit
		}
	} else if desiredReplicas < currentReplicas {
		rm, ok := f.historicalScaleDownReplicaMemory.Load(keyForAutoscaler)
		if !ok {
			logger.V(4).Info("Initialize scale down replica memory")
			rm = f.newReplicaMemory()
			f.historicalScaleDownReplicaMemory.Store(keyForAutoscaler, rm)
		}

		// 应用 stabilization window
		stabilizationWindow := f.options.ScaleDownStabilizationWindow
		if w := fluxPreference.ScaleDownRuleSet.StabilizationWindowSeconds; w != nil {
			stabilizationWindow = time.Second * time.Duration(*w)
		}
		cutoff := time.Now().Add(-stabilizationWindow)
		snapshots := rm.(ReplicaMemory).GetMemorySince(cutoff, f.options.MemoryCutoffJitterToleration)
		if len(snapshots) > 0 {
			// 在窗口期内取最大值以避免过度收缩
			stableReplicas := snapshots[0].Replicas
			for _, snapshot := range snapshots {
				stableReplicas = max(stableReplicas, snapshot.Replicas)
			}
			desiredReplicas = max(desiredReplicas, stableReplicas)
		}

		limit := f.getScaleDownLimit(logger, rm.(ReplicaMemory), currentReplicas, fluxPreference.ScaleDownRuleSet)
		if limit != nil && desiredReplicas < *limit {
			logger.V(2).Info("Scale down limit reached", "limit", limit)
			desiredReplicas = *limit
		}
	}

	logger.V(2).Info("Recommendation", "desiredReplicas", desiredReplicas)
	return desiredReplicas
}

func (f *FluxTuner) newReplicaMemory() ReplicaMemory {
	return NewSimpleReplicaMemory(f.options.ReplicaMemoryMaxSize, f.options.ReplicaMemoryRetention)
}

func (f *FluxTuner) AcceptRecommendation(keyForAutoscaler string, currentReplicas int32, desiredReplicas int32) {
	snapshot := ReplicaSnapshot{
		Timestamp: time.Now(),
		Replicas:  desiredReplicas,
	}

	if currentReplicas < desiredReplicas {
		f.addSnapshot(f.historicalScaleUpReplicaMemory, keyForAutoscaler, snapshot)
	} else if currentReplicas > desiredReplicas {
		f.addSnapshot(f.historicalScaleDownReplicaMemory, keyForAutoscaler, snapshot)
	} else {
		// Add both memory for holding the same replicas, as it should be retain for later scale up or down decision
		f.addSnapshot(f.historicalScaleUpReplicaMemory, keyForAutoscaler, snapshot)
		f.addSnapshot(f.historicalScaleDownReplicaMemory, keyForAutoscaler, snapshot)
	}
}

func (f *FluxTuner) addSnapshot(memory *sync.Map, key string, snapshot ReplicaSnapshot) {
	recordedMemory, ok := memory.Load(key)
	if !ok {
		recordedMemory = f.newReplicaMemory()
		memory.Store(key, recordedMemory)
	}
	recordedMemory.(ReplicaMemory).Add(snapshot)
}

func (f *FluxTuner) getScaleUpLimit(logger logr.Logger, replicaMemory ReplicaMemory, currentReplicas int32, ruleSet *FluxRuleSet) *int32 {
	limit := int32(math.MaxInt32)
	choosePolicy := min
	if ruleSet.Strategy == RuleStrategyMax {
		choosePolicy = max
		limit = int32(math.MinInt32)
	}
	gotEffectiveLimit := false
	for _, rule := range ruleSet.Rules {
		ruleValue, err := intstr.GetScaledValueFromIntOrPercent(&rule.Value, 100, true)
		if err != nil {
			logger.Info("Invalid rule value for calculating scale up limit", "rule", rule.Value.String())
			continue
		}
		cutoff := time.Now().Add(-time.Duration(rule.PeriodSeconds) * time.Second)
		snapshotAfterCutoff := replicaMemory.GetFirstSnapshotAfter(cutoff, f.options.MemoryCutoffJitterToleration)
		var replicasBase int32
		if snapshotAfterCutoff != nil {
			replicasBase = snapshotAfterCutoff.Replicas
		} else {
			replicasBase = currentReplicas
		}
		switch rule.Type {
		case RuleTypeReplicaCount:
			replicasBase += int32(ruleValue)
		case RuleTypeReplicaPercent:
			replicasBase += int32(math.Ceil(float64(replicasBase) * float64(ruleValue) / 100))
		}
		gotEffectiveLimit = true
		limit = choosePolicy(limit, replicasBase)
	}
	if !gotEffectiveLimit {
		logger.Error(fmt.Errorf("no effective rule for calculating scale up limit"), "Unable to get scale up limit")
		return nil
	}
	return &limit
}

func (f *FluxTuner) getScaleDownLimit(logger logr.Logger, replicaMemory ReplicaMemory, currentReplicas int32, ruleSet *FluxRuleSet) *int32 {
	limit := int32(math.MinInt32)
	choosePolicy := max
	if ruleSet.Strategy == RuleStrategyMin {
		choosePolicy = min
		limit = int32(math.MaxInt32)
	}
	gotEffectiveLimit := false
	for _, rule := range ruleSet.Rules {
		ruleValue, err := intstr.GetScaledValueFromIntOrPercent(&rule.Value, 100, true)
		if err != nil {
			logger.Info("Invalid rule value for calculating scale down limit", "rule", rule.Value.String())
			continue
		}
		cutoff := time.Now().Add(-time.Duration(rule.PeriodSeconds) * time.Second)
		snapshotAfterCutoff := replicaMemory.GetFirstSnapshotAfter(cutoff, f.options.MemoryCutoffJitterToleration)
		var replicasBase int32
		if snapshotAfterCutoff != nil {
			replicasBase = snapshotAfterCutoff.Replicas
		} else {
			replicasBase = currentReplicas
		}
		switch rule.Type {
		case RuleTypeReplicaCount:
			replicasBase -= int32(ruleValue)
		case RuleTypeReplicaPercent:
			replicasBase -= int32(math.Ceil(float64(replicasBase) * float64(rule.Value.IntValue()) / 100))
		}
		gotEffectiveLimit = true
		limit = choosePolicy(limit, replicasBase)
	}
	if !gotEffectiveLimit {
		logger.Error(fmt.Errorf("no effective rule for calculating scale down limit"), "Unable to get scale down limit")
		return nil
	}
	return &limit
}
