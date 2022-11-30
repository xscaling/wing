package utils

import (
	"sync"

	wingv1 "github.com/xscaling/wing/api/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	defaultVersion  = "v1"
	defaultGroup    = "apps"
	defaultKind     = "Deployment"
	defaultResource = "deployments"
)

// ParseGVKR returns GroupVersionKindResource for specified apiVersion (groupVersion) and Kind
func ParseGVKR(restMapper meta.RESTMapper, apiVersion string, kind string) (wingv1.GroupVersionKindResource, error) {
	var group, version, resource string

	// if apiVersion is not specified, we suppose the default one should be used
	if apiVersion == "" {
		group = defaultGroup
		version = defaultVersion
	} else {
		groupVersion, err := schema.ParseGroupVersion(apiVersion)
		if err != nil {
			return wingv1.GroupVersionKindResource{}, err
		}

		group = groupVersion.Group
		version = groupVersion.Version
	}

	// if kind is not specified, we suppose that default one should be used
	if kind == "" {
		kind = defaultKind
	}

	// get resource
	resource, err := getResource(restMapper, group, version, kind)
	if err != nil {
		return wingv1.GroupVersionKindResource{}, err
	}

	return wingv1.GroupVersionKindResource{
		Group:    group,
		Version:  version,
		Kind:     kind,
		Resource: resource,
	}, nil
}

func getResource(restMapper meta.RESTMapper, group string, version string, kind string) (string, error) {
	// tricky: return most frequently used resource first
	switch kind {
	case defaultKind:
		return defaultResource, nil
	case "StatefulSet":
		return "statefulsets", nil
	default:
		restmapping, err := restMapper.RESTMapping(schema.GroupKind{Group: group, Kind: kind}, version)
		if err == nil {
			return restmapping.Resource.GroupResource().Resource, nil
		}

		return "", err
	}
}

// A cache mapping "resource.group" to true or false if we know whether this resource is scalable.
var (
	isScalableCache   map[string]bool
	scalableCacheLock sync.RWMutex
)

func init() {
	// Prefill the cache with some known values for core resources in case of future parallelism to avoid stampeding herd on startup.
	isScalableCache = map[string]bool{
		"deployments.apps": true,
		"statefusets.apps": true,
	}
}

func GetGroupResourceKnownScalable(groupResource string) (scalable, ok bool) {
	scalableCacheLock.RLock()
	defer scalableCacheLock.RUnlock()
	scalable, ok = isScalableCache[groupResource]
	return
}

func SetGroupResourceKnownScalable(groupResource string, scalable bool) {
	scalableCacheLock.Lock()
	defer scalableCacheLock.Unlock()
	isScalableCache[groupResource] = scalable
}

func GetGVKRUnstructured(gvkr wingv1.GroupVersionKindResource) *unstructured.Unstructured {
	payload := &unstructured.Unstructured{}
	payload.SetGroupVersionKind(gvkr.GroupVersionKind())
	return payload
}
