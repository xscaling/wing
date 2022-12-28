package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionType string

const (
	ConditionReplicaPatched ConditionType = "ReplicaPatched"
	ConditionScaleLimited   ConditionType = "ScaleLimited"
	ConditionReady          ConditionType = "Ready"
)

type Conditions []Condition

type Condition struct {
	// Type of condition
	// +required
	Type ConditionType `json:"type" description:"type of status condition"`

	// Status of the condition, one of True, False, Unknown.
	// +required
	Status metav1.ConditionStatus `json:"status" description:"status of the condition, one of True, False, Unknown"`

	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty" description:"one-word CamelCase reason for the condition's last transition"`

	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty" description:"human-readable message indicating details about last transition"`

	// Last time the condition transitioned from one status to another.
	// +required
	LastTransitionTime metav1.Time `json:"lastTransitionTime" description:"last time the condition transitioned from one status to another"`
}

// InitializeConditions to the default -> Status: Unknown
func InitializeConditions(c *Conditions) {
	*c = append(*c, Condition{Type: ConditionReady, Status: metav1.ConditionUnknown})
}

func SetCondition(conditions Conditions, condition Condition) Conditions {
	if len(conditions) == 0 {
		InitializeConditions(&conditions)
	}
	found := false
	for i, cond := range conditions {
		if cond.Type == condition.Type {
			found = true
			if conditions[i].Status != condition.Status || conditions[i].Reason != condition.Reason {
				conditions[i].LastTransitionTime = metav1.Now()
			}
			conditions[i].Status = condition.Status
			conditions[i].Reason = condition.Reason
			conditions[i].Message = condition.Message
		}
	}
	if !found {
		condition.LastTransitionTime = metav1.Now()
		conditions = append(conditions, condition)
	}
	return conditions
}

func GetCondition(conditions Conditions, conditionType ConditionType) Condition {
	if len(conditions) == 0 {
		InitializeConditions(&conditions)
	}
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return conditions[i]
		}
	}
	return Condition{}
}

func DeleteCondition(conditions Conditions, conditionType ConditionType) Conditions {
	if len(conditions) == 0 {
		InitializeConditions(&conditions)
	}
	for i := range conditions {
		if conditions[i].Type == conditionType {
			conditions = append(conditions[:i], conditions[i+1:]...)
			return conditions
		}
	}
	return conditions
}
