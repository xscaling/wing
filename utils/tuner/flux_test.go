package tuner

import (
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestFluxTuner_GetRecommendation(t *testing.T) {
	tests := []struct {
		name            string
		currentReplicas int32
		desiredReplicas int32
		preference      interface{}
		history         []ReplicaSnapshot
		want            int32
	}{
		{
			name:            "scale up with no rules",
			currentReplicas: 1,
			desiredReplicas: 2,
			// failed on type cast
			preference: struct{}{},
			want:       2,
		},
		{
			name:            "scale up with no rules",
			currentReplicas: 1,
			desiredReplicas: 2,
			preference:      FluxPreference{},
			want:            2,
		},
		{
			name:            "scale down with no rules",
			currentReplicas: 10,
			desiredReplicas: 2,
			preference:      FluxPreference{},
			// limit by default rule set
			want: 5,
		},
		{
			name:            "scale up with replica count rule",
			currentReplicas: 1,
			desiredReplicas: 5,
			preference: FluxPreference{
				ScaleUpRuleSet: &FluxRuleSet{
					Strategy: RuleStrategyMax,
					Rules: []FluxRule{
						{
							Type:          RuleTypeReplicaCount,
							Value:         intstr.FromInt(2),
							PeriodSeconds: 60,
						},
					},
				},
			},
			want: 3,
		},
		{
			name:            "scale up with invalid replica count rule(loss flux control)",
			currentReplicas: 1,
			desiredReplicas: 5,
			preference: FluxPreference{
				ScaleUpRuleSet: &FluxRuleSet{
					Strategy: RuleStrategyMax,
					Rules: []FluxRule{
						{
							Type:          RuleTypeReplicaCount,
							Value:         intstr.FromString("200%"),
							PeriodSeconds: 60,
						},
					},
				},
			},
			want: 5,
		},
		{
			name:            "scale down with invalid replica count rule(loss flux control)",
			currentReplicas: 100,
			desiredReplicas: 1,
			preference: FluxPreference{
				ScaleUpRuleSet: &FluxRuleSet{
					Strategy: RuleStrategyMax,
					Rules: []FluxRule{
						{
							Type:          RuleTypeReplicaCount,
							Value:         intstr.FromString("200%"),
							PeriodSeconds: 60,
						},
					},
				},
			},
			want: 1,
		},
		{
			name:            "scale down with replica count rule",
			currentReplicas: 5,
			desiredReplicas: 1,
			preference: FluxPreference{
				ScaleDownRuleSet: &FluxRuleSet{
					Strategy: RuleStrategyMin,
					Rules: []FluxRule{
						{
							Type:          RuleTypeReplicaCount,
							Value:         intstr.FromInt(2),
							PeriodSeconds: 60,
						},
					},
				},
			},
			want: 3,
		},
		{
			name:            "scale up with replica percent rule",
			currentReplicas: 10,
			desiredReplicas: 20,
			preference: FluxPreference{
				ScaleUpRuleSet: &FluxRuleSet{
					Strategy: RuleStrategyMax,
					Rules: []FluxRule{
						{
							Type:          RuleTypeReplicaPercent,
							Value:         intstr.FromInt(50),
							PeriodSeconds: 60,
						},
					},
				},
			},
			want: 15,
		},
		{
			name:            "scale down with replica percent rule",
			currentReplicas: 20,
			desiredReplicas: 10,
			preference: FluxPreference{
				ScaleDownRuleSet: &FluxRuleSet{
					Strategy: RuleStrategyMin,
					Rules: []FluxRule{
						{
							Type:          RuleTypeReplicaPercent,
							Value:         intstr.FromInt(50),
							PeriodSeconds: 60,
						},
					},
				},
			},
			want: 10,
		},
		{
			name:            "scale up with multiple rules using max strategy",
			currentReplicas: 10,
			desiredReplicas: 30,
			preference: FluxPreference{
				ScaleUpRuleSet: &FluxRuleSet{
					Strategy: RuleStrategyMax,
					Rules: []FluxRule{
						{
							Type:          RuleTypeReplicaCount,
							Value:         intstr.FromInt(5),
							PeriodSeconds: 60,
						},
						{
							Type:          RuleTypeReplicaPercent,
							Value:         intstr.FromInt(50),
							PeriodSeconds: 60,
						},
					},
				},
			},
			want: 15,
		},
		{
			name:            "scale down with multiple rules using min strategy",
			currentReplicas: 30,
			desiredReplicas: 10,
			preference: FluxPreference{
				ScaleDownRuleSet: &FluxRuleSet{
					Strategy: RuleStrategyMin,
					Rules: []FluxRule{
						{
							Type:          RuleTypeReplicaCount,
							Value:         intstr.FromInt(5),
							PeriodSeconds: 60,
						},
						{
							Type:          RuleTypeReplicaPercent,
							Value:         intstr.FromInt(50),
							PeriodSeconds: 60,
						},
					},
				},
			},
			want: 15,
		},
		{
			name:            "scale up with history",
			currentReplicas: 10,
			desiredReplicas: 20,
			preference: FluxPreference{
				ScaleUpRuleSet: &FluxRuleSet{
					Strategy: RuleStrategyMax,
					Rules: []FluxRule{
						{
							Type:          RuleTypeReplicaCount,
							Value:         intstr.FromInt(5),
							PeriodSeconds: 60,
						},
					},
				},
			},
			history: []ReplicaSnapshot{
				{
					Timestamp: time.Now().Add(-30 * time.Second),
					Replicas:  8,
				},
				{
					Timestamp: time.Now().Add(-20 * time.Second),
					Replicas:  10,
				},
			},
			want: 13,
		},
		{
			name:            "scale down with history",
			currentReplicas: 20,
			desiredReplicas: 10,
			preference: FluxPreference{
				ScaleDownRuleSet: &FluxRuleSet{
					Strategy: RuleStrategyMin,
					Rules: []FluxRule{
						{
							Type:          RuleTypeReplicaCount,
							Value:         intstr.FromInt(5),
							PeriodSeconds: 60,
						},
					},
				},
			},
			history: []ReplicaSnapshot{
				{
					Timestamp: time.Now().Add(-70 * time.Second),
					Replicas:  22,
				},
				{
					Timestamp: time.Now().Add(-30 * time.Second),
					Replicas:  22,
				},
				{
					Timestamp: time.Now().Add(-20 * time.Second),
					Replicas:  20,
				},
			},
			want: 17,
		},
		{
			name:            "scale up with history and percent rule",
			currentReplicas: 10,
			desiredReplicas: 20,
			preference: FluxPreference{
				ScaleUpRuleSet: &FluxRuleSet{
					Strategy: RuleStrategyMin,
					Rules: []FluxRule{
						{
							Type:          RuleTypeReplicaCount,
							Value:         intstr.FromInt(5),
							PeriodSeconds: 60,
						},
						{
							Type:          RuleTypeReplicaPercent,
							Value:         intstr.FromInt(50),
							PeriodSeconds: 60,
						},
					},
				},
			},
			want: 15,
		},
		{
			name:            "hybrid scale with history",
			currentReplicas: 20,
			desiredReplicas: 10,
			preference: FluxPreference{
				ScaleDownRuleSet: &FluxRuleSet{
					Strategy: RuleStrategyMin,
					Rules: []FluxRule{
						{
							Type:          RuleTypeReplicaCount,
							Value:         intstr.FromInt(5),
							PeriodSeconds: 60,
						},
					},
				},
			},
			history: []ReplicaSnapshot{
				{
					Timestamp: time.Now().Add(-40 * time.Second),
					Replicas:  8,
				},
				{
					Timestamp: time.Now().Add(-30 * time.Second),
					Replicas:  22,
				},
				{
					Timestamp: time.Now().Add(-20 * time.Second),
					Replicas:  20,
				},
			},
			want: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := NewFluxTuner(NewDefaultFluxOptions())
			if tt.history != nil {
				for i, h := range tt.history {
					var prevReplicas int32
					if i > 0 {
						prevReplicas = tt.history[i-1].Replicas
					}
					fc.AcceptRecommendation("test", prevReplicas, h.Replicas)
				}
			}
			got := fc.GetRecommendation("test", tt.currentReplicas, tt.desiredReplicas, tt.preference)
			if got != tt.want {
				t.Errorf("FluxTuner.GetRecommendation() = %v, want %v", got, tt.want)
			}
			fc.AcceptRecommendation("test", tt.currentReplicas, got)
		})
	}
}
