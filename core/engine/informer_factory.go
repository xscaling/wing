package engine

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
)

type InformerFactory struct {
	factory informers.SharedInformerFactory
}

func NewInformerFactory(clientSet kubernetes.Interface) *InformerFactory {
	f := &InformerFactory{
		factory: informers.NewSharedInformerFactory(clientSet, time.Minute),
	}
	return f
}

func (f *InformerFactory) Run(stopCh <-chan struct{}) {
	defer runtime.HandleCrash()

	// Add informer here
	f.factory.Core().V1().Pods().Informer()

	f.factory.Start(stopCh)

	syncs := f.factory.WaitForCacheSync(stopCh)
	for typ, syn := range syncs {
		if !syn {
			runtime.HandleError(fmt.Errorf("wait for cache %s sync timeout", typ.String()))
		}
	}
}

func (f InformerFactory) PodLister() listerscorev1.PodLister {
	return f.factory.Core().V1().Pods().Lister()
}
