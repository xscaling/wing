package utils

import (
	"reflect"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

var DeepEqual = equality.Semantic.DeepEqual

// Checks if cluster-independent, user provided data in ObjectMeta and Spec in two given top
// level api objects are equivalent.
func RuntimeObjectMetaAndSpecEquivalent(a, b runtime.Object) bool {
	objectMetaA := reflect.ValueOf(a).Elem().FieldByName("ObjectMeta").Interface().(metav1.ObjectMeta)
	objectMetaB := reflect.ValueOf(b).Elem().FieldByName("ObjectMeta").Interface().(metav1.ObjectMeta)
	specA := reflect.ValueOf(a).Elem().FieldByName("Spec").Interface()
	specB := reflect.ValueOf(b).Elem().FieldByName("Spec").Interface()
	return ObjectMetaEquivalent(objectMetaA, objectMetaB) && DeepEqual(specA, specB)
}

func UnstructuredObjectMetaAndSpecEquivalent(a, b *unstructured.Unstructured) (bool, error) {
	if a.GetName() != b.GetName() ||
		a.GetNamespace() != b.GetNamespace() ||
		!DeepEqual(a.GetLabels(), b.GetLabels()) ||
		!DeepEqual(a.GetAnnotations(), b.GetAnnotations()) {
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
	return DeepEqual(aSpec, bSpec), nil
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
	if !DeepEqual(a.Labels, b.Labels) {
		return false
	}
	if !DeepEqual(a.Annotations, b.Annotations) {
		return false
	}
	return true
}
