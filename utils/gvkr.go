package utils

import (
	"reflect"
	"sync"

	wingv1 "github.com/xscaling/wing/api/v1"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	defaultVersion  = "v1"
	defaultGroup    = "apps"
	defaultKind     = "Deployment"
	defaultResource = "deployments"
)

var deepEqual = equality.Semantic.DeepEqual

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

// Checks if cluster-independent, user provided data in ObjectMeta and Spec in two given top
// level api objects are equivalent.
func RuntimeObjectMetaAndSpecEquivalent(a, b runtime.Object) bool {
	objectMetaA := reflect.ValueOf(a).Elem().FieldByName("ObjectMeta").Interface().(metav1.ObjectMeta)
	objectMetaB := reflect.ValueOf(b).Elem().FieldByName("ObjectMeta").Interface().(metav1.ObjectMeta)
	specA := reflect.ValueOf(a).Elem().FieldByName("Spec").Interface()
	specB := reflect.ValueOf(b).Elem().FieldByName("Spec").Interface()
	return ObjectMetaEquivalent(objectMetaA, objectMetaB) && deepEqual(specA, specB)
}

func UnstructuredObjectMetaAndSpecEquivalent(a, b *unstructured.Unstructured) (bool, error) {
	if a.GetName() != b.GetName() ||
		a.GetNamespace() != b.GetNamespace() ||
		!deepEqual(a.GetLabels(), b.GetLabels()) ||
		!deepEqual(a.GetAnnotations(), b.GetAnnotations()) {
		return false, nil
	}
	// Compare spec
	aSpec, _, err := unstructured.NestedFieldCopy(a.Object, "spec")
	if err != nil {
		return false, err
	}
	bSpec, _, err := unstructured.NestedFieldCopy(b.Object, "spec")
	if err != nil {
		return false, err
	}
	return deepEqual(aSpec, bSpec), nil
}

// Checks if cluster-independent, user provided data in two given ObjectMeta are equal. If in
// the future the ObjectMeta structure is expanded then any field that is not populated
// by the api server should be included here.
func ObjectMetaEquivalent(a, b metav1.ObjectMeta) bool {
	if a.Name != b.Name {
		return false
	}
	if a.Namespace != b.Namespace {
		return false
	}
	if !deepEqual(a.Labels, b.Labels) {
		return false
	}
	if !deepEqual(a.Annotations, b.Annotations) {
		return false
	}
	return true
}
