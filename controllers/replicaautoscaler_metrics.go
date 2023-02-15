/*
Copyright 2023 xScaling.

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
	"github.com/prometheus/client_golang/prometheus"
	runtimemetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	metricPluginElapsed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "plugin_elapsed",
		Help: "The time elapsed for the scaler/replicator plugin to run",
	}, []string{"namespace", "replicaautoscaler", "plugin", "kind"})
)

func init() {
	runtimemetrics.Registry.MustRegister(metricPluginElapsed)
}
