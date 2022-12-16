/*
Copyright 2022 xScaling.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"time"

	wingv1 "github.com/xscaling/wing/api/v1"

	"github.com/go-logr/logr"
)

// finalizer will do some recovery works
func (r *ReplicaAutoscalerReconciler) finalizeAutoscaler(logger logr.Logger, autoscaler *wingv1.ReplicaAutoscaler) time.Duration {
	return DefaultRequeueDelay
}
