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
	"github.com/xscaling/wing/utils"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ReplicaAutoscalerReconciler reconciles a ReplicaAutoscaler object
type ReplicaAutoscalerReconciler struct {
	runtimeclient.Client
	cache.Cache

	EventRecorder record.EventRecorder

	Config           ReplicaAutoscalerControllerConfig
	KubernetesConfig *rest.Config
	Scheme           *runtime.Scheme
	Engine           *engine.Engine
	DryRun           bool

	restMapper  meta.RESTMapper
	scaleClient scale.ScalesGetter
}

//+kubebuilder:rbac:groups=wing.xscaling.dev,resources=replicaautoscalers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=wing.xscaling.dev,resources=replicaautoscalers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=wing.xscaling.dev,resources=replicaautoscalers/finalizers,verbs=update
//+kubebuilder:rbac:groups=*,resources=*/scale,verbs=*
//+kubebuilder:rbac:groups="core",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="core",resources=events,verbs="*"
//+kubebuilder:rbac:groups="metrics.k8s.io",resources=*,verbs=get;list;watch

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
	observedAutoscaler := replicaAutoscaler.DeepCopy()

	if err := utils.PurgeUnusedReplicaPatches(replicaAutoscaler); err != nil {
		logger.Error(err, "Failed to purge unused replica patches")
	}

	requeueDelay := r.reconcile(logger, replicaAutoscaler)

	// Patch the autoscaler if needed
	if updateRequeueDelay, err := r.updateAutoscalerIfNeeded(ctx, observedAutoscaler, replicaAutoscaler); err != nil {
		logger.Error(err, "Failed to update autoscaler")
		return ctrl.Result{
			RequeueAfter: updateRequeueDelay,
		}, err
	}

	// If we didn't requeue here then in this case one request would be dropped
	// and RA would processed after 2 x resyncPeriod.
	return ctrl.Result{
		RequeueAfter: requeueDelay,
	}, nil
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

func (r *ReplicaAutoscalerReconciler) updateAutoscalerIfNeeded(ctx context.Context,
	observedAutoscaler, replicaAutoscaler *wingv1.ReplicaAutoscaler) (time.Duration, error) {
	logger := log.FromContext(ctx)

	// Check annotations is equal
	annotationsEqual := utils.DeepEqual(replicaAutoscaler.Annotations, observedAutoscaler.Annotations)
	statusEqual := utils.DeepEqual(replicaAutoscaler.Status, observedAutoscaler.Status)

	if annotationsEqual && statusEqual {
		logger.V(4).Info("Autoscaler annotations and status are equal, no update needed")
		return 0, nil
	}

	patch := runtimeclient.MergeFrom(observedAutoscaler.DeepCopy())
	observedAutoscaler.Annotations = make(map[string]string)
	for k, v := range replicaAutoscaler.Annotations {
		observedAutoscaler.Annotations[k] = v
	}
	observedAutoscaler.Status = replicaAutoscaler.Status

	if annotationsEqual && !statusEqual {
		// patch status only
		logger.V(4).Info("Patching status only")
		err := r.Client.Status().Patch(ctx, observedAutoscaler, patch)
		if err != nil {
			logger.Error(err, "Failed to update autoscaler status")
			return RequeueDelayOnErrorState, err
		}
	} else {
		// patch both annotations and status
		logger.V(4).Info("Patching both annotations and status")
		err := r.Client.Patch(ctx, observedAutoscaler, patch)
		if err != nil {
			logger.Error(err, "Failed to update autoscaler object")
			return RequeueDelayOnErrorState, err
		}
	}
	return 0, nil
}
