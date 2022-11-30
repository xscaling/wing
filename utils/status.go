package utils

import (
	wingv1 "github.com/xscaling/wing/api/v1"
)

func SetTargetStatus(status *wingv1.ReplicaAutoscalerStatus, targetStatus wingv1.TargetStatus) {
	for i := range status.Targets {
		if status.Targets[i].Target == targetStatus.Target {
			status.Targets[i] = targetStatus
			return
		}
	}
	status.Targets = append(status.Targets, targetStatus)
}
