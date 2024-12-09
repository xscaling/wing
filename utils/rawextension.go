package utils

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/runtime"
)

func ExtractRawExtension(settings *runtime.RawExtension, v any) error {
	if settings == nil || settings.Raw == nil {
		return nil
	}
	return json.Unmarshal(settings.Raw, v)
}
