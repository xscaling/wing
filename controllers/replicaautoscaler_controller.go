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
	"context"
	"time"

	wingv1 "github.com/xscaling/wing/api/v1"
	"github.com/xscaling/wing/core/engine"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/scale"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ReplicaAutoscalerReconciler reconciles a ReplicaAutoscaler object
type ReplicaAutoscalerReconciler struct {
	client.Client
	cache.Cache

	Config           ReplicaAutoscalerControllerConfig
	KubernetesConfig *rest.Config
	Scheme           *runtime.Scheme
	Engine           *engine.Engine

	restMapper  meta.RESTMapper
	scaleClient scale.ScalesGetter
}

//+kubebuilder:rbac:groups=wing.xscaling.dev,resources=replicaautoscalers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=wing.xscaling.dev,resources=replicaautoscalers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=wing.xscaling.dev,resources=replicaautoscalers/finalizers,verbs=update
//+kubebuilder:rbac:groups=*,resources=*/scale,verbs=*
//+kubebuilder:rbac:groups="core",resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *ReplicaAutoscalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(4).Info("Reconciling")
	replicaAutoscaler := &wingv1.ReplicaAutoscaler{}

	if err := r.Cache.Get(ctx, req.NamespacedName, replicaAutoscaler); err != nil {
		logger.Error(err, "Unable to get ReplicaAutoscaler")
		return ctrl.Result{}, err
	}
	requeue, err := r.reconcile(logger, replicaAutoscaler)
	if err != nil {
		logger.Error(err, "Failed to reconcile ReplicaAutoscaler")
		return ctrl.Result{}, err
	}
	// If we didn't requeue here then in this case one request would be dropped
	// and RA would processed after 2 x resyncPeriod.
	if requeue {
		return ctrl.Result{
			RequeueAfter: time.Second * 3,
		}, nil
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReplicaAutoscalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	logger := mgr.GetLogger()
	clientset, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		logger.Error(err, "Not able to create Discovery clientset")
		return err
	}
	scaleKindResolver := scale.NewDiscoveryScaleKindResolver(clientset)
	r.scaleClient = scale.New(
		clientset.RESTClient(), mgr.GetRESTMapper(),
		dynamic.LegacyAPIPathResolverFunc,
		scaleKindResolver,
	)
	r.restMapper = mgr.GetRESTMapper()
	logger.Info("Setting up controller with manager", "reconcileConcurrent", r.Config.Workers)
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: r.Config.Workers,
		}).
		For(&wingv1.ReplicaAutoscaler{}).
		Complete(r)
}
