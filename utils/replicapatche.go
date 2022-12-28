package utils

import (
	"encoding/json"
	"time"

	wingv1 "github.com/xscaling/wing/api/v1"
	"github.com/xscaling/wing/utils/timerange"
)

func GetReplicaPatches(replicaAutoscaler wingv1.ReplicaAutoscaler) (wingv1.ReplicaPatches, error) {
	if replicaAutoscaler.Annotations == nil {
		return nil, nil
	}
	rawString, ok := replicaAutoscaler.Annotations[wingv1.ReplicaPatchesAnnotation]
	if !ok {
		return nil, nil
	}
	var replicaPatches wingv1.ReplicaPatches
	err := json.Unmarshal([]byte(rawString), &replicaPatches)
	if err != nil {
		return nil, err
	}
	return replicaPatches, nil
}

func PurgeUnusedReplicaPatches(replicaAutoscaler *wingv1.ReplicaAutoscaler) (changed bool, err error) {
	patches, err := GetReplicaPatches(*replicaAutoscaler)
	if err != nil {
		return false, err
	}
	if patches == nil {
		return false, nil
	}
	var newPatches wingv1.ReplicaPatches
	for _, patch := range patches {
		timezone, err := time.LoadLocation(patch.Timezone)
		if err == nil {
			dateScheduler, err := timerange.NewDateScheduler(timezone, patch.Start, patch.End)
			if err == nil {
				expiredTime := dateScheduler.GetEndTime()
				if patch.RetentionSeconds != nil {
					expiredTime = expiredTime.Add(time.Duration(*patch.RetentionSeconds) * time.Second)
				}
				// Delete it
				if time.Now().After(expiredTime) {
					continue
				}
			}
		}
		newPatches = append(newPatches, patch)
	}
	if len(newPatches) == len(patches) {
		// nothing to update
		return false, nil
	}
	if len(newPatches) == 0 {
		delete(replicaAutoscaler.Annotations, wingv1.ReplicaPatchesAnnotation)
	} else {
		newPatchesString, err := json.Marshal(newPatches)
		if err != nil {
			return false, err
		}
		replicaAutoscaler.Annotations[wingv1.ReplicaPatchesAnnotation] = string(newPatchesString)
	}
	return true, nil
}
