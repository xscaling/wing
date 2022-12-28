package utils

import (
	"context"
	"encoding/json"
	"time"

	wingv1 "github.com/xscaling/wing/api/v1"
	"github.com/xscaling/wing/utils/timerange"

	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
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

func PurgeUnusedReplicaPatches(client runtimeclient.Client, replicaAutoscaler *wingv1.ReplicaAutoscaler) error {
	patches, err := GetReplicaPatches(*replicaAutoscaler)
	if err != nil {
		return err
	}
	if patches == nil {
		return nil
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
		return nil
	}
	raw := replicaAutoscaler.DeepCopy()
	if len(newPatches) == 0 {
		delete(replicaAutoscaler.Annotations, wingv1.ReplicaPatchesAnnotation)
	} else {
		newPatchesString, err := json.Marshal(newPatches)
		if err != nil {
			return err
		}
		replicaAutoscaler.Annotations[wingv1.ReplicaPatchesAnnotation] = string(newPatchesString)
	}
	patch := runtimeclient.MergeFrom(raw.DeepCopy())
	raw.Annotations = make(map[string]string)
	for k, v := range replicaAutoscaler.Annotations {
		raw.Annotations[k] = v
	}
	return client.Patch(context.TODO(), raw, patch)
}
