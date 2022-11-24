package utils

import (
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func ClientOrDie(kubeConfig rest.Config, userAgent string) kubernetes.Interface {
	patchedConfig := rest.AddUserAgent(&kubeConfig, userAgent)
	return kubernetes.NewForConfigOrDie(patchedConfig)
}

func DiscoveryClientOrDie(kubeConfig rest.Config, userAgent string) discovery.DiscoveryInterface {
	patchedConfig := rest.AddUserAgent(&kubeConfig, userAgent)
	// Discovery makes a lot of requests infrequently.  This allows the burst to succeed and refill to happen
	// in just a few seconds.
	patchedConfig.Burst = 200
	patchedConfig.QPS = 20
	return kubernetes.NewForConfigOrDie(patchedConfig)
}
